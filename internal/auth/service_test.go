package auth_test

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/biel-ferreira/yield-forge/internal/auth"
)

// --- hand-written fakes for the ports (CLAUDE.md: no gomock/mockery) ---

type fakeUsers struct {
	byEmail map[string]auth.User
	byID    map[string]auth.User
}

func newFakeUsers() *fakeUsers {
	return &fakeUsers{byEmail: map[string]auth.User{}, byID: map[string]auth.User{}}
}

func (f *fakeUsers) CreateUser(_ context.Context, email, hash string) (auth.User, error) {
	if _, ok := f.byEmail[email]; ok {
		return auth.User{}, auth.ErrEmailTaken
	}
	u := auth.User{ID: "user-" + strconv.Itoa(len(f.byEmail)+1), Email: email, PasswordHash: hash}
	f.byEmail[email] = u
	f.byID[u.ID] = u
	return u, nil
}

func (f *fakeUsers) UserByEmail(_ context.Context, email string) (auth.User, error) {
	u, ok := f.byEmail[email]
	if !ok {
		return auth.User{}, auth.ErrUserNotFound
	}
	return u, nil
}

func (f *fakeUsers) UserByID(_ context.Context, id string) (auth.User, error) {
	u, ok := f.byID[id]
	if !ok {
		return auth.User{}, auth.ErrUserNotFound
	}
	return u, nil
}

type fakeSessions struct {
	byHash map[string]auth.Session
}

func newFakeSessions() *fakeSessions { return &fakeSessions{byHash: map[string]auth.Session{}} }

func (f *fakeSessions) CreateSession(_ context.Context, userID, tokenHash string, expiresAt time.Time) (auth.Session, error) {
	s := auth.Session{ID: "sess-" + strconv.Itoa(len(f.byHash)+1), UserID: userID, TokenHash: tokenHash, ExpiresAt: expiresAt}
	f.byHash[tokenHash] = s
	return s, nil
}

func (f *fakeSessions) SessionByTokenHash(_ context.Context, tokenHash string) (auth.Session, error) {
	s, ok := f.byHash[tokenHash]
	if !ok {
		return auth.Session{}, auth.ErrSessionNotFound
	}
	return s, nil
}

func (f *fakeSessions) DeleteSession(_ context.Context, tokenHash string) error {
	delete(f.byHash, tokenHash)
	return nil
}

// fakeHasher is a trivial, fast stand-in for bcrypt (tests must not pay bcrypt cost).
type fakeHasher struct{}

func (fakeHasher) Hash(p string) (string, error) { return "hash:" + p, nil }
func (fakeHasher) Compare(hash, p string) error {
	if hash == "hash:"+p {
		return nil
	}
	return auth.ErrInvalidCredentials
}

// fakeClock returns a fixed, mutable time so expiry is deterministic.
type fakeClock struct{ t time.Time }

func (c *fakeClock) Now() time.Time { return c.t }

func newService(clk *fakeClock, ttl time.Duration) (*auth.Service, *fakeUsers, *fakeSessions) {
	users, sessions := newFakeUsers(), newFakeSessions()
	return auth.NewService(users, sessions, fakeHasher{}, clk, ttl), users, sessions
}

// --- tests ---

func TestService_Register(t *testing.T) {
	ctx := context.Background()
	clk := &fakeClock{t: time.Unix(1_700_000_000, 0).UTC()}

	t.Run("creates a user and normalizes email", func(t *testing.T) {
		svc, _, _ := newService(clk, time.Hour)
		u, err := svc.Register(ctx, "  Alice@Example.com ", "supersecret")
		require.NoError(t, err)
		require.Equal(t, "alice@example.com", u.Email)
		require.Equal(t, "hash:supersecret", u.PasswordHash)
	})

	t.Run("duplicate email is ErrEmailTaken", func(t *testing.T) {
		svc, _, _ := newService(clk, time.Hour)
		_, err := svc.Register(ctx, "bob@example.com", "supersecret")
		require.NoError(t, err)
		_, err = svc.Register(ctx, "bob@example.com", "anothersecret")
		require.ErrorIs(t, err, auth.ErrEmailTaken)
	})

	t.Run("invalid email and weak password are rejected", func(t *testing.T) {
		svc, _, _ := newService(clk, time.Hour)
		_, err := svc.Register(ctx, "not-an-email", "supersecret")
		require.ErrorIs(t, err, auth.ErrInvalidEmail)
		_, err = svc.Register(ctx, "carol@example.com", "short")
		require.ErrorIs(t, err, auth.ErrWeakPassword)
	})
}

