package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	appconfig "github.com/sherlock/service/internal/config"
	"github.com/sherlock/service/internal/database"
	"github.com/sherlock/service/internal/queue"
	"github.com/sherlock/service/internal/commands"
	"github.com/sherlock/service/internal/services/cache"
	"github.com/sherlock/service/internal/services/comment"
	repoconfig "github.com/sherlock/service/internal/services/config"
	"github.com/sherlock/service/internal/services/git"
	"github.com/sherlock/service/internal/services/github"
	"github.com/sherlock/service/internal/services/indexer"
	"github.com/sherlock/service/internal/services/metrics"
	"github.com/sherlock/service/internal/services/review"
	"github.com/sherlock/service/internal/types"
)

type WorkerPool struct {
	server           *asynq.Server
	db               *database.DB
	config           *appconfig.Config
	gitService       *git.CloneService
	reviewService    *review.SherlockService
	commandService   *review.CommandService
	githubCommentSvc *comment.GitHubCommentService
	gitlabCommentSvc *comment.GitLabCommentService
	commandRouter         *commands.CommandRouter
	reviewCache           *cache.ReviewCache
	redisClient           *redis.Client
	incrementalReviewSvc  *review.IncrementalReviewService
	codebaseIndexer       *indexer.CodebaseIndexer
	metricsService        *metrics.MetricsService
	tokenService          *github.TokenService
}

