package database

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// LogUsage logs a usage event
func (db *DB) LogUsage(orgID string, eventType string, metadata map[string]interface{}) error {
	id := uuid.New().String()
	now := time.Now()

	metadataJSON := "{}"
	if len(metadata) > 0 {
		metadataBytes, err := json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
		metadataJSON = string(metadataBytes)
	}

	query := `
		INSERT INTO usage_logs (id, org_id, event_type, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := db.conn.Exec(query, id, orgID, eventType, metadataJSON, now)
	if err != nil {
		return fmt.Errorf("failed to log usage: %w", err)
	}

	return nil
}

// GetUsageStats returns usage statistics for an organization
type UsageStats struct {
	TotalReviews    int
	TotalCommands   int
	TotalAPICalls   int
	ReviewsThisMonth int
	CommandsThisMonth int
}

func (db *DB) GetUsageStats(orgID string, startDate time.Time, endDate time.Time) (*UsageStats, error) {
	stats := &UsageStats{}

	// Total reviews
	query := `
		SELECT COUNT(*)
		FROM reviews
		WHERE org_id = $1 AND created_at >= $2 AND created_at <= $3
	`
	err := db.conn.QueryRow(query, orgID, startDate, endDate).Scan(&stats.TotalReviews)
	if err != nil {
		return nil, fmt.Errorf("failed to get review count: %w", err)
	}

	// Total commands
	query = `
		SELECT COUNT(*)
		FROM usage_logs
		WHERE org_id = $1 AND event_type = 'command' AND created_at >= $2 AND created_at <= $3
	`
	err = db.conn.QueryRow(query, orgID, startDate, endDate).Scan(&stats.TotalCommands)
	if err != nil {
		return nil, fmt.Errorf("failed to get command count: %w", err)
	}

	// Total API calls
	query = `
		SELECT COUNT(*)
		FROM usage_logs
		WHERE org_id = $1 AND event_type = 'api_call' AND created_at >= $2 AND created_at <= $3
	`
	err = db.conn.QueryRow(query, orgID, startDate, endDate).Scan(&stats.TotalAPICalls)
	if err != nil {
		return nil, fmt.Errorf("failed to get API call count: %w", err)
	}

	// Reviews this month
	query = `
		SELECT COUNT(*)
		FROM reviews
		WHERE org_id = $1
		AND created_at >= DATE_TRUNC('month', CURRENT_DATE)
	`
	err = db.conn.QueryRow(query, orgID).Scan(&stats.ReviewsThisMonth)
	if err != nil {
		return nil, fmt.Errorf("failed to get monthly review count: %w", err)
	}

	// Commands this month
	query = `
		SELECT COUNT(*)
		FROM usage_logs
		WHERE org_id = $1
		AND event_type = 'command'
		AND created_at >= DATE_TRUNC('month', CURRENT_DATE)
	`
	err = db.conn.QueryRow(query, orgID).Scan(&stats.CommandsThisMonth)
	if err != nil {
		return nil, fmt.Errorf("failed to get monthly command count: %w", err)
	}

	return stats, nil
}

