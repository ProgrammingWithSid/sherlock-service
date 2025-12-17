package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/rs/zerolog/log"
	"github.com/sherlock/service/internal/config"
	"github.com/sherlock/service/internal/database"
	"github.com/sherlock/service/internal/queue"
	repoconfig "github.com/sherlock/service/internal/services/config"
	"github.com/sherlock/service/internal/services/metrics"
	"github.com/sherlock/service/internal/types"
)

type Handler struct {
	db             *database.DB
	reviewQueue    *queue.ReviewQueue
	config         *config.Config
	metricsService *metrics.MetricsService
}

func NewHandler(db *database.DB, reviewQueue *queue.ReviewQueue, cfg *config.Config, metricsService *metrics.MetricsService) *Handler {
	return &Handler{
		db:             db,
		reviewQueue:    reviewQueue,
		config:         cfg,
		metricsService: metricsService,
	}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	// Apply authentication middleware
	r.Use(RequireOrgID)

	r.Route("/reviews", func(r chi.Router) {
		r.Get("/", h.ListReviews)
		r.Get("/{id}", h.GetReview)
		r.Post("/{id}/retry", h.RetryReview)
		r.Delete("/{id}", h.CancelReview)
	})

	r.Route("/repositories", func(r chi.Router) {
		r.Get("/", h.ListRepositories)
		r.Post("/", h.ConnectRepository)
		r.Get("/{id}", h.GetRepository)
		r.Put("/{id}/active", h.SetRepositoryActive)
	})

	r.Route("/repos", func(r chi.Router) {
		r.Route("/{owner}/{repo}", func(r chi.Router) {
			r.Get("/reviews", h.ListRepoReviews)
			r.Get("/config", h.GetRepoConfig)
			r.Put("/config", h.UpdateRepoConfig)
		})
	})

	r.Route("/stats", func(r chi.Router) {
		r.Get("/", h.GetStats)
	})

	r.Route("/queue", func(r chi.Router) {
		r.Get("/status", h.GetQueueStatus)
	})

	r.Route("/metrics", func(r chi.Router) {
		r.Get("/", h.GetMetrics)
	})
}

func (h *Handler) ListReviews(w http.ResponseWriter, r *http.Request) {
	orgID := r.Header.Get("X-Org-ID")
	if orgID == "" {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{"error": "Organization ID required"})
		return
	}

	limit := 50
	offset := 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil {
			limit = parsedLimit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil {
			offset = parsedOffset
		}
	}

	reviews, err := h.db.GetReviewsByOrgID(orgID, limit, offset)
	if err != nil {
		log.Error().Err(err).Str("org_id", orgID).Int("limit", limit).Int("offset", offset).Msg("Failed to get reviews")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	render.JSON(w, r, reviews)
}

func (h *Handler) GetReview(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	review, err := h.db.GetReviewByID(id)
	if err != nil {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	render.JSON(w, r, review)
}

func (h *Handler) RetryReview(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	review, err := h.db.GetReviewByID(id)
	if err != nil {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	repo, err := h.db.GetRepositoryByID(review.RepoID)
	if err != nil {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "Repository not found"})
		return
	}

	// Parse owner and name from FullName
	parts := strings.Split(repo.FullName, "/")
	if len(parts) != 2 {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid repository full name"})
		return
	}
	owner := parts[0]
	repoName := parts[1]

	// Construct clone URL based on platform
	var cloneURL string
	switch repo.Platform {
	case types.PlatformGitHub:
		cloneURL = fmt.Sprintf("https://github.com/%s.git", repo.FullName)
	case types.PlatformGitLab:
		// GitLab clone URL format: https://gitlab.com/{owner}/{repo}.git
		cloneURL = fmt.Sprintf("https://gitlab.com/%s.git", repo.FullName)
	default:
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Unsupported platform"})
		return
	}

	// Get base branch - try to get from review result or default to "main"
	baseBranch := "main"
	if review.Result != "" {
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(review.Result), &result); err == nil {
			if pr, ok := result["pr"].(map[string]interface{}); ok {
				if bb, ok := pr["baseBranch"].(string); ok && bb != "" {
					baseBranch = bb
				}
			}
		}
	}

	job := &types.ReviewJob{
		ID:       id,
		Type:     "full_review",
		Platform: repo.Platform,
		OrgID:    review.OrgID,
		Repo: types.RepoInfo{
			Owner:    owner,
			Name:     repoName,
			FullName: repo.FullName,
			CloneURL: cloneURL,
		},
		PR: types.PRInfo{
			Number:     review.PRNumber,
			HeadSHA:   review.HeadSHA,
			BaseBranch: baseBranch,
		},
	}

	_, err = h.reviewQueue.EnqueueReviewJob(job, 1)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	render.JSON(w, r, map[string]string{"status": "queued"})
}

func (h *Handler) CancelReview(w http.ResponseWriter, r *http.Request) {
	render.Status(r, http.StatusNotImplemented)
	render.JSON(w, r, map[string]string{"error": "Not implemented"})
}