func NewWorkerPool(reviewQueue *queue.ReviewQueue, db *database.DB, cfg *appconfig.Config, redisClient *redis.Client) *WorkerPool {
	gitService := git.NewCloneService(cfg.ReposPath, cfg.MaxRepoAgeHours)
	reviewService := review.NewSherlockService("") // Use default node path
	commandService := review.NewCommandService("")

	// Initialize command handlers
	parser := commands.NewParser("sherlock")

	reviewHandler := commands.NewReviewHandler(func(job *types.ReviewJob, priority int) (string, error) {
		return reviewQueue.EnqueueReviewJob(job, priority)
	})

	explainHandler := commands.NewExplainHandler(
		commandService,
		func(repoPath string, branch string) (string, error) {
			// Clone and create worktree
			repoPath, err := gitService.CloneRepository(repoPath, false)
			if err != nil {
				return "", err
			}
			return gitService.CreateWorktree(repoPath, branch, branch)
		},
		func(orgID string, repoID string) (review.ReviewConfig, error) {
			// Get config from database
			repo, err := db.GetRepositoryByID(repoID)
			if err != nil {
				return review.ReviewConfig{}, err
			}
			org, err := db.GetOrganizationByID(orgID)
			if err != nil {
				return review.ReviewConfig{}, err
			}
			configLoader := repoconfig.NewLoader()
			repoConfig, _ := configLoader.LoadFromJSON(repo.Config)
			return buildReviewConfigFromRepo(org, repo, repoConfig, cfg), nil
		},
	)

	securityHandler := commands.NewSecurityHandler(
		commandService,
		func(repoPath string, branch string) (string, error) {
			repoPath, err := gitService.CloneRepository(repoPath, false)
			if err != nil {
				return "", err
			}
			return gitService.CreateWorktree(repoPath, branch, branch)
		},
		func(orgID string, repoID string) (review.ReviewConfig, error) {
			repo, err := db.GetRepositoryByID(repoID)
			if err != nil {
				return review.ReviewConfig{}, err
			}
			org, err := db.GetOrganizationByID(orgID)
			if err != nil {
				return review.ReviewConfig{}, err
			}
			configLoader := repoconfig.NewLoader()
			repoConfig, _ := configLoader.LoadFromJSON(repo.Config)
			return buildReviewConfigFromRepo(org, repo, repoConfig, cfg), nil
		},
	)

	performanceHandler := commands.NewPerformanceHandler(
		commandService,
		func(repoPath string, branch string) (string, error) {
			repoPath, err := gitService.CloneRepository(repoPath, false)
			if err != nil {
				return "", err
			}
			return gitService.CreateWorktree(repoPath, branch, branch)
		},
		func(orgID string, repoID string) (review.ReviewConfig, error) {
			repo, err := db.GetRepositoryByID(repoID)
			if err != nil {
				return review.ReviewConfig{}, err
			}
			org, err := db.GetOrganizationByID(orgID)
			if err != nil {
				return review.ReviewConfig{}, err
			}
			configLoader := repoconfig.NewLoader()
			repoConfig, _ := configLoader.LoadFromJSON(repo.Config)
			return buildReviewConfigFromRepo(org, repo, repoConfig, cfg), nil
		},
	)

	fixHandler := commands.NewFixHandler()
	helpHandler := commands.NewHelpHandler(parser)

	commandRouter := commands.NewCommandRouter(
		reviewHandler,
		explainHandler,
		fixHandler,
		securityHandler,
		performanceHandler,
		helpHandler,
	)

	// Initialize review cache
	reviewCache := cache.NewReviewCache(redisClient, cfg.ReviewCacheTTLHours)

	// Initialize incremental review service (with optional rust-indexer)
	incrementalReviewSvc := review.NewIncrementalReviewService(
		gitService,
		reviewCache,
		reviewService,
		cfg.RustIndexerURL,
	)

	// Initialize codebase indexer (with Rust or chunkyyy integration)
	codebaseIndexer := indexer.NewCodebaseIndexer(db, cfg.ReposPath, "", cfg.RustIndexerURL)

	// Initialize metrics service
	metricsService := metrics.NewMetricsService(redisClient)

	// Initialize GitHub token service (if GitHub App is configured)
	var tokenService *github.TokenService
	if cfg.GitHubAppID > 0 && cfg.GitHubPrivateKeyPath != "" {
		privateKeyPath := cfg.GitHubPrivateKeyPath

		// If path is relative, try common locations
		if !filepath.IsAbs(privateKeyPath) {
			// Try common EC2/service locations
			possiblePaths := []string{
				privateKeyPath, // Original path
				filepath.Join("/home/ubuntu/sherlock-service/backend", privateKeyPath), // EC2 location
				filepath.Join("backend", privateKeyPath), // If running from project root
			}

			for _, path := range possiblePaths {
				if _, err := os.Stat(path); err == nil {
					privateKeyPath = path
					break
				}
			}
		}

		privateKeyData, err := os.ReadFile(privateKeyPath)
		if err != nil {
			log.Warn().Err(err).Str("path", privateKeyPath).Msg("Failed to read GitHub private key, token refresh will not work")
		} else {
			tokenService, err = github.NewTokenService(cfg.GitHubAppID, privateKeyData)
			if err != nil {
				log.Warn().Err(err).Msg("Failed to initialize GitHub token service, token refresh will not work")
			}
		}
	}

	return &WorkerPool{
		server:              reviewQueue.GetServer(),
		db:                  db,
		config:              cfg,
		gitService:          gitService,
		reviewService:       reviewService,
		commandService:      commandService,
		commandRouter:       commandRouter,
		reviewCache:         reviewCache,
		redisClient:         redisClient,
		incrementalReviewSvc: incrementalReviewSvc,
		codebaseIndexer:     codebaseIndexer,
		metricsService:      metricsService,
		tokenService:         tokenService,
	}
}

