package analytics

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/sherlock/service/internal/types"
)


// AnalyticsService provides analytics and reporting capabilities
type AnalyticsService struct {
	db *sql.DB
}

// NewAnalyticsService creates a new analytics service
func NewAnalyticsService(db *sql.DB) *AnalyticsService {
	return &AnalyticsService{
		db: db,
	}
}

// TimeSeriesPoint represents a single data point in a time series
type TimeSeriesPoint struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"`
}

// IssueCategoryBreakdown represents issue counts by category
type IssueCategoryBreakdown struct {
	Category string `json:"category"`
	Count    int    `json:"count"`
	Errors   int    `json:"errors"`
	Warnings int    `json:"warnings"`
	Info     int    `json:"info"`
}

// RepositoryMetrics represents metrics for a specific repository
type RepositoryMetrics struct {
	RepositoryID   string  `json:"repository_id"`
	RepositoryName string  `json:"repository_name"`
	TotalReviews   int     `json:"total_reviews"`
	TotalIssues    int     `json:"total_issues"`
	AverageScore   float64 `json:"average_score"`
	SuccessRate    float64 `json:"success_rate"`
}

// QualityTrend represents quality metrics over time
type QualityTrend struct {
	Date          string  `json:"date"`
	OverallScore  float64 `json:"overall_score"`
	Accuracy      float64 `json:"accuracy"`
	Actionability float64 `json:"actionability"`
	Coverage      float64 `json:"coverage"`
}

// GetQualityTrends returns quality metrics over time for an organization
func (as *AnalyticsService) GetQualityTrends(orgID string, days int) ([]QualityTrend, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	query := `
		SELECT
			DATE(created_at) as date,
			AVG(
				CASE
					WHEN result::jsonb->>'qualityMetrics' IS NOT NULL
					THEN (result::jsonb->>'qualityMetrics')::jsonb->>'overallScore'
					ELSE NULL
				END::float
			) as overall_score,
			AVG(
				CASE
					WHEN result::jsonb->>'qualityMetrics' IS NOT NULL
					THEN (result::jsonb->>'qualityMetrics')::jsonb->>'accuracy'
					ELSE NULL
				END::float
			) as accuracy,
			AVG(
				CASE
					WHEN result::jsonb->>'qualityMetrics' IS NOT NULL
					THEN (result::jsonb->>'qualityMetrics')::jsonb->>'actionability'
					ELSE NULL
				END::float
			) as actionability,
			AVG(
				CASE
					WHEN result::jsonb->>'qualityMetrics' IS NOT NULL
					THEN (result::jsonb->>'qualityMetrics')::jsonb->>'coverage'
					ELSE NULL
				END::float
			) as coverage
		FROM reviews
		WHERE org_id = $1
			AND created_at >= $2
			AND status = 'completed'
			AND result IS NOT NULL
		GROUP BY DATE(created_at)
		ORDER BY date ASC
	`

	rows, err := as.db.Query(query, orgID, startDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query quality trends: %w", err)
	}
	defer rows.Close()

	var trends []QualityTrend
	for rows.Next() {
		var trend QualityTrend
		var date time.Time
		var overallScore, accuracy, actionability, coverage sql.NullFloat64

		err := rows.Scan(&date, &overallScore, &accuracy, &actionability, &coverage)
		if err != nil {
			log.Error().Err(err).Msg("Failed to scan quality trend row")
			continue
		}


		trend.Date = date.Format("2006-01-02")
		if overallScore.Valid {
			trend.OverallScore = overallScore.Float64
		}
		if accuracy.Valid {
			trend.Accuracy = accuracy.Float64
		}
		if actionability.Valid {
			trend.Actionability = actionability.Float64
		}
		if coverage.Valid {
			trend.Coverage = coverage.Float64
		}

		trends = append(trends, trend)
	}

	return trends, nil
}

// GetIssueTrends returns issue counts over time
func (as *AnalyticsService) GetIssueTrends(orgID string, days int) ([]TimeSeriesPoint, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	query := `
		SELECT
			DATE(created_at) as date,
			SUM(
				CASE
					WHEN result::jsonb->>'summary' IS NOT NULL
					THEN ((result::jsonb->>'summary')::jsonb->>'total_issues')::int
					ELSE 0
				END
			) as total_issues
		FROM reviews
		WHERE org_id = $1
			AND created_at >= $2
			AND status = 'completed'
			AND result IS NOT NULL
		GROUP BY DATE(created_at)
		ORDER BY date ASC
	`

	rows, err := as.db.Query(query, orgID, startDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query issue trends: %w", err)
	}
	defer rows.Close()

	var points []TimeSeriesPoint
	for rows.Next() {
		var point TimeSeriesPoint
		var date time.Time
		var value sql.NullFloat64

		err := rows.Scan(&date, &value)
		if err != nil {
			log.Error().Err(err).Msg("Failed to scan issue trend row")
			continue
		}


		point.Date = date.Format("2006-01-02")
		if value.Valid {
			point.Value = value.Float64
		}

		points = append(points, point)
	}

	return points, nil
}

