package insight

import (
	"encoding/json"
	"fmt"
	"strings"
)

// systemPrompt is the STABLE guardrail + output-format envelope shared by every
// provider adapter (SPEC-005 BR-502 / FR-014). It is the FIRST line of defense —
// steering the model away from orders and fabrication; the deterministic gate is the
// backstop. It requests structured JSON (D2) so the explainability gate is reliable.
//
// Instructions are in English on purpose: LLMs (especially smaller local models) follow
// English instructions more reliably, and the model is told to write the user-facing
// text in Brazilian Portuguese. This prompt is deliberately MINIMAL and task-agnostic —
// the per-task intelligence (the rich prompt built from the user's profile + holdings +
// market facts) is the Fact Builder + task prompts owned by SPEC-104, which fill the
// user message; this constant only carries the safety + format contract.
const systemPrompt = `You are a portfolio-analysis assistant for a Brazilian retail investor.

Hard rules (never violate):
- Reason ONLY over the facts given in the user message. NEVER invent numbers, prices, or percentages — if a figure is not in the facts, do not state it.
- Produce explainable observations framed as AREAS / CONSIDERATIONS for the user's own analysis.
- NEVER issue a transaction order: no imperative buy/sell of an asset, no quantities, no price or entry/exit targets, no guaranteed-return claims.
- You MAY name a sector or asset as a candidate to research.
- Write all user-facing text (title, detail, explanation) in Brazilian Portuguese (pt-BR).

Respond with ONLY valid JSON in exactly this shape (no text outside the JSON):
{"insights":[{"category":"...","title":"...","detail":"...","explanation":"..."}]}
Every insight MUST include "explanation" (why it was raised).`

// ReAskSuffix is appended to the user prompt on a single retry after a malformed reply.
const ReAskSuffix = "\n\nIMPORTANT: respond with ONLY valid JSON in the requested shape, no extra text."

// BuildPrompt builds the provider-neutral system + user prompts from a request. The
// facts are serialized into the user prompt; an empty facts set is ErrInsufficientFacts
// (BR-502 — no facts, no fabricated answer). Adapters wrap these strings in their own
// chat-message format.
func BuildPrompt(req InsightRequest) (system, user string, err error) {
	if len(req.Facts) == 0 {
		return "", "", ErrInsufficientFacts
	}
	factsJSON, err := json.Marshal(req.Facts)
	if err != nil {
		return "", "", fmt.Errorf("marshal facts: %w", err)
	}
	user = fmt.Sprintf("Tarefa: %s\n\nFatos da carteira (JSON):\n%s", req.Task, factsJSON)
	return systemPrompt, user, nil
}

// llmResponse is the JSON shape adapters request from the model.
type llmResponse struct {
	Insights []struct {
		Category    string `json:"category"`
		Title       string `json:"title"`
		Detail      string `json:"detail"`
		Explanation string `json:"explanation"`
	} `json:"insights"`
}

// ParseResult parses a model's reply into an InsightResult (no Disclaimer — the gate
// attaches it). It tolerates a model wrapping JSON in prose/code-fences by extracting
// the outermost {...}. A reply that still won't parse returns ErrMalformedResponse, the
// signal an adapter uses to re-ask once.
func ParseResult(raw string) (InsightResult, error) {
	var resp llmResponse
	if err := json.Unmarshal([]byte(extractJSON(raw)), &resp); err != nil {
		return InsightResult{}, fmt.Errorf("parse insights: %w", ErrMalformedResponse)
	}
	result := InsightResult{Insights: make([]Insight, 0, len(resp.Insights))}
	for _, in := range resp.Insights {
		result.Insights = append(result.Insights, Insight{
			Category:    in.Category,
			Title:       in.Title,
			Detail:      in.Detail,
			Explanation: in.Explanation,
		})
	}
	return result, nil
}

// extractJSON returns the substring from the first '{' to the last '}', so a reply that
// wraps the JSON in prose or ```json fences still parses. Returns s unchanged if no
// brace pair is found (Unmarshal then yields ErrMalformedResponse).
func extractJSON(s string) string {
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start < 0 || end <= start {
		return s
	}
	return s[start : end+1]
}
