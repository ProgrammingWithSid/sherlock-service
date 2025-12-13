package api

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/sherlock/service/internal/database"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/gitlab"
)

type AuthHandler struct {
	db          *database.DB
	githubOAuth *oauth2.Config
	gitlabOAuth *oauth2.Config
}

func NewAuthHandler(db *database.DB, githubClientID, githubClientSecret, gitlabClientID, gitlabClientSecret, baseURL string) *AuthHandler {
	githubOAuth := &oauth2.Config{
		ClientID:     githubClientID,
		ClientSecret: githubClientSecret,
		RedirectURL:  fmt.Sprintf("%s/api/v1/auth/github/callback", baseURL),
		Scopes:       []string{"repo", "read:org", "read:user"},
		Endpoint:     github.Endpoint,
	}

	gitlabOAuth := &oauth2.Config{
		ClientID:     gitlabClientID,
		ClientSecret: gitlabClientSecret,
		RedirectURL:  fmt.Sprintf("%s/api/v1/auth/gitlab/callback", baseURL),
		Scopes:       []string{"api", "read_repository", "write_repository"},
		Endpoint:     gitlab.Endpoint,
	}

	return &AuthHandler{
		db:          db,
		githubOAuth: githubOAuth,
		gitlabOAuth: gitlabOAuth,
	}
}

func (h *AuthHandler) RegisterRoutes(r chi.Router) {
	r.Route("/auth", func(r chi.Router) {
		r.Get("/github", h.InitiateGitHubOAuth)
		r.Get("/github/callback", h.GitHubCallback)
		r.Get("/gitlab", h.InitiateGitLabOAuth)
		r.Get("/gitlab/callback", h.GitLabCallback)
		r.Get("/me", h.GetCurrentUser)
		r.Post("/logout", h.Logout)
	})
}

// InitiateGitHubOAuth starts the GitHub OAuth flow
func (h *AuthHandler) InitiateGitHubOAuth(w http.ResponseWriter, r *http.Request) {
	state := generateStateToken()
	redirectURI := r.URL.Query().Get("redirect_uri")
	if redirectURI == "" {
		redirectURI = "/"
	}

	// Store state and redirect_uri in cookies
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600, // 10 minutes
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_redirect_uri",
		Value:    redirectURI,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600, // 10 minutes
	})

	url := h.githubOAuth.AuthCodeURL(state, oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// InitiateGitLabOAuth starts the GitLab OAuth flow
func (h *AuthHandler) InitiateGitLabOAuth(w http.ResponseWriter, r *http.Request) {
	state := generateStateToken()
	redirectURI := r.URL.Query().Get("redirect_uri")
	if redirectURI == "" {
		redirectURI = "/"
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_redirect_uri",
		Value:    redirectURI,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600,
	})

	url := h.gitlabOAuth.AuthCodeURL(state, oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// GitHubCallback handles GitHub OAuth callback
func (h *AuthHandler) GitHubCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	// Verify state
	cookie, err := r.Cookie("oauth_state")
	if err != nil || cookie.Value != state {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid state"})
		return
	}

	// Exchange code for token
	token, err := h.githubOAuth.Exchange(context.Background(), code)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to exchange token"})
		return
	}

	// Get user info from GitHub
	user, _, err := h.getGitHubUserInfo(token.AccessToken)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to get user info"})
		return
	}

	// Create or get organization
	// Generate slug from username and try to find existing org by slug
	slug := sanitizeSlug(user.Login)
	if slug == "" {
		slug = fmt.Sprintf("user-%d", user.ID)
	}

	// Try to find organization by slug first
	org, err := h.db.GetOrganizationBySlug(slug)
	if err != nil {
		// Organization doesn't exist, create it
		org, err = h.db.CreateOrganization(user.Login, slug)
		if err != nil {
			// If slug already exists (race condition), try with user ID appended
			slug = fmt.Sprintf("%s-%d", slug, user.ID)
			org, err = h.db.CreateOrganization(user.Login, slug)
			if err != nil {
				log.Error().Err(err).Str("slug", slug).Str("username", user.Login).Msg("Failed to create organization")
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, map[string]string{"error": fmt.Sprintf("Failed to create organization: %v", err)})
				return
			}
		}
	}

	// Store GitHub installation/token
	installationID := int64(0) // Would get from GitHub API
	expiresAt := token.Expiry
	h.db.CreateOrUpdateInstallation(org.ID, installationID, token.AccessToken, &expiresAt)

	// Create session token
	sessionToken := generateSessionToken()

	// Store session (simplified - in production use Redis or database)
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    sessionToken,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400 * 7, // 7 days
	})

	// Get redirect URI from cookie
	redirectCookie, err := r.Cookie("oauth_redirect_uri")
	frontendURL := "/"
	if err == nil && redirectCookie != nil {
		frontendURL = redirectCookie.Value
	}

	// Clear the redirect cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_redirect_uri",
		Value:    "",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	http.Redirect(w, r, fmt.Sprintf("%s?token=%s&org_id=%s", frontendURL, sessionToken, org.ID), http.StatusTemporaryRedirect)
}