func buildReviewConfigFromRepo(org *types.Organization, repo *types.Repository, repoConfig *repoconfig.RepoConfig, cfg *appconfig.Config) review.ReviewConfig {
	aiProvider := cfg.AIProvider
	if repoConfig != nil && repoConfig.AI != nil && repoConfig.AI.Provider != "" {
		aiProvider = repoConfig.AI.Provider
	}

	config := review.ReviewConfig{
		AIProvider:  aiProvider,
		GlobalRules: []string{},
	}

	// Use repository-specific rules only
	if repoConfig != nil && len(repoConfig.Rules) > 0 {
		config.GlobalRules = repoConfig.Rules
	}

	if aiProvider == "openai" && cfg.OpenAIAPIKey != "" {
		model := "gpt-4o" // Default model (can be overridden via .sherlock.yml)
		if repoConfig != nil && repoConfig.AI != nil && repoConfig.AI.Model != "" {
			model = repoConfig.AI.Model
		}
		config.OpenAI = &review.OpenAIConfig{
			APIKey: cfg.OpenAIAPIKey,
			Model:   model,
		}
	} else if aiProvider == "claude" && cfg.ClaudeAPIKey != "" {
		model := "claude-3-5-sonnet-20241022"
		if repoConfig != nil && repoConfig.AI != nil && repoConfig.AI.Model != "" {
			model = repoConfig.AI.Model
		}
		config.Claude = &review.ClaudeConfig{
			APIKey: cfg.ClaudeAPIKey,
			Model:   model,
		}
	}

	return config
}

func (wp *WorkerPool) Start(ctx context.Context) {
	mux := asynq.NewServeMux()

	mux.HandleFunc("review", wp.handleReviewJob)
	mux.HandleFunc("command", wp.handleCommandJob)

	if err := wp.server.Run(mux); err != nil {
		log.Fatal().Err(err).Msg("Worker pool failed")
	}
}

func (wp *WorkerPool) Stop() {
	wp.server.Shutdown()
}

func (wp *WorkerPool) handleReviewJob(ctx context.Context, t *asynq.Task) error {
	var job types.ReviewJob
	if err := json.Unmarshal(t.Payload(), &job); err != nil {
		return fmt.Errorf("failed to unmarshal job: %w", err)
	}

	logger := log.With().
		Str("job_id", job.ID).
		Str("repo", job.Repo.FullName).
		Int("pr_number", job.PR.Number).
		Logger()

	logger.Info().Msg("Processing review job")

	// Add timeout context
	timeout := time.Duration(wp.config.ReviewTimeoutMs) * time.Millisecond
	if timeout == 0 {
		timeout = 5 * time.Minute // Default 5 minutes
	}
	reviewCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Update review status
	if err := wp.db.UpdateReviewStatus(job.ID, types.ReviewStatusProcessing, nil, nil); err != nil {
		logger.Error().Err(err).Msg("Failed to update review status")
	}

	startTime := time.Now()

	// Process with retry logic
	maxRetries := 3
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(attempt) * time.Second
			logger.Warn().
				Int("attempt", attempt).
				Dur("backoff", backoff).
				Msg("Retrying review job")
			time.Sleep(backoff)
		}

		err := wp.processReviewJob(reviewCtx, &job, logger, startTime)
		if err == nil {
			return nil
		}

		lastErr = err
		// Don't retry on certain errors (e.g., invalid config, not found)
		if isNonRetryableError(err) {
			logger.Error().Err(err).Msg("Non-retryable error, aborting")
			break
		}

		if attempt < maxRetries {
			logger.Warn().Err(err).Int("attempt", attempt+1).Msg("Review job failed, will retry")
		}
	}

	// All retries failed
	logger.Error().Err(lastErr).Msg("Review job failed after all retries")
	wp.updateReviewStatusFailed(job.ID, lastErr.Error(), startTime)
	return lastErr
}

// isNonRetryableError checks if an error should not be retried
func isNonRetryableError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	nonRetryablePatterns := []string{
		"not found",
		"invalid",
		"unauthorized",
		"forbidden",
		"validation",
	}
	for _, pattern := range nonRetryablePatterns {
		if strings.Contains(strings.ToLower(errStr), pattern) {
			return true
		}
	}
	return false
}

