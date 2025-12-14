package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sherlock/service/internal/services/metrics"
)

// MetricsHandler handles metrics endpoints
type MetricsHandler struct {
	metricsService *metrics.MetricsService
}

// NewMetricsHandler creates a new metrics handler
func NewMetricsHandler(metricsService *metrics.MetricsService) *MetricsHandler {
	return &MetricsHandler{
		metricsService: metricsService,
	}
}

// GetMetrics returns review metrics
// GET /api/metrics
func (h *MetricsHandler) GetMetrics(c *gin.Context) {
	reviewMetrics, err := h.metricsService.GetReviewMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get metrics",
		})
		return
	}

	cacheHitRate := h.metricsService.GetCacheHitRate()
	successRate := h.metricsService.GetSuccessRate()

	c.JSON(http.StatusOK, gin.H{
		"reviews": gin.H{
			"total":            reviewMetrics.TotalReviews,
			"successful":       reviewMetrics.SuccessfulReviews,
			"failed":           reviewMetrics.FailedReviews,
			"average_duration": reviewMetrics.AverageDuration,
			"cache_hits":       reviewMetrics.CacheHits,
			"cache_misses":     reviewMetrics.CacheMisses,
			"incremental":      reviewMetrics.IncrementalReviews,
			"full":             reviewMetrics.FullReviews,
		},
		"rates": gin.H{
			"cache_hit_rate": cacheHitRate,
			"success_rate":    successRate,
		},
	})
}

