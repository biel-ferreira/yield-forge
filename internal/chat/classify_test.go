package chat

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClassify(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		intent  Intent
		amount  int64
		horizon int
	}{
		{"contribution with mil", "tenho 2 mil pra aportar esse mês", IntentContribution, 200_000, 0},
		{"contribution with R$", "onde foco os R$ 1.500 de aporte?", IntentContribution, 150_000, 0},
		{"contribution R$ + cents", "quero aportar R$2.000,50", IntentContribution, 200_050, 0},
		{"contribution reais", "vou contribuir 3000 reais", IntentContribution, 300_000, 0},
		{"projection with horizon", "como fica meu patrimônio daqui a 10 anos?", IntentProjection, 0, 10},
		{"projection passive income", "quanto de renda passiva isso gera?", IntentProjection, 0, 0},
		{"general question", "estou concentrado demais em logística?", IntentGeneral, 0, 0},
		{"amount without contribution signal → general", "gastei R$ 500 no mercado", IntentGeneral, 0, 0},
		{"aportar without amount → general", "qual a melhor forma de aportar?", IntentGeneral, 0, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			intent, amount, horizon := Classify(tc.text)
			require.Equal(t, tc.intent, intent)
			require.Equal(t, tc.amount, amount)
			require.Equal(t, tc.horizon, horizon)
		})
	}
}

func TestClassify_Deterministic(t *testing.T) {
	i1, a1, h1 := Classify("tenho 2 mil pra aportar")
	i2, a2, h2 := Classify("tenho 2 mil pra aportar")
	require.Equal(t, i1, i2)
	require.Equal(t, a1, a2)
	require.Equal(t, h1, h2)
}

func TestBRLToCentavos(t *testing.T) {
	require.Equal(t, int64(150_000), brlToCentavos("1.500"))
	require.Equal(t, int64(150_000), brlToCentavos("1.500,00"))
	require.Equal(t, int64(250), brlToCentavos("2,50"))
	require.Equal(t, int64(200), brlToCentavos("2"))
	require.Equal(t, int64(0), brlToCentavos("abc"))
}

func TestParseHorizonYears_Bounds(t *testing.T) {
	require.Equal(t, 10, parseHorizonYears("daqui a 10 anos"))
	require.Equal(t, 0, parseHorizonYears("daqui a 99 anos"), "out of 1..40 range")
	require.Equal(t, 0, parseHorizonYears("sem horizonte"))
}

func TestParseRole(t *testing.T) {
	r, err := ParseRole("  Assistant ")
	require.NoError(t, err)
	require.Equal(t, RoleAssistant, r)
	_, err = ParseRole("system")
	require.ErrorIs(t, err, ErrUnknownRole)
}
