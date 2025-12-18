package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sherlock/service/internal/types"
)

func (db *DB) CreateOrganization(name string, slug string) (*types.Organization, error) {
	return db.CreateOrganizationWithClaimToken(name, slug, true)
}

// CreateOrganizationWithClaimToken creates an organization optionally with a claim token
func (db *DB) CreateOrganizationWithClaimToken(name string, slug string, generateToken bool) (*types.Organization, error) {
	id := uuid.New().String()
	now := time.Now()
	
	var claimToken *string
	var claimTokenExpires *time.Time
	
	if generateToken {
		// Generate secure random token (32 bytes = 64 hex characters)
		token := uuid.New().String() + uuid.New().String() // 64 character token
		claimToken = &token
		expires := now.Add(7 * 24 * time.Hour) // Token expires in 7 days
		claimTokenExpires = &expires
	}

	query := `
		INSERT INTO organizations (id, name, slug, global_rules, claim_token, claim_token_expires, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, name, slug, plan, stripe_customer_id, stripe_subscription_id,
		          plan_activated_at, global_rules, claim_token, claim_token_expires, created_at, updated_at
	`

	org := &types.Organization{}
	err := db.conn.QueryRow(query, id, name, slug, "[]", claimToken, claimTokenExpires, now, now).Scan(
		&org.ID, &org.Name, &org.Slug, &org.Plan,
		&org.StripeCustomerID, &org.StripeSubscriptionID,
		&org.PlanActivatedAt, &org.GlobalRules, &org.ClaimToken, &org.ClaimTokenExpires,
		&org.CreatedAt, &org.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create organization: %w", err)
	}

	return org, nil
}

