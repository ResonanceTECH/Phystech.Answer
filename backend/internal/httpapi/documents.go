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

// listDocuments returns all documents. If ?userId= is provided, each document
// carries the read status for that user.
func (s *Server) listDocuments(w http.ResponseWriter, r *http.Request) {
	docs, err := s.store.ListDocuments(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list documents")
		return
	}

	// Read status: explicit ?userId= or the signed-in user.
	userID := r.URL.Query().Get("userId")
	if userID == "" {
		if u, ok := s.currentUser(r); ok {
			userID = u.ID
		}
	}
	brief := r.URL.Query().Get("brief") == "1"
	var reads map[string]time.Time
	if userID != "" {
		if reads, err = s.store.ReadStatus(r.Context(), userID); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to load read status")
			return
		}
	}

	resp := make([]documentResponse, 0, len(docs))
	for _, d := range docs {
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
	doc, err := s.store.GetDocument(r.Context(), r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "document not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get document")
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
	var req markReadRequest
	if r.ContentLength != 0 && !decodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.UserID) == "" {
		if u, ok := s.currentUser(r); ok {
			req.UserID = u.ID
		}
	}
	if strings.TrimSpace(req.UserID) == "" {
		writeError(w, http.StatusBadRequest, "userId is required")
		return
	}

	readAt, err := s.store.MarkRead(r.Context(), r.PathValue("id"), req.UserID)
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
		"userId":     req.UserID,
		"readAt":     readAt,
	})
}
