package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sherlock/service/internal/types"
)

func (db *DB) CreateReview(review *types.Review) error {
	if review.ID == "" {
		review.ID = uuid.New().String()
	}
	if review.CreatedAt.IsZero() {
		review.CreatedAt = time.Now()
	}

	query := `
		INSERT INTO reviews (id, org_id, repo_id, pr_number, head_sha, status,
		                    result, comments_posted, duration_ms, ai_provider, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := db.conn.Exec(query,
		review.ID, review.OrgID, review.RepoID, review.PRNumber,
		review.HeadSHA, review.Status, review.Result, review.CommentsPosted,
		review.DurationMs, review.AIProvider, review.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create review: %w", err)
	}

	return nil
}

func (db *DB) GetReviewByPRAndSHA(repoID string, prNumber int, headSHA string) (*types.Review, error) {
	query := `
		SELECT id, org_id, repo_id, pr_number, head_sha, status, result,
		       comments_posted, duration_ms, ai_provider, created_at, completed_at
		FROM reviews
		WHERE repo_id = $1 AND pr_number = $2 AND head_sha = $3
	`

	review := &types.Review{}
	var result sql.NullString
	var completedAt sql.NullTime
	err := db.conn.QueryRow(query, repoID, prNumber, headSHA).Scan(
		&review.ID, &review.OrgID, &review.RepoID, &review.PRNumber,
		&review.HeadSHA, &review.Status, &result,
		&review.CommentsPosted, &review.DurationMs, &review.AIProvider,
		&review.CreatedAt, &completedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get review by PR and SHA: %w", err)
	}

	if result.Valid {
		review.Result = result.String
	}
	if completedAt.Valid {
		review.CompletedAt = &completedAt.Time
	}

	return review, nil
}

func (db *DB) GetReviewByID(id string) (*types.Review, error) {
	query := `
		SELECT id, org_id, repo_id, pr_number, head_sha, status, result,
		       comments_posted, duration_ms, ai_provider, created_at, completed_at
		FROM reviews
		WHERE id = $1
	`

	review := &types.Review{}
	var result sql.NullString
	var completedAt sql.NullTime
	err := db.conn.QueryRow(query, id).Scan(
		&review.ID, &review.OrgID, &review.RepoID, &review.PRNumber,
		&review.HeadSHA, &review.Status, &result,
		&review.CommentsPosted, &review.DurationMs, &review.AIProvider,
		&review.CreatedAt, &completedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("review not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get review: %w", err)
	}

	if result.Valid {
		review.Result = result.String
	}
	if completedAt.Valid {
		review.CompletedAt = &completedAt.Time
	}

	return review, nil
}

func (db *DB) UpdateReviewStatus(id string, status types.ReviewStatus, result *string, durationMs *int) error {
	now := time.Now()
	query := `
		UPDATE reviews
		SET status = $1, result = $2, duration_ms = $3, completed_at = $4
		WHERE id = $5
	`

	_, err := db.conn.Exec(query, status, result, durationMs, now, id)
	if err != nil {
		return fmt.Errorf("failed to update review status: %w", err)
	}

	return nil
}

func (db *DB) GetReviewsByOrgID(orgID string, limit int, offset int) ([]*types.Review, error) {
	query := `
		SELECT id, org_id, repo_id, pr_number, head_sha, status, result,
		       comments_posted, duration_ms, ai_provider, created_at, completed_at
		FROM reviews
		WHERE org_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := db.conn.Query(query, orgID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query reviews: %w", err)
	}
	defer rows.Close()

	reviews := []*types.Review{}
	for rows.Next() {
		review := &types.Review{}
		var result sql.NullString
		var completedAt sql.NullTime
		err := rows.Scan(
			&review.ID, &review.OrgID, &review.RepoID, &review.PRNumber,
			&review.HeadSHA, &review.Status, &result,
			&review.CommentsPosted, &review.DurationMs, &review.AIProvider,
			&review.CreatedAt, &completedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan review: %w", err)
		}
		if result.Valid {
			review.Result = result.String
		}
		if completedAt.Valid {
			review.CompletedAt = &completedAt.Time
		}
		reviews = append(reviews, review)
	}

	return reviews, nil
}

func (db *DB) GetReviewsByRepoID(repoID string, limit int, offset int) ([]*types.Review, error) {
	query := `
		SELECT id, org_id, repo_id, pr_number, head_sha, status, result,
		       comments_posted, duration_ms, ai_provider, created_at, completed_at
		FROM reviews
		WHERE repo_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := db.conn.Query(query, repoID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query reviews: %w", err)
	}
	defer rows.Close()

	reviews := []*types.Review{}
	for rows.Next() {
		review := &types.Review{}
		var result sql.NullString
		var completedAt sql.NullTime
		err := rows.Scan(
			&review.ID, &review.OrgID, &review.RepoID, &review.PRNumber,
			&review.HeadSHA, &review.Status, &result,
			&review.CommentsPosted, &review.DurationMs, &review.AIProvider,
			&review.CreatedAt, &completedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan review: %w", err)
		}
		if result.Valid {
			review.Result = result.String
		}
		if completedAt.Valid {
			review.CompletedAt = &completedAt.Time
		}
		reviews = append(reviews, review)
	}

	return reviews, nil
}