func (db *DB) GetOrganizationByID(id string) (*types.Organization, error) {
	query := `
		SELECT id, name, slug, plan, stripe_customer_id, stripe_subscription_id,
		       plan_activated_at, global_rules, claim_token, claim_token_expires, created_at, updated_at
		FROM organizations
		WHERE id = $1
	`

	org := &types.Organization{}
	err := db.conn.QueryRow(query, id).Scan(
		&org.ID, &org.Name, &org.Slug, &org.Plan,
		&org.StripeCustomerID, &org.StripeSubscriptionID,
		&org.PlanActivatedAt, &org.GlobalRules, &org.ClaimToken, &org.ClaimTokenExpires,
		&org.CreatedAt, &org.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("organization not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}

	return org, nil
}

func (db *DB) GetOrganizationBySlug(slug string) (*types.Organization, error) {
	query := `
		SELECT id, name, slug, plan, stripe_customer_id, stripe_subscription_id,
		       plan_activated_at, global_rules, claim_token, claim_token_expires, created_at, updated_at
		FROM organizations
		WHERE slug = $1
	`

	org := &types.Organization{}
	err := db.conn.QueryRow(query, slug).Scan(
		&org.ID, &org.Name, &org.Slug, &org.Plan,
		&org.StripeCustomerID, &org.StripeSubscriptionID,
		&org.PlanActivatedAt, &org.GlobalRules, &org.ClaimToken, &org.ClaimTokenExpires,
		&org.CreatedAt, &org.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("organization not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}

	return org, nil
}

// ValidateClaimToken validates a claim token and returns the organization if valid
func (db *DB) ValidateClaimToken(token string) (*types.Organization, error) {
	query := `
		SELECT id, name, slug, plan, stripe_customer_id, stripe_subscription_id,
		       plan_activated_at, global_rules, claim_token, claim_token_expires, created_at, updated_at
		FROM organizations
		WHERE claim_token = $1 AND claim_token_expires > NOW()
	`

	org := &types.Organization{}
	err := db.conn.QueryRow(query, token).Scan(
		&org.ID, &org.Name, &org.Slug, &org.Plan,
		&org.StripeCustomerID, &org.StripeSubscriptionID,
		&org.PlanActivatedAt, &org.GlobalRules, &org.ClaimToken, &org.ClaimTokenExpires,
		&org.CreatedAt, &org.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invalid or expired claim token")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to validate claim token: %w", err)
	}

	return org, nil
}

// ClearClaimToken clears the claim token after successful organization claim
func (db *DB) ClearClaimToken(orgID string) error {
	query := `
		UPDATE organizations
		SET claim_token = NULL, claim_token_expires = NULL, updated_at = NOW()
		WHERE id = $1
	`

	_, err := db.conn.Exec(query, orgID)
	if err != nil {
		return fmt.Errorf("failed to clear claim token: %w", err)
	}

	return nil
}

func (db *DB) UpdateOrganizationPlan(id string, plan types.Plan, subscriptionID *string) error {
	now := time.Now()
	query := `
		UPDATE organizations
		SET plan = $1, stripe_subscription_id = $2, plan_activated_at = $3, updated_at = $4
		WHERE id = $5
	`

	_, err := db.conn.Exec(query, plan, subscriptionID, now, now, id)
	if err != nil {
		return fmt.Errorf("failed to update organization plan: %w", err)
	}

	return nil
}

func (db *DB) GetMonthlyReviewCount(orgID string) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM reviews
		WHERE org_id = $1
		AND created_at >= DATE_TRUNC('month', CURRENT_DATE)
	`

	var count int
	err := db.conn.QueryRow(query, orgID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get monthly review count: %w", err)
	}

	return count, nil
}

func (db *DB) GetRepoCount(orgID string) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM repositories
		WHERE org_id = $1 AND is_active = true
	`

	var count int
	err := db.conn.QueryRow(query, orgID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get repo count: %w", err)
	}

	return count, nil
}

// UpdateOrganizationGlobalRules updates global rules for an organization
func (db *DB) UpdateOrganizationGlobalRules(orgID string, rules []string) error {
	// Convert rules to JSON
	rulesJSON, err := json.Marshal(rules)
	if err != nil {
		return fmt.Errorf("failed to marshal rules: %w", err)
	}

	query := `
		UPDATE organizations
		SET global_rules = $1, updated_at = NOW()
		WHERE id = $2
	`

	_, err = db.conn.Exec(query, string(rulesJSON), orgID)
	if err != nil {
		return fmt.Errorf("failed to update global rules: %w", err)
	}

	return nil
}

// ListOrganizationsByToken returns all organizations that have installations with the given token
func (db *DB) ListOrganizationsByToken(token string) ([]*types.Organization, error) {
	query := `
		SELECT DISTINCT o.id, o.name, o.slug, o.plan, o.stripe_customer_id, o.stripe_subscription_id,
		       o.plan_activated_at, o.global_rules, o.claim_token, o.claim_token_expires, o.created_at, o.updated_at
		FROM organizations o
		INNER JOIN github_installations gi ON o.id = gi.org_id
		WHERE gi.token = $1
		ORDER BY o.created_at DESC
	`

	rows, err := db.conn.Query(query, token)
	if err != nil {
		return nil, fmt.Errorf("failed to list organizations: %w", err)
	}
	defer rows.Close()

	var orgs []*types.Organization
	for rows.Next() {
		org := &types.Organization{}
		err := rows.Scan(
			&org.ID, &org.Name, &org.Slug, &org.Plan,
			&org.StripeCustomerID, &org.StripeSubscriptionID,
			&org.PlanActivatedAt, &org.GlobalRules, &org.ClaimToken, &org.ClaimTokenExpires,
			&org.CreatedAt, &org.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan organization: %w", err)
		}
		orgs = append(orgs, org)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating organizations: %w", err)
	}

	return orgs, nil
}

// ListAllOrganizations returns all organizations that have installations (connected accounts)
func (db *DB) ListAllOrganizations() ([]*types.Organization, error) {
	query := `
		SELECT DISTINCT o.id, o.name, o.slug, o.plan, o.stripe_customer_id, o.stripe_subscription_id,
		       o.plan_activated_at, o.global_rules, o.claim_token, o.claim_token_expires, o.created_at, o.updated_at
		FROM organizations o
		INNER JOIN github_installations gi ON o.id = gi.org_id
		ORDER BY o.created_at DESC
	`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list organizations: %w", err)
	}
	defer rows.Close()

	var orgs []*types.Organization
	for rows.Next() {
		org := &types.Organization{}
		err := rows.Scan(
			&org.ID, &org.Name, &org.Slug, &org.Plan,
			&org.StripeCustomerID, &org.StripeSubscriptionID,
			&org.PlanActivatedAt, &org.GlobalRules, &org.ClaimToken, &org.ClaimTokenExpires,
			&org.CreatedAt, &org.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan organization: %w", err)
		}
		orgs = append(orgs, org)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating organizations: %w", err)
	}

	return orgs, nil
}

// GetOrganizationsByUserID returns organizations for a specific user
// If user has org_id, returns that organization
// If user is super admin, returns all organizations
func (db *DB) GetOrganizationsByUserID(userID string, userRole string, userOrgID *string) ([]*types.Organization, error) {
	// Super admins can see all organizations
	if userRole == string(types.RoleSuperAdmin) {
		return db.ListAllOrganizations()
	}

	// Regular users only see their own organization
	if userOrgID == nil {
		return []*types.Organization{}, nil
	}

	org, err := db.GetOrganizationByID(*userOrgID)
	if err != nil {
		return []*types.Organization{}, nil // Return empty if not found
	}

	return []*types.Organization{org}, nil
}
