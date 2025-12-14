package review

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/sherlock/service/internal/services/cache"
	"github.com/sherlock/service/internal/services/git"
	"github.com/sherlock/service/internal/services/indexer"
	"github.com/sherlock/service/internal/types"
)

// IncrementalReviewService provides incremental review capabilities
type IncrementalReviewService struct {
	gitService      *git.CloneService
	reviewCache     *cache.ReviewCache
	reviewService   *SherlockService
	rustIndexer     *indexer.RustIndexerService // Optional rust indexer for chunk hashing
	useRustIndexer  bool
	repoPath        string // Current repo path for rust indexer
}

// NewIncrementalReviewService creates a new incremental review service
func NewIncrementalReviewService(
	gitService *git.CloneService,
	reviewCache *cache.ReviewCache,
	reviewService *SherlockService,
	rustIndexerURL string,
) *IncrementalReviewService {
	rustIndexer := indexer.NewRustIndexerService(rustIndexerURL)
	useRust := rustIndexer.IsEnabled()
	
	if useRust {
		log.Info().Str("url", rustIndexerURL).Msg("Rust indexer enabled for incremental reviews")
	}
	
	return &IncrementalReviewService{
		gitService:     gitService,
		reviewCache:    reviewCache,
		reviewService:  reviewService,
		rustIndexer:    rustIndexer,
		useRustIndexer: useRust,
	}
}

// ChangedFileInfo contains information about a changed file
type ChangedFileInfo struct {
	Path        string
	Status      string
	ChangedLines []int
	ChunkHashes []string
}

// ReviewDiff reviews only the changed portions of code
func (irs *IncrementalReviewService) ReviewDiff(
	ctx context.Context,
	repoPath string,
	repoID string,
	baseBranch string,
	headBranch string,
	config ReviewConfig,
) (*ReviewResult, error) {
	// Store repo path for rust indexer
	irs.repoPath = repoPath
	log.Info().
		Str("repo_path", repoPath).
		Str("base_branch", baseBranch).
		Str("head_branch", headBranch).
		Msg("Starting incremental review")

	// Step 1: Get changed files
	changedFiles, err := irs.gitService.GetChangedFiles(repoPath, baseBranch, headBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to get changed files: %w", err)
	}

	if len(changedFiles) == 0 {
		return &ReviewResult{
			Summary: "No files changed",
			Stats: ReviewStats{},
			Comments: []ReviewComment{},
			Recommendation: "APPROVE",
		}, nil
	}

	log.Info().Int("changed_files", len(changedFiles)).Msg("Found changed files")

	// Step 2: Get detailed diff information for each file
	changedFileInfos := make([]ChangedFileInfo, 0, len(changedFiles))
	for _, filePath := range changedFiles {
		diff, err := irs.gitService.GetFileDiff(repoPath, baseBranch, headBranch, filePath)
		if err != nil {
			log.Warn().Err(err).Str("file", filePath).Msg("Failed to get file diff, skipping")
			continue
		}

		// Get changed lines
		changedLines, err := irs.gitService.GetChangedLines(repoPath, baseBranch, headBranch, filePath)
		if err != nil {
			log.Warn().Err(err).Str("file", filePath).Msg("Failed to get changed lines, skipping")
			continue
		}

		// Compute chunk hashes for caching (use rust-indexer if available, else simple hash)
		chunkHashes := make([]string, 0)
		for _, hunk := range diff.Hunks {
			var hash string
			
			// Try rust-indexer first if enabled
			if irs.useRustIndexer && irs.rustIndexer != nil {
				rustHash, err := irs.rustIndexer.GetChunkHash(
					ctx,
					repoPath,
					filePath,
					hunk.NewStart,
					hunk.NewStart+hunk.NewLines,
				)
				if err == nil {
					hash = rustHash
					log.Debug().
						Str("file", filePath).
						Int("start", hunk.NewStart).
						Int("end", hunk.NewStart+hunk.NewLines).
						Msg("Using rust-indexer for chunk hash")
				} else {
					log.Debug().Err(err).Msg("Rust indexer hash failed, using simple hash")
					hash = fmt.Sprintf("%s:%d-%d", filePath, hunk.NewStart, hunk.NewStart+hunk.NewLines)
				}
			} else {
				// Simple hash based on file path and line range
				hash = fmt.Sprintf("%s:%d-%d", filePath, hunk.NewStart, hunk.NewStart+hunk.NewLines)
			}
			
			chunkHashes = append(chunkHashes, hash)
		}

		changedFileInfos = append(changedFileInfos, ChangedFileInfo{
			Path:        filePath,
			Status:      diff.Status,
			ChangedLines: changedLines,
			ChunkHashes: chunkHashes,
		})
	}

	// Step 3: Check cache for unchanged chunks
	cachedResults := make([]*ReviewResult, 0)
	newChunks := make([]ChangedFileInfo, 0)

	for _, fileInfo := range changedFileInfos {
		hasNewChunks := false
		for _, hash := range fileInfo.ChunkHashes {
			cached, found := irs.reviewCache.GetCachedReview(repoID, fileInfo.Path, hash)
			if found {
				// Convert types.ReviewResult to ReviewResult
				converted := convertToReviewResult(cached)
				cachedResults = append(cachedResults, converted)
			} else {
				hasNewChunks = true
			}
		}
		if hasNewChunks {
			newChunks = append(newChunks, fileInfo)
		}
	}

	log.Info().
		Int("cached_chunks", len(cachedResults)).
		Int("new_chunks", len(newChunks)).
		Msg("Cache analysis complete")

	// Step 4: Review new/changed chunks only
	var newResults *ReviewResult
	if len(newChunks) > 0 {
		// For now, fall back to full review for new chunks
		// TODO: Implement chunk-based review
		reviewReq := ReviewRequest{
			WorktreePath: repoPath,
			TargetBranch: headBranch,
			BaseBranch:   baseBranch,
			Config:       config,
		}

		var err error
		newResults, err = irs.reviewService.RunReview(reviewReq)
		if err != nil {
			return nil, fmt.Errorf("failed to review new chunks: %w", err)
		}

		// Cache new results
		for _, fileInfo := range newChunks {
			for i, hash := range fileInfo.ChunkHashes {
				if i < len(newResults.Comments) {
					// Cache individual comments (simplified)
					_ = hash // Use hash for caching
				}
			}
		}
	} else {
		// All chunks were cached
		newResults = &ReviewResult{
			Summary: "All changes were previously reviewed (cached)",
			Stats: ReviewStats{},
			Comments: []ReviewComment{},
			Recommendation: "APPROVE",
		}
	}

	// Step 5: Merge cached and new results
	mergedResult := mergeReviewResults(cachedResults, newResults)

	log.Info().
		Int("total_comments", len(mergedResult.Comments)).
		Int("errors", mergedResult.Stats.Errors).
		Int("warnings", mergedResult.Stats.Warnings).
		Msg("Incremental review completed")

	return mergedResult, nil
}

