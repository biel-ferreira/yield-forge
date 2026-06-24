package insight

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
)

// stubInsighter returns a preset result/error, for testing the decorators.
type stubInsighter struct {
	result InsightResult
	err    error
	calls  int
}

func (s *stubInsighter) Generate(context.Context, InsightRequest) (InsightResult, error) {
	s.calls++
	return s.result, s.err
}

func discardLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestGated_PassesAndAttachesDisclaimer(t *testing.T) {
	stub := &stubInsighter{result: InsightResult{Insights: []Insight{{
		Title:       "Logística sub-representada",
		Detail:      "Sua carteira tem pouca exposição ao setor de logística.",
		Explanation: "Logística é ~5% da carteira, abaixo do típico para diversificação.",
	}}}}

	res, err := Gated(stub, discardLogger()).Generate(context.Background(), InsightRequest{})
	require.NoError(t, err)
	require.Equal(t, Disclaimer, res.Disclaimer, "every gated result carries the disclaimer")
	require.Len(t, res.Insights, 1)
}

func TestGated_RejectsMissingExplanation(t *testing.T) {
	stub := &stubInsighter{result: InsightResult{Insights: []Insight{
		{Title: "x", Detail: "y", Explanation: "   "}, // blank explanation
	}}}

	_, err := Gated(stub, discardLogger()).Generate(context.Background(), InsightRequest{})
	require.ErrorIs(t, err, ErrMissingExplanation)
}

func TestGated_RejectsOrder(t *testing.T) {
	stub := &stubInsighter{result: InsightResult{Insights: []Insight{
		{Title: "Ação sugerida", Detail: "Compre 100 cotas de HGLG11.", Explanation: "diversificação"},
	}}}

	_, err := Gated(stub, discardLogger()).Generate(context.Background(), InsightRequest{})
	require.ErrorIs(t, err, ErrAdviceDetected)
}

func TestGated_PropagatesProviderError(t *testing.T) {
	stub := &stubInsighter{err: ErrInsightsUnavailable}

	_, err := Gated(stub, discardLogger()).Generate(context.Background(), InsightRequest{})
	require.ErrorIs(t, err, ErrInsightsUnavailable)
}
