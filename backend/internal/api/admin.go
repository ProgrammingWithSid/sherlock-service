package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/sherlock/service/internal/database"
	"github.com/sherlock/service/internal/types"
)

type AdminHandler struct {
	db *database.DB
}

func NewAdminHandler(db *database.DB) *AdminHandler {
	return &AdminHandler{db: db}
}

func (h *AdminHandler) RegisterRoutes(r chi.Router) {
	r.Route("/admin", func(r chi.Router) {
		r.Use(RequireAuth)
		r.Use(RequireSuperAdmin)

		r.Get("/organizations", h.ListAllOrganizations)
		r.Get("/organizations/{id}", h.GetOrganization)
		r.Get("/stats", h.GetSystemStats)
		r.Get("/users", h.ListAllUsers)
	})
}

// ListAllOrganizations returns all organizations (super admin only)
func (h *AdminHandler) ListAllOrganizations(w http.ResponseWriter, r *http.Request) {
	orgs, err := h.db.ListAllOrganizations()
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	render.JSON(w, r, orgs)
}

// GetOrganization returns organization details (super admin only)
func (h *AdminHandler) GetOrganization(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "id")
	if orgID == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Organization ID required"})
		return
	}

	org, err := h.db.GetOrganizationByID(orgID)
	if err != nil {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "Organization not found"})
		return
	}

	// Get repo count
	repoCount, _ := h.db.GetRepoCount(orgID)

	// Get monthly review count
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	reviewCount, _ := h.db.GetMonthlyReviewCount(orgID)

	response := map[string]interface{}{
		"organization": org,
		"stats": map[string]interface{}{
			"repo_count":        repoCount,
			"reviews_this_month": reviewCount,
		},
	}

	render.JSON(w, r, response)
}

// SystemStats represents system-wide statistics
type SystemStats struct {
	TotalOrganizations int `json:"total_organizations"`
	TotalUsers         int `json:"total_users"`
	TotalRepositories  int `json:"total_repositories"`
	TotalReviews       int `json:"total_reviews"`
	ReviewsThisMonth   int `json:"reviews_this_month"`
}

// GetSystemStats returns system-wide statistics (super admin only)
func (h *AdminHandler) GetSystemStats(w http.ResponseWriter, r *http.Request) {
	// Get all organizations
	orgs, err := h.db.ListAllOrganizations()
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to get organizations"})
		return
	}

	// Get all users
	users, err := h.db.ListUsers(nil)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to get users"})
		return
	}

	// Calculate totals
	totalRepos := 0
	totalReviews := 0
	reviewsThisMonth := 0

	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	for _, org := range orgs {
		repoCount, _ := h.db.GetRepoCount(org.ID)
		totalRepos += repoCount

		monthlyReviews, _ := h.db.GetMonthlyReviewCount(org.ID)
		reviewsThisMonth += monthlyReviews

		// Get total reviews (would need a new method for this)
		// For now, we'll use monthly count as approximation
		totalReviews += monthlyReviews
	}

	stats := SystemStats{
		TotalOrganizations: len(orgs),
		TotalUsers:         len(users),
		TotalRepositories:  totalRepos,
		TotalReviews:       totalReviews,
		ReviewsThisMonth:   reviewsThisMonth,
	}

	render.JSON(w, r, stats)
}

// ListAllUsers returns all users (super admin only)
func (h *AdminHandler) ListAllUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.db.ListUsers(nil)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	// Remove password hashes from response
	userList := make([]map[string]interface{}, len(users))
	for i, user := range users {
		userList[i] = map[string]interface{}{
			"id":        user.ID,
			"email":     user.Email,
			"name":      user.Name,
			"role":      user.Role,
			"org_id":    user.OrgID,
			"is_active": user.IsActive,
			"created_at": user.CreatedAt.Format(time.RFC3339),
			"updated_at": user.UpdatedAt.Format(time.RFC3339),
		}
	}

	render.JSON(w, r, userList)
}
