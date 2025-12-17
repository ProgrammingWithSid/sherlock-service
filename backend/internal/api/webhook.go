package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/sherlock/service/internal/commands"
	"github.com/sherlock/service/internal/config"
	"github.com/sherlock/service/internal/database"
	"github.com/sherlock/service/internal/plan"
	"github.com/sherlock/service/internal/queue"
	"github.com/sherlock/service/internal/types"
)

type WebhookHandler struct {
	db          *database.DB
	reviewQueue *queue.ReviewQueue
	planService *plan.Service
	config      *config.Config
}

func NewWebhookHandler(db *database.DB, reviewQueue *queue.ReviewQueue, cfg *config.Config) *WebhookHandler {
	planService := plan.NewService(
		db.GetMonthlyReviewCount,
		db.GetRepoCount,
	)

	return &WebhookHandler{
		db:          db,
		reviewQueue: reviewQueue,
		planService: planService,
		config:      cfg,
	}
}

func (h *WebhookHandler) RegisterRoutes(r chi.Router) {
	r.Post("/github", h.HandleGitHubWebhook)
	r.Post("/gitlab", h.HandleGitLabWebhook)
}

func (h *WebhookHandler) HandleGitHubWebhook(w http.ResponseWriter, r *http.Request) {
	eventType := r.Header.Get("X-GitHub-Event")
	signature := r.Header.Get("X-Hub-Signature-256")

	log.Info().
		Str("event_type", eventType).
		Str("method", r.Method).
		Str("path", r.URL.Path).
		Int64("content_length", r.ContentLength).
		Msg("Received GitHub webhook")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error().Err(err).Msg("Failed to read webhook body")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Failed to read body"})
		return
	}

	if len(body) == 0 {
		log.Warn().Msg("Webhook body is empty")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Empty body"})
		return
	}

	log.Debug().Int("body_size", len(body)).Str("content_type", r.Header.Get("Content-Type")).Msg("Read webhook body")

	// Handle form-encoded payload (GitHub webhook form format)
	bodyStr := string(body)
	var jsonBody []byte

	if strings.HasPrefix(bodyStr, "payload=") {
		// Form-encoded format: payload=<url-encoded-json>
		log.Info().Msg("Detected form-encoded webhook payload")
		values, err := url.ParseQuery(bodyStr)
		if err != nil {
			log.Error().Err(err).Msg("Failed to parse form-encoded payload")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, map[string]string{"error": "Invalid form data"})
			return
		}

		payloadParam := values.Get("payload")
		if payloadParam == "" {
			log.Error().Msg("Form payload missing 'payload' parameter")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, map[string]string{"error": "Missing payload parameter"})
			return
		}

		// URL decode the payload
		decoded, err := url.QueryUnescape(payloadParam)
		if err != nil {
			log.Error().Err(err).Msg("Failed to URL decode payload")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, map[string]string{"error": "Failed to decode payload"})
			return
		}

		jsonBody = []byte(decoded)
	} else {
		// Raw JSON format
		jsonBody = body
	}

	// Verify signature (use original body for signature verification)
	if signature != "" && h.config.GitHubWebhookSecret != "" {
		if !verifyGitHubSignature(h.config.GitHubWebhookSecret, signature, body) {
			log.Warn().Msg("Invalid webhook signature")
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, map[string]string{"error": "Invalid signature"})
			return
		}
	} else {
		log.Warn().Msg("Webhook signature verification skipped (no secret configured)")
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(jsonBody, &payload); err != nil {
		bodyPreview := string(jsonBody)
		if len(bodyPreview) > 200 {
			bodyPreview = bodyPreview[:200]
		}
		log.Error().Err(err).Str("body_preview", bodyPreview).Msg("Failed to parse webhook JSON")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid JSON"})
		return
	}

	// Handle GitHub ping event (webhook test)
	if eventType == "ping" {
		log.Info().Msg("Received GitHub ping event (webhook test)")
		render.JSON(w, r, map[string]string{"status": "ok", "message": "Webhook configured successfully"})
		return
	}

	switch eventType {
	case "installation":
		// Handle GitHub App installation events
		if err := h.handleInstallation(payload); err != nil {
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, map[string]string{"error": err.Error()})
			return
		}

	case "pull_request":
		action, ok := payload["action"].(string)
		if !ok {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, map[string]string{"error": "Invalid action"})
			return
		}

		if action == "opened" || action == "synchronize" || action == "reopened" {
			if err := h.handlePullRequest(payload); err != nil {
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, map[string]string{"error": err.Error()})
				return
			}
		}

	case "issue_comment":
		if err := h.handleIssueComment(payload); err != nil {
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, map[string]string{"error": err.Error()})
			return
		}
	}

	render.JSON(w, r, map[string]string{"status": "ok"})
}

