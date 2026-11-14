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
	"phystech-answer/ml/internal/decide"
	"phystech-answer/ml/internal/embed"
	"phystech-answer/ml/internal/httpapi"
	"phystech-answer/ml/internal/rag"
	"phystech-answer/ml/internal/store"
)

func main() {
	cfg := config.Load()

	st := store.New(cfg.IndexPath)
	em := embed.New(cfg.EmbedBaseURL, cfg.EmbedAPIKey, cfg.EmbedModel)
	ch := chat.New(cfg.LLMBaseURL, cfg.LLMAPIKey, cfg.LLMModel, cfg.LLMReasoning)

	if em == nil {
		log.Print("ml: embeddings disabled (EMBED_BASE_URL=off), lexical search only")
	}
	if ch == nil {
		log.Print("ml: chat model disabled (LLM_BASE_URL=off), excerpt fallback only")
	}

	dc := decide.New(cfg.DecisionURL)
	if cfg.Reranker == "decision" && dc == nil {
		log.Print("ml: RAG_RERANK=decision but DECISION_BASE_URL is off — reranking disabled")
	}
	svc := rag.New(st, em, ch, dc, rag.Options{
		Mode:       cfg.RetrievalMode,
		Reranker:   cfg.Reranker,
		Amendments: cfg.Amendments,
		Decompose:  cfg.Decompose,
		TopK:       cfg.TopK,
		MinVector:  cfg.MinVector,
		MinBM25:    cfg.MinBM25,
	})

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           httpapi.New(svc, cfg.CORSOrigin).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		// LLM calls can take a while; keep write timeout generous.
		WriteTimeout: 360 * time.Second,
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

	log.Printf("ml service listening on %s (llm=%s @ %s, embed=%s, mode=%s, rerank=%q)",
		cfg.Addr, cfg.LLMModel, cfg.LLMBaseURL, cfg.EmbedModel, cfg.RetrievalMode, cfg.Reranker)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server error: %v", err)
	}
}
