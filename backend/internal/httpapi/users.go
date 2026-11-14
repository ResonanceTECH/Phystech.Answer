package httpapi

import (
	"net/http"
	"strings"

	"phystech-answer/backend/internal/model"
)

func (s *Server) listUsers(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	users, err := s.store.ListUsers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list users")
		return
	}
	writeJSON(w, http.StatusOK, users)
}

type createUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (s *Server) createUser(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var req createUserRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	user, err := s.store.CreateUser(r.Context(), model.User{
		Name:  req.Name,
		Email: req.Email,
	}, "")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create user")
		return
	}
	writeJSON(w, http.StatusCreated, user)
}

func (s *Server) unreadDocuments(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	target := r.PathValue("id")
	if target != user.ID && user.Role != "admin" && user.Role != "expert" {
		writeError(w, http.StatusForbidden, "cannot view unread documents of another user")
		return
	}

	docs, err := s.store.ListDocuments(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list documents")
		return
	}
	reads, err := s.store.ReadStatus(r.Context(), target)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load read status")
		return
	}

	privileged := user.Role == "admin" || user.Role == "expert"
	unread := make([]model.Document, 0, len(docs))
	for _, d := range docs {
		if d.AccessLevel == "restricted" && !privileged {
			continue
		}
		if _, ok := reads[d.ID]; !ok {
			unread = append(unread, d)
		}
	}
	writeJSON(w, http.StatusOK, unread)
}
