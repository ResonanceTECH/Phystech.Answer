// Package rerank is a client for OpenAI-compatible /v1/rerank endpoints
// (llama.cpp server --reranking, Jina/Cohere-style APIs).
package rerank

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	apiKey  string
	model   string
	hc      *http.Client
}

func New(baseURL, apiKey, model string) *Client {
	if baseURL == "" {
		return nil
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		model:   model,
		hc:      &http.Client{Timeout: 120 * time.Second},
	}
}

type Score struct {
	Index          int     `json:"index"`
	RelevanceScore float64 `json:"relevance_score"`
}

type rerankResponse struct {
	Results []Score `json:"results"`
}

// Rerank scores query-document pairs and returns scores sorted descending.
// The returned slice has one entry per input document; Index refers to the
// original position in docs.
func (c *Client) Rerank(ctx context.Context, query string, docs []string) ([]Score, error) {
	if len(docs) == 0 {
		return nil, nil
	}
	body, err := json.Marshal(map[string]any{
		"model":     c.model,
		"query":     query,
		"documents": docs,
		"top_n":     len(docs),
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/rerank", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("rerank API returned %s", resp.Status)
	}
	var out rerankResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode rerank response: %w", err)
	}
	return out.Results, nil
}