func (wp *WorkerPool) processReviewJob(ctx context.Context, job *types.ReviewJob, logger zerolog.Logger, startTime time.Time) error {
	var worktreePath string
	var repoPath string

	defer func() {
		// Cleanup worktree
		if worktreePath != "" {
			if err := wp.gitService.RemoveWorktree(worktreePath); err != nil {
				logger.Warn().Err(err).Str("worktree_path", worktreePath).Msg("Failed to cleanup worktree")
			}
		}
	}()

	// Step 1: Clone repository (or use existing)
	repoPath, err := wp.gitService.CloneRepository(job.Repo.CloneURL, false)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to clone repository")
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	// Step 2: Create worktree for the review
	worktreePath, err = wp.gitService.CreateWorktree(repoPath, job.PR.HeadSHA, job.PR.HeadSHA)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create worktree")
		return fmt.Errorf("failed to create worktree: %w", err)
	}

	// Step 3: Get repository from database
	// We need to find the repository by full name
	org, err := wp.db.GetOrganizationByID(job.OrgID)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get organization")
		return fmt.Errorf("organization not found: %w", err)
	}

	repos, err := wp.db.GetRepositoriesByOrgID(org.ID)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to get repositories, using defaults")
	}

	var repo *types.Repository
	for _, r := range repos {
		if r.FullName == job.Repo.FullName {
			repo = r
			break
		}
	}

	// Step 4: Load repository config
	configLoader := repoconfig.NewLoader()
	var repoConfig *repoconfig.RepoConfig

	if repo != nil {
		// Try to load from .sherlock.yml file first
		fileConfig, err := configLoader.LoadFromFile(worktreePath)
		if err != nil {
			logger.Warn().Err(err).Msg("Failed to load .sherlock.yml, using database config")
			// Fall back to database config
			if repo.Config != "" {
				dbConfig, err := configLoader.LoadFromJSON(repo.Config)
				if err == nil {
					repoConfig = dbConfig
				}
			}
		} else {
			repoConfig = fileConfig
			// Merge with database config if available
			if repo.Config != "" {
				dbConfig, err := configLoader.LoadFromJSON(repo.Config)
				if err == nil {
					// File config takes precedence, but merge rules and other settings
					if dbConfig.Rules != nil && len(dbConfig.Rules) > 0 {
						repoConfig.Rules = dbConfig.Rules
					}
				}
			}
		}
	}

	if repoConfig == nil {
		defaultConfig, _ := configLoader.LoadFromJSON("{}")
		repoConfig = defaultConfig
	}

	// Step 5: Build review config
	reviewConfig := wp.buildReviewConfig(*job, repo, repoConfig)

	// Validate that GitHub/GitLab token is present (code-sherlock requires it)
	if job.Platform == types.PlatformGitHub && reviewConfig.GitHub == nil {
		logger.Error().Str("org_id", job.OrgID).Msg("GitHub token is required but not available")
		return fmt.Errorf("GitHub token is required but not available for org %s - ensure GitHub App is properly installed and configured", job.OrgID)
	}
	if job.Platform == types.PlatformGitLab && reviewConfig.GitLab == nil {
		logger.Error().Msg("GitLab token is required but not configured")
		return fmt.Errorf("GitLab token is required but not configured")
	}

	// Step 6: Run code-sherlock review (use incremental if enabled)
	var reviewResult *review.ReviewResult
	var usedIncremental bool
	var usedCache bool

	if wp.config.EnableIncrementalReviews && repo != nil {
		// Use incremental review service
		logger.Info().Msg("Using incremental review service")
		usedIncremental = true
		reviewResult, err = wp.incrementalReviewSvc.ReviewDiff(
			ctx,
			repoPath,
			repo.ID,
			job.PR.BaseBranch,
			job.PR.HeadSHA,
			reviewConfig,
		)
		if err != nil {
			logger.Warn().Err(err).Msg("Incremental review failed, falling back to full review")
			usedIncremental = false
			// Fall back to full review
			reviewReq := review.ReviewRequest{
				WorktreePath: worktreePath,
				TargetBranch: job.PR.HeadSHA,
				BaseBranch:   job.PR.BaseBranch,
				Config:       reviewConfig,
			}
			reviewResult, err = wp.reviewService.RunReview(reviewReq)
		} else {
			// Check if cache was used (incremental review service tracks this)
			// For now, assume cache was used if incremental succeeded
			usedCache = true
		}
	} else {
		// Use full review
		reviewReq := review.ReviewRequest{
			WorktreePath: worktreePath,
			TargetBranch: job.PR.HeadSHA,
			BaseBranch:   job.PR.BaseBranch,
			Config:       reviewConfig,
		}
		reviewResult, err = wp.reviewService.RunReview(reviewReq)
	}

	if err != nil {
		logger.Error().Err(err).Msg("Review execution failed")
		// Record failed review metrics
		if wp.metricsService != nil {
			duration := time.Since(startTime)
			wp.metricsService.RecordReview(duration, false, false, usedIncremental)
		}
		return fmt.Errorf("review execution failed: %w", err)
	}

	// Step 7: Convert to internal types
	result := wp.convertReviewResult(reviewResult)

	// Step 8: Cache review results (for future incremental reviews)
	if repo != nil {
		// Cache individual comments by file/line (simplified - full implementation would cache by chunk hash)
		// This is a placeholder for future incremental review implementation
		_ = wp.reviewCache // Will be used when implementing chunk-based caching
	}

	// Step 9: Post comments if configured
	if repoConfig != nil && repoConfig.Comments != nil && repoConfig.Comments.PostSummary {
		if err := wp.postComments(*job, result); err != nil {
			logger.Warn().Err(err).Msg("Failed to post comments")
			// Don't fail the job if comments fail
		}
	}

	// Step 10: Log usage
	if repo != nil {
		_ = wp.db.LogUsage(job.OrgID, "review", map[string]interface{}{
			"repo_id":   repo.ID,
			"pr_number": job.PR.Number,
			"issues":    result.Summary.TotalIssues,
			"errors":    result.Summary.Errors,
		})
	}

	duration := int(time.Since(startTime).Milliseconds())

	// Step 11: Record metrics (only if quality metrics weren't already recorded)
	if wp.metricsService != nil && result.QualityMetrics == nil {
		reviewDuration := time.Since(startTime)
		wp.metricsService.RecordReview(reviewDuration, true, usedCache, usedIncremental)
		logger.Info().
			Dur("duration", reviewDuration).
			Bool("incremental", usedIncremental).
			Bool("cache_used", usedCache).
			Msg("Review metrics recorded")
	}

	// Step 12: Save result
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	resultStr := string(resultJSON)

	// Update review status
	if err := wp.db.UpdateReviewStatus(job.ID, types.ReviewStatusCompleted, &resultStr, &duration); err != nil {
		logger.Error().Err(err).Msg("Failed to update review status")
		return fmt.Errorf("failed to update review status: %w", err)
	}

	logger.Info().
		Int("duration_ms", duration).
		Int("comments", len(result.Comments)).
		Int("errors", result.Summary.Errors).
		Int("warnings", result.Summary.Warnings).
		Msg("Review job completed successfully")

	return nil
}

