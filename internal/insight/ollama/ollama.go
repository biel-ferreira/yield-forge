// Package ollama is the local-LLM adapter for the Insighter port (SPEC-005 FR-502): it
// talks to a local Ollama server's chat API. HTTP/wire details live here only; the
// prompt and parsing are shared in the insight core (BR-503). The local model means a
// user's portfolio facts never leave the machine — the privacy-preserving dev default.
package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/biel-ferreira/yield-forge/internal/insight"
)

// Adapter is an insight.Insighter backed by Ollama's /api/chat endpoint.
type Adapter struct {
	baseURL string
	model   string
	client  *http.Client
}

// Compile-time check that Adapter satisfies the port.
var _ insight.Insighter = (*Adapter)(nil)

// New returns an Ollama adapter. timeout bounds each generation so a hung model can't
// hang the caller (SPEC-005 FR-506).
func New(baseURL, model string, timeout time.Duration) *Adapter {
	return &Adapter{
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		client:  &http.Client{Timeout: timeout},
	}
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
	Format   string        `json:"format"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
}

// Generate prompts the local model and returns parsed insights. On a malformed reply it
// re-asks once; any other failure (or a second malformed reply) degrades to
// ErrInsightsUnavailable, preserving the cause for debugging (SPEC-005 FR-506).
func (a *Adapter) Generate(ctx context.Context, req insight.InsightRequest) (insight.InsightResult, error) {
	system, user, err := insight.BuildPrompt(req)
	if err != nil {
		return insight.InsightResult{}, err // ErrInsufficientFacts
	}

	result, genErr := a.callOnce(ctx, system, user)
	if genErr == nil {
		return result, nil
	}
	if errors.Is(genErr, insight.ErrMalformedResponse) {
		result, retryErr := a.callOnce(ctx, system, user+insight.ReAskSuffix)
		if retryErr == nil {
			return result, nil
		}
		genErr = retryErr
	}
	return insight.InsightResult{}, fmt.Errorf("%w: %v", insight.ErrInsightsUnavailable, genErr)
}

func (a *Adapter) callOnce(ctx context.Context, system, user string) (insight.InsightResult, error) {
	body, err := json.Marshal(chatRequest{
		Model: a.model,
		Messages: []chatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		Stream: false,
		Format: "json",
	})
	if err != nil {
		return insight.InsightResult{}, fmt.Errorf("encode request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return insight.InsightResult{}, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return insight.InsightResult{}, fmt.Errorf("ollama request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return insight.InsightResult{}, fmt.Errorf("ollama status %d", resp.StatusCode)
	}

	var cr chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return insight.InsightResult{}, fmt.Errorf("ollama decode: %w", err)
	}
	return insight.ParseResult(cr.Message.Content)
}
