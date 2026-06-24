package insight

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildPrompt(t *testing.T) {
	t.Run("empty facts is ErrInsufficientFacts", func(t *testing.T) {
		_, _, err := BuildPrompt(InsightRequest{Facts: nil})
		require.ErrorIs(t, err, ErrInsufficientFacts)
	})

	t.Run("builds system + user from facts", func(t *testing.T) {
		sys, user, err := BuildPrompt(InsightRequest{Facts: Facts{"total_centavos": 100000}, Task: "overview"})
		require.NoError(t, err)
		require.NotEmpty(t, sys)
		require.Contains(t, user, "overview")
		require.Contains(t, user, "total_centavos")
	})
}

func TestParseResult(t *testing.T) {
	t.Run("clean json", func(t *testing.T) {
		raw := `{"insights":[{"category":"c","title":"t","detail":"d","explanation":"e"}]}`
		res, err := ParseResult(raw)
		require.NoError(t, err)
		require.Len(t, res.Insights, 1)
		require.Equal(t, "e", res.Insights[0].Explanation)
		require.Empty(t, res.Disclaimer, "the gate attaches the disclaimer, not the parser")
	})

	t.Run("json wrapped in prose and code fences", func(t *testing.T) {
		raw := "Claro! Aqui está:\n```json\n{\"insights\":[{\"title\":\"t\",\"explanation\":\"e\"}]}\n```"
		res, err := ParseResult(raw)
		require.NoError(t, err)
		require.Len(t, res.Insights, 1)
	})

	t.Run("malformed is ErrMalformedResponse", func(t *testing.T) {
		_, err := ParseResult("desculpe, não consegui gerar")
		require.ErrorIs(t, err, ErrMalformedResponse)
	})
}
