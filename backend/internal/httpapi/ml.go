package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"phystech-answer/backend/internal/model"
)

// mlDo performs a best-effort call to the ML service. Indexing failures are
// logged but never fail the user-facing request — the document remains
// available in the catalog and can be re-synced on restart.
func (s *Server) mlDo(ctx context.Context, method, path string, body any) {
	if s.mlURL == "" {
		return
	}
	var rdr io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			log.Printf("ml sync: marshal body: %v", err)
			return
		}
		rdr = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, s.mlURL+path, rdr)
	if err != nil {
		log.Printf("ml sync: build request: %v", err)
		return
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := s.mlClient.Do(req)
	if err != nil {
		log.Printf("ml sync %s %s: %v", method, path, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		log.Printf("ml sync %s %s: status %s", method, path, resp.Status)
	}
}

// indexDocument pushes a document to the ML index (PUT /api/documents/{id}).
func (s *Server) indexDocument(ctx context.Context, doc model.Document) {
	s.mlDo(ctx, http.MethodPut, "/api/documents/"+doc.ID, doc)
}

// removeFromIndex drops a document from the ML index.
func (s *Server) removeFromIndex(ctx context.Context, id string) {
	s.mlDo(ctx, http.MethodDelete, "/api/documents/"+id, nil)
}

// SyncDocuments pushes the full document set to the ML service
// (POST /api/sync). Called on startup so the index mirrors the store.
func (s *Server) SyncDocuments(ctx context.Context) {
	if s.mlURL == "" {
		return
	}
	docs, err := s.store.ListDocuments(ctx)
	if err != nil {
		log.Printf("ml sync: list documents: %v", err)
		return
	}
	s.mlDo(ctx, http.MethodPost, "/api/sync", map[string]any{"documents": docs})
}