func (wp *WorkerPool) buildReviewConfig(job types.ReviewJob, repo *types.Repository, repoConfig *repoconfig.RepoConfig) review.ReviewConfig {
	// Determine AI provider (repo config takes precedence over env var)
	aiProvider := wp.config.AIProvider // Start with env var default
	if repoConfig != nil && repoConfig.AI != nil && repoConfig.AI.Provider != "" {
		// Repo config (.sherlock.yml) overrides env var
		aiProvider = repoConfig.AI.Provider
	}

	config := review.ReviewConfig{
		AIProvider:  aiProvider,
		GlobalRules: []string{},
		Repository: review.RepositoryConfig{
			Owner:      job.Repo.Owner,
			Repo:       job.Repo.Name,
			BaseBranch: job.PR.BaseBranch,
		},
		PR: review.PRConfig{
			Number:     job.PR.Number,
			BaseBranch: job.PR.BaseBranch,
		},
	}

	// Set AI provider config
	if aiProvider == "openai" && wp.config.OpenAIAPIKey != "" {
		// Default model - change this to your preferred model
		// Available models: gpt-4o, gpt-4-turbo-preview, gpt-4, gpt-3.5-turbo
		model := "gpt-4o" // Default model (can be overridden via .sherlock.yml)
		if repoConfig != nil && repoConfig.AI != nil && repoConfig.AI.Model != "" {
			model = repoConfig.AI.Model
		}
		config.OpenAI = &review.OpenAIConfig{
			APIKey: wp.config.OpenAIAPIKey,
			Model:   model,
		}
	} else if aiProvider == "claude" && wp.config.ClaudeAPIKey != "" {
		model := "claude-3-5-sonnet-20241022"
		if repoConfig != nil && repoConfig.AI != nil && repoConfig.AI.Model != "" {
			model = repoConfig.AI.Model
		}
		config.Claude = &review.ClaudeConfig{
			APIKey: wp.config.ClaudeAPIKey,
			Model:   model,
		}
	}

	// Use repository-specific rules only
	if repoConfig != nil && len(repoConfig.Rules) > 0 {
		config.GlobalRules = repoConfig.Rules
	}

	// Get GitHub token from installation (refresh if needed)
	if job.Platform == types.PlatformGitHub {
		log.Debug().Str("org_id", job.OrgID).Str("platform", string(job.Platform)).Msg("Building review config for GitHub platform")
		inst, err := wp.db.GetInstallationByOrgID(job.OrgID)
		if err != nil {
			log.Error().Err(err).Str("org_id", job.OrgID).Msg("Failed to get GitHub installation - code-sherlock requires GitHub token")
			// Return config without token - this will cause code-sherlock to fail with a clear error
			return config
		}

		token := inst.Token
		log.Info().
			Int64("installation_id", inst.InstallationID).
			Bool("has_token", token != "").
			Bool("has_token_expires", inst.TokenExpires != nil).
			Bool("has_token_service", wp.tokenService != nil).
			Str("org_id", job.OrgID).
			Msg("Installation found, checking token")

		// Refresh token if expired or missing
		if wp.tokenService == nil {
			log.Error().
				Int64("installation_id", inst.InstallationID).
				Str("org_id", job.OrgID).
				Msg("TokenService is nil - cannot refresh token. Check GITHUB_APP_ID and GITHUB_PRIVATE_KEY_PATH environment variables")
		} else {
			needsRefresh := token == "" || inst.TokenExpires == nil || (inst.TokenExpires != nil && time.Until(*inst.TokenExpires) < 5*time.Minute)
			if needsRefresh {
				log.Info().Int64("installation_id", inst.InstallationID).Msg("Refreshing GitHub installation token")
				newToken, newExpiresAt, err := wp.tokenService.GetInstallationTokenWithRefresh(
					inst.InstallationID,
					token,
					inst.TokenExpires,
				)
				if err != nil {
					log.Error().Err(err).Int64("installation_id", inst.InstallationID).Msg("Failed to refresh GitHub token")
				} else {
					token = newToken
					log.Info().Int64("installation_id", inst.InstallationID).Bool("has_new_token", token != "").Msg("Token refreshed successfully")
					// Update token in database
					if updateErr := wp.db.UpdateInstallationToken(inst.InstallationID, newToken, newExpiresAt); updateErr != nil {
						log.Warn().Err(updateErr).Int64("installation_id", inst.InstallationID).Msg("Failed to update token in database")
					}
				}
			} else {
				log.Debug().Int64("installation_id", inst.InstallationID).Msg("Token is still valid, no refresh needed")
			}
		}

		if token != "" {
			config.GitHub = &review.GitHubConfig{
				Token: token,
			}
			log.Debug().Int64("installation_id", inst.InstallationID).Msg("GitHub token added to review config")
		} else {
			log.Error().
				Int64("installation_id", inst.InstallationID).
				Str("org_id", job.OrgID).
				Bool("has_token_service", wp.tokenService != nil).
				Msg("GitHub token is empty after refresh attempt")
		}
	} else if job.Platform == types.PlatformGitLab {
		// For GitLab, check if token is available
		if wp.config.GitLabToken != "" {
			config.GitLab = &review.GitLabConfig{
				Token:     wp.config.GitLabToken,
				ProjectID: "", // Project ID should be set from repo config if needed
			}
			log.Debug().Msg("GitLab token added to review config")
		} else {
			log.Error().Msg("GitLab token is not configured - code-sherlock requires GitHub or GitLab token to function")
		}
	} else {
		log.Debug().Str("platform", string(job.Platform)).Msg("Unknown platform, code-sherlock requires GitHub or GitLab token")
	}

	return config
}