func (h *WebhookHandler) HandleGitLabWebhook(w http.ResponseWriter, r *http.Request) {
	render.Status(r, http.StatusNotImplemented)
	render.JSON(w, r, map[string]string{"error": "GitLab webhooks not yet implemented"})
}

// handleInstallation handles GitHub App installation events
func (h *WebhookHandler) handleInstallation(payload map[string]interface{}) error {
	action, ok := payload["action"].(string)
	if !ok {
		return fmt.Errorf("invalid installation action")
	}

	installationData, ok := payload["installation"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid installation data")
	}

	installationIDFloat, ok := installationData["id"].(float64)
	if !ok {
		return fmt.Errorf("invalid installation ID")
	}
	installationID := int64(installationIDFloat)

	accountData, ok := installationData["account"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid account data")
	}

	accountLogin, ok := accountData["login"].(string)
	if !ok {
		return fmt.Errorf("invalid account login")
	}

	log.Info().
		Str("action", action).
		Int64("installation_id", installationID).
		Str("account", accountLogin).
		Msg("Processing GitHub App installation event")

	switch action {
	case "created":
		// New installation - create organization and installation record
		slug := sanitizeSlug(accountLogin)
		if slug == "" {
			slug = fmt.Sprintf("org-%d", installationID)
		}

		// Try to find existing organization by slug
		org, err := h.db.GetOrganizationBySlug(slug)
		if err != nil {
			// Create new organization
			org, err = h.db.CreateOrganization(accountLogin, slug)
			if err != nil {
				return fmt.Errorf("failed to create organization: %w", err)
			}
			log.Info().Str("org_id", org.ID).Str("slug", slug).Msg("Created organization for GitHub App installation")
		}

		// Create or update installation record
		// Token will be fetched when needed using TokenService
		err = h.db.CreateOrUpdateInstallation(org.ID, installationID, "", nil)
		if err != nil {
			return fmt.Errorf("failed to create installation record: %w", err)
		}

		log.Info().
			Str("org_id", org.ID).
			Int64("installation_id", installationID).
			Msg("GitHub App installation created")

	case "deleted":
		// Installation removed - mark as inactive (don't delete to preserve history)
		inst, err := h.db.GetInstallationByID(installationID)
		if err != nil {
			log.Warn().Err(err).Int64("installation_id", installationID).Msg("Installation not found for deletion")
			return nil // Not an error if already deleted
		}

		log.Info().
			Str("org_id", inst.OrgID).
			Int64("installation_id", installationID).
			Msg("GitHub App installation deleted")

	case "suspend", "unsuspend":
		// Installation suspended/unsuspended
		log.Info().
			Str("action", action).
			Int64("installation_id", installationID).
			Msg("GitHub App installation status changed")
	}

	return nil
}

// sanitizeSlug converts a string to a URL-friendly slug
func sanitizeSlug(input string) string {
	// Convert to lowercase
	slug := strings.ToLower(input)

	// Remove special characters, keep only alphanumeric and hyphens
	reg := regexp.MustCompile("[^a-z0-9-]")
	slug = reg.ReplaceAllString(slug, "-")

	// Remove multiple consecutive hyphens
	reg = regexp.MustCompile("-+")
	slug = reg.ReplaceAllString(slug, "-")

	// Remove leading/trailing hyphens
	slug = strings.Trim(slug, "-")

	// Limit length
	if len(slug) > 50 {
		slug = slug[:50]
	}

	return slug
}

