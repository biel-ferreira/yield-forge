package auth_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/biel-ferreira/yield-forge/internal/auth"
)

func TestNewSessionToken_UniqueAndNonEmpty(t *testing.T) {
	t1, err := auth.NewSessionToken()
	require.NoError(t, err)
	require.NotEmpty(t, t1)

	t2, err := auth.NewSessionToken()
	require.NoError(t, err)
	require.NotEqual(t, t1, t2, "two tokens must differ (high entropy)")
}

func TestHashToken(t *testing.T) {
	raw, err := auth.NewSessionToken()
	require.NoError(t, err)

	h := auth.HashToken(raw)
	require.NotEqual(t, raw, h, "stored hash must not equal the raw token (BR-303)")
	require.Equal(t, h, auth.HashToken(raw), "hashing is deterministic")

	other, err := auth.NewSessionToken()
	require.NoError(t, err)
	require.NotEqual(t, h, auth.HashToken(other), "different tokens hash differently")
}