func (wp *WorkerPool) convertReviewResult(reviewResult *review.ReviewResult) types.ReviewResult {
	comments := make([]types.ReviewComment, 0, len(reviewResult.Comments))

	// Convert quality metrics if present
	var qualityMetrics *types.ReviewQualityMetrics
	if reviewResult.QualityMetrics != nil {
		qualityMetrics = &types.ReviewQualityMetrics{
			Accuracy:     reviewResult.QualityMetrics.Accuracy,
			Actionability: reviewResult.QualityMetrics.Actionability,
			Coverage:     reviewResult.QualityMetrics.Coverage,
			Precision:    reviewResult.QualityMetrics.Precision,
			Recall:       reviewResult.QualityMetrics.Recall,
			OverallScore: reviewResult.QualityMetrics.OverallScore,
			Confidence:   reviewResult.QualityMetrics.Confidence,
		}
	}

	for _, c := range reviewResult.Comments {
		severity := types.SeverityInfo
		switch strings.ToLower(c.Severity) {
		case "error":
			severity = types.SeverityError
		case "warning":
			severity = types.SeverityWarning
		}

		category := types.CategoryQuality
		switch strings.ToLower(c.Category) {
		case "bugs":
			category = types.CategoryBugs
		case "security":
			category = types.CategorySecurity
		case "performance":
			category = types.CategoryPerformance
		case "architecture":
			category = types.CategoryArchitecture
		}

		comments = append(comments, types.ReviewComment{
			File:     c.File,
			Line:     c.Line,
			Severity: severity,
			Category: category,
			Message:  c.Message,
			Fix:      c.Fix,
		})
	}

	recommendation := types.RecommendationComment
	switch strings.ToUpper(reviewResult.Recommendation) {
	case "APPROVE":
		recommendation = types.RecommendationApprove
	case "REQUEST_CHANGES":
		recommendation = types.RecommendationRequestChanges
	}

	result := types.ReviewResult{
		Recommendation: recommendation,
		Summary: types.ReviewSummary{
			TotalIssues: reviewResult.Stats.Errors + reviewResult.Stats.Warnings + reviewResult.Stats.Suggestions,
			Errors:      reviewResult.Stats.Errors,
			Warnings:    reviewResult.Stats.Warnings,
			Suggestions: reviewResult.Stats.Suggestions,
		},
		Comments:        comments,
		QualityMetrics:  qualityMetrics,
	}
	return result
}

