package http

import (
	"net/http"

	"github.com/biel-ferreira/yield-forge/internal/auth"
)

// requireAuth enforces deny-by-default authentication (SPEC-003 FR-305 / BR-301):
// every route requires a valid session except the explicit public allowlist. On
// success it injects the authenticated UserID into the request context (the seam
// feature repositories scope by, FR-306) and records it for the request log.
func requireAuth(authn AuthService, cookieName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isPublicRoute(r.Method, r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			token := sessionTokenFromRequest(r, cookieName)
			user, err := authn.Authenticate(r.Context(), token)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "authentication required")
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
// and version are public for probes; register and login must be public to bootstrap
// a session. Everything else (including unknown routes) requires auth (BR-301).
func isPublicRoute(method, path string) bool {
	switch path {
	case "/healthz", "/readyz", "/version":
		return true
	case "/auth/register", "/auth/login":
		return method == http.MethodPost
	default:
		return false
	}
}
