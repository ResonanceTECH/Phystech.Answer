package httpapi

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"phystech-answer/backend/internal/mlclient"
)

type askRequest struct {
	Question string `json:"question"`
	UserID   string `json:"userId,omitempty"`
}

type askResponse struct {
	Answer  string            `json:"answer"`
	Sources []mlclient.Source `json:"sources"`
	// Stub is true when the ML service was unreachable and the answer is a fallback.
	Stub   bool   `json:"stub,omitempty"`
	Status string `json:"status,omitempty"`
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
	body, _ := json.Marshal(map[string]string{"question": question})
	raw, err := s.ml.AskRaw(r.Context(), body)
	if err != nil {
		log.Printf("ask: ML unavailable: %v", err)
		s.writeStubAnswer(w)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(raw)
}

func (s *Server) writeStubAnswer(w http.ResponseWriter) {
	writeJSON(w, http.StatusOK, askResponse{
		Answer:  "Сервис поиска по документам временно недоступен. Попробуйте повторить запрос через минуту.",
		Sources: []mlclient.Source{},
		Stub:    true,
		Status:  "error",
	})
}
