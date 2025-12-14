package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/rs/zerolog/log"
	"github.com/sherlock/service/internal/database"
)

type OrganizationHandler struct {
	db *database.DB
}

func NewOrganizationHandler(db *database.DB) *OrganizationHandler {
	return &OrganizationHandler{db: db}
}

func (h *OrganizationHandler) RegisterRoutes(r chi.Router) {
	r.Route("/organizations", func(r chi.Router) {
		r.Use(RequireAuth)
		r.Get("/{id}/rules", h.GetGlobalRules)
		r.Put("/{id}/rules", h.UpdateGlobalRules)
	})
}

type GlobalRulesRequest struct {
	Rules []string `json:"rules"`
}

type GlobalRulesResponse struct {
	Rules []string `json:"rules"`
}

// GetGlobalRules gets global rules for an organization
func (h *OrganizationHandler) GetGlobalRules(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "id")
	
	// Verify user has access to this organization
	userRole, _ := r.Context().Value("user_role").(string)
	userOrgID, _ := r.Context().Value("org_id").(string)

	// Check authorization
	if userRole != "super_admin" && userOrgID != orgID {
		render.Status(r, http.StatusForbidden)
		render.JSON(w, r, map[string]string{"error": "Access denied"})
		return
	}

	org, err := h.db.GetOrganizationByID(orgID)
	if err != nil {
		log.Error().Err(err).Str("org_id", orgID).Msg("Failed to get organization")
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "Organization not found"})
		return
	}

	var rules []string
	if org.GlobalRules != "" && org.GlobalRules != "[]" {
		if err := json.Unmarshal([]byte(org.GlobalRules), &rules); err != nil {
			log.Warn().Err(err).Str("org_id", orgID).Msg("Failed to parse global rules")
			rules = []string{}
		}
	}

	render.JSON(w, r, GlobalRulesResponse{Rules: rules})
}

// UpdateGlobalRules updates global rules for an organization
func (h *OrganizationHandler) UpdateGlobalRules(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "id")
	
	// Verify user has access to this organization
	userRole, _ := r.Context().Value("user_role").(string)
	userOrgID, _ := r.Context().Value("org_id").(string)

	// Check authorization (org admin or super admin)
	if userRole != "super_admin" && userOrgID != orgID {
		render.Status(r, http.StatusForbidden)
		render.JSON(w, r, map[string]string{"error": "Access denied"})
		return
	}

	var req GlobalRulesRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid request body"})
		return
	}

	// Validate rules (check plan limits)
	org, err := h.db.GetOrganizationByID(orgID)
	if err != nil {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "Organization not found"})
		return
	}

	// Check plan limits (can be enhanced later)
	maxRules := 20 // Default limit
	if org.Plan == "enterprise" {
		maxRules = -1 // Unlimited
	} else if org.Plan == "team" {
		maxRules = 50
	} else if org.Plan == "pro" {
		maxRules = 10
	}

	if maxRules > 0 && len(req.Rules) > maxRules {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{
			"error": fmt.Sprintf("Plan limit exceeded. Maximum %d rules allowed.", maxRules),
		})
		return
	}

	if err := h.db.UpdateOrganizationGlobalRules(orgID, req.Rules); err != nil {
		log.Error().Err(err).Str("org_id", orgID).Msg("Failed to update global rules")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to update global rules"})
		return
	}

	render.JSON(w, r, GlobalRulesResponse{Rules: req.Rules})
}

