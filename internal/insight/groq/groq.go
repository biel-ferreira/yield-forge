// Package groq is the hosted-LLM adapter for the Insighter port (SPEC-005 FR-502): it
// talks to Groq's OpenAI-compatible chat-completions API — the deployed path. Built on
// the OpenAI shape so OpenAI (paid) is a near-free future drop-in. HTTP/wire details and
// the API key live here only (BR-503); the prompt, parsing, and re-ask/degrade policy
// are shared in the insight core.
package groq

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/biel-ferreira/yield-forge/internal/insight"
)

// maxResponseBytes caps the response body read so a malfunctioning or hostile endpoint
// can't exhaust memory; an insight reply is small JSON, well under this (SPEC-005 FR-506).
const maxResponseBytes = 4 << 20 // 4 MiB

// Adapter is an insight.Insighter backed by Groq's /chat/completions endpoint.
type Adapter struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

// Compile-time check that Adapter satisfies the port.
var _ insight.Insighter = (*Adapter)(nil)

// New returns a Groq adapter. timeout bounds each generation (SPEC-005 FR-506).
func New(baseURL, apiKey, model string, timeout time.Duration) *Adapter {
	return &Adapter{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		model:   model,
		client:  &http.Client{Timeout: timeout},
	}
}

// Generate prompts the hosted model and returns parsed insights, with the shared re-ask
// + graceful-degradation policy (SPEC-005 FR-506). A non-200 (incl. 429 rate-limit) is
// not a malformed reply, so it degrades immediately with no charge-incurring retry
// (BR-504).
func (a *Adapter) Generate(ctx context.Context, req insight.InsightRequest) (insight.InsightResult, error) {
	return insight.GenerateWithReask(ctx, req, a.callOnce)
}

type chatRequest struct {
	Model          string          `json:"model"`
	Messages       []chatMessage   `json:"messages"`
	ResponseFormat *responseFormat `json:"response_format,omitempty"`
	Stream         bool            `json:"stream"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type responseFormat struct {
	Type string `json:"type"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func (a *Adapter) callOnce(ctx context.Context, system, user string) (insight.InsightResult, error) {
	body, err := json.Marshal(chatRequest{
		Model: a.model,
		Messages: []chatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		ResponseFormat: &responseFormat{Type: "json_object"},
		Stream:         false,
	})
	if err != nil {
		return insight.InsightResult{}, fmt.Errorf("encode request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return insight.InsightResult{}, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+a.apiKey)

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return insight.InsightResult{}, fmt.Errorf("groq request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Auth/rate-limit/server errors degrade (no re-ask, no charge-incurring retry).
		return insight.InsightResult{}, fmt.Errorf("groq status %d", resp.StatusCode)
	}

	var cr chatResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes)).Decode(&cr); err != nil {
		return insight.InsightResult{}, fmt.Errorf("groq decode: %w", err)
	}
	if len(cr.Choices) == 0 {
		return insight.InsightResult{}, fmt.Errorf("groq: empty choices: %w", insight.ErrMalformedResponse)
	}
	return insight.ParseResult(cr.Choices[0].Message.Content)
}
