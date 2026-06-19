package bcrypt_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/biel-ferreira/yield-forge/internal/auth"
	authbcrypt "github.com/biel-ferreira/yield-forge/internal/auth/bcrypt"
)

func TestHasher_HashAndCompare(t *testing.T) {
	h := authbcrypt.New()
	const password = "correct horse battery staple"

	hash, err := h.Hash(password)
	require.NoError(t, err)
	require.NotEqual(t, password, hash, "the hash must never equal the plaintext (BR-302)")

	require.NoError(t, h.Compare(hash, password), "correct password should verify")
	require.ErrorIs(t, h.Compare(hash, "wrong password"), auth.ErrInvalidCredentials,
		"wrong password should return the generic credentials error")
}

func TestHasher_MalformedHashIsNotInvalidCredentials(t *testing.T) {
	h := authbcrypt.New()

	err := h.Compare("not-a-bcrypt-hash", "whatever")
	require.Error(t, err)
	require.NotErrorIs(t, err, auth.ErrInvalidCredentials,
		"a malformed stored hash is an internal error, not a wrong-password result")
}