// GetIssueCategoryBreakdown returns issue counts by category
func (as *AnalyticsService) GetIssueCategoryBreakdown(orgID string, days int) ([]IssueCategoryBreakdown, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	query := `
		SELECT
			result,
			created_at
		FROM reviews
		WHERE org_id = $1
			AND created_at >= $2
			AND status = 'completed'
			AND result IS NOT NULL
	`

	rows, err := as.db.Query(query, orgID, startDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query reviews: %w", err)
	}
	defer rows.Close()

	categoryMap := make(map[string]*IssueCategoryBreakdown)

	for rows.Next() {
		var resultJSON string
		var createdAt time.Time

		err := rows.Scan(&resultJSON, &createdAt)
		if err != nil {
			log.Error().Err(err).Msg("Failed to scan review row")
			continue
		}


		var result map[string]interface{}
		if err := json.Unmarshal([]byte(resultJSON), &result); err != nil {
			log.Warn().Err(err).Msg("Failed to unmarshal review result")
			continue
		}


		comments, ok := result["comments"].([]interface{})
		if !ok {
			continue
		}

		for _, commentRaw := range comments {
			comment, ok := commentRaw.(map[string]interface{})
			if !ok {
				continue
			}

			category, _ := comment["category"].(string)
			if category == "" {
				category = "general"
			}

			severity, _ := comment["severity"].(string)

			if _, exists := categoryMap[category]; !exists {
				categoryMap[category] = &IssueCategoryBreakdown{
					Category: category,
				}
			}

			catBreakdown := categoryMap[category]
			catBreakdown.Count++

			switch severity {
			case "error":
				catBreakdown.Errors++
			case "warning":
				catBreakdown.Warnings++
			case "info":
				catBreakdown.Info++
			}
		}
	}

	breakdowns := make([]IssueCategoryBreakdown, 0, len(categoryMap))
	for _, breakdown := range categoryMap {
		breakdowns = append(breakdowns, *breakdown)
	}

	return breakdowns, nil
}

// GetRepositoryComparison returns metrics comparing repositories
func (as *AnalyticsService) GetRepositoryComparison(orgID string, days int) ([]RepositoryMetrics, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	query := `
		SELECT
			r.repo_id,
			repo.full_name,
			COUNT(*) as total_reviews,
			SUM(
				CASE
					WHEN r.result::jsonb->>'summary' IS NOT NULL
					THEN ((r.result::jsonb->>'summary')::jsonb->>'total_issues')::int
					ELSE 0
				END
			) as total_issues,
			AVG(
				CASE
					WHEN r.result::jsonb->>'qualityMetrics' IS NOT NULL
					THEN (r.result::jsonb->>'qualityMetrics')::jsonb->>'overallScore'
					ELSE NULL
				END::float
			) as avg_score,
			SUM(CASE WHEN r.status = 'completed' THEN 1 ELSE 0 END)::float / COUNT(*)::float * 100 as success_rate
		FROM reviews r
		LEFT JOIN repositories repo ON r.repo_id = repo.id
		WHERE r.org_id = $1
			AND r.created_at >= $2
		GROUP BY r.repo_id, repo.full_name
		ORDER BY total_reviews DESC
	`

	rows, err := as.db.Query(query, orgID, startDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query repository comparison: %w", err)
	}
	defer rows.Close()

	var metrics []RepositoryMetrics
	for rows.Next() {
		var metric RepositoryMetrics
		var repositoryName sql.NullString
		var avgScore sql.NullFloat64

		err := rows.Scan(
			&metric.RepositoryID,
			&repositoryName,
			&metric.TotalReviews,
			&metric.TotalIssues,
			&avgScore,
			&metric.SuccessRate,
		)
		if err != nil {
			log.Error().Err(err).Msg("Failed to scan repository comparison row")
			continue
		}

		if repositoryName.Valid {
			metric.RepositoryName = repositoryName.String
		} else {
			metric.RepositoryName = "Unknown"
		}


		if avgScore.Valid {
			metric.AverageScore = avgScore.Float64
		}

		metrics = append(metrics, metric)
	}

	return metrics, nil
}

// GetSeverityTrends returns trends for errors, warnings, and suggestions
func (as *AnalyticsService) GetSeverityTrends(orgID string, days int) (map[string][]TimeSeriesPoint, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	query := `
		SELECT
			DATE(created_at) as date,
			SUM(
				CASE
					WHEN result::jsonb->>'summary' IS NOT NULL
					THEN ((result::jsonb->>'summary')::jsonb->>'errors')::int
					ELSE 0
				END
			) as errors,
			SUM(
				CASE
					WHEN result::jsonb->>'summary' IS NOT NULL
					THEN ((result::jsonb->>'summary')::jsonb->>'warnings')::int
					ELSE 0
				END
			) as warnings,
			SUM(
				CASE
					WHEN result::jsonb->>'summary' IS NOT NULL
					THEN ((result::jsonb->>'summary')::jsonb->>'suggestions')::int
					ELSE 0
				END
			) as suggestions
		FROM reviews
		WHERE org_id = $1
			AND created_at >= $2
			AND status = 'completed'
			AND result IS NOT NULL
		GROUP BY DATE(created_at)
		ORDER BY date ASC
	`

	rows, err := as.db.Query(query, orgID, startDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query severity trends: %w", err)
	}
	defer rows.Close()

	trends := map[string][]TimeSeriesPoint{
		"errors":      {},
		"warnings":    {},
		"suggestions": {},
	}

	for rows.Next() {
		var date time.Time
		var errors, warnings, suggestions sql.NullFloat64

		err := rows.Scan(&date, &errors, &warnings, &suggestions)
		if err != nil {
			log.Error().Err(err).Msg("Failed to scan severity trend row")
			continue
		}


		dateStr := date.Format("2006-01-02")

		if errors.Valid {
			trends["errors"] = append(trends["errors"], TimeSeriesPoint{
				Date:  dateStr,
				Value: errors.Float64,
			})
		}
		if warnings.Valid {
			trends["warnings"] = append(trends["warnings"], TimeSeriesPoint{
				Date:  dateStr,
				Value: warnings.Float64,
			})
		}
		if suggestions.Valid {
			trends["suggestions"] = append(trends["suggestions"], TimeSeriesPoint{
				Date:  dateStr,
				Value: suggestions.Float64,
			})
		}
	}

	return trends, nil
}