func (h *WebhookHandler) handlePullRequest(payload map[string]interface{}) error {
	prData, ok := payload["pull_request"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid pull_request data")
	}

	repoData, ok := payload["repository"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid repository data")
	}

	repoFullName, ok := repoData["full_name"].(string)
	if !ok {
		return fmt.Errorf("invalid repository full_name")
	}

	// Get organization from GitHub App installation
	var org *types.Organization
	var err error

	// Check if this is a GitHub App installation webhook
	if installationData, ok := payload["installation"].(map[string]interface{}); ok {
		installationID, ok := installationData["id"].(float64)
		if !ok {
			return fmt.Errorf("invalid installation ID")
		}

		// Get or create organization from installation
		inst, err := h.db.GetInstallationByID(int64(installationID))
		if err != nil {
			// Installation not found - create organization and installation
			org, createErr := h.db.CreateOrganization(
				fmt.Sprintf("Organization %d", int64(installationID)),
				fmt.Sprintf("org-%d", int64(installationID)),
			)
			if createErr != nil {
				return fmt.Errorf("failed to create organization: %w", createErr)
			}

			// Create installation record (token would be fetched separately)
			_ = h.db.CreateOrUpdateInstallation(org.ID, int64(installationID), "", nil)
			inst, err = h.db.GetInstallationByID(int64(installationID))
			if err != nil {
				return fmt.Errorf("failed to get installation: %w", err)
			}
		}

		org, err = h.db.GetOrganizationByID(inst.OrgID)
		if err != nil {
			return fmt.Errorf("organization not found: %w", err)
		}
	} else {
		// GitHub App installation not found in payload
		// This should not happen with GitHub App webhooks
		log.Warn().Str("repo", repoFullName).Msg("GitHub App installation not found in webhook payload")
		return fmt.Errorf("GitHub App installation required - please install the app on repository %s", repoFullName)
	}

	// Check plan limits
	canReview, reason := h.planService.CheckCanReview(org.ID, org.Plan)
	if !canReview {
		// Would post comment to PR here
		return fmt.Errorf("review limit: %s", reason)
	}

	prNumberFloat, ok := prData["number"].(float64)
	if !ok {
		return fmt.Errorf("invalid PR number")
	}
	prNumber := int(prNumberFloat)

	headData, ok := prData["head"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid head data")
	}

	headSHA, ok := headData["sha"].(string)
	if !ok {
		return fmt.Errorf("invalid head SHA")
	}

	baseData, ok := prData["base"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid base data")
	}

	baseRef, ok := baseData["ref"].(string)
	if !ok {
		return fmt.Errorf("invalid base ref")
	}

	// Get or create repository
	repos, err := h.db.GetRepositoriesByOrgID(org.ID)
	if err != nil {
		return fmt.Errorf("failed to get repositories: %w", err)
	}

	var repo *types.Repository
	for _, r := range repos {
		if r.FullName == repoFullName {
			repo = r
			break
		}
	}

	if repo == nil {
		// Create new repository
		repoIDFloat, ok := repoData["id"].(float64)
		if !ok {
			return fmt.Errorf("invalid repository ID")
		}
		repoID := fmt.Sprintf("%.0f", repoIDFloat)

		repoName, ok := repoData["name"].(string)
		if !ok {
			return fmt.Errorf("invalid repository name")
		}

		isPrivate, ok := repoData["private"].(bool)
		if !ok {
			isPrivate = false
		}

		repo = &types.Repository{
			OrgID:      org.ID,
			Platform:   types.PlatformGitHub,
			ExternalID: repoID,
			Name:       repoName,
			FullName:   repoFullName,
			IsPrivate:  isPrivate,
			IsActive:   true,
			Config:     "{}",
		}
		if err := h.db.CreateRepository(repo); err != nil {
			return fmt.Errorf("failed to create repository: %w", err)
		}
	}

	// Create review record
	review := &types.Review{
		OrgID:      org.ID,
		RepoID:     repo.ID,
		PRNumber:   prNumber,
		HeadSHA:    headSHA,
		Status:     types.ReviewStatusPending,
		AIProvider: h.planService.GetAIProvider(org.Plan, ""),
	}
	if err := h.db.CreateReview(review); err != nil {
		return fmt.Errorf("failed to create review: %w", err)
	}

	// Extract owner and clone URL
	ownerData, ok := repoData["owner"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid owner data")
	}
	ownerLogin, ok := ownerData["login"].(string)
	if !ok {
		return fmt.Errorf("invalid owner login")
	}

	// Try to get clone_url from payload, or construct it
	cloneURL, ok := repoData["clone_url"].(string)
	if !ok || cloneURL == "" {
		// Construct clone URL from repository full name
		// Format: https://github.com/{owner}/{repo}.git
		cloneURL = fmt.Sprintf("https://github.com/%s.git", repoFullName)
		log.Info().Str("clone_url", cloneURL).Str("repo", repoFullName).Msg("Constructed clone URL from full name")
	}

	// Enqueue review job
	job := &types.ReviewJob{
		ID:       review.ID,
		Type:     "full_review",
		Platform: types.PlatformGitHub,
		OrgID:    org.ID,
		Repo: types.RepoInfo{
			Owner:    ownerLogin,
			Name:     repo.Name,
			FullName: repo.FullName,
			CloneURL: cloneURL,
		},
		PR: types.PRInfo{
			Number:     prNumber,
			HeadSHA:   headSHA,
			BaseBranch: baseRef,
		},
	}

	priority := h.planService.GetQueuePriority(org.Plan)
	if _, err := h.reviewQueue.EnqueueReviewJob(job, priority); err != nil {
		return fmt.Errorf("failed to enqueue review job: %w", err)
	}

	return nil
}

