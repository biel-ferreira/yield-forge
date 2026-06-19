package http

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/biel-ferreira/yield-forge/internal/auth"
	authbcrypt "github.com/biel-ferreira/yield-forge/internal/auth/bcrypt"
	authpostgres "github.com/biel-ferreira/yield-forge/internal/auth/postgres"
	"github.com/biel-ferreira/yield-forge/internal/platform/buildinfo"
	"github.com/biel-ferreira/yield-forge/internal/platform/clock"
	"github.com/biel-ferreira/yield-forge/internal/platform/config"
	"github.com/biel-ferreira/yield-forge/internal/platform/database"
)

// authIntegrationEnv wires the REAL auth stack (Postgres repos + bcrypt + system
// clock) behind the router, against TEST_DATABASE_URL. It applies migrations and
// truncates the auth tables so each test starts clean. Gated like the other
// integration tests (skips in -short mode and without TEST_DATABASE_URL).
func authIntegrationEnv(t *testing.T) (http.Handler, *sql.DB) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping auth integration test in -short mode")
	}
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL (a disposable Postgres) to run auth integration tests")
	}

	require.NoError(t, database.MigrateUp(url), "apply migrations")

	cfg := config.Config{
		DatabaseURL:       url,
		DBMaxOpenConns:    5,
		DBMaxIdleConns:    2,
		DBConnMaxLifetime: 5 * time.Minute,
		DBConnMaxIdleTime: 5 * time.Minute,
		DBConnectTimeout:  5 * time.Second,
	}
	db, err := database.Connect(context.Background(), cfg)
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	// Clean slate: TRUNCATE users cascades to sessions.
	_, err = db.ExecContext(context.Background(), "TRUNCATE users CASCADE")
	require.NoError(t, err)

	svc := auth.NewService(
		authpostgres.NewUserRepository(db),
		authpostgres.NewSessionRepository(db),
		authbcrypt.New(),
		clock.System{},
		time.Hour,
	)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := NewRouter(Deps{
		Logger:     logger,
		Build:      buildinfo.Info{},
		Ready:      db,
		Auth:       svc,
		CookieName: "yf_session",
		SessionTTL: time.Hour,
	})
	return router, db
}

func TestAuth_FullFlow_Integration(t *testing.T) {
	router, _ := authIntegrationEnv(t)
	body := `{"email":"flow@example.com","password":"supersecret"}`

	// register
	require.Equal(t, http.StatusCreated, doReq(router, http.MethodPost, "/auth/register", body).Code)

	// login → session cookie
	rr := doReq(router, http.MethodPost, "/auth/login", body)
	require.Equal(t, http.StatusOK, rr.Code)
	cookie := sessionCookie(rr, "yf_session")
	require.NotNil(t, cookie)

	// me with the cookie
	require.Equal(t, http.StatusOK, doReq(router, http.MethodGet, "/auth/me", "", cookie).Code)

	// logout
	require.Equal(t, http.StatusNoContent, doReq(router, http.MethodPost, "/auth/logout", "", cookie).Code)

	// me with the now-revoked cookie
	require.Equal(t, http.StatusUnauthorized, doReq(router, http.MethodGet, "/auth/me", "", cookie).Code)
}

func TestAuth_NoPlaintextStored_Integration(t *testing.T) {
	router, db := authIntegrationEnv(t)
	const email, password = "secret@example.com", "supersecret"

	require.Equal(t, http.StatusCreated,
		doReq(router, http.MethodPost, "/auth/register",
			`{"email":"secret@example.com","password":"supersecret"}`).Code)

	// The stored password hash must not be the plaintext (BR-302).
	var passwordHash string
	require.NoError(t, db.QueryRowContext(context.Background(),
		"SELECT password_hash FROM users WHERE email = $1", email).Scan(&passwordHash))
	require.NotEqual(t, password, passwordHash)
	require.NotEmpty(t, passwordHash)

	// Log in, then the stored session token hash must be sha256(token), never the raw
	// token the client holds (BR-303).
	rr := doReq(router, http.MethodPost, "/auth/login",
		`{"email":"secret@example.com","password":"supersecret"}`)
	require.Equal(t, http.StatusOK, rr.Code)
	rawToken := sessionCookie(rr, "yf_session").Value

	var tokenHash string
	require.NoError(t, db.QueryRowContext(context.Background(),
		"SELECT token_hash FROM sessions LIMIT 1").Scan(&tokenHash))
	require.NotEqual(t, rawToken, tokenHash, "raw token must not be stored")
	require.Equal(t, auth.HashToken(rawToken), tokenHash, "stored value must be sha256(token)")
}

func TestAuth_IsolationSeam_Integration(t *testing.T) {
	router, _ := authIntegrationEnv(t)

	// Two users, each with their own session.
	login := func(email string) *http.Cookie {
		body := `{"email":"` + email + `","password":"supersecret"}`
		require.Equal(t, http.StatusCreated, doReq(router, http.MethodPost, "/auth/register", body).Code)
		rr := doReq(router, http.MethodPost, "/auth/login", body)
		require.Equal(t, http.StatusOK, rr.Code)
		c := sessionCookie(rr, "yf_session")
		require.NotNil(t, c)
		return c
	}
	cookieA := login("alice@example.com")
	cookieB := login("bob@example.com")

	// Each session resolves to its OWN user — never the other's (FR-306 / BR-304).
	meA := doReq(router, http.MethodGet, "/auth/me", "", cookieA)
	require.Equal(t, http.StatusOK, meA.Code)
	require.Contains(t, meA.Body.String(), "alice@example.com")
	require.NotContains(t, meA.Body.String(), "bob@example.com")

	meB := doReq(router, http.MethodGet, "/auth/me", "", cookieB)
	require.Equal(t, http.StatusOK, meB.Code)
	require.Contains(t, meB.Body.String(), "bob@example.com")
	require.NotContains(t, meB.Body.String(), "alice@example.com")
}
