package profile

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseObjectives(t *testing.T) {
	t.Run("dedupes preserving first-seen order", func(t *testing.T) {
		got, err := ParseObjectives([]string{"retirement", "Passive_Income", "retirement", " long_term_growth "})
		require.NoError(t, err)
		require.Equal(t, []Objective{ObjectiveRetirement, ObjectivePassiveIncome, ObjectiveLongTermGrowth}, got)
	})

	t.Run("empty is rejected", func(t *testing.T) {
		_, err := ParseObjectives(nil)
		require.ErrorIs(t, err, ErrNoObjectives)
	})

	t.Run("unknown objective is rejected", func(t *testing.T) {
		_, err := ParseObjectives([]string{"retirement", "yolo"})
		require.ErrorIs(t, err, ErrInvalidObjective)
	})
}
