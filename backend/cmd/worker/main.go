// Standalone background worker (spec §4.3): runs the same job queue the
// API server embeds, so heavy parsing/generation can scale separately.
package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"phystech-answer/backend/internal/config"
	"phystech-answer/backend/internal/db"
	"phystech-answer/backend/internal/jobs"
	"phystech-answer/backend/internal/mlclient"
	"phystech-answer/backend/internal/storage"
	"phystech-answer/backend/internal/store"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required (see .env.example)")
	}
	pool, err := db.Connect(ctx, cfg.DatabaseURL, cfg.DBMaxConns)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool); err != nil {
		log.Fatalf("migrations: %v", err)
	}

	st := store.NewPostgres(pool)
	fs, err := storage.NewLocal(cfg.StoragePath)
	if err != nil {
		log.Fatalf("storage: %v", err)
	}
	ml := mlclient.New(cfg.MLServiceURL, time.Duration(cfg.MLTimeoutSec)*time.Second)

	worker := jobs.NewRunner(st, fs, ml, cfg.WorkerConcurrency)

	sigCtx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Printf("worker started (concurrency=%d)", cfg.WorkerConcurrency)
	worker.Run(sigCtx)
	log.Printf("worker stopped")
}
