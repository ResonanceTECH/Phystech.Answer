package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"phystech-answer/backend/internal/model"
	"phystech-answer/backend/internal/store"
)

type documentResponse struct {
	model.Document
	Read   bool       `json:"read"`
	ReadAt *time.Time `json:"readAt,omitempty"`
}

func isPrivileged(u model.User) bool {
	return u.Role == "admin" || u.Role == "expert"
}

// listDocuments returns all documents visible to the caller. If ?userId= is
// provided (admin only), each document carries the read status for that user.
func (s *Server) listDocuments(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	docs, err := s.store.ListDocuments(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list documents")
		return
	}

	// Access gate (spec §13): restricted docs are visible only to admin/expert.
	visible := docs[:0]
	for _, d := range docs {
		if d.AccessLevel == "restricted" && !isPrivileged(user) {
			continue
		}
		visible = append(visible, d)
	}
	docs = visible

	readFor := user.ID
	if q := r.URL.Query().Get("userId"); q != "" {
		if !isPrivileged(user) {
			writeError(w, http.StatusForbidden, "cannot read status of another user")
			return
		}
		readFor = q
	}
	brief := r.URL.Query().Get("brief") == "1"
	reads, err := s.store.ReadStatus(r.Context(), readFor)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load read status")
		return
	}

	// Скрытые и архивные документы видят только администраторы.
	isAdmin := isPrivileged(user)
	resp := make([]documentResponse, 0, len(docs))
	for _, d := range docs {
		if !isAdmin && d.Visibility != "" && d.Visibility != "active" {
			continue
		}
		if brief {
			d.Content = "" // catalog/onboarding lists don't need full texts
		}
		item := documentResponse{Document: d}
		if t, ok := reads[d.ID]; ok {
			item.Read = true
			item.ReadAt = &t
		}
		resp = append(resp, item)
	}
	writeJSON(w, http.StatusOK, resp)
}

type documentRequest struct {
	Title   string   `json:"title"`
	Content string   `json:"content"`
	Tags    []string `json:"tags"`
}

func (s *Server) createDocument(w http.ResponseWriter, r *http.Request) {
	// Changing the knowledge base is admin/expert-only.
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var req documentRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	doc, err := s.store.CreateDocument(r.Context(), model.Document{
		Title:   req.Title,
		Content: req.Content,
		Tags:    req.Tags,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create document")
		return
	}
	s.indexDocument(r.Context(), doc)
	writeJSON(w, http.StatusCreated, doc)
}

func (s *Server) getDocument(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	doc, err := s.store.GetDocument(r.Context(), r.PathValue("id"))
	if err == nil && doc.Visibility != "" && doc.Visibility != "active" && !isPrivileged(user) {
		err = store.ErrNotFound
	}
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "document not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get document")
		return
	}
	if doc.AccessLevel == "restricted" && !isPrivileged(user) {
		writeError(w, http.StatusNotFound, "document not found")
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

func (s *Server) updateDocument(w http.ResponseWriter, r *http.Request) {
	// Changing the knowledge base is admin/expert-only.
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var req documentRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	doc, err := s.store.UpdateDocument(r.Context(), model.Document{
		ID:      r.PathValue("id"),
		Title:   req.Title,
		Content: req.Content,
		Tags:    req.Tags,
	})
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "document not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update document")
		return
	}
	s.indexDocument(r.Context(), doc)
	writeJSON(w, http.StatusOK, doc)
}

func (s *Server) deleteDocument(w http.ResponseWriter, r *http.Request) {
	// Changing the knowledge base is admin/expert-only.
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	err := s.store.DeleteDocument(r.Context(), r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "document not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete document")
		return
	}
	s.removeFromIndex(r.Context(), r.PathValue("id"))
	w.WriteHeader(http.StatusNoContent)
}

type markReadRequest struct {
	UserID string `json:"userId"`
}

func (s *Server) markRead(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	var req markReadRequest
	if r.ContentLength != 0 && !decodeJSON(w, r, &req) {
		return
	}
	// Read marks always apply to the session user; admins may mark on
	// behalf of another user explicitly.
	target := user.ID
	if req.UserID != "" && req.UserID != user.ID {
		if !isPrivileged(user) {
			writeError(w, http.StatusForbidden, "cannot mark read for another user")
			return
		}
		target = req.UserID
	}

	readAt, err := s.store.MarkRead(r.Context(), r.PathValue("id"), target)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "document or user not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to mark document as read")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"documentId": r.PathValue("id"),
		"userId":     target,
		"readAt":     readAt,
	})
}
