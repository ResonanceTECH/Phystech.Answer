package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"phystech-answer/backend/internal/config"
	"phystech-answer/backend/internal/httpapi"
	"phystech-answer/backend/internal/store"
)

func main() {
	cfg := config.Load()

	st := store.NewMemory()
	if cfg.SeedDemoData {
		st.SeedDemo()
	}

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           httpapi.New(st, cfg).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("graceful shutdown failed: %v", err)
		}
	}()

	log.Printf("listening on %s", cfg.Addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server error: %v", err)
	}
}
