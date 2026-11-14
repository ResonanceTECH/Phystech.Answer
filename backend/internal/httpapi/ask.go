package httpapi

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

type askRequest struct {
	Question string `json:"question"`
	UserID   string `json:"userId,omitempty"`
}

type askResponse struct {
	Answer  string   `json:"answer"`
	Sources []string `json:"sources"`
	// Stub is true when the ML service was unreachable and the answer is a fallback.
	Stub bool `json:"stub"`
}

// ask forwards the question to the ML service (POST {ML_SERVICE_URL}/ask with
// {"question": ...}, expecting {"answer": ..., "sources": [...]}). While the ML
// service is offline it returns a stub answer with stub=true so the frontend
// can be developed independently.
func (s *Server) ask(w http.ResponseWriter, r *http.Request) {
	var req askRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	question := strings.TrimSpace(req.Question)
	if question == "" {
		writeError(w, http.StatusBadRequest, "question is required")
		return
	}

	body, err := json.Marshal(map[string]string{"question": question})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to encode question")
		return
	}
	mlReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost, s.mlURL+"/ask", bytes.NewReader(body))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "invalid ML service URL")
		return
	}
	mlReq.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(mlReq)
	if err != nil {
		log.Printf("ml service unreachable: %v", err)
		s.writeStubAnswer(w)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("ml service returned %s", resp.Status)
		s.writeStubAnswer(w)
		return
	}

	var ml askResponse
	if err := json.NewDecoder(resp.Body).Decode(&ml); err != nil {
		log.Printf("ml service returned malformed response: %v", err)
		s.writeStubAnswer(w)
		return
	}
	writeJSON(w, http.StatusOK, ml)
}

func (s *Server) writeStubAnswer(w http.ResponseWriter) {
	writeJSON(w, http.StatusOK, askResponse{
		Answer:  "ML-сервис пока недоступен, это ответ-заглушка.",
		Sources: []string{},
		Stub:    true,
	})
}
