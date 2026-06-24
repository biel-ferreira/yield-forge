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
			// Imperative buy/sell + asset/quantity.
			"Compre 100 cotas de HGLG11",
			"compre HGLG11 agora",
			"venda HGLG11 imediatamente",
			"venda 200 ações",
			"Buy 50 shares of HGLG11",
			"sell HGLG11 now",
			"Compre HGLG11 a R$ 160",
			"buy at $100",
			"venda a 120",
			// Price / fair-value / entry-exit targets.
			"preço-alvo de R$ 120",
			"target price of $120",
			"O valor justo é R$ 160 por cota",
			"ponto de entrada em R$ 95",
			"entry point at 95",
			// Guaranteed-return claims.
			"retorno garantido de 12%",
			"rentabilidade garantida",
			"guaranteed return of 12%",
			"garantido 15% ao ano",
			// Advisory / infinitive moods (security-review bypasses — must now be caught).
			"Recomendo aumentar sua posição em HGLG11 para 200 cotas",
			"Sugiro adquirir mais KNRI11",
			"Minha sugestão é comprar HGLG11",
			"Você deveria vender tudo",
			"Sell half of your position now",
			"You should add 100 shares of HGLG11",
			"Consider increasing your allocation to HGLG11 by 200 cotas",
			"Recomendo comprar",
			"I recommend buying HGLG11",
			"Aconselho investir em KNRI11",
			// Order split onto its own line.
			"Compre.\nHGLG11 está barato, 100 cotas.",
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
			// Asset-class / diversification framing — guidance, not a per-asset order (FR-019).
			"Esta análise sugere que o setor de logística está concentrado",
			"Os dados sugerem manter uma carteira diversificada",
			"Recomendamos diversificar entre diferentes setores",
			"Aumentar a diversificação pode reduzir o risco da carteira",
			"Vale considerar uma maior exposição à renda fixa neste cenário",
			"Reduzir a concentração em um único setor tende a diminuir o risco",
			"Compreender o risco de cada ativo é parte da análise", // "Compre" is not a word boundary here
		}
		for _, s := range ok {
			require.False(t, containsOrder(s), "should NOT flag a consideration/holding: %q", s)
		}
	})
}
