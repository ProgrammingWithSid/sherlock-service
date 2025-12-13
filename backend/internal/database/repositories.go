package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sherlock/service/internal/types"
)

func (db *DB) CreateRepository(repo *types.Repository) error {
	if repo.ID == "" {
		repo.ID = uuid.New().String()
	}
	if repo.CreatedAt.IsZero() {
		repo.CreatedAt = time.Now()
	}

	query := `
		INSERT INTO repositories (id, org_id, platform, external_id, name, full_name,
		                         is_private, is_active, config, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := db.conn.Exec(query,
		repo.ID, repo.OrgID, repo.Platform, repo.ExternalID,
		repo.Name, repo.FullName, repo.IsPrivate, repo.IsActive,
		repo.Config, repo.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create repository: %w", err)
	}

	return nil
}

func (db *DB) GetRepositoryByID(id string) (*types.Repository, error) {
	query := `
		SELECT id, org_id, platform, external_id, name, full_name, is_private,
		       is_active, config, created_at
		FROM repositories
		WHERE id = $1
	`

	repo := &types.Repository{}
	err := db.conn.QueryRow(query, id).Scan(
		&repo.ID, &repo.OrgID, &repo.Platform, &repo.ExternalID,
		&repo.Name, &repo.FullName, &repo.IsPrivate, &repo.IsActive,
		&repo.Config, &repo.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("repository not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	return repo, nil
}

func (db *DB) GetRepositoriesByOrgID(orgID string) ([]*types.Repository, error) {
	query := `
		SELECT id, org_id, platform, external_id, name, full_name, is_private,
		       is_active, config, created_at
		FROM repositories
		WHERE org_id = $1
		ORDER BY created_at DESC
	`

	rows, err := db.conn.Query(query, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to query repositories: %w", err)
	}
	defer rows.Close()

	repos := []*types.Repository{}
	for rows.Next() {
		repo := &types.Repository{}
		err := rows.Scan(
			&repo.ID, &repo.OrgID, &repo.Platform, &repo.ExternalID,
			&repo.Name, &repo.FullName, &repo.IsPrivate, &repo.IsActive,
			&repo.Config, &repo.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan repository: %w", err)
		}
		repos = append(repos, repo)
	}

	return repos, nil
}

func (db *DB) UpdateRepositoryConfig(id string, config string) error {
	query := `
		UPDATE repositories
		SET config = $1
		WHERE id = $2
	`

	_, err := db.conn.Exec(query, config, id)
	if err != nil {
		return fmt.Errorf("failed to update repository config: %w", err)
	}

	return nil
}

func (db *DB) SetRepositoryActive(id string, isActive bool) error {
	query := `
		UPDATE repositories
		SET is_active = $1
		WHERE id = $2
	`

	_, err := db.conn.Exec(query, isActive, id)
	if err != nil {
		return fmt.Errorf("failed to set repository active: %w", err)
	}

	return nil
}

func (db *DB) GetRepositoryByFullName(fullName string) (*types.Repository, error) {
	query := `
		SELECT id, org_id, platform, external_id, name, full_name, is_private,
		       is_active, config, created_at
		FROM repositories
		WHERE full_name = $1
		LIMIT 1
	`

	repo := &types.Repository{}
	err := db.conn.QueryRow(query, fullName).Scan(
		&repo.ID, &repo.OrgID, &repo.Platform, &repo.ExternalID,
		&repo.Name, &repo.FullName, &repo.IsPrivate, &repo.IsActive,
		&repo.Config, &repo.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("repository not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	return repo, nil
}
