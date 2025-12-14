package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/rs/zerolog/log"
	"github.com/sherlock/service/internal/services/learning"
)

// FeedbackHandler handles review feedback endpoints
type FeedbackHandler struct {
	learningService *learning.LearningService
}

// NewFeedbackHandler creates a new feedback handler
func NewFeedbackHandler(learningService *learning.LearningService) *FeedbackHandler {
	return &FeedbackHandler{
		learningService: learningService,
	}
}

// RegisterRoutes registers feedback routes
func (h *FeedbackHandler) RegisterRoutes(r chi.Router) {
	r.Route("/feedback", func(r chi.Router) {
		r.Post("/", h.RecordFeedback)
		r.Get("/patterns", h.GetFeedbackPatterns)
		r.Get("/preferences", h.GetTeamPreferences)
	})
}

// RecordFeedbackRequest represents a feedback request
type RecordFeedbackRequest struct {
	ReviewID   string `json:"review_id"`
	CommentID  string `json:"comment_id"`
	FilePath   string `json:"file_path"`
	LineNumber int    `json:"line_number"`
	Feedback   string `json:"feedback"` // "accepted", "dismissed", "fixed"
}

// RecordFeedback records user feedback on a review comment
// POST /api/v1/feedback
func (h *FeedbackHandler) RecordFeedback(w http.ResponseWriter, r *http.Request) {
	orgID := r.Header.Get("X-Org-ID")
	if orgID == "" {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{"error": "Organization ID required"})
		return
	}

	var req RecordFeedbackRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid request body"})
		return
	}

	// Validate feedback type
	if req.Feedback != "accepted" && req.Feedback != "dismissed" && req.Feedback != "fixed" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid feedback type. Must be 'accepted', 'dismissed', or 'fixed'"})
		return
	}

	// Get user ID from session (simplified - would get from auth context)
	userID := "system" // TODO: Get from auth context

	feedback := learning.ReviewFeedback{
		ReviewID:   req.ReviewID,
		CommentID:  req.CommentID,
		FilePath:   req.FilePath,
		LineNumber: req.LineNumber,
		Feedback:   req.Feedback,
		UserID:     userID,
		OrgID:      orgID,
		CreatedAt:  time.Now(),
	}

	if err := h.learningService.RecordFeedback(r.Context(), feedback); err != nil {
		log.Error().Err(err).Msg("Failed to record feedback")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to record feedback"})
		return
	}

	render.JSON(w, r, map[string]string{"status": "success"})
}

// GetFeedbackPatterns returns feedback patterns for the organization
// GET /api/v1/feedback/patterns
func (h *FeedbackHandler) GetFeedbackPatterns(w http.ResponseWriter, r *http.Request) {
	orgID := r.Header.Get("X-Org-ID")
	if orgID == "" {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{"error": "Organization ID required"})
		return
	}

	patterns, err := h.learningService.GetFeedbackPatterns(r.Context(), orgID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get feedback patterns")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to get feedback patterns"})
		return
	}

	render.JSON(w, r, patterns)
}

// GetTeamPreferences returns learned preferences for the team
// GET /api/v1/feedback/preferences
func (h *FeedbackHandler) GetTeamPreferences(w http.ResponseWriter, r *http.Request) {
	orgID := r.Header.Get("X-Org-ID")
	if orgID == "" {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{"error": "Organization ID required"})
		return
	}

	preferences, err := h.learningService.GetTeamPreferences(r.Context(), orgID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get team preferences")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to get team preferences"})
		return
	}

	render.JSON(w, r, preferences)
}
