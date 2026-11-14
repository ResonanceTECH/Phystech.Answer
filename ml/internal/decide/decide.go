// Package decide is a client for decision models served with the
// /v1/systemone API (H2O-Lightning on llama.cpp's llama-server). A decision
// model answers a typed question about a record with a calibrated
// probability in one forward pass — used here as a relevance judge:
// "does this fragment answer the employee's question?".
package decide

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/text/unicode/norm"
)

type Client struct {
	baseURL string
	hc      *http.Client
}

func New(baseURL string) *Client {
	if baseURL == "" {
		return nil
	}
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), hc: &http.Client{Timeout: 60 * time.Second}}
}

type question struct {
	Type         string            `json:"type"`
	Instructions string            `json:"instructions"`
	Criteria     map[string]string `json:"criteria,omitempty"`
}

type request struct {
	State     map[string]string   `json:"state"`
	Questions map[string]question `json:"questions"`
}

type response struct {
	Answers map[string]struct {
		Noul float64 `json:"noul"`
	} `json:"answers"`
}

const maxFragmentRunes = 2400

// Relevance returns P(fragment answers the question) in [0, 1].
func (c *Client) Relevance(ctx context.Context, q, header, fragment string) (float64, error) {
	if r := []rune(fragment); len(r) > maxFragmentRunes {
		fragment = string(r[:maxFragmentRunes])
	}
	// The model was trained on NFC text; llama.cpp does not normalise.
	nfc := func(s string) string { return norm.NFC.String(strings.ToValidUTF8(s, "")) }
	req := request{
		State: map[string]string{
			"вопрос сотрудника":  nfc(q),
			"документ":           nfc(header),
			"фрагмент документа": nfc(fragment),
		},
		Questions: map[string]question{
			"relevant": {
				Type:         "noul",
				Instructions: "Фрагмент документа содержит ответ на вопрос сотрудника или его существенную часть (шаг процедуры, срок, условие, ответственного).",
				Criteria: map[string]string{
					"true":  "Фрагмент отвечает на вопрос",
					"false": "Фрагмент не отвечает на вопрос",
				},
			},
		},
	}
	body, err := json.Marshal(req)
	if err != nil {
		return 0, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/systemone", bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.hc.Do(httpReq)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 300))
		return 0, fmt.Errorf("decision API returned %d: %s", resp.StatusCode, strings.TrimSpace(string(msg)))
	}
	var out response
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return 0, fmt.Errorf("decode decision response: %w", err)
	}
	a, ok := out.Answers["relevant"]
	if !ok {
		return 0, fmt.Errorf("decision API: no answer")
	}
	return a.Noul, nil
}
