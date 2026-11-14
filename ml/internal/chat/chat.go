// Package chat is a minimal OpenAI-compatible chat client.
// Works with llama.cpp server (http://127.0.0.1:8081/v1), Ollama, LM Studio,
// OpenAI and other providers exposing POST {base}/chat/completions.
package chat

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
	baseURL    string
	apiKey     string
	model      string
	noThinking bool
	hc         *http.Client
}

func New(baseURL, apiKey, model string) *Client {
	return newWith(baseURL, apiKey, model, false)
}

// NewNoThinking creates a client that sends
// chat_template_kwargs.enable_thinking=false (llama.cpp reasoning models).
func NewNoThinking(baseURL, apiKey, model string) *Client {
	return newWith(baseURL, apiKey, model, true)
}

func newWith(baseURL, apiKey, model string, noThinking bool) *Client {
	if baseURL == "" {
		return nil
	}
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		model:      model,
		noThinking: noThinking,
		hc:         &http.Client{Timeout: 300 * time.Second},
	}
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model              string         `json:"model"`
	Messages           []message      `json:"messages"`
	Temperature        float64        `json:"temperature"`
	MaxTokens          int            `json:"max_tokens,omitempty"`
	ChatTemplateKwargs map[string]any `json:"chat_template_kwargs,omitempty"`
	ResponseFormat     map[string]any `json:"response_format,omitempty"`
}

type usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}

type timings struct {
	PredictedPerSecond float64 `json:"predicted_per_second"`
	PredictedMS        float64 `json:"predicted_ms"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Role             string `json:"role"`
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage   usage   `json:"usage"`
	Timings timings `json:"timings"`
}

// Result carries the assistant reply plus generation stats.
type Result struct {
	Content       string
	FinishReason  string
	PromptTokens  int
	CompletionTok int
	TokensPerSec  float64
}

// Chat sends a system+user prompt pair and returns the assistant reply.
func (c *Client) Chat(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	res, err := c.ChatEx(ctx, ChatOptions{System: systemPrompt, User: userPrompt})
	if err != nil {
		return "", err
	}
	return res.Content, nil
}

type ChatOptions struct {
	System      string
	User        string
	Messages    []message // optional full history; takes precedence over System/User
	Temperature float64
	MaxTokens   int
	JSONMode    bool // request application/json response format
	NoThinking  *bool
}

func (c *Client) ChatEx(ctx context.Context, opts ChatOptions) (Result, error) {
	msgs := opts.Messages
	if msgs == nil {
		msgs = []message{{Role: "system", Content: opts.System}}
		msgs = append(msgs, message{Role: "user", Content: opts.User})
	}
	temp := opts.Temperature
	if temp == 0 {
		temp = 0.2
	}
	req := chatRequest{
		Model:       c.model,
		Messages:    msgs,
		Temperature: temp,
		MaxTokens:   opts.MaxTokens,
	}
	noThink := c.noThinking
	if opts.NoThinking != nil {
		noThink = *opts.NoThinking
	}
	if noThink {
		req.ChatTemplateKwargs = map[string]any{"enable_thinking": false}
	}
	if opts.JSONMode {
		req.ResponseFormat = map[string]any{"type": "json_object"}
	}
	body, err := json.Marshal(req)
	if err != nil {
		return Result{}, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return Result{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.hc.Do(httpReq)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var eb bytes.Buffer
		buf := make([]byte, 4096)
		n, _ := resp.Body.Read(buf)
		eb.Write(buf[:n])
		return Result{}, fmt.Errorf("chat API returned %s: %s", resp.Status, eb.String())
	}

	var out chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return Result{}, fmt.Errorf("decode chat response: %w", err)
	}
	if len(out.Choices) == 0 {
		return Result{}, fmt.Errorf("chat API returned no choices")
	}
	ch := out.Choices[0]
	content := strings.TrimSpace(ch.Message.Content)
	// Reasoning models may put the answer after a long thinking trace; when
	// thinking is disabled content holds the reply directly. If the model
	// still emits only reasoning, surface that as the content.
	if content == "" && ch.Message.ReasoningContent != "" {
		content = strings.TrimSpace(ch.Message.ReasoningContent)
	}
	return Result{
		Content:       content,
		FinishReason:  ch.FinishReason,
		PromptTokens:  out.Usage.PromptTokens,
		CompletionTok: out.Usage.CompletionTokens,
		TokensPerSec:  out.Timings.PredictedPerSecond,
	}, nil
}