func (h *WebhookHandler) handleIssueComment(payload map[string]interface{}) error {
	commentData, ok := payload["comment"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid comment data")
	}

	body, ok := commentData["body"].(string)
	if !ok {
		return fmt.Errorf("invalid comment body")
	}

	// Parse @sherlock commands
	parser := commands.NewParser("sherlock")
	if !parser.IsCommandComment(body) {
		return nil // Not a command comment
	}

	cmds, err := parser.ParseComment(body)
	if err != nil {
		return fmt.Errorf("failed to parse commands: %w", err)
	}

	if len(cmds) == 0 {
		return nil
	}

	// Get issue/PR information
	issueData, ok := payload["issue"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid issue data")
	}

	prNumberFloat, ok := issueData["number"].(float64)
	if !ok {
		return fmt.Errorf("invalid PR number")
	}
	prNumber := int(prNumberFloat)

	repoData, ok := payload["repository"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid repository data")
	}

	repoFullName, ok := repoData["full_name"].(string)
	if !ok {
		return fmt.Errorf("invalid repository full_name")
	}

	installationData, ok := payload["installation"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid installation data")
	}

	installationID, ok := installationData["id"].(float64)
	if !ok {
		return fmt.Errorf("invalid installation ID")
	}

	// Get organization from installation
	inst, err := h.db.GetInstallationByID(int64(installationID))
	if err != nil {
		return fmt.Errorf("installation not found: %w", err)
	}

	// Get repository
	repos, err := h.db.GetRepositoriesByOrgID(inst.OrgID)
	if err != nil {
		return fmt.Errorf("failed to get repositories: %w", err)
	}

	var repo *types.Repository
	for _, r := range repos {
		if r.FullName == repoFullName {
			repo = r
			break
		}
	}

	if repo == nil {
		return fmt.Errorf("repository not found")
	}

	// Get PR head SHA from comment context
	// For issue comments, we need to get the PR number and fetch the SHA
	// For now, we'll extract from the comment body or use a placeholder
	// In production, this would fetch from GitHub API using the PR number
	headSHA := ""
	if issueData, ok := payload["issue"].(map[string]interface{}); ok {
		if pullRequest, ok := issueData["pull_request"].(map[string]interface{}); ok {
			if head, ok := pullRequest["head"].(map[string]interface{}); ok {
				if sha, ok := head["sha"].(string); ok {
					headSHA = sha
				}
			}
		}
	}

	// If still empty, try to get from comment body or use PR number to fetch
	if headSHA == "" {
		// Fallback: would need to fetch from GitHub API using PR number
		// For now, we'll leave it empty and let the command handler fetch it
	}

	// Create command job for each command
	for _, cmd := range cmds {
		commandJob := &types.CommandJob{
			ID:       uuid.New().String(),
			Type:     "command",
			Platform: types.PlatformGitHub,
			OrgID:    inst.OrgID,
			Repo: types.RepoInfo{
				Owner:    repoFullName, // Would parse owner/repo
				Name:     repo.Name,
				FullName: repo.FullName,
			},
			PR: types.PRInfo{
				Number:     prNumber,
				HeadSHA:   headSHA,
				BaseBranch: "main", // Would fetch from PR
			},
			Comment: types.CommentInfo{
				ID:     int(commentData["id"].(float64)),
				Body:   body,
				Author: commentData["user"].(map[string]interface{})["login"].(string),
			},
			Command: types.CommandInfo{
				Name: cmd.Name,
				Args: cmd.Args,
			},
		}

		if _, err := h.reviewQueue.EnqueueCommandJob(commandJob); err != nil {
			log.Error().Err(err).Msg("Failed to enqueue command job")
			continue
		}
	}

	return nil
}

func verifyGitHubSignature(secret string, signature string, body []byte) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expectedSignature := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}
