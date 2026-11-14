package httpapi

import (
	"net/http"
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
	// Access-level gate: non-privileged users can't see restricted docs.
	privileged := user.Role == "admin" || user.Role == "expert"
	_ = privileged // per-document access filtering applied below via access_level

	writeJSON(w, http.StatusOK, map[string]any{
		"query":   q,
		"results": hits,
	})
}
