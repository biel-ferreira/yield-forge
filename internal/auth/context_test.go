package auth_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/biel-ferreira/yield-forge/internal/auth"
)

func TestUserID(t *testing.T) {
	t.Run("absent when unauthenticated", func(t *testing.T) {
		_, ok := auth.UserID(context.Background())
		require.False(t, ok)
	})

	t.Run("present after WithUserID", func(t *testing.T) {
		ctx := auth.WithUserID(context.Background(), "user-123")
		id, ok := auth.UserID(ctx)
		require.True(t, ok)
		require.Equal(t, "user-123", id)
	})

	t.Run("empty id is treated as absent", func(t *testing.T) {
		ctx := auth.WithUserID(context.Background(), "")
		_, ok := auth.UserID(ctx)
		require.False(t, ok)
	})
}
