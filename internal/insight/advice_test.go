package insight

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestContainsOrder is the non-advice corpus (SPEC-005 FR-504): order phrasings must be
// detected, and legitimate considerations / holding restatements must pass. The
// true-negatives are as important as the positives — over-rejection silently neuters
// legitimate portfolio intelligence (PRD FR-019).
func TestContainsOrder(t *testing.T) {
	t.Run("order signatures are detected", func(t *testing.T) {
		orders := []string{
			"Compre 100 cotas de HGLG11",
			"compre HGLG11 agora",
			"venda HGLG11 imediatamente",
			"venda 200 ações",
			"Buy 50 shares of HGLG11",
			"sell HGLG11 now",
			"Compre HGLG11 a R$ 160",
			"buy at $100",
			"venda a 120",
			"preço-alvo de R$ 120",
			"target price of $120",
			"ponto de entrada em R$ 95",
			"entry point at 95",
			"retorno garantido de 12%",
			"rentabilidade garantida",
			"guaranteed return of 12%",
			"garantido 15% ao ano",
		}
		for _, s := range orders {
			require.True(t, containsOrder(s), "should detect an order: %q", s)
		}
	})

	t.Run("considerations and holdings pass", func(t *testing.T) {
		ok := []string{
			"HGLG11 é um FII de logística que vale analisar",
			"HGLG11 is a logistics FII worth researching",
			"Sua carteira está sub-exposta ao setor de logística",
			"Você possui 100 cotas de HGLG11, cerca de 30% da carteira", // a holding, not an order
			"You currently hold 100 shares of HGLG11 (30% of the portfolio)",
			"Consider increasing your fixed-income exposure",
			"O setor de papel (ex.: KNCR11) está sub-representado na sua carteira",
			"A renda fixa pode merecer atenção no cenário atual de juros altos",
			"Vale pesquisar mais sobre FIIs de logística para a sua análise",
		}
		for _, s := range ok {
			require.False(t, containsOrder(s), "should NOT flag a consideration/holding: %q", s)
		}
	})
}
