package httpapi

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"phystech-answer/backend/internal/config"
	"phystech-answer/backend/internal/store"
)

type Server struct {
	store     store.Store
	mlURL     string
	corsAllow string
	client    *http.Client
	mlClient  *http.Client
}

func New(st store.Store, cfg config.Config) *Server {
	return &Server{
		store:     st,
		mlURL:     cfg.MLServiceURL,
		corsAllow: cfg.CORSOrigin,
		client:    &http.Client{Timeout: 60 * time.Second},
		mlClient:  &http.Client{Timeout: 15 * time.Second},
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", s.health)

	mux.HandleFunc("GET /api/documents", s.listDocuments)
	mux.HandleFunc("POST /api/documents", s.createDocument)
	mux.HandleFunc("GET /api/documents/{id}", s.getDocument)
	mux.HandleFunc("PUT /api/documents/{id}", s.updateDocument)
	mux.HandleFunc("DELETE /api/documents/{id}", s.deleteDocument)
	mux.HandleFunc("POST /api/documents/{id}/read", s.markRead)

	mux.HandleFunc("GET /api/users", s.listUsers)
	mux.HandleFunc("POST /api/users", s.createUser)
	mux.HandleFunc("GET /api/users/{id}/unread", s.unreadDocuments)

	mux.HandleFunc("POST /api/ask", s.ask)

	return s.cors(s.recoverer(s.logging(mux)))
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", s.corsAllow)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
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
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
}
