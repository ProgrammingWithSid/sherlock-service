package database

import (
	"database/sql"
	"fmt"
	"time"
)

// GitHubInstallation represents a GitHub App installation
type GitHubInstallation struct {
	ID           string    `db:"id"`
	OrgID        string    `db:"org_id"`
	InstallationID int64   `db:"installation_id"`
	Token        string    `db:"token"`
	TokenExpires *time.Time `db:"token_expires"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

func (db *DB) CreateInstallationTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS github_installations (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
			installation_id BIGINT NOT NULL UNIQUE,
			token TEXT,
			token_expires TIMESTAMP,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)
	`

	_, err := db.conn.Exec(query)
	return err
}

func (db *DB) CreateOrUpdateInstallation(orgID string, installationID int64, token string, expiresAt *time.Time) error {
	now := time.Now()
	query := `
		INSERT INTO github_installations (org_id, installation_id, token, token_expires, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (installation_id)
		DO UPDATE SET
			org_id = EXCLUDED.org_id,
			token = EXCLUDED.token,
			token_expires = EXCLUDED.token_expires,
			updated_at = EXCLUDED.updated_at
	`

	_, err := db.conn.Exec(query, orgID, installationID, token, expiresAt, now, now)
	if err != nil {
		return fmt.Errorf("failed to create/update installation: %w", err)
	}

	return nil
}

func (db *DB) UpdateInstallationToken(installationID int64, token string, expiresAt *time.Time) error {
	now := time.Now()
	query := `
		UPDATE github_installations
		SET token = $1, token_expires = $2, updated_at = $3
		WHERE installation_id = $4
	`

	_, err := db.conn.Exec(query, token, expiresAt, now, installationID)
	if err != nil {
		return fmt.Errorf("failed to update installation token: %w", err)
	}

	return nil
}

func (db *DB) GetInstallationByID(installationID int64) (*GitHubInstallation, error) {
	query := `
		SELECT id, org_id, installation_id, token, token_expires, created_at, updated_at
		FROM github_installations
		WHERE installation_id = $1
	`

	inst := &GitHubInstallation{}
	var tokenExpires sql.NullTime
	err := db.conn.QueryRow(query, installationID).Scan(
		&inst.ID, &inst.OrgID, &inst.InstallationID, &inst.Token,
		&tokenExpires, &inst.CreatedAt, &inst.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("installation not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get installation: %w", err)
	}

	if tokenExpires.Valid {
		inst.TokenExpires = &tokenExpires.Time
	}

	return inst, nil
}

func (db *DB) GetInstallationByOrgID(orgID string) (*GitHubInstallation, error) {
	query := `
		SELECT id, org_id, installation_id, token, token_expires, created_at, updated_at
		FROM github_installations
		WHERE org_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	inst := &GitHubInstallation{}
	var tokenExpires sql.NullTime
	err := db.conn.QueryRow(query, orgID).Scan(
		&inst.ID, &inst.OrgID, &inst.InstallationID, &inst.Token,
		&tokenExpires, &inst.CreatedAt, &inst.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("installation not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get installation: %w", err)
	}

	if tokenExpires.Valid {
		inst.TokenExpires = &tokenExpires.Time
	}

	return inst, nil
}

func (db *DB) DeleteInstallation(installationID int64) error {
	query := `DELETE FROM github_installations WHERE installation_id = $1`
	_, err := db.conn.Exec(query, installationID)
	if err != nil {
		return fmt.Errorf("failed to delete installation: %w", err)
	}
	return nil
}