func (h *Handler) ListRepoReviews(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoName := chi.URLParam(r, "repo")
	fullName := owner + "/" + repoName

	orgID := r.Header.Get("X-Org-ID")
	if orgID == "" {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{"error": "Organization ID required"})
		return
	}

	repos, err := h.db.GetRepositoriesByOrgID(orgID)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	var repo *types.Repository
	for _, r := range repos {
		if r.FullName == fullName {
			repo = r
			break
		}
	}

	if repo == nil {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "Repository not found"})
		return
	}

	limit := 50
	offset := 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil {
			limit = parsedLimit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil {
			offset = parsedOffset
		}
	}

	reviews, err := h.db.GetReviewsByRepoID(repo.ID, limit, offset)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	render.JSON(w, r, reviews)
}

func (h *Handler) GetRepoConfig(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoName := chi.URLParam(r, "repo")
	fullName := owner + "/" + repoName

	orgID := r.Header.Get("X-Org-ID")
	if orgID == "" {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{"error": "Organization ID required"})
		return
	}

	repos, err := h.db.GetRepositoriesByOrgID(orgID)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	var repo *types.Repository
	for _, r := range repos {
		if r.FullName == fullName {
			repo = r
			break
		}
	}

	if repo == nil {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "Repository not found"})
		return
	}

	// Use config loader to properly parse and return config with defaults
	loader := repoconfig.NewLoader()
	repoConfig, err := loader.LoadFromJSON(repo.Config)
	if err != nil {
		log.Warn().Err(err).Str("repo_id", repo.ID).Msg("Failed to parse repo config, using defaults")
		// Load defaults by passing empty JSON
		repoConfig, _ = loader.LoadFromJSON("{}")
	}

	// Convert to JSON for response
	configJSON, err := json.Marshal(repoConfig)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to marshal config"})
		return
	}

	var config map[string]interface{}
	if err := json.Unmarshal(configJSON, &config); err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to parse config"})
		return
	}

	render.JSON(w, r, config)
}

func (h *Handler) UpdateRepoConfig(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoName := chi.URLParam(r, "repo")
	fullName := owner + "/" + repoName

	orgID := r.Header.Get("X-Org-ID")
	if orgID == "" {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{"error": "Organization ID required"})
		return
	}

	repos, err := h.db.GetRepositoriesByOrgID(orgID)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	var repo *types.Repository
	for _, r := range repos {
		if r.FullName == fullName {
			repo = r
			break
		}
	}

	if repo == nil {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "Repository not found"})
		return
	}

	var config map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid JSON"})
		return
	}

	// Load existing config and merge with new values
	loader := repoconfig.NewLoader()
	existingConfig, err := loader.LoadFromJSON(repo.Config)
	if err != nil {
		log.Warn().Err(err).Str("repo_id", repo.ID).Msg("Failed to parse existing config, using defaults")
		existingConfig, _ = loader.LoadFromJSON("{}")
	}

	// Update rules if provided
	if rules, ok := config["rules"].([]interface{}); ok {
		rulesStr := make([]string, 0, len(rules))
		for _, r := range rules {
			if ruleStr, ok := r.(string); ok && ruleStr != "" {
				rulesStr = append(rulesStr, ruleStr)
			}
		}
		existingConfig.Rules = rulesStr
	}

	// Update other config fields if provided (for future extensibility)
	if aiConfig, ok := config["ai"].(map[string]interface{}); ok {
		if existingConfig.AI == nil {
			existingConfig.AI = &repoconfig.AIConfig{}
		}
		if provider, ok := aiConfig["provider"].(string); ok {
			existingConfig.AI.Provider = provider
		}
		if model, ok := aiConfig["model"].(string); ok {
			existingConfig.AI.Model = model
		}
	}

	// Convert to JSON for storage
	configJSON, err := json.Marshal(existingConfig)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to marshal config"})
		return
	}

	if err := h.db.UpdateRepositoryConfig(repo.ID, string(configJSON)); err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	render.JSON(w, r, map[string]string{"status": "updated"})
}

func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	orgID := r.Header.Get("X-Org-ID")
	if orgID == "" {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{"error": "Organization ID required"})
		return
	}

	// Get usage stats for current month
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	endOfMonth := startOfMonth.AddDate(0, 1, 0).Add(-time.Second)

	stats, err := h.db.GetUsageStats(orgID, startOfMonth, endOfMonth)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	// Get organization for plan info
	org, err := h.db.GetOrganizationByID(orgID)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Organization not found"})
		return
	}

	response := map[string]interface{}{
		"reviews_this_month": stats.ReviewsThisMonth,
		"commands_this_month": stats.CommandsThisMonth,
		"total_reviews": stats.TotalReviews,
		"total_commands": stats.TotalCommands,
		"total_api_calls": stats.TotalAPICalls,
		"plan": org.Plan,
		"period": map[string]interface{}{
			"start": startOfMonth.Format(time.RFC3339),
			"end":   endOfMonth.Format(time.RFC3339),
		},
	}

	render.JSON(w, r, response)
}

