package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/sherlock/service/internal/database"
	"github.com/sherlock/service/internal/types"
)

type AuthHandler struct {
	db *database.DB
}

func NewAuthHandler(db *database.DB) *AuthHandler {
	return &AuthHandler{
		db: db,
	}
}

func (h *AuthHandler) RegisterRoutes(r chi.Router) {
	r.Route("/auth", func(r chi.Router) {
		r.Post("/signup", h.Signup)
		r.Post("/login", h.Login)
		r.Get("/me", h.GetCurrentUser)
		r.Get("/organizations", h.ListOrganizations)
		r.Post("/logout", h.Logout)
		r.Get("/github/callback", h.GitHubCallback)
	})
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


func generateSessionToken() string {
	return uuid.New().String()
}

// GitHubCallback handles GitHub App installation callback
// This is called when a user installs the GitHub App on their repository
func (h *AuthHandler) GitHubCallback(w http.ResponseWriter, r *http.Request) {
	// Extract query parameters
	installationIDStr := r.URL.Query().Get("installation_id")
	setupAction := r.URL.Query().Get("setup_action")
	code := r.URL.Query().Get("code")

	log.Info().
		Str("installation_id", installationIDStr).
		Str("setup_action", setupAction).
		Str("code", code).
		Msg("GitHub App installation callback received")

	// Validate installation_id
	if installationIDStr == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Missing installation_id"})
		return
	}

	installationID, err := strconv.ParseInt(installationIDStr, 10, 64)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid installation_id"})
		return
	}

	// Check if setup_action is "install"
	if setupAction != "install" {
		log.Warn().
			Str("setup_action", setupAction).
			Msg("Unexpected setup_action in callback")
	}

	// Check if installation already exists
	_, err = h.db.GetInstallationByID(installationID)
	if err != nil {
		// Installation not found - it will be created via webhook
		// But we can log it here for reference
		log.Info().
			Int64("installation_id", installationID).
			Msg("Installation callback received, waiting for webhook to create record")
	}

	// Return success response
	// The actual installation processing happens via webhook
	response := map[string]interface{}{
		"status":          "success",
		"message":         "GitHub App installation received",
		"installation_id": installationID,
		"note":            "Installation will be processed via webhook. You can close this page.",
	}

	render.JSON(w, r, response)
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
