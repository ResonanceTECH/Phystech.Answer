// Package httpapi exposes the ML service over HTTP: document indexing,
// full sync, the /ask endpoint consumed by the Go backend, and the
// /internal/v1/* computation API (parse/ocr/embeddings/rerank/analyze/
// generate/verify/relations).
package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"phystech-answer/ml/internal/model"
	"phystech-answer/ml/internal/rag"
	"phystech-answer/ml/internal/store"
)

type Server struct {
	rag       *rag.Service
	corsAllow string
}

func New(svc *rag.Service, corsAllow string) *Server {
	return &Server{rag: svc, corsAllow: corsAllow}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("GET /health/live", s.health)

	mux.HandleFunc("GET /api/documents", s.listDocuments)
	mux.HandleFunc("POST /api/documents", s.upsertDocument)
	mux.HandleFunc("PUT /api/documents/{id}", s.upsertDocument)
	mux.HandleFunc("DELETE /api/documents/{id}", s.deleteDocument)
	mux.HandleFunc("POST /api/sync", s.syncDocuments)

	// The Go backend proxies POST {ML_SERVICE_URL}/ask; keep both aliases.
	mux.HandleFunc("POST /api/ask", s.ask)
	mux.HandleFunc("POST /ask", s.ask)

	// Internal computation API (spec §20).
	mux.HandleFunc("POST /internal/v1/documents/parse", s.parseDocument)
	mux.HandleFunc("POST /internal/v1/documents/ocr", s.ocrDocument)
	mux.HandleFunc("POST /internal/v1/embeddings", s.embeddings)
	mux.HandleFunc("POST /internal/v1/rerank", s.rerankEndpoint)
	mux.HandleFunc("POST /internal/v1/questions/analyze", s.analyzeQuestion)
	mux.HandleFunc("POST /internal/v1/answers/generate", s.generateAnswer)
	mux.HandleFunc("POST /internal/v1/answers/verify", s.verifyAnswer)
	mux.HandleFunc("POST /internal/v1/relations/extract", s.extractRelations)
	mux.HandleFunc("GET /internal/v1/relations", s.listRelations)
	mux.HandleFunc("GET /internal/v1/stats", s.stats)

	return s.cors(s.recoverer(s.logging(mux)))
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	docs, chunks := s.rag.Stats()
	writeJSON(w, http.StatusOK, map[string]any{
		"status":    "ok",
		"documents": docs,
		"chunks":    chunks,
	})
}

func (s *Server) stats(w http.ResponseWriter, _ *http.Request) {
	docs, chunks := s.rag.Stats()
	rels := s.rag.Relations()
	n := 0
	for _, v := range rels {
		n += len(v)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"documents": docs,
		"chunks":    chunks,
		"relations": n,
	})
}

type documentRequest struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	Content        string   `json:"content"`
	Tags           []string `json:"tags"`
	DocumentNumber string   `json:"documentNumber"`
	DocumentType   string   `json:"documentType"`
	DocumentDate   string   `json:"documentDate"`
	Category       string   `json:"category"`
	LegalStatus    string   `json:"legalStatus"`
}

func (s *Server) listDocuments(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.rag.ListDocuments())
}

func (s *Server) upsertDocument(w http.ResponseWriter, r *http.Request) {
	var req documentRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if id := r.PathValue("id"); id != "" {
		req.ID = id
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		writeError(w, http.StatusBadRequest, "content is required")
		return
	}
	if req.ID == "" {
		req.ID = "doc_" + req.Title // fallback; backend always sends an ID
	}

	doc := model.Document{
		ID: req.ID, Title: req.Title, Content: req.Content, Tags: req.Tags,
		DocumentNumber: req.DocumentNumber, DocumentType: req.DocumentType,
		DocumentDate: req.DocumentDate, Category: req.Category,
		LegalStatus: req.LegalStatus,
	}
	if err := s.rag.IndexDocument(r.Context(), doc); err != nil {
		log.Printf("index document %s: %v", req.ID, err)
		writeError(w, http.StatusInternalServerError, "failed to index document")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"id": req.ID, "status": "indexed"})
}

