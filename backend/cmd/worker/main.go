package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog/log"

	"github.com/sherlock/service/internal/config"
	"github.com/sherlock/service/internal/database"
	"github.com/sherlock/service/internal/queue"
	"github.com/sherlock/service/internal/workers"
)

func main() {
	cfg := config.Load()

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

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatal().Err(err).Msg("Invalid configuration")
	}

	// Initialize workers
	workerPool := workers.NewWorkerPool(reviewQueue, db, cfg, redisClient)

	// Start workers
	go workerPool.Start(context.Background())

	log.Info().Msg("Worker started")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down worker...")
	workerPool.Stop()
	log.Info().Msg("Worker stopped")
}
