package http

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/biel-ferreira/yield-forge/internal/auth"
	"github.com/biel-ferreira/yield-forge/internal/platform/buildinfo"
)

// authRouter builds the full router (incl. the deny-by-default middleware) with the
// given fake auth service.
func authRouter(fa fakeAuth) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewRouter(Deps{
		Logger:     logger,
		Build:      buildinfo.Info{},
		Ready:      fakePinger{},
		Auth:       fa,
		CookieName: "yf_session",
		SessionTTL: time.Hour,
	})
}

// doReq issues a request through router, optionally with a JSON body and cookies.
func doReq(router http.Handler, method, path, body string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	var rdr io.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, rdr)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

// sessionCookie extracts the Set-Cookie with the given name from a response.
func sessionCookie(rr *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, c := range rr.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func TestRegisterHandler(t *testing.T) {
	body := `{"email":"new@example.com","password":"supersecret"}`

	t.Run("created", func(t *testing.T) {
		r := authRouter(fakeAuth{registerUser: auth.User{ID: "u1", Email: "new@example.com"}})
		rr := doReq(r, http.MethodPost, "/auth/register", body)
		require.Equal(t, http.StatusCreated, rr.Code)
		require.JSONEq(t, `{"id":"u1","email":"new@example.com"}`, rr.Body.String())
	})

	t.Run("duplicate email is 409", func(t *testing.T) {
		r := authRouter(fakeAuth{registerErr: auth.ErrEmailTaken})
		rr := doReq(r, http.MethodPost, "/auth/register", body)
		require.Equal(t, http.StatusConflict, rr.Code)
	})

	t.Run("invalid email is 400", func(t *testing.T) {
		r := authRouter(fakeAuth{registerErr: auth.ErrInvalidEmail})
		rr := doReq(r, http.MethodPost, "/auth/register", body)
		require.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("weak password is 400", func(t *testing.T) {
		r := authRouter(fakeAuth{registerErr: auth.ErrWeakPassword})
		rr := doReq(r, http.MethodPost, "/auth/register", body)
		require.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("malformed body is 400", func(t *testing.T) {
		r := authRouter(fakeAuth{})
		rr := doReq(r, http.MethodPost, "/auth/register", `{not json`)
		require.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

func TestLoginHandler(t *testing.T) {
	body := `{"email":"u@example.com","password":"supersecret"}`

	t.Run("success sets a hardened session cookie", func(t *testing.T) {
		r := authRouter(fakeAuth{
			loginUser:  auth.User{ID: "u1", Email: "u@example.com"},
			loginToken: "raw-token-123",
		})
		rr := doReq(r, http.MethodPost, "/auth/login", body)
		require.Equal(t, http.StatusOK, rr.Code)

		c := sessionCookie(rr, "yf_session")
		require.NotNil(t, c, "login must set the session cookie")
		require.Equal(t, "raw-token-123", c.Value)
		require.True(t, c.HttpOnly, "cookie must be HttpOnly")
		require.Equal(t, http.SameSiteLaxMode, c.SameSite)
		require.Positive(t, c.MaxAge)
	})

	t.Run("bad credentials is a generic 401", func(t *testing.T) {
		r := authRouter(fakeAuth{loginErr: auth.ErrInvalidCredentials})
		rr := doReq(r, http.MethodPost, "/auth/login", body)
		require.Equal(t, http.StatusUnauthorized, rr.Code)
		require.Contains(t, rr.Body.String(), "invalid email or password")
		require.Nil(t, sessionCookie(rr, "yf_session"), "no cookie on failed login")
	})
}

func TestMeHandler(t *testing.T) {
	cookie := &http.Cookie{Name: "yf_session", Value: "tok"}

	t.Run("authenticated returns identity", func(t *testing.T) {
		user := auth.User{ID: "u1", Email: "me@example.com"}
		r := authRouter(fakeAuth{authUser: user, meUser: user})
		rr := doReq(r, http.MethodGet, "/auth/me", "", cookie)
		require.Equal(t, http.StatusOK, rr.Code)
		require.JSONEq(t, `{"id":"u1","email":"me@example.com"}`, rr.Body.String())
	})

	t.Run("no session is 401", func(t *testing.T) {
		r := authRouter(fakeAuth{authErr: auth.ErrSessionNotFound})
		rr := doReq(r, http.MethodGet, "/auth/me", "")
		require.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}

func TestLogoutHandler(t *testing.T) {
	user := auth.User{ID: "u1", Email: "u@example.com"}
	r := authRouter(fakeAuth{authUser: user})
	cookie := &http.Cookie{Name: "yf_session", Value: "tok"}

	rr := doReq(r, http.MethodPost, "/auth/logout", "", cookie)
	require.Equal(t, http.StatusNoContent, rr.Code)

	cleared := sessionCookie(rr, "yf_session")
	require.NotNil(t, cleared)
	require.Negative(t, cleared.MaxAge, "logout must expire the cookie (MaxAge < 0)")
}

func TestAuthMiddleware_DenyByDefault(t *testing.T) {
	t.Run("public routes bypass auth even with no session", func(t *testing.T) {
		r := authRouter(fakeAuth{authErr: auth.ErrSessionNotFound})
		for _, path := range []string{"/healthz", "/readyz", "/version"} {
			rr := doReq(r, http.MethodGet, path, "")
			require.Equal(t, http.StatusOK, rr.Code, "%s should be public", path)
		}
	})

	t.Run("protected route without a session is 401", func(t *testing.T) {
		r := authRouter(fakeAuth{authErr: auth.ErrSessionNotFound})
		rr := doReq(r, http.MethodPost, "/auth/logout", "")
		require.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("authenticated unknown route reaches the 404 handler", func(t *testing.T) {
		r := authRouter(fakeAuth{authUser: auth.User{ID: "u1"}})
		rr := doReq(r, http.MethodGet, "/nope", "", &http.Cookie{Name: "yf_session", Value: "tok"})
		require.Equal(t, http.StatusNotFound, rr.Code)
	})
}
