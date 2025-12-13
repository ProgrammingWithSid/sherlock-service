package api

import (
	"net/http"

	"github.com/go-chi/render"
	"github.com/sherlock/service/internal/database"
)

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, map[string]string{
		"status": "ok",
	})
}

func ReadinessCheck(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(); err != nil {
			render.Status(r, http.StatusServiceUnavailable)
			render.JSON(w, r, map[string]string{
				"status": "not ready",
				"error":  err.Error(),
			})
			return
		}

		render.JSON(w, r, map[string]string{
			"status": "ready",
		})
	}
}

func LivenessCheck(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, map[string]string{
		"status": "alive",
	})
}


