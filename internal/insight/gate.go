package insight

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// Disclaimer is the non-advice disclaimer attached to every gated result (SPEC-005
// FR-014, PRD FR-021). It is a deliberate PT-BR user-facing string (the audience is
// Brazilian retail investors), mirroring the insight content language.
const Disclaimer = "Estas são considerações educacionais para a sua própria análise — " +
	"não são recomendações de compra ou venda nem aconselhamento financeiro. " +
	"A decisão final é sempre sua."

// gated wraps an Insighter with the two binding-guard gates (SPEC-005 FR-503/FR-504,
// BR-501): every returned insight must carry an explanation, none may contain an order
// signature, and the non-advice disclaimer is always attached. The gates fail closed —
// a violation rejects the whole result rather than passing it through (BR-506) — and a
// rejection logs the reason code only, never the content (BR-505).
//
// (FR-020 risk/assumption disclosure for suggestions extends this same loop when
// suggestions land in SPEC-105 — a clean addition, no rework.)
type gated struct {
	next   Insighter
	logger *slog.Logger
}

// Gated returns inner wrapped with the explainability + non-advice gates. Apply it
// centrally so every provider is guarded by construction.
func Gated(inner Insighter, logger *slog.Logger) Insighter {
	return gated{next: inner, logger: logger}
}

func (g gated) Generate(ctx context.Context, req InsightRequest) (InsightResult, error) {
	result, err := g.next.Generate(ctx, req)
	if err != nil {
		return InsightResult{}, err
	}

	for _, in := range result.Insights {
		if strings.TrimSpace(in.Explanation) == "" {
			g.reject(ctx, "insight rejected: missing explanation", req.Task)
			return InsightResult{}, fmt.Errorf("gate: %w", ErrMissingExplanation)
		}
		if containsOrder(insightText(in)) {
			g.reject(ctx, "insight rejected: order signature", req.Task)
			return InsightResult{}, fmt.Errorf("gate: %w", ErrAdviceDetected)
		}
	}

	// Always attach the non-advice disclaimer (FR-014).
	result.Disclaimer = Disclaimer
	return result, nil
}

// reject logs a gate rejection with the reason code + task only — never the insight
// content (BR-505).
func (g gated) reject(ctx context.Context, msg string, task Task) {
	if g.logger != nil {
		g.logger.WarnContext(ctx, msg, slog.String("task", string(task)))
	}
}