func (s *Server) deleteDocument(w http.ResponseWriter, r *http.Request) {
	err := s.rag.DeleteDocument(r.Context(), r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "document not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete document")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type syncRequest struct {
	Documents []model.Document `json:"documents"`
}

// syncDocuments replaces the whole index — used by the backend on startup
// to push its authoritative document set.
func (s *Server) syncDocuments(w http.ResponseWriter, r *http.Request) {
	var req syncRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.rag.SyncDocuments(r.Context(), req.Documents); err != nil {
		log.Printf("sync documents: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to sync documents")
		return
	}
	docs, chunks := s.rag.Stats()
	writeJSON(w, http.StatusOK, map[string]any{"status": "synced", "documents": docs, "chunks": chunks})
}

type askRequest struct {
	Question string `json:"question"`
	UserID   string `json:"userId,omitempty"`
}

func (s *Server) ask(w http.ResponseWriter, r *http.Request) {
	var req askRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	resp, err := s.rag.Ask(r.Context(), req.Question)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// ---------- internal/v1 endpoints ----------

type analyzeRequest struct {
	Question string `json:"question"`
}

func (s *Server) analyzeQuestion(w http.ResponseWriter, r *http.Request) {
	var req analyzeRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	out, err := s.rag.AnalyzeQuestion(r.Context(), req.Question)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) generateAnswer(w http.ResponseWriter, r *http.Request) {
	var req askRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	payload, err := s.rag.AskStructured(r.Context(), req.Question)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

type verifyRequest struct {
	Question string              `json:"question"`
	Answer   model.AnswerPayload `json:"answer"`
}

func (s *Server) verifyAnswer(w http.ResponseWriter, r *http.Request) {
	var req verifyRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	// Re-run retrieval for the question to get the same context.
	out, err := s.rag.VerifyAnswerWithContext(r.Context(), req.Question, req.Answer)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) extractRelations(w http.ResponseWriter, r *http.Request) {
	var req documentRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		writeError(w, http.StatusBadRequest, "content is required")
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	doc := model.Document{ID: req.ID, Title: req.Title, Content: req.Content}
	out, err := s.rag.ExtractRelationsLLM(r.Context(), doc)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"relations": out})
}

func (s *Server) listRelations(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"relations": s.rag.Relations()})
}

// parseDocument is a stub that reports the service cannot parse files
// itself — parsing happens in backend jobs / offline pipeline. Kept for
// contract completeness; returns method-not-allowed style guidance.
func (s *Server) parseDocument(w http.ResponseWriter, _ *http.Request) {
	writeError(w, http.StatusNotImplemented,
		"document parsing is performed by the backend pipeline; use /api/documents to index extracted text")
}

// ocrDocument is likewise handled by the offline OCR pipeline
// (ml/scripts/parse_corpus.sh). Endpoint reserved for future runtime OCR.
func (s *Server) ocrDocument(w http.ResponseWriter, _ *http.Request) {
	writeError(w, http.StatusNotImplemented,
		"OCR runs in the offline pipeline (scripts/parse_corpus.sh); endpoint reserved")
}

type embedRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model,omitempty"`
}

func (s *Server) embeddings(w http.ResponseWriter, r *http.Request) {
	var req embedRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if len(req.Input) == 0 {
		writeError(w, http.StatusBadRequest, "input is required")
		return
	}
	if len(req.Input) > 256 {
		writeError(w, http.StatusBadRequest, "too many inputs (max 256)")
		return
	}
	vecs, err := s.rag.EmbedTexts(r.Context(), req.Input)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	type item struct {
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	}
	data := make([]item, len(vecs))
	for i, v := range vecs {
		data[i] = item{Index: i, Embedding: v}
	}
	writeJSON(w, http.StatusOK, map[string]any{"object": "list", "data": data})
}

type rerankRequest struct {
	Query     string   `json:"query"`
	Documents []string `json:"documents"`
	TopN      int      `json:"top_n,omitempty"`
}

func (s *Server) rerankEndpoint(w http.ResponseWriter, r *http.Request) {
	var req rerankRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Query) == "" || len(req.Documents) == 0 {
		writeError(w, http.StatusBadRequest, "query and documents are required")
		return
	}
	out, err := s.rag.RerankTexts(r.Context(), req.Query, req.Documents, req.TopN)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"object": "list", "results": out})
}

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", s.corsAllow)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic handling %s %s: %v", r.Method, r.URL.Path, err)
				writeError(w, http.StatusInternalServerError, "internal error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (s *Server) logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s (%s)", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 32<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read body")
		return false
	}
	if err := json.Unmarshal(body, v); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
}
