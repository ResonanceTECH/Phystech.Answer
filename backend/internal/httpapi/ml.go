package httpapi

import (
	"context"
	"log"

	"phystech-answer/backend/internal/model"
)

// indexDocument pushes a document to the ML index. Indexing failures are
// logged but never fail the user-facing request — the document remains
// available in the catalog and is re-synced on restart.
func (s *Server) indexDocument(ctx context.Context, doc model.Document) {
	if s.ml == nil {
		return
	}
	if err := s.ml.IndexDocument(ctx, doc); err != nil {
		log.Printf("ml index %s: %v", doc.ID, err)
	}
}

// removeFromIndex drops a document from the ML index.
func (s *Server) removeFromIndex(ctx context.Context, id string) {
	if s.ml == nil {
		return
	}
	if err := s.ml.RemoveDocument(ctx, id); err != nil {
		log.Printf("ml remove %s: %v", id, err)
	}
}

// SyncDocuments pushes the full document set to the ML service
// (POST /api/sync). Called on startup so the index mirrors the store.
func (s *Server) SyncDocuments(ctx context.Context) {
	if s.ml == nil || !s.ml.Enabled() {
		return
	}
	docs, err := s.store.ListDocuments(ctx)
	if err != nil {
		log.Printf("ml sync: list documents: %v", err)
		return
	}
	if err := s.ml.SyncDocuments(ctx, docs); err != nil {
		log.Printf("ml sync: %v", err)
	}
}
