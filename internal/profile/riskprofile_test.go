package profile

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseRiskProfile(t *testing.T) {
	t.Run("valid (normalized)", func(t *testing.T) {
		cases := map[string]RiskProfile{
			"conservative": RiskConservative,
			"Moderate":     RiskModerate,
			" AGGRESSIVE ": RiskAggressive,
		}
		for in, want := range cases {
			got, err := ParseRiskProfile(in)
			require.NoError(t, err, "input %q", in)
			require.Equal(t, want, got)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		for _, in := range []string{"", "risky", "balanced", "conservador"} {
			_, err := ParseRiskProfile(in)
			require.ErrorIs(t, err, ErrInvalidRiskProfile, "should reject %q", in)
		}
	})
}
