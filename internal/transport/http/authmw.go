package http

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/biel-ferreira/yield-forge/internal/auth"
)

// requireAuth enforces deny-by-default authentication (SPEC-003 FR-305 / BR-301):
// every route requires a valid session except the explicit public allowlist. On
// success it injects the authenticated UserID into the request context (the seam
// feature repositories scope by, FR-306) and records it for the request log.
func requireAuth(authn AuthService, cookieName string, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isPublicRoute(r.Method, r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			token := sessionTokenFromRequest(r, cookieName)
			user, err := authn.Authenticate(r.Context(), token)
			if err != nil {
				// A missing/expired/unknown session is a normal 401. Any other error
				// is an infrastructure failure (e.g. the DB is down) — surface it as a
				// 500 and log it, rather than masking it as "not authenticated".
				if errors.Is(err, auth.ErrSessionNotFound) {
					writeError(w, http.StatusUnauthorized, "authentication required")
					return
				}
				logger.ErrorContext(r.Context(), "authentication check failed", slog.String("error", err.Error()))
				writeError(w, http.StatusInternalServerError, "internal error")
				return
			}

			if lf := logFieldsFromContext(r.Context()); lf != nil {
				lf.userID = user.ID
			}
			next.ServeHTTP(w, r.WithContext(auth.WithUserID(r.Context(), user.ID)))
		})
	}
}

// isPublicRoute is the allowlist of routes reachable without authentication. Health
// and version are public for probes; the API docs (spec + Swagger UI) expose only the
// schema, never data, so they are public too; register and login must be public to
// bootstrap a session. Everything else (including unknown routes) requires auth (BR-301).
func isPublicRoute(method, path string) bool {
	switch path {
	case "/healthz", "/readyz", "/version", "/docs", "/openapi.yaml":
		return true
	case "/auth/register", "/auth/login":
		return method == http.MethodPost
	default:
		return false
	}
}
