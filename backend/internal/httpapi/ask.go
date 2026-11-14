package httpapi

import (
	"net/http"
	"strings"

	"phystech-answer/backend/internal/mlclient"
)

type askRequest struct {
	Question string `json:"question"`
	UserID   string `json:"userId,omitempty"`
}

type askResponse struct {
	Answer  string             `json:"answer"`
	Sources []mlclient.Source  `json:"sources"`
	// Stub is true when the ML service was unreachable and the answer is a fallback.
	Stub bool `json:"stub,omitempty"`
}

// ask forwards the question to the ML service synchronously — the legacy
// stateless endpoint kept for the existing frontend. The conversations
// API (POST /api/v1/conversations/{id}/messages) is the persistent path.
func (s *Server) ask(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireUser(w, r); !ok {
		return
	}
	var req askRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	question := strings.TrimSpace(req.Question)
	if question == "" {
		writeError(w, http.StatusBadRequest, "question is required")
		return
	}

	if s.ml == nil || !s.ml.Enabled() {
		s.writeStubAnswer(w)
		return
	}
	res, err := s.ml.Ask(r.Context(), question)
	if err != nil {
		s.writeStubAnswer(w)
		return
	}
	writeJSON(w, http.StatusOK, askResponse{
		Answer:  res.Answer,
		Sources: res.Sources,
		Stub:    res.Stub,
	})
}

func (s *Server) writeStubAnswer(w http.ResponseWriter) {
	writeJSON(w, http.StatusOK, askResponse{
		Answer:  "ML-сервис пока недоступен, это ответ-заглушка.",
		Sources: []mlclient.Source{},
		Stub:    true,
	})
}
