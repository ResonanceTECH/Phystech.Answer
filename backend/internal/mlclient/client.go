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
	Title      string  `json:"title"`
	Section    string  `json:"section,omitempty"`
	Excerpt    string  `json:"excerpt"`
	Score      float64 `json:"score"`
}

// AskResult is the ML answer plus the degraded-mode flag.
type AskResult struct {
	Answer  string
	Sources []Source
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
		Answer  string   `json:"answer"`
		Sources []Source `json:"sources"`
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
	return AskResult{Answer: ml.Answer, Sources: ml.Sources}, nil
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
		if strings.TrimSpace(d.Content) == "" {
			continue // not parsed yet / requires review
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
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Content string   `json:"content"`
	Tags    []string `json:"tags"`
	Number  string   `json:"number,omitempty"`
	Type    string   `json:"type,omitempty"`
	Date    string   `json:"date,omitempty"` // YYYY-MM-DD
}

func toMLDoc(d model.Document) mlDoc {
	m := mlDoc{ID: d.ID, Title: d.Title, Content: d.Content, Tags: d.Tags,
		Number: d.DocumentNumber, Type: d.DocumentType}
	if d.DocumentDate != nil {
		m.Date = d.DocumentDate.Format("2006-01-02")
	}
	return m
}
