package workers

import (
	"context"
	"encoding/json"
	"fmt"
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
	"github.com/sherlock/service/internal/services/indexer"
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

	// Initialize incremental review service
	incrementalReviewSvc := review.NewIncrementalReviewService(
		gitService,
		reviewCache,
		reviewService,
	)

	// Initialize codebase indexer (with chunkyyy integration)
	codebaseIndexer := indexer.NewCodebaseIndexer(db, cfg.ReposPath, "")

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

	// Step 6: Run code-sherlock review (use incremental if enabled)
	var reviewResult *review.ReviewResult

	if wp.config.EnableIncrementalReviews && repo != nil {
		// Use incremental review service
		logger.Info().Msg("Using incremental review service")
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
			// Fall back to full review
			reviewReq := review.ReviewRequest{
				WorktreePath: worktreePath,
				TargetBranch: job.PR.HeadSHA,
				BaseBranch:   job.PR.BaseBranch,
				Config:       reviewConfig,
			}
			reviewResult, err = wp.reviewService.RunReview(reviewReq)
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

	// Step 11: Save result
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

	// Set rules from repo config
	if repoConfig != nil && len(repoConfig.Rules) > 0 {
		config.GlobalRules = repoConfig.Rules
	}

	// Get GitHub token from installation
	if job.Platform == types.PlatformGitHub {
		inst, err := wp.db.GetInstallationByOrgID(job.OrgID)
		if err == nil && inst.Token != "" {
			config.GitHub = &review.GitHubConfig{
				Token: inst.Token,
			}
		}
	}

	return config
}

func (wp *WorkerPool) convertReviewResult(reviewResult *review.ReviewResult) types.ReviewResult {
	comments := make([]types.ReviewComment, 0, len(reviewResult.Comments))
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

	return types.ReviewResult{
		Recommendation: recommendation,
		Summary: types.ReviewSummary{
			TotalIssues: reviewResult.Stats.Errors + reviewResult.Stats.Warnings + reviewResult.Stats.Suggestions,
			Errors:      reviewResult.Stats.Errors,
			Warnings:    reviewResult.Stats.Warnings,
			Suggestions: reviewResult.Stats.Suggestions,
		},
		Comments: comments,
	}
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
