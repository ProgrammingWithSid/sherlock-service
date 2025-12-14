package learning

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/sherlock/service/internal/database"
)

// LearningService learns from review feedback to improve future reviews
type LearningService struct {
	db *database.DB
}

// NewLearningService creates a new learning service
func NewLearningService(db *database.DB) *LearningService {
	return &LearningService{
		db: db,
	}
}

// ReviewFeedback represents feedback on a review comment
type ReviewFeedback struct {
	ReviewID    string
	CommentID   string
	FilePath    string
	LineNumber  int
	Feedback    string // "accepted", "dismissed", "fixed"
	UserID      string
	OrgID       string
	CreatedAt   time.Time
}

// RecordFeedback records user feedback on a review comment
func (ls *LearningService) RecordFeedback(ctx context.Context, feedback ReviewFeedback) error {
	feedbackJSON, err := json.Marshal(feedback)
	if err != nil {
		return fmt.Errorf("failed to marshal feedback: %w", err)
	}

	// Store feedback in database (using review_feedback table from migration)
	query := `
		INSERT INTO review_feedback (review_id, comment_id, file_path, line_number, feedback, user_id, org_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (review_id, comment_id) DO UPDATE
		SET feedback = EXCLUDED.feedback, updated_at = NOW()
	`

	_, err = ls.db.DB().ExecContext(ctx, query,
		feedback.ReviewID,
		feedback.CommentID,
		feedback.FilePath,
		feedback.LineNumber,
		feedback.Feedback,
		feedback.UserID,
		feedback.OrgID,
		feedback.CreatedAt,
	)

	if err != nil {
		log.Error().Err(err).Msg("Failed to record feedback")
		return fmt.Errorf("failed to record feedback: %w", err)
	}

	log.Info().
		Str("review_id", feedback.ReviewID).
		Str("comment_id", feedback.CommentID).
		Str("feedback", feedback.Feedback).
		Msg("Feedback recorded")

	return nil
}

// GetFeedbackPatterns analyzes feedback patterns for an organization
func (ls *LearningService) GetFeedbackPatterns(ctx context.Context, orgID string) (map[string]interface{}, error) {
	// Analyze accepted vs dismissed patterns
	query := `
		SELECT
			feedback,
			COUNT(*) as count,
			COUNT(*) * 100.0 / SUM(COUNT(*)) OVER () as percentage
		FROM review_feedback
		WHERE org_id = $1
		GROUP BY feedback
	`

	rows, err := ls.db.DB().QueryContext(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to query feedback patterns: %w", err)
	}
	defer rows.Close()

	patterns := make(map[string]interface{})
	feedbackCounts := make(map[string]int)
	total := 0

	for rows.Next() {
		var feedback string
		var count int
		var percentage float64

		if err := rows.Scan(&feedback, &count, &percentage); err != nil {
			continue
		}

		feedbackCounts[feedback] = count
		total += count
	}

	patterns["feedback_distribution"] = feedbackCounts
	patterns["total_feedback"] = total

	// Calculate acceptance rate
	if accepted, ok := feedbackCounts["accepted"]; ok && total > 0 {
		patterns["acceptance_rate"] = float64(accepted) / float64(total) * 100
	}

	return patterns, nil
}

// ShouldSuppressComment checks if a similar comment was dismissed before
func (ls *LearningService) ShouldSuppressComment(ctx context.Context, orgID string, filePath string, lineNumber int, commentText string) (bool, error) {
	// Check if similar comments were dismissed
	query := `
		SELECT COUNT(*)
		FROM review_feedback
		WHERE org_id = $1
		  AND file_path = $2
		  AND line_number = $3
		  AND feedback = 'dismissed'
		  AND created_at > NOW() - INTERVAL '30 days'
	`

	var count int
	err := ls.db.DB().QueryRowContext(ctx, query, orgID, filePath, lineNumber).Scan(&count)
	if err != nil {
		return false, err
	}

	// If 3+ dismissals in last 30 days, suppress similar comments
	return count >= 3, nil
}

// GetTeamPreferences gets learned preferences for a team
func (ls *LearningService) GetTeamPreferences(ctx context.Context, orgID string) (map[string]interface{}, error) {
	patterns, err := ls.GetFeedbackPatterns(ctx, orgID)
	if err != nil {
		return nil, err
	}

	preferences := make(map[string]interface{})
	preferences["patterns"] = patterns

	// Add learned rules based on feedback
	// Example: If team dismisses "use const" comments often, reduce them
	preferences["learned_rules"] = []string{
		// Will be populated based on feedback analysis
	}

	return preferences, nil
}
