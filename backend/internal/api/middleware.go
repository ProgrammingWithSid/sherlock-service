package api

import (
	"context"
	"net/http"

	"github.com/go-chi/render"
	"github.com/sherlock/service/internal/database"
	"github.com/sherlock/service/internal/types"
)

// sessionStore will be initialized in setupRouter
var sessionStore *SessionStore

func InitSessionStore(db *database.DB) {
	sessionStore = NewSessionStore(db)
}

// RequireAuth middleware ensures user is authenticated
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		ctx := r.Context()
		ctx = context.WithValue(ctx, "user_id", session.UserID)
		ctx = context.WithValue(ctx, "user_role", session.Role)
		if session.OrgID != nil {
			ctx = context.WithValue(ctx, "org_id", *session.OrgID)
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireSuperAdmin middleware ensures user is super admin
func RequireSuperAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, ok := r.Context().Value("user_role").(string)
		if !ok || role != string(types.RoleSuperAdmin) {
			render.Status(r, http.StatusForbidden)
			render.JSON(w, r, map[string]string{"error": "Super admin access required"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireOrgID middleware ensures X-Org-ID header is present or user has org
func RequireOrgID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		orgID := r.Header.Get("X-Org-ID")
		if orgID == "" {
			// Try to get from context (set by RequireAuth)
			if ctxOrgID, ok := r.Context().Value("org_id").(string); ok {
				orgID = ctxOrgID
			}
		}

		if orgID == "" {
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, map[string]string{
				"error": "Organization ID required",
			})
			return
		}

		// Add org ID to request context
		ctx := r.Context()
		ctx = context.WithValue(ctx, "org_id", orgID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OptionalOrgID middleware adds org ID to context if present
func OptionalOrgID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		orgID := r.Header.Get("X-Org-ID")
		if orgID != "" {
			ctx := r.Context()
			ctx = context.WithValue(ctx, "org_id", orgID)
			r = r.WithContext(ctx)
		}
		next.ServeHTTP(w, r)
	})
}

func getSessionToken(r *http.Request) string {
	// Try cookie first
	if cookie, err := r.Cookie("session_token"); err == nil {
		return cookie.Value
	}
	// Try Authorization header
	if auth := r.Header.Get("Authorization"); auth != "" {
		if len(auth) > 7 && auth[:7] == "Bearer " {
			return auth[7:]
		}
	}
	return ""
}
