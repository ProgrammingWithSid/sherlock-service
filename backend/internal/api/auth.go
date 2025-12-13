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
	"github.com/sherlock/service/internal/types"
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
		r.Post("/signup", h.Signup)
		r.Post("/login", h.Login)
		r.Get("/github", h.InitiateGitHubOAuth)
		r.Get("/github/callback", h.GitHubCallback)
		r.Get("/gitlab", h.InitiateGitLabOAuth)
		r.Get("/gitlab/callback", h.GitLabCallback)
		r.Get("/me", h.GetCurrentUser)
		r.Get("/organizations", h.ListOrganizations)
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

	// Get or create user
	dbUser, err := h.db.GetUserByEmail(user.Email)
	if err != nil {
		// User doesn't exist, create one
		// Use a random password since OAuth users don't need passwords
		randomPassword := generateSessionToken() // Use session token generator for random string
		dbUser, err = h.db.CreateUser(user.Email, randomPassword, user.Name, types.RoleOrgAdmin, &org.ID)
		if err != nil {
			log.Error().Err(err).Str("email", user.Email).Msg("Failed to create user")
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, map[string]string{"error": "Failed to create user"})
			return
		}
	}

	// Create session token
	sessionToken := generateSessionToken()
	sessionStore.Set(sessionToken, dbUser.ID, string(dbUser.Role), &org.ID)

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
	sessionStore.Set(sessionToken, user.ID, string(user.Role), &org.ID)

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
	token := getSessionToken(r)
	if token == "" {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{"error": "Authentication required"})
		return
	}

	session, ok := sessionStore.Get(token)
	if !ok {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{"error": "Invalid or expired session"})
		return
	}

	// Get user
	user, err := h.db.GetUserByID(session.UserID)
	if err != nil {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{"error": "User not found"})
		return
	}

	response := map[string]interface{}{
		"org_id": user.OrgID,
		"name":   user.Name,
		"role":   user.Role,
		"token":  token,
	}

	// Get organization if user has one
	if user.OrgID != nil {
		org, err := h.db.GetOrganizationByID(*user.OrgID)
		if err == nil {
			response["plan"] = org.Plan
		}
	}

	render.JSON(w, r, response)
}

// SignupRequest represents signup request
type SignupRequest struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	Name       string `json:"name"`
	OrgName    string `json:"org_name"`
	OrgSlug    string `json:"org_slug"`
}

// Signup handles user and organization signup
func (h *AuthHandler) Signup(w http.ResponseWriter, r *http.Request) {
	var req SignupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid request"})
		return
	}

	// Validate input
	if req.Email == "" || req.Password == "" || req.Name == "" || req.OrgName == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "All fields are required"})
		return
	}

	// Generate slug if not provided
	slug := req.OrgSlug
	if slug == "" {
		slug = sanitizeSlug(req.OrgName)
		if slug == "" {
			slug = uuid.New().String()[:8]
		}
	}

	// Create organization
	org, err := h.db.CreateOrganization(req.OrgName, slug)
	if err != nil {
		// If slug exists, try with UUID
		slug = fmt.Sprintf("%s-%s", slug, uuid.New().String()[:8])
		org, err = h.db.CreateOrganization(req.OrgName, slug)
		if err != nil {
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, map[string]string{"error": "Failed to create organization"})
			return
		}
	}

	// Create user
	user, err := h.db.CreateUser(req.Email, req.Password, req.Name, types.RoleOrgAdmin, &org.ID)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to create user"})
		return
	}

	// Create session token
	sessionToken := generateSessionToken()
	sessionStore.Set(sessionToken, user.ID, string(user.Role), &org.ID)

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    sessionToken,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400 * 7,
	})

	response := map[string]interface{}{
		"user": map[string]interface{}{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
			"role":  user.Role,
			"org_id": org.ID,
		},
		"organization": map[string]interface{}{
			"id":   org.ID,
			"name": org.Name,
			"slug": org.Slug,
			"plan": org.Plan,
		},
		"token": sessionToken,
	}

	render.JSON(w, r, response)
}

// LoginRequest represents login request
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Login handles user login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid request"})
		return
	}

	// Get user
	user, err := h.db.GetUserByEmail(req.Email)
	if err != nil {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{"error": "Invalid credentials"})
		return
	}

	// Verify password
	if !database.VerifyPassword(user.PasswordHash, req.Password) {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{"error": "Invalid credentials"})
		return
	}

	if !user.IsActive {
		render.Status(r, http.StatusForbidden)
		render.JSON(w, r, map[string]string{"error": "Account is inactive"})
		return
	}

	// Get organization if user has one
	var org *types.Organization
	if user.OrgID != nil {
		org, err = h.db.GetOrganizationByID(*user.OrgID)
		if err != nil {
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, map[string]string{"error": "Organization not found"})
			return
		}
	}

	// Create session token
	sessionToken := generateSessionToken()
	sessionStore.Set(sessionToken, user.ID, string(user.Role), user.OrgID)

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    sessionToken,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400 * 7,
	})

	response := map[string]interface{}{
		"user": map[string]interface{}{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
			"role":  user.Role,
			"org_id": user.OrgID,
		},
		"token": sessionToken,
	}

	if org != nil {
		response["organization"] = map[string]interface{}{
			"id":   org.ID,
			"name": org.Name,
			"slug": org.Slug,
			"plan": org.Plan,
		}
	}

	render.JSON(w, r, response)
}

// ListOrganizations returns organizations accessible by the current user
// Super admins see all organizations, regular users only see their own organization
func (h *AuthHandler) ListOrganizations(w http.ResponseWriter, r *http.Request) {
	// Get user info from session (set by RequireAuth middleware if authenticated)
	// If not authenticated, try to get from session token
	token := getSessionToken(r)
	var userID, userRole string
	var userOrgID *string

	if token != "" {
		if session, ok := sessionStore.Get(token); ok {
			userID = session.UserID
			userRole = session.Role
			userOrgID = session.OrgID
		}
	}

	// If no session, try to get from context (set by middleware)
	if userID == "" {
		if ctxUserID, ok := r.Context().Value("user_id").(string); ok {
			userID = ctxUserID
		}
		if ctxRole, ok := r.Context().Value("user_role").(string); ok {
			userRole = ctxRole
		}
		if ctxOrgID, ok := r.Context().Value("org_id").(string); ok {
			userOrgID = &ctxOrgID
		}
	}

	// If still no user info, return empty list (or could require auth)
	if userID == "" {
		render.JSON(w, r, []*types.Organization{})
		return
	}

	orgs, err := h.db.GetOrganizationsByUserID(userID, userRole, userOrgID)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	render.JSON(w, r, orgs)
}

// Logout logs out the current user
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	token := getSessionToken(r)
	if token != "" {
		sessionStore.Delete(token)
	}

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
