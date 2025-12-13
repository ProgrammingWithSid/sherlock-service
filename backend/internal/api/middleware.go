package api

import (
	"context"
	"net/http"

	"github.com/go-chi/render"
)

// RequireOrgID middleware ensures X-Org-ID header is present
func RequireOrgID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		orgID := r.Header.Get("X-Org-ID")
		if orgID == "" {
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, map[string]string{
				"error": "X-Org-ID header required",
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