func (wp *WorkerPool) postComments(job types.ReviewJob, result types.ReviewResult) error {
	switch job.Platform {
	case types.PlatformGitHub:
		// Get token from installation
		inst, err := wp.db.GetInstallationByOrgID(job.OrgID)
		if err != nil {
			return fmt.Errorf("failed to get installation: %w", err)
		}
		if inst.Token == "" {
			return fmt.Errorf("GitHub token not available")
		}

		// Initialize comment service with token
		githubCommentSvc := comment.NewGitHubCommentService(inst.Token)
		return githubCommentSvc.PostReview(
			job.Repo.Owner,
			job.Repo.Name,
			job.PR.Number,
			job.PR.HeadSHA,
			&result,
		)
	case types.PlatformGitLab:
		// GitLab would need project ID and token
		return fmt.Errorf("GitLab posting not fully implemented")
	default:
		return fmt.Errorf("unsupported platform: %s", job.Platform)
	}
}

func (wp *WorkerPool) updateReviewStatusFailed(reviewID string, errorMsg string, startTime time.Time) {
	duration := int(time.Since(startTime).Milliseconds())
	errorResult := map[string]interface{}{
		"error": errorMsg,
	}
	resultJSON, _ := json.Marshal(errorResult)
	resultStr := string(resultJSON)
	_ = wp.db.UpdateReviewStatus(reviewID, types.ReviewStatusFailed, &resultStr, &duration)
}

