package metrics

import (
	"context"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog/log"
)

// MetricsService provides metrics collection and reporting
type MetricsService struct {
	redis *redis.Client
	ctx   context.Context
}

// NewMetricsService creates a new metrics service
func NewMetricsService(redisClient *redis.Client) *MetricsService {
	return &MetricsService{
		redis: redisClient,
		ctx:   context.Background(),
	}
}

// ReviewMetrics tracks review performance metrics
type ReviewMetrics struct {
	TotalReviews      int64
	SuccessfulReviews int64
	FailedReviews     int64
	AverageDuration   float64
	CacheHits         int64
	CacheMisses       int64
	IncrementalReviews int64
	FullReviews       int64
}

// RecordReview records a review metric
func (ms *MetricsService) RecordReview(duration time.Duration, success bool, usedCache bool, incremental bool) {
	key := "metrics:reviews"

	// Increment counters
	pipe := ms.redis.Pipeline()
	pipe.Incr(ms.ctx, key+":total")
	if success {
		pipe.Incr(ms.ctx, key+":success")
	} else {
		pipe.Incr(ms.ctx, key+":failed")
	}
	if usedCache {
		pipe.Incr(ms.ctx, key+":cache_hits")
	} else {
		pipe.Incr(ms.ctx, key+":cache_misses")
	}
	if incremental {
		pipe.Incr(ms.ctx, key+":incremental")
	} else {
		pipe.Incr(ms.ctx, key+":full")
	}

	// Record duration
	durationMs := duration.Milliseconds()
	pipe.ZAdd(ms.ctx, key+":durations", &redis.Z{
		Score:  float64(time.Now().Unix()),
		Member: durationMs,
	})

	// Keep only last 1000 durations
	pipe.ZRemRangeByRank(ms.ctx, key+":durations", 0, -1001)

	if _, err := pipe.Exec(ms.ctx); err != nil {
		log.Warn().Err(err).Msg("Failed to record review metrics")
	}
}

// GetReviewMetrics gets review metrics
func (ms *MetricsService) GetReviewMetrics() (*ReviewMetrics, error) {
	key := "metrics:reviews"

	pipe := ms.redis.Pipeline()
	totalCmd := pipe.Get(ms.ctx, key+":total")
	successCmd := pipe.Get(ms.ctx, key+":success")
	failedCmd := pipe.Get(ms.ctx, key+":failed")
	cacheHitsCmd := pipe.Get(ms.ctx, key+":cache_hits")
	cacheMissesCmd := pipe.Get(ms.ctx, key+":cache_misses")
	incrementalCmd := pipe.Get(ms.ctx, key+":incremental")
	fullCmd := pipe.Get(ms.ctx, key+":full")
	durationsCmd := pipe.ZRange(ms.ctx, key+":durations", -1000, -1)

	if _, err := pipe.Exec(ms.ctx); err != nil && err != redis.Nil {
		return nil, err
	}

	metrics := &ReviewMetrics{}

	if val, err := totalCmd.Int64(); err == nil {
		metrics.TotalReviews = val
	}
	if val, err := successCmd.Int64(); err == nil {
		metrics.SuccessfulReviews = val
	}
	if val, err := failedCmd.Int64(); err == nil {
		metrics.FailedReviews = val
	}
	if val, err := cacheHitsCmd.Int64(); err == nil {
		metrics.CacheHits = val
	}
	if val, err := cacheMissesCmd.Int64(); err == nil {
		metrics.CacheMisses = val
	}
	if val, err := incrementalCmd.Int64(); err == nil {
		metrics.IncrementalReviews = val
	}
	if val, err := fullCmd.Int64(); err == nil {
		metrics.FullReviews = val
	}

	// Calculate average duration
	durations, err := durationsCmd.Result()
	if err == nil && len(durations) > 0 {
		var sum int64
		count := 0
		for _, d := range durations {
			if val, err := strconv.ParseInt(d, 10, 64); err == nil {
				sum += val
				count++
			}
		}
		if count > 0 {
			metrics.AverageDuration = float64(sum) / float64(count)
		}
	}

	return metrics, nil
}

// GetCacheHitRate calculates cache hit rate
func (ms *MetricsService) GetCacheHitRate() float64 {
	metrics, err := ms.GetReviewMetrics()
	if err != nil {
		return 0
	}

	total := metrics.CacheHits + metrics.CacheMisses
	if total == 0 {
		return 0
	}

	return float64(metrics.CacheHits) / float64(total) * 100
}

// GetSuccessRate calculates success rate
func (ms *MetricsService) GetSuccessRate() float64 {
	metrics, err := ms.GetReviewMetrics()
	if err != nil {
		return 0
	}

	if metrics.TotalReviews == 0 {
		return 0
	}

	return float64(metrics.SuccessfulReviews) / float64(metrics.TotalReviews) * 100
}

// ResetMetrics resets all metrics (for testing)
func (ms *MetricsService) ResetMetrics() error {
	keys := []string{
		"metrics:reviews:total",
		"metrics:reviews:success",
		"metrics:reviews:failed",
		"metrics:reviews:cache_hits",
		"metrics:reviews:cache_misses",
		"metrics:reviews:incremental",
		"metrics:reviews:full",
		"metrics:reviews:durations",
	}

	return ms.redis.Del(ms.ctx, keys...).Err()
}
