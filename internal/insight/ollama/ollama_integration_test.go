package ollama_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/biel-ferreira/yield-forge/internal/insight"
	"github.com/biel-ferreira/yield-forge/internal/insight/ollama"
)

// TestOllama_LiveGeneration_Integration runs a real generation against a local Ollama
// when TEST_OLLAMA_URL is set (model from TEST_OLLAMA_MODEL, default llama3.1). It skips
// cleanly in -short mode and when no Ollama is configured — CI has none, and the
// deterministic fake covers CI. This proves real JSON-mode parsing end-to-end.
func TestOllama_LiveGeneration_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live Ollama test in -short mode")
	}
	url := os.Getenv("TEST_OLLAMA_URL")
	if url == "" {
		t.Skip("set TEST_OLLAMA_URL (a running Ollama) to run this integration test")
	}
	model := os.Getenv("TEST_OLLAMA_MODEL")
	if model == "" {
		model = "llama3.1"
	}

	req := insight.InsightRequest{
		Facts: insight.Facts{
			"logistica_pct":  5,
			"tipico_pct":     20,
			"total_centavos": 10_000_000,
			"renda_fixa_pct": 60,
			"selic_bps":      10_500,
		},
		Task:   "concentration",
		UserID: "u1",
	}

	res, err := ollama.New(url, model, 60*time.Second).Generate(context.Background(), req)
	require.NoError(t, err)
	require.NotEmpty(t, res.Insights, "a live model should return at least one insight")
	for _, in := range res.Insights {
		require.NotEmpty(t, in.Explanation, "every insight must carry an explanation")
	}
}