func TestService_Login(t *testing.T) {
	ctx := context.Background()
	clk := &fakeClock{t: time.Unix(1_700_000_000, 0).UTC()}

	t.Run("valid credentials issue a session", func(t *testing.T) {
		svc, _, sessions := newService(clk, time.Hour)
		_, err := svc.Register(ctx, "dora@example.com", "supersecret")
		require.NoError(t, err)

		user, token, err := svc.Login(ctx, "Dora@example.com", "supersecret")
		require.NoError(t, err)
		require.Equal(t, "dora@example.com", user.Email)
		require.NotEmpty(t, token)
		// The stored session is keyed by the HASH of the token, never the raw token.
		require.Contains(t, sessions.byHash, auth.HashToken(token))
		require.NotContains(t, sessions.byHash, token)
	})

	t.Run("wrong password and unknown email both return the generic error", func(t *testing.T) {
		svc, _, _ := newService(clk, time.Hour)
		_, err := svc.Register(ctx, "erin@example.com", "supersecret")
		require.NoError(t, err)

		_, _, err = svc.Login(ctx, "erin@example.com", "wrongpassword")
		require.ErrorIs(t, err, auth.ErrInvalidCredentials)

		_, _, err = svc.Login(ctx, "nobody@example.com", "whatever123")
		require.ErrorIs(t, err, auth.ErrInvalidCredentials)
	})
}

func TestService_Authenticate(t *testing.T) {
	ctx := context.Background()

	t.Run("valid token resolves to the user", func(t *testing.T) {
		clk := &fakeClock{t: time.Unix(1_700_000_000, 0).UTC()}
		svc, _, _ := newService(clk, time.Hour)
		_, err := svc.Register(ctx, "finn@example.com", "supersecret")
		require.NoError(t, err)
		_, token, err := svc.Login(ctx, "finn@example.com", "supersecret")
		require.NoError(t, err)

		user, err := svc.Authenticate(ctx, token)
		require.NoError(t, err)
		require.Equal(t, "finn@example.com", user.Email)
	})

	t.Run("expired session is rejected and cleaned up", func(t *testing.T) {
		clk := &fakeClock{t: time.Unix(1_700_000_000, 0).UTC()}
		svc, _, sessions := newService(clk, time.Hour)
		_, err := svc.Register(ctx, "gwen@example.com", "supersecret")
		require.NoError(t, err)
		_, token, err := svc.Login(ctx, "gwen@example.com", "supersecret")
		require.NoError(t, err)

		clk.t = clk.t.Add(2 * time.Hour) // past the 1h TTL
		_, err = svc.Authenticate(ctx, token)
		require.ErrorIs(t, err, auth.ErrSessionNotFound)
		require.Empty(t, sessions.byHash, "expired session should be deleted lazily")
	})

	t.Run("missing or unknown token is ErrSessionNotFound", func(t *testing.T) {
		clk := &fakeClock{t: time.Unix(1_700_000_000, 0).UTC()}
		svc, _, _ := newService(clk, time.Hour)
		_, err := svc.Authenticate(ctx, "")
		require.ErrorIs(t, err, auth.ErrSessionNotFound)
		_, err = svc.Authenticate(ctx, "never-issued")
		require.ErrorIs(t, err, auth.ErrSessionNotFound)
	})
}

func TestService_Logout(t *testing.T) {
	ctx := context.Background()
	clk := &fakeClock{t: time.Unix(1_700_000_000, 0).UTC()}
	svc, _, sessions := newService(clk, time.Hour)
	_, err := svc.Register(ctx, "hugo@example.com", "supersecret")
	require.NoError(t, err)
	_, token, err := svc.Login(ctx, "hugo@example.com", "supersecret")
	require.NoError(t, err)

	require.NoError(t, svc.Logout(ctx, token))
	require.Empty(t, sessions.byHash)

	_, err = svc.Authenticate(ctx, token)
	require.ErrorIs(t, err, auth.ErrSessionNotFound)

	// Logging out again (token already gone, or empty) is a no-op.
	require.NoError(t, svc.Logout(ctx, token))
	require.NoError(t, svc.Logout(ctx, ""))
}