// GitLabCallback handles GitLab OAuth callback
func (h *AuthHandler) GitLabCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	// Verify state
	cookie, err := r.Cookie("oauth_state")
	if err != nil || cookie.Value != state {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid state"})
		return
	}

	// Exchange code for token
	token, err := h.gitlabOAuth.Exchange(context.Background(), code)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to exchange token"})
		return
	}

	// Get user info from GitLab
	user, _, err := h.getGitLabUserInfo(token.AccessToken)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to get user info"})
		return
	}

	// Create or get organization
	// Generate slug from username and try to find existing org by slug
	slug := sanitizeSlug(user.Username)
	if slug == "" {
		slug = fmt.Sprintf("user-%d", user.ID)
	}

	// Try to find organization by slug first
	org, err := h.db.GetOrganizationBySlug(slug)
	if err != nil {
		// Organization doesn't exist, create it
		org, err = h.db.CreateOrganization(user.Username, slug)
		if err != nil {
			// If slug already exists (race condition), try with user ID appended
			slug = fmt.Sprintf("%s-%d", slug, user.ID)
			org, err = h.db.CreateOrganization(user.Username, slug)
			if err != nil {
				log.Error().Err(err).Str("slug", slug).Str("username", user.Username).Msg("Failed to create organization")
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, map[string]string{"error": fmt.Sprintf("Failed to create organization: %v", err)})
				return
			}
		}
	}

	// Create session token
	sessionToken := generateSessionToken()

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    sessionToken,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400 * 7,
	})

	// Get redirect URI from cookie
	redirectCookie, err := r.Cookie("oauth_redirect_uri")
	frontendURL := "/"
	if err == nil && redirectCookie != nil {
		frontendURL = redirectCookie.Value
	}

	// Clear the redirect cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_redirect_uri",
		Value:    "",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	http.Redirect(w, r, fmt.Sprintf("%s?token=%s&org_id=%s", frontendURL, sessionToken, org.ID), http.StatusTemporaryRedirect)
}

// GetCurrentUser returns current authenticated user
func (h *AuthHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	// Get org ID from header (frontend sends this)
	orgID := r.Header.Get("X-Org-ID")
	if orgID == "" {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{"error": "Organization ID required"})
		return
	}

	org, err := h.db.GetOrganizationByID(orgID)
	if err != nil {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{"error": "Organization not found"})
		return
	}

	// Get token from cookie if available
	token := ""
	if sessionTokenCookie, err := r.Cookie("session_token"); err == nil {
		token = sessionTokenCookie.Value
	}

	response := map[string]interface{}{
		"org_id": org.ID,
		"name":   org.Name,
		"plan":   org.Plan,
		"token":  token,
	}

	render.JSON(w, r, response)
}

// Logout logs out the current user
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	render.JSON(w, r, map[string]string{"status": "logged_out"})
}

type GitHubUser struct {
	Login string `json:"login"`
	ID    int64  `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

func (h *AuthHandler) getGitHubUserInfo(token string) (*GitHubUser, string, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	var user GitHubUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, "", err
	}

	// Use user ID as org ID (simplified - in production would handle orgs differently)
	orgID := fmt.Sprintf("org_%d", user.ID)
	return &user, orgID, nil
}

type GitLabUser struct {
	Username string `json:"username"`
	ID       int64  `json:"id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
}

func (h *AuthHandler) getGitLabUserInfo(token string) (*GitLabUser, string, error) {
	req, err := http.NewRequest("GET", "https://gitlab.com/api/v4/user", nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	var user GitLabUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, "", err
	}

	orgID := fmt.Sprintf("org_%d", user.ID)
	return &user, orgID, nil
}

func generateStateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func generateSessionToken() string {
	return uuid.New().String()
}

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