// convertTypesReviewResultToReviewResult converts types.ReviewResult to ReviewResult
func convertTypesReviewResultToReviewResult(tr *types.ReviewResult) *ReviewResult {
	if tr == nil {
		return nil
	}

	// Convert comments
	comments := make([]ReviewComment, 0, len(tr.Comments))
	for _, tc := range tr.Comments {
		comments = append(comments, ReviewComment{
			File:     tc.File,
			Line:     tc.Line,
			Severity: string(tc.Severity),
			Category: string(tc.Category),
			Message:  tc.Message,
			Fix:      tc.Fix,
		})
	}

	// Convert recommendation
	recommendation := "APPROVE"
	if tr.Recommendation == types.RecommendationRequestChanges {
		recommendation = "REQUEST_CHANGES"
	} else if tr.Recommendation == types.RecommendationComment {
		recommendation = "COMMENT"
	}

	return &ReviewResult{
		Summary: fmt.Sprintf("Cached review: %d issues found", tr.Summary.TotalIssues),
		Stats: ReviewStats{
			Errors:      tr.Summary.Errors,
			Warnings:    tr.Summary.Warnings,
			Suggestions: tr.Summary.Suggestions,
		},
		Comments:       comments,
		Recommendation: recommendation,
	}
}


// mergeReviewResults merges multiple review results into one
func mergeReviewResults(cached []*ReviewResult, new *ReviewResult) *ReviewResult {
	merged := &ReviewResult{
		Summary:     new.Summary,
		Stats:       new.Stats,
		Comments:    make([]ReviewComment, 0),
		Recommendation: new.Recommendation,
	}

	// Add cached comments
	for _, result := range cached {
		merged.Comments = append(merged.Comments, result.Comments...)
		merged.Stats.Errors += result.Stats.Errors
		merged.Stats.Warnings += result.Stats.Warnings
		merged.Stats.Suggestions += result.Stats.Suggestions
	}

	// Add new comments
	merged.Comments = append(merged.Comments, new.Comments...)

	// Update recommendation based on merged stats
	if merged.Stats.Errors > 0 {
		merged.Recommendation = "REQUEST_CHANGES"
	} else if merged.Stats.Warnings > 0 {
		merged.Recommendation = "COMMENT"
	} else {
		merged.Recommendation = "APPROVE"
	}

	return merged
}

// convertToReviewResult converts types.ReviewResult to ReviewResult
func convertToReviewResult(tr *types.ReviewResult) *ReviewResult {
	if tr == nil {
		return nil
	}

	comments := make([]ReviewComment, 0, len(tr.Comments))
	for _, c := range tr.Comments {
		comments = append(comments, ReviewComment{
			File:     c.File,
			Line:     int(c.Line),
			Severity: string(c.Severity),
			Category: string(c.Category),
			Message:  c.Message,
			Fix:      c.Fix,
		})
	}

	return &ReviewResult{
		Summary: fmt.Sprintf("Cached review: %d issues found", tr.Summary.TotalIssues),
		Stats: ReviewStats{
			Errors:      tr.Summary.Errors,
			Warnings:    tr.Summary.Warnings,
			Suggestions: tr.Summary.Suggestions,
		},
		Comments:       comments,
		Recommendation: string(tr.Recommendation),
	}
}
