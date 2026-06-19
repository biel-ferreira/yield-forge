package http

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/biel-ferreira/yield-forge/internal/platform/buildinfo"
)

// Deps are the dependencies NewRouter wires into the HTTP handler.
type Deps struct {
	Logger       *slog.Logger
	Build        buildinfo.Info
	Ready        Pinger      // readiness dependency (the DB pool)
	Auth         AuthService // authentication use cases
	CookieName   string      // session cookie name
	CookieSecure bool        // set the cookie's Secure flag (off in dev)
	SessionTTL   time.Duration
}

// NewRouter builds the application's HTTP handler: the route table plus the
// middleware chain. Auth is deny-by-default — every route requires a valid session
// except the public allowlist in isPublicRoute (SPEC-003 FR-305).
//
// Chain (outermost first): requestID → logRequests → requireAuth → mux. requireAuth
// is inside logRequests so failed-auth responses are still logged and the resolved
// user_id reaches the log line.
func NewRouter(d Deps) http.Handler {
	api := apiHandler{build: d.Build, ready: d.Ready, logger: d.Logger}
	authH := authHandler{
		service:      d.Auth,
		logger:       d.Logger,
		cookieName:   d.CookieName,
		cookieSecure: d.CookieSecure,
		sessionTTL:   d.SessionTTL,
	}

	mux := http.NewServeMux()
	// Public (see isPublicRoute).
	mux.HandleFunc("GET /healthz", api.healthz)
	mux.HandleFunc("GET /readyz", api.readyz)
	mux.HandleFunc("GET /version", api.version)
	mux.HandleFunc("POST /auth/register", authH.register)
	mux.HandleFunc("POST /auth/login", authH.login)
	// Protected (require a valid session).
	mux.HandleFunc("POST /auth/logout", authH.logout)
	mux.HandleFunc("GET /auth/me", authH.me)
	mux.HandleFunc("/", api.notFound) // catch-all → JSON 404 (when authenticated)

	return requestID(logRequests(d.Logger)(requireAuth(d.Auth, d.CookieName)(mux)))
}
