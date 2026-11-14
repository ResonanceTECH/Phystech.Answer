package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"phystech-answer/ml/internal/chat"
	"phystech-answer/ml/internal/config"
	"phystech-answer/ml/internal/embed"
	"phystech-answer/ml/internal/httpapi"
	"phystech-answer/ml/internal/rag"
	"phystech-answer/ml/internal/rerank"
	"phystech-answer/ml/internal/store"
)

func main() {
	cfg := config.Load()

	st := store.New(cfg.IndexPath)
	em := embed.New(cfg.EmbedBaseURL, cfg.EmbedAPIKey, cfg.EmbedModel)
	var ch *chat.Client
	if cfg.LLMBaseURL != "" {
		if cfg.LLMDisableThinking {
			ch = chat.NewNoThinking(cfg.LLMBaseURL, cfg.LLMAPIKey, cfg.LLMModel)
		} else {
			ch = chat.New(cfg.LLMBaseURL, cfg.LLMAPIKey, cfg.LLMModel)
		}
	}
	rr := rerank.New(cfg.RerankBaseURL, cfg.RerankAPIKey, cfg.RerankModel)
	if !cfg.RerankEnabled {
		rr = nil
	}

	if em == nil {
		log.Print("ml: embeddings disabled (EMBED_BASE_URL=off), lexical search only")
	}
	if ch == nil {
		log.Print("ml: chat model disabled (LLM_BASE_URL=off), excerpt fallback only")
	}
	if rr == nil {
		log.Print("ml: reranker disabled (RERANK_ENABLED=false or RERANK_BASE_URL=off)")
	}

	svc := rag.New(st, em, ch, rr, cfg)

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           httpapi.New(svc, cfg.CORSOrigin).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		// LLM calls can take a while; keep write timeout generous.
		WriteTimeout: 300 * time.Second,
		IdleTimeout:  60 * time.Second,
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

	log.Printf("ml service listening on %s (llm=%s embed=%s rerank=%v)", cfg.Addr, cfg.LLMModel, cfg.EmbedModel, rr != nil)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server error: %v", err)
	}
}
