package httpapi

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"phystech-answer/backend/internal/config"
	"phystech-answer/backend/internal/mlclient"
	"phystech-answer/backend/internal/model"
	"phystech-answer/backend/internal/storage"
	"phystech-answer/backend/internal/store"
)

const sessionCookie = "fiztech_session"

type Server struct {
	store      store.Store
	fs         storage.Storage
	ml         *mlclient.Client
	corsAllow  string
	client     *http.Client
	sessionTTL time.Duration
}

func New(st store.Store, fs storage.Storage, ml *mlclient.Client, cfg config.Config) *Server {
	return &Server{
		store:      st,
		fs:         fs,
		ml:         ml,
		corsAllow:  cfg.CORSOrigin,
		client:     &http.Client{Timeout: 60 * time.Second},
		sessionTTL: 72 * time.Hour,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", s.health)
	mux.HandleFunc("GET /api/v1/health", s.health)
	mux.HandleFunc("GET /health/live", s.health)
	mux.HandleFunc("GET /health/ready", s.ready)

	// Auth.
	mux.HandleFunc("POST /api/v1/auth/login", s.login)
	mux.HandleFunc("POST /api/v1/auth/logout", s.logout)
	mux.HandleFunc("GET /api/v1/auth/me", s.me)
	mux.HandleFunc("GET /api/v1/users/me", s.me)

	// Documents catalog (v1).
	mux.HandleFunc("GET /api/v1/documents", s.listDocuments)
	mux.HandleFunc("POST /api/v1/documents", s.createDocument)
	mux.HandleFunc("GET /api/v1/documents/{id}", s.getDocument)
	mux.HandleFunc("PUT /api/v1/documents/{id}", s.updateDocument)
	mux.HandleFunc("DELETE /api/v1/documents/{id}", s.deleteDocument)
	mux.HandleFunc("POST /api/v1/documents/{id}/read", s.markRead)
	mux.HandleFunc("GET /api/v1/documents/{id}/versions", s.listVersions)
	mux.HandleFunc("GET /api/v1/documents/{id}/sections", s.listSections)
	mux.HandleFunc("GET /api/v1/documents/{id}/relations", s.listRelations)
	mux.HandleFunc("GET /api/v1/documents/{id}/attachments", s.listAttachments)
	mux.HandleFunc("GET /api/v1/documents/{id}/file", s.getDocumentFile)
	mux.HandleFunc("GET /api/v1/documents/{id}/content", s.getDocumentContent)
	mux.HandleFunc("GET /api/v1/versions/{id}/chunks", s.listChunks)
	mux.HandleFunc("GET /api/v1/sources/{id}", s.getSourceFile)

	mux.HandleFunc("POST /api/v1/search", s.search)

	// Users.
	mux.HandleFunc("GET /api/v1/users", s.listUsers)
	mux.HandleFunc("POST /api/v1/users", s.createUser)

	// Conversations + runs.
	mux.HandleFunc("POST /api/v1/conversations", s.createConversation)
	mux.HandleFunc("GET /api/v1/conversations", s.listConversations)
	mux.HandleFunc("GET /api/v1/conversations/{id}", s.getConversation)
	mux.HandleFunc("PATCH /api/v1/conversations/{id}", s.renameConversation)
	mux.HandleFunc("DELETE /api/v1/conversations/{id}", s.deleteConversation)
	mux.HandleFunc("POST /api/v1/conversations/{id}/messages", s.postMessage)
	mux.HandleFunc("GET /api/v1/conversations/{id}/messages", s.listMessages)

	mux.HandleFunc("GET /api/v1/runs/{id}", s.getRun)
	mux.HandleFunc("GET /api/v1/runs/{id}/events", s.runEvents)
	mux.HandleFunc("GET /api/v1/runs/{id}/answer", s.getRunAnswer)
	mux.HandleFunc("POST /api/v1/runs/{id}/cancel", s.cancelRun)

	// Admin.
	mux.HandleFunc("POST /api/v1/admin/documents/upload", s.uploadDocuments)
	mux.HandleFunc("GET /api/v1/admin/processing", s.listJobs)
	mux.HandleFunc("GET /api/v1/admin/processing/{id}", s.getJob)
	mux.HandleFunc("POST /api/v1/admin/processing/{id}/retry", s.retryJob)
	mux.HandleFunc("POST /api/v1/admin/documents/{id}/publish", s.publishDocument)
	mux.HandleFunc("POST /api/v1/admin/documents/{id}/reprocess", s.reprocessDocument)
	mux.HandleFunc("POST /api/v1/admin/document-relations", s.createRelation)
	mux.HandleFunc("PATCH /api/v1/admin/document-relations/{id}", s.patchRelation)
	mux.HandleFunc("POST /api/v1/admin/document-relations/{id}/verify", s.verifyRelation)
	mux.HandleFunc("GET /api/v1/admin/dashboard", s.dashboard)

	// Legacy unversioned API kept for the existing frontend.
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

// ready checks the dependencies the API needs to serve requests.
func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	if _, err := s.store.ListDocuments(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", s.corsAllow)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
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

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

// ---------- auth helpers ----------

// currentUser resolves the session cookie to a user. ok=false means the
// request is anonymous.
func (s *Server) currentUser(r *http.Request) (model.User, bool) {
	c, err := r.Cookie(sessionCookie)
	if err != nil || c.Value == "" {
		return model.User{}, false
	}
	u, err := s.store.GetSessionUser(r.Context(), c.Value)
	if err != nil {
		return model.User{}, false
	}
	return u, true
}

func (s *Server) requireUser(w http.ResponseWriter, r *http.Request) (model.User, bool) {
	u, ok := s.currentUser(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
	}
	return u, ok
}

// requireAdmin allows admin and expert roles (spec: для хакатона Admin и
// Expert объединены).
func (s *Server) requireAdmin(w http.ResponseWriter, r *http.Request) (model.User, bool) {
	u, ok := s.requireUser(w, r)
	if !ok {
		return model.User{}, false
	}
	if u.Role != "admin" && u.Role != "expert" {
		writeError(w, http.StatusForbidden, "admin role required")
		return model.User{}, false
	}
	return u, true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write response: %v", err)
	}
}

// writeError emits the spec §29 error shape.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]any{
			"code":    http.StatusText(status),
			"message": msg,
		},
	})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32<<20)).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
}
