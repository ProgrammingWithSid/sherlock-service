package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog/log"
	"github.com/sherlock/service/internal/types"
)

// ReviewCache provides caching for review results to avoid re-reviewing unchanged code
type ReviewCache struct {
	redis *redis.Client
	ttl   time.Duration
	ctx   context.Context
}

// NewReviewCache creates a new review cache instance
func NewReviewCache(redisClient *redis.Client, ttlHours int) *ReviewCache {
	ttl := time.Duration(ttlHours) * time.Hour
	if ttl == 0 {
		ttl = 24 * time.Hour // Default 24 hours
	}

	return &ReviewCache{
		redis: redisClient,
		ttl:   ttl,
		ctx:   context.Background(),
	}
}

// ComputeChunkHash computes a SHA256 hash of code chunk content for cache key
func ComputeChunkHash(filePath string, content string, startLine int, endLine int) string {
	data := fmt.Sprintf("%s:%d:%d:%s", filePath, startLine, endLine, content)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// GetCachedReview retrieves a cached review result for a specific chunk
func (rc *ReviewCache) GetCachedReview(
	repoID string,
	filePath string,
	chunkHash string,
) (*types.ReviewResult, bool) {
	key := rc.buildCacheKey(repoID, filePath, chunkHash)

	val, err := rc.redis.Get(rc.ctx, key).Result()
	if err == redis.Nil {
		return nil, false
	}
	if err != nil {
		log.Warn().Err(err).Str("key", key).Msg("Failed to get cached review")
		return nil, false
	}

	var result types.ReviewResult
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		log.Warn().Err(err).Str("key", key).Msg("Failed to unmarshal cached review")
		return nil, false
	}

	log.Debug().
		Str("repo_id", repoID).
		Str("file", filePath).
		Msg("Cache hit for review")

	return &result, true
}

// CacheReview stores a review result in cache
func (rc *ReviewCache) CacheReview(
	repoID string,
	filePath string,
	chunkHash string,
	result *types.ReviewResult,
) error {
	key := rc.buildCacheKey(repoID, filePath, chunkHash)

	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal review result: %w", err)
	}

	if err := rc.redis.Set(rc.ctx, key, data, rc.ttl).Err(); err != nil {
		return fmt.Errorf("failed to cache review: %w", err)
	}

	log.Debug().
		Str("repo_id", repoID).
		Str("file", filePath).
		Dur("ttl", rc.ttl).
		Msg("Cached review result")

	return nil
}

// InvalidateCache invalidates cached reviews for a repository or specific file
func (rc *ReviewCache) InvalidateCache(repoID string, filePath string) error {
	pattern := rc.buildCacheKey(repoID, filePath, "*")

	// Use SCAN to find all matching keys
	iter := rc.redis.Scan(rc.ctx, 0, pattern, 100).Iterator()
	var keys []string
	for iter.Next(rc.ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return fmt.Errorf("failed to scan cache keys: %w", err)
	}

	if len(keys) > 0 {
		if err := rc.redis.Del(rc.ctx, keys...).Err(); err != nil {
			return fmt.Errorf("failed to delete cache keys: %w", err)
		}
		log.Info().
			Str("repo_id", repoID).
			Str("file", filePath).
			Int("keys_deleted", len(keys)).
			Msg("Invalidated cache")
	}

	return nil
}

// GetCacheStats returns cache statistics
func (rc *ReviewCache) GetCacheStats(repoID string) (int, error) {
	pattern := rc.buildCacheKey(repoID, "*", "*")

	iter := rc.redis.Scan(rc.ctx, 0, pattern, 1000).Iterator()
	count := 0
	for iter.Next(rc.ctx) {
		count++
	}
	if err := iter.Err(); err != nil {
		return 0, fmt.Errorf("failed to scan cache keys: %w", err)
	}

	return count, nil
}

// buildCacheKey builds a Redis cache key
func (rc *ReviewCache) buildCacheKey(repoID string, filePath string, chunkHash string) string {
	return fmt.Sprintf("review:cache:%s:%s:%s", repoID, filePath, chunkHash)
}