func (h *Handler) ListRepositories(w http.ResponseWriter, r *http.Request) {
	orgID := r.Header.Get("X-Org-ID")
	if orgID == "" {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{"error": "Organization ID required"})
		return
	}

	repos, err := h.db.GetRepositoriesByOrgID(orgID)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	render.JSON(w, r, repos)
}

func (h *Handler) GetRepository(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	repo, err := h.db.GetRepositoryByID(id)
	if err != nil {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	render.JSON(w, r, repo)
}

func (h *Handler) ConnectRepository(w http.ResponseWriter, r *http.Request) {
	orgID := r.Header.Get("X-Org-ID")
	if orgID == "" {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{"error": "Organization ID required"})
		return
	}

	var body struct {
		Platform string `json:"platform"`
		Owner    string `json:"owner"`
		Repo     string `json:"repo"`
		URL      string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid JSON"})
		return
	}

	// Parse URL if provided
	if body.URL != "" {
		// Extract owner/repo from URL
		// Format: https://github.com/owner/repo or https://gitlab.com/owner/repo
		// This is a simplified parser - in production, use proper URL parsing
		parts := strings.Split(strings.TrimSuffix(body.URL, ".git"), "/")
		if len(parts) >= 2 {
			body.Owner = parts[len(parts)-2]
			body.Repo = parts[len(parts)-1]
			if body.Platform == "" {
				if strings.Contains(body.URL, "gitlab.com") {
					body.Platform = "gitlab"
				} else {
					body.Platform = "github"
				}
			}
		}
	}

	if body.Owner == "" || body.Repo == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Owner and repo are required"})
		return
	}

	if body.Platform == "" {
		body.Platform = "github"
	}

	// Check if repository already exists
	repos, err := h.db.GetRepositoriesByOrgID(orgID)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	fullName := fmt.Sprintf("%s/%s", body.Owner, body.Repo)
	for _, repo := range repos {
		if repo.FullName == fullName {
			render.JSON(w, r, repo)
			return
		}
	}

	// Create new repository
	repo := &types.Repository{
		OrgID:      orgID,
		Platform:   types.Platform(body.Platform),
		ExternalID: fmt.Sprintf("%s/%s", body.Owner, body.Repo),
		Name:       body.Repo,
		FullName:   fullName,
		IsPrivate:  false, // Would need to fetch from GitHub/GitLab API
		IsActive:   true,
		Config:     "{}",
	}

	if err := h.db.CreateRepository(repo); err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, repo)
}

func (h *Handler) SetRepositoryActive(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var body struct {
		IsActive bool `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid JSON"})
		return
	}

	if err := h.db.SetRepositoryActive(id, body.IsActive); err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	render.JSON(w, r, map[string]string{"status": "updated"})
}

func (h *Handler) GetQueueStatus(w http.ResponseWriter, r *http.Request) {
	// Get queue stats from Asynq inspector
	inspector := h.reviewQueue.GetInspector()
	if inspector == nil {
		render.Status(r, http.StatusServiceUnavailable)
		render.JSON(w, r, map[string]string{"error": "Queue inspector not available"})
		return
	}

	// Get queue information
	queueInfo, err := inspector.GetQueueInfo("review")
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	response := map[string]interface{}{
		"queue":    "review",
		"pending":  queueInfo.Pending,
		"active":   queueInfo.Active,
		"scheduled": queueInfo.Scheduled,
		"retry":    queueInfo.Retry,
		"archived": queueInfo.Archived,
	}

	render.JSON(w, r, response)
}

func (h *Handler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	if h.metricsService == nil {
		render.Status(r, http.StatusServiceUnavailable)
		render.JSON(w, r, map[string]string{"error": "Metrics service not available"})
		return
	}

	reviewMetrics, err := h.metricsService.GetReviewMetrics()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get review metrics")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to get metrics"})
		return
	}

	cacheHitRate := h.metricsService.GetCacheHitRate()
	successRate := h.metricsService.GetSuccessRate()

	response := map[string]interface{}{
		"reviews": map[string]interface{}{
			"total":            reviewMetrics.TotalReviews,
			"successful":       reviewMetrics.SuccessfulReviews,
			"failed":           reviewMetrics.FailedReviews,
			"average_duration_ms": reviewMetrics.AverageDuration,
			"cache_hits":       reviewMetrics.CacheHits,
			"cache_misses":     reviewMetrics.CacheMisses,
			"incremental":      reviewMetrics.IncrementalReviews,
			"full":             reviewMetrics.FullReviews,
		},
		"rates": map[string]interface{}{
			"cache_hit_rate": cacheHitRate,
			"success_rate":    successRate,
		},
	}

	// Add quality metrics if available
	if reviewMetrics.TotalQualityScores > 0 {
		response["quality"] = map[string]interface{}{
			"average_score":    reviewMetrics.AverageQualityScore,
			"total_scores":    reviewMetrics.TotalQualityScores,
			"score_percentage": reviewMetrics.AverageQualityScore,
		}
	}

	render.JSON(w, r, response)
}
