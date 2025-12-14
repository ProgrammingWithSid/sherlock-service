package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/rs/zerolog/log"

	"github.com/sherlock/service/internal/api"
	"github.com/sherlock/service/internal/config"
	"github.com/sherlock/service/internal/database"
	"github.com/sherlock/service/internal/queue"
	"github.com/sherlock/service/internal/services/metrics"
	"github.com/sherlock/service/internal/workers"
)

func main() {
	cfg := config.Load()

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatal().Err(err).Msg("Invalid configuration")
	}

	// Initialize database
	db, err := database.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

	// Initialize Redis client
	redisClient := queue.NewRedisClient(cfg.RedisURL)
	defer redisClient.Close()

	// Initialize queue
	reviewQueue := queue.NewReviewQueue(redisClient)

	// Initialize metrics service
	metricsService := metrics.NewMetricsService(redisClient)

	// Initialize workers
	workerPool := workers.NewWorkerPool(reviewQueue, db, cfg, redisClient)
	go workerPool.Start(context.Background())

	// Initialize session store
	api.InitSessionStore(db)

	// Initialize API router
	router := setupRouter(cfg, db, reviewQueue, metricsService)

	// Start server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Server failed to start")
		}
	}()

	log.Info().Int("port", cfg.Port).Msg("Server started")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Server forced to shutdown")
	}

	workerPool.Stop()
	log.Info().Msg("Server stopped")
}

func setupRouter(cfg *config.Config, db *database.DB, reviewQueue *queue.ReviewQueue, metricsService *metrics.MetricsService) *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Org-ID"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Allow all origins for development (remove in production)
	if cfg.Environment == "development" {
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins:   []string{"*"},
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"*"},
			AllowCredentials: false,
		}))
	}

	// Health checks
	r.Get("/health", api.HealthCheck)
	r.Get("/health/ready", api.ReadinessCheck(db))
	r.Get("/health/live", api.LivenessCheck)

	// Auth routes (public)
	r.Route("/api/v1", func(r chi.Router) {
		authHandler := api.NewAuthHandler(
			db,
			cfg.GitHubClientID,
			cfg.GitHubClientSecret,
			cfg.GitLabClientID,
			cfg.GitLabClientSecret,
			cfg.BaseURL,
		)
		authHandler.RegisterRoutes(r)

		// Protected API routes
		r.Route("/", func(r chi.Router) {
			apiHandler := api.NewHandler(db, reviewQueue, cfg, metricsService)
			apiHandler.RegisterRoutes(r)

			// Admin routes (requires super admin)
			adminHandler := api.NewAdminHandler(db)
			adminHandler.RegisterRoutes(r)
		})
	})

	// Webhook routes
	r.Route("/webhooks", func(r chi.Router) {
		webhookHandler := api.NewWebhookHandler(db, reviewQueue, cfg)
		webhookHandler.RegisterRoutes(r)
	})

	// Serve static files (frontend)
	// Check if frontend/dist exists
	if _, err := os.Stat("./frontend/dist"); err == nil {
		// Serve static assets with long cache headers (they're hashed)
		assetsHandler := http.StripPrefix("/assets", http.FileServer(http.Dir("./frontend/dist/assets")))
		r.Handle("/assets/*", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Cache hashed assets for 1 year
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			assetsHandler.ServeHTTP(w, r)
		}))

		// Handle SPA routing - serve index.html for all non-API routes
		r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
			// Don't serve index.html for API routes, webhooks, or health checks
			if strings.HasPrefix(req.URL.Path, "/api") ||
			   strings.HasPrefix(req.URL.Path, "/webhooks") ||
			   strings.HasPrefix(req.URL.Path, "/health") ||
			   strings.HasPrefix(req.URL.Path, "/assets") {
				http.NotFound(w, req)
				return
			}
			// Prevent caching of index.html to ensure users get latest version
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate, max-age=0")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")
			// Add ETag with timestamp to force revalidation
			w.Header().Set("ETag", `"v20251213-211000"`)
			http.ServeFile(w, req, "./frontend/dist/index.html")
		})
	}

	return r
}
