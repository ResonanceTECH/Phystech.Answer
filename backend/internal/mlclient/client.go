// Package mlclient is the adapter to the ML service (embeddings, retrieval,
// answer generation). It owns request/response formats so callers don't
// re-implement them.
package mlclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"phystech-answer/backend/internal/model"
)

type Client struct {
	baseURL string
	http    *http.Client
}

func New(baseURL string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), http: &http.Client{Timeout: timeout}}
}

func (c *Client) Enabled() bool { return c != nil && c.baseURL != "" }

// Source mirrors the citation format returned by the ML service.
type Source struct {
	DocumentID string  `json:"documentId"`
	ChunkID    string  `json:"chunkId,omitempty"`
	Title      string  `json:"title"`
	Section    string  `json:"section,omitempty"`
	Page       int     `json:"page,omitempty"`
	Excerpt    string  `json:"excerpt"`
	Score      float64 `json:"score"`
	Ref        int     `json:"ref,omitempty"` // [n] marker in our pipeline's answer
}

// Step mirrors the structured step returned by the ML answer payload.
type Step struct {
	Order       int      `json:"order"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Responsible *string  `json:"responsible"`
	Deadline    *string  `json:"deadline"`
	Condition   *string  `json:"condition"`
	CitationIDs []string `json:"citation_ids"`
	// Our pipeline's compact step form: text with [n] markers + source refs.
	Text string `json:"text,omitempty"`
	Refs []int  `json:"refs,omitempty"`
}

// Conflict mirrors a detected contradiction between sources.
type Conflict struct {
	Description string   `json:"description"`
	ChunkIDs    []string `json:"chunk_ids"`
}

// AskResult is the ML answer plus the degraded-mode flag.
type AskResult struct {
	Answer             string
	Summary            string
	Sources            []Source
	Status             string
	Steps              []Step
	Warnings           []string
	MissingInformation []string
	Conflicts          []Conflict
	// Stub is true when ML was unreachable and this is a fallback reply.
	Stub bool
}

func (c *Client) do(ctx context.Context, method, path string, body any, out any) (int, error) {
	var rdr io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return 0, err
		}
		rdr = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, rdr)
	if err != nil {
		return 0, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if out != nil && resp.StatusCode == http.StatusOK {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return resp.StatusCode, fmt.Errorf("decode ML response: %w", err)
		}
	}
	return resp.StatusCode, nil
}

// Ask runs the RAG pipeline in the ML service.
func (c *Client) Ask(ctx context.Context, question string) (AskResult, error) {
	var ml struct {
		Answer             string     `json:"answer"`
		Summary            string     `json:"summary"`
		Sources            []Source   `json:"sources"`
		Status             string     `json:"status"`
		Steps              []Step     `json:"steps"`
		Notes              []string   `json:"notes"`
		Warnings           []string   `json:"warnings"`
		Missing            string     `json:"missing"`
		MissingInformation []string   `json:"missing_information"`
		Conflicts          []Conflict `json:"conflicts"`
	}
	code, err := c.do(ctx, http.MethodPost, "/api/ask",
		map[string]string{"question": question}, &ml)
	if err != nil {
		return AskResult{}, err
	}
	if code != http.StatusOK {
		return AskResult{}, fmt.Errorf("ML service returned status %d", code)
	}
	if ml.Sources == nil {
		ml.Sources = []Source{}
	}
	normalizeCompactSteps(ml.Steps, ml.Sources)
	warnings := append(ml.Notes, ml.Warnings...)
	missing := ml.MissingInformation
	if strings.TrimSpace(ml.Missing) != "" {
		missing = append(missing, ml.Missing)
	}
	return AskResult{
		Answer: ml.Answer, Summary: ml.Summary, Sources: ml.Sources, Status: ml.Status,
		Steps: ml.Steps, Warnings: warnings,
		MissingInformation: missing, Conflicts: ml.Conflicts,
	}, nil
}

// normalizeCompactSteps maps our pipeline's {text, refs} steps onto the
// structured form: the text becomes title/description (keeping [n] markers
// for the UI) and refs are linked to sources through synthetic chunk ids.
func normalizeCompactSteps(steps []Step, sources []Source) {
	byRef := map[int]int{}
	for i := range sources {
		ref := sources[i].Ref
		if ref <= 0 {
			ref = i + 1
		}
		byRef[ref] = i
	}
	for i := range steps {
		st := &steps[i]
		if st.Text == "" || st.Description != "" {
			continue
		}
		st.Description = st.Text
		st.Title = truncateRunes(st.Text, 120)
		for _, ref := range st.Refs {
			idx, ok := byRef[ref]
			if !ok {
				continue
			}
			if sources[idx].ChunkID == "" {
				sources[idx].ChunkID = fmt.Sprintf("ref:%d", ref)
			}
			st.CitationIDs = append(st.CitationIDs, sources[idx].ChunkID)
		}
	}
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

// AskRaw returns the full ML answer JSON (status, summary, steps,
// conflicts, sources, trace) without re-shaping it, so new answer fields
// reach the UI without backend changes.
func (c *Client) AskRaw(ctx context.Context, body []byte) (json.RawMessage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/ask", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ML service returned status %d", resp.StatusCode)
	}
	return data, nil
}

// IndexDocument upserts a document into the ML search index.
func (c *Client) IndexDocument(ctx context.Context, doc model.Document) error {
	if !c.Enabled() {
		return nil
	}
	if doc.Visibility != "" && doc.Visibility != "active" {
		return c.RemoveDocument(ctx, doc.ID) // hidden documents are not searchable
	}
	code, err := c.do(ctx, http.MethodPut, "/api/documents/"+doc.ID, toMLDoc(doc), nil)
	if err != nil {
		return err
	}
	if code >= 300 {
		return fmt.Errorf("ML index returned status %d", code)
	}
	return nil
}

// RemoveDocument drops a document from the ML index.
func (c *Client) RemoveDocument(ctx context.Context, id string) error {
	if !c.Enabled() {
		return nil
	}
	_, err := c.do(ctx, http.MethodDelete, "/api/documents/"+id, nil, nil)
	return err
}

// SyncDocuments replaces the whole ML index with the given set.
func (c *Client) SyncDocuments(ctx context.Context, docs []model.Document) error {
	if !c.Enabled() {
		return nil
	}
	out := make([]mlDoc, 0, len(docs))
	for _, d := range docs {
		if strings.TrimSpace(d.Content) == "" || (d.Visibility != "" && d.Visibility != "active") {
			continue // not parsed yet / requires review / hidden from the assistant
		}
		out = append(out, toMLDoc(d))
	}
	code, err := c.do(ctx, http.MethodPost, "/api/sync",
		map[string]any{"documents": out}, nil)
	if err != nil {
		return err
	}
	if code >= 300 {
		return fmt.Errorf("ML sync returned status %d", code)
	}
	return nil
}

// mlDoc is the document payload of the ML index API.
type mlDoc struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	Content        string   `json:"content"`
	Tags           []string `json:"tags"`
	DocumentNumber string   `json:"documentNumber,omitempty"`
	DocumentType   string   `json:"documentType,omitempty"`
	DocumentDate   string   `json:"documentDate,omitempty"` // YYYY-MM-DD
	Category       string   `json:"category,omitempty"`
	LegalStatus    string   `json:"legalStatus,omitempty"`
}

func toMLDoc(d model.Document) mlDoc {
	m := mlDoc{ID: d.ID, Title: d.Title, Content: d.Content, Tags: d.Tags,
		DocumentNumber: d.DocumentNumber, DocumentType: d.DocumentType,
		Category: d.Category, LegalStatus: d.LegalStatus}
	if d.DocumentDate != nil {
		m.DocumentDate = d.DocumentDate.Format("2006-01-02")
	}
	return m
}
