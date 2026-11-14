package httpapi

import (
	"net/http"
	"strings"

	"phystech-answer/backend/internal/model"
)

func (s *Server) listUsers(w http.ResponseWriter, r *http.Request) {
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
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create user")
		return
	}
	writeJSON(w, http.StatusCreated, user)
}

func (s *Server) unreadDocuments(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")

	docs, err := s.store.ListDocuments(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list documents")
		return
	}
	reads, err := s.store.ReadStatus(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load read status")
		return
	}

	unread := make([]model.Document, 0, len(docs))
	for _, d := range docs {
		if _, ok := reads[d.ID]; !ok {
			unread = append(unread, d)
		}
	}
	writeJSON(w, http.StatusOK, unread)
}
