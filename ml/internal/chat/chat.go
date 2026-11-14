// Package chat is a minimal OpenAI-compatible chat client.
// Works with Ollama (local or Ollama Cloud), vLLM, LM Studio, OpenAI and
// other providers exposing POST {base}/chat/completions.
package chat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

type Client struct {
	baseURL string
	apiKey  string
	model   string
	hc      *http.Client
	// reasoning is sent as reasoning_effort ("none" disables the thinking
	// phase of hybrid reasoning models such as Qwen 3.5 — grounded RAG
	// answers do not need it and it multiplies latency).
	reasoning string
	// noJSONMode is set once the provider rejects response_format for this
	// model; later calls rely on the prompt alone.
	noJSONMode atomic.Bool
}

func New(baseURL, apiKey, model, reasoning string) *Client {
	if baseURL == "" {
		return nil
	}
	return &Client{
		baseURL:   baseURL,
		apiKey:    apiKey,
		model:     model,
		reasoning: reasoning,
		hc:        &http.Client{Timeout: 180 * time.Second},
	}
}

func (c *Client) Model() string { return c.model }

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model          string          `json:"model"`
	Messages       []message       `json:"messages"`
	Temperature    float64         `json:"temperature"`
	Seed           int             `json:"seed,omitempty"`
	Reasoning      string          `json:"reasoning_effort,omitempty"`
	ResponseFormat *responseFormat `json:"response_format,omitempty"`
}

type responseFormat struct {
	Type string `json:"type"`
}

type chatResponse struct {
	Choices []struct {
		Message message `json:"message"`
	} `json:"choices"`
}

// Chat sends a system+user prompt pair and returns the assistant reply.
func (c *Client) Chat(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	return c.do(ctx, systemPrompt, userPrompt, false)
}

// ChatJSON asks for a JSON object reply (response_format=json_object). If
// the provider rejects the parameter, it retries without it — the prompt
// itself requests JSON.
func (c *Client) ChatJSON(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	if c.noJSONMode.Load() {
		return c.do(ctx, systemPrompt, userPrompt, false)
	}
	out, err := c.do(ctx, systemPrompt, userPrompt, true)
	if err != nil && isUnsupportedJSONMode(err) {
		c.noJSONMode.Store(true)
		return c.do(ctx, systemPrompt, userPrompt, false)
	}
	return out, err
}

// isUnsupportedJSONMode: providers answer 400 ("unknown parameter") or 501
// ("structured output is unavailable") when response_format is unsupported.
func isUnsupportedJSONMode(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, " 400:") || strings.Contains(msg, " 501:") ||
		strings.Contains(msg, "structured output") || strings.Contains(msg, "response_format")
}

func (c *Client) do(ctx context.Context, systemPrompt, userPrompt string, jsonMode bool) (string, error) {
	req := chatRequest{
		Model: c.model,
		Messages: []message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0, // deterministic, grounded answers
		Seed:        42,
		Reasoning:   c.reasoning,
	}
	if jsonMode {
		req.ResponseFormat = &responseFormat{Type: "json_object"}
	}
	body, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(time.Duration(attempt) * time.Second):
			}
		}
		out, retry, err := c.send(ctx, body)
		if err == nil {
			return out, nil
		}
		lastErr = err
		if !retry {
			break
		}
	}
	return "", lastErr
}

// send performs one request; retry reports whether the error is transient.
func (c *Client) send(ctx context.Context, body []byte) (string, bool, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", false, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	resp, err := c.hc.Do(httpReq)
	if err != nil {
		return "", true, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 300))
		retry := resp.StatusCode == 429 || (resp.StatusCode >= 500 && resp.StatusCode != 501)
		return "", retry, fmt.Errorf("chat API returned %d: %s", resp.StatusCode, strings.TrimSpace(string(msg)))
	}
	var out chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", true, fmt.Errorf("decode chat response: %w", err)
	}
	if len(out.Choices) == 0 || strings.TrimSpace(out.Choices[0].Message.Content) == "" {
		return "", true, fmt.Errorf("chat API returned empty reply")
	}
	return strings.TrimSpace(out.Choices[0].Message.Content), false, nil
}
