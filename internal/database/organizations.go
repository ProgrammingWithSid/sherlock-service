package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sherlock/service/internal/types"
)

func (db *DB) CreateOrganization(name string, slug string) (*types.Organization, error) {
	id := uuid.New().String()
	now := time.Now()

	query := `
		INSERT INTO organizations (id, name, slug, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, name, slug, plan, stripe_customer_id, stripe_subscription_id,
		          plan_activated_at, created_at, updated_at
	`

	org := &types.Organization{}
	err := db.conn.QueryRow(query, id, name, slug, now, now).Scan(
		&org.ID, &org.Name, &org.Slug, &org.Plan,
		&org.StripeCustomerID, &org.StripeSubscriptionID,
		&org.PlanActivatedAt, &org.CreatedAt, &org.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create organization: %w", err)
	}

	return org, nil
}

func (db *DB) GetOrganizationByID(id string) (*types.Organization, error) {
	query := `
		SELECT id, name, slug, plan, stripe_customer_id, stripe_subscription_id,
		       plan_activated_at, created_at, updated_at
		FROM organizations
		WHERE id = $1
	`

	org := &types.Organization{}
	err := db.conn.QueryRow(query, id).Scan(
		&org.ID, &org.Name, &org.Slug, &org.Plan,
		&org.StripeCustomerID, &org.StripeSubscriptionID,
		&org.PlanActivatedAt, &org.CreatedAt, &org.UpdatedAt,
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
		       plan_activated_at, created_at, updated_at
		FROM organizations
		WHERE slug = $1
	`

	org := &types.Organization{}
	err := db.conn.QueryRow(query, slug).Scan(
		&org.ID, &org.Name, &org.Slug, &org.Plan,
		&org.StripeCustomerID, &org.StripeSubscriptionID,
		&org.PlanActivatedAt, &org.CreatedAt, &org.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("organization not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}

	return org, nil
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
