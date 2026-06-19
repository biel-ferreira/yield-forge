package auth_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/biel-ferreira/yield-forge/internal/auth"
)

func TestNormalizeEmail(t *testing.T) {
	require.Equal(t, "user@example.com", auth.NormalizeEmail("  User@Example.COM "))
}

func TestValidateEmail(t *testing.T) {
	t.Run("valid is normalized", func(t *testing.T) {
		got, err := auth.ValidateEmail("  Alice@Example.com ")
		require.NoError(t, err)
		require.Equal(t, "alice@example.com", got)
	})

	t.Run("invalid returns ErrInvalidEmail", func(t *testing.T) {
		for _, in := range []string{"", "   ", "not-an-email", "no-at-sign.com"} {
			_, err := auth.ValidateEmail(in)
			require.ErrorIs(t, err, auth.ErrInvalidEmail, "input %q should be invalid", in)
		}
	})
}

func TestValidatePassword(t *testing.T) {
	require.NoError(t, auth.ValidatePassword("longenough1"))
	require.ErrorIs(t, auth.ValidatePassword("short"), auth.ErrWeakPassword)
}
