package learning

import (
	"context"
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
	// Store feedback in database (using review_feedback table from migration)
	query := `
		INSERT INTO review_feedback (review_id, comment_id, file_path, line_number, feedback, user_id, org_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (review_id, comment_id) DO UPDATE
		SET feedback = EXCLUDED.feedback, updated_at = NOW()
	`

	_, err := ls.db.Conn().ExecContext(ctx, query,
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
	// Use subquery to calculate total first, then calculate percentage
	query := `
		SELECT
			feedback,
			COUNT(*) as count
		FROM review_feedback
		WHERE org_id = $1
		GROUP BY feedback
	`

	rows, err := ls.db.Conn().QueryContext(ctx, query, orgID)
	if err != nil {
		log.Error().Err(err).Str("org_id", orgID).Msg("Failed to query feedback patterns")
		// Return empty patterns if table doesn't exist or query fails
		return map[string]interface{}{
			"feedback_distribution": map[string]int{},
			"total_feedback":        0,
		}, nil
	}
	defer rows.Close()

	patterns := make(map[string]interface{})
	feedbackCounts := make(map[string]int)
	total := 0

	for rows.Next() {
		var feedback string
		var count int

		if err := rows.Scan(&feedback, &count); err != nil {
			log.Warn().Err(err).Msg("Failed to scan feedback row")
			continue
		}

		feedbackCounts[feedback] = count
		total += count
	}

	if err := rows.Err(); err != nil {
		log.Error().Err(err).Msg("Error iterating feedback rows")
		return map[string]interface{}{
			"feedback_distribution": map[string]int{},
			"total_feedback":        0,
		}, nil
	}

	patterns["feedback_distribution"] = feedbackCounts
	patterns["total_feedback"] = total

	// Calculate acceptance rate (only if we have feedback)
	if total > 0 {
		if accepted, ok := feedbackCounts["accepted"]; ok {
			patterns["acceptance_rate"] = float64(accepted) / float64(total) * 100
		} else {
			patterns["acceptance_rate"] = 0.0
		}
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
	err := ls.db.Conn().QueryRowContext(ctx, query, orgID, filePath, lineNumber).Scan(&count)
	if err != nil {
		// If table doesn't exist or query fails, don't suppress
		log.Warn().Err(err).Msg("Failed to check if comment should be suppressed")
		return false, nil
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

	// Add learned rules based on feedback analysis
	learnedRules := make([]string, 0)

	// Analyze feedback patterns to generate rules
	if patternsMap, ok := patterns["feedback_distribution"].(map[string]int); ok {
		total := patterns["total_feedback"].(int)
		
		// Rule 1: If >50% of feedback is "dismissed", suggest reducing similar comments
		if dismissed, ok := patternsMap["dismissed"]; ok && total > 10 {
			dismissRate := float64(dismissed) / float64(total) * 100
			if dismissRate > 50 {
				learnedRules = append(learnedRules, "High dismissal rate detected - consider reducing similar comment types")
			}
		}

		// Rule 2: If >70% acceptance rate, team is satisfied with review quality
		if accepted, ok := patternsMap["accepted"]; ok && total > 10 {
			acceptRate := float64(accepted) / float64(total) * 100
			if acceptRate > 70 {
				learnedRules = append(learnedRules, "High acceptance rate - review quality is good")
			}
		}

		// Rule 3: If many "fixed" feedbacks, reviews are actionable
		if fixed, ok := patternsMap["fixed"]; ok && total > 10 {
			fixedRate := float64(fixed) / float64(total) * 100
			if fixedRate > 30 {
				learnedRules = append(learnedRules, "High fix rate - reviews are actionable and helpful")
			}
		}
	}

	preferences["learned_rules"] = learnedRules

	return preferences, nil
}
