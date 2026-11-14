package httpapi

import (
	"net/http"
	"phystech-answer/backend/internal/model"
	"strings"
)

type searchRequest struct {
	Query   string `json:"query"`
	Filters struct {
		DocumentIDs []string `json:"document_ids"`
		Categories  []string `json:"categories"`
		AsOfDate    *string  `json:"as_of_date"`
	} `json:"filters"`
	Limit int `json:"limit"`
}

// search is the catalog/fragment search endpoint (spec §12). Full-text
// search over published chunks; semantic search lives in the ML service.
func (s *Server) search(w http.ResponseWriter, r *http.Request) {
	var req searchRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	q := strings.TrimSpace(req.Query)
	if q == "" {
		writeError(w, http.StatusBadRequest, "query is required")
		return
	}

	hits, err := s.store.SearchChunks(r.Context(), q, req.Filters.DocumentIDs, req.Filters.Categories, req.Limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "search failed")
		return
	}

	user, _ := s.currentUser(r)
	// Access gate (spec §13): restricted docs are visible only to
	// admin/expert; public/internal are visible to everyone.
	privileged := user.Role == "admin" || user.Role == "expert"
	visible := hits[:0]
	for _, h := range hits {
		if !privileged && h.AccessLevel == "restricted" {
			continue
		}
		visible = append(visible, h)
	}
	if visible == nil {
		visible = []model.SearchHit{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"query":   q,
		"results": visible,
	})
}