func (wp *WorkerPool) handleCommandJob(ctx context.Context, t *asynq.Task) error {
	var job types.CommandJob
	if err := json.Unmarshal(t.Payload(), &job); err != nil {
		return fmt.Errorf("failed to unmarshal job: %w", err)
	}

	log.Info().
		Str("job_id", job.ID).
		Str("command", job.Command.Name).
		Msg("Processing command job")

	// Log usage
	_ = wp.db.LogUsage(job.OrgID, "command", map[string]interface{}{
		"command": job.Command.Name,
		"args":    job.Command.Args,
		"pr":      job.PR.Number,
	})

	// Get repository
	repos, err := wp.db.GetRepositoriesByOrgID(job.OrgID)
	if err != nil {
		return fmt.Errorf("failed to get repositories: %w", err)
	}

	var repo *types.Repository
	for _, r := range repos {
		if r.FullName == job.Repo.FullName {
			repo = r
			break
		}
	}

	if repo == nil {
		return fmt.Errorf("repository not found: %s", job.Repo.FullName)
	}

	// Create command context
	cmdContext := commands.CommandContext{
		OrgID:    job.OrgID,
		RepoID:   repo.ID,
		Repo:     job.Repo,
		PR:       job.PR,
		Platform: job.Platform,
	}

	// Create command
	cmd := commands.Command{
		Name: job.Command.Name,
		Args: job.Command.Args,
	}

	// Route and execute command
	response, err := wp.commandRouter.Route(cmd, cmdContext)
	if err != nil {
		log.Error().Err(err).Str("command", cmd.Name).Msg("Command execution failed")
		response = fmt.Sprintf("❌ Error executing command: %s", err.Error())
	}

	// Post response as comment
	if err := wp.postCommandResponse(job, response); err != nil {
		log.Error().Err(err).Msg("Failed to post command response")
		return fmt.Errorf("failed to post response: %w", err)
	}

	log.Info().
		Str("job_id", job.ID).
		Str("command", cmd.Name).
		Msg("Command job completed")

	return nil
}

func (wp *WorkerPool) postCommandResponse(job types.CommandJob, response string) error {
	switch job.Platform {
	case types.PlatformGitHub:
		inst, err := wp.db.GetInstallationByOrgID(job.OrgID)
		if err != nil {
			return fmt.Errorf("failed to get installation: %w", err)
		}
		if inst.Token == "" {
			return fmt.Errorf("GitHub token not available")
		}

		githubCommentSvc := comment.NewGitHubCommentService(inst.Token)
		return githubCommentSvc.PostComment(
			job.Repo.Owner,
			job.Repo.Name,
			job.PR.Number,
			response,
		)
	case types.PlatformGitLab:
		return fmt.Errorf("GitLab command responses not yet implemented")
	default:
		return fmt.Errorf("unsupported platform: %s", job.Platform)
	}
}
