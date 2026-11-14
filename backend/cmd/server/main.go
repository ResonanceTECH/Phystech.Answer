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
	"phystech-answer/backend/internal/db"
	"phystech-answer/backend/internal/docmeta"
	"phystech-answer/backend/internal/httpapi"
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
	store.SeedDemo(ctx, st, cfg.SeedDemoData)
	if docs, err := st.ListDocuments(ctx); err == nil {
		for _, doc := range docs {
			if doc.Category != "" && doc.Category != "Прочие документы" {
				continue
			}
			folder := docmeta.RefineFolder(doc.Category, doc.Title, doc.Content)
			if folder != doc.Category {
				if err := st.SetDocumentFolder(ctx, doc.ID, folder); err != nil {
					log.Printf("backfill folder %s: %v", doc.ID, err)
				}
			}
		}
	}

	fs, err := storage.NewLocal(cfg.StoragePath)
	if err != nil {
		log.Fatalf("storage: %v", err)
	}
	ml := mlclient.New(cfg.MLServiceURL, time.Duration(cfg.MLTimeoutSec)*time.Second)

	srv := httpapi.New(st, fs, ml, cfg)
	// Push the document set to the ML index and import the knowledge base
	// folder in the background: first-time embedding of the corpus can take
	// minutes on CPU and must not block the API. Parsing of imported files
	// runs in the job worker; each parsed document is indexed on completion.
	store.SeedOrg(ctx, st)
	go func() {
		srv.SyncDocuments(ctx)
		srv.ImportDirectory(ctx, cfg.DocsDir)
		// Demo assignments need parsed documents (numbers/titles).
		if cfg.SeedDemoData || cfg.SeedOnboarding {
			srv.WaitForProcessing(ctx, 20*time.Minute)
			store.SeedAssignments(ctx, st)
		}
	}()

	// Background worker: document parsing, answer generation.
	worker := jobs.NewRunner(st, fs, ml, cfg.WorkerConcurrency)

	httpSrv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	sigCtx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go worker.Run(sigCtx)

	go func() {
		<-sigCtx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpSrv.Shutdown(shutdownCtx); err != nil {
			log.Printf("graceful shutdown failed: %v", err)
		}
	}()

	log.Printf("listening on %s", cfg.Addr)
	if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server error: %v", err)
	}
}
