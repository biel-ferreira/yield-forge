package http

import (
	"log/slog"
	"net/http"

	"github.com/biel-ferreira/yield-forge/internal/platform/buildinfo"
)

// NewRouter builds the application's HTTP handler: the route table plus the
// middleware chain (request id, then request logging). It owns transport concerns
// only — handlers delegate to feature services (none yet) and to ready for the
// readiness probe's dependency check.
func NewRouter(logger *slog.Logger, build buildinfo.Info, ready Pinger) http.Handler {
	h := apiHandler{build: build, ready: ready, logger: logger}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", h.healthz)
	mux.HandleFunc("GET /readyz", h.readyz)
	mux.HandleFunc("GET /version", h.version)
	mux.HandleFunc("/", h.notFound) // catch-all → JSON 404

	// Middleware applied outermost-first: requestID → logRequests → mux.
	return requestID(logRequests(logger)(mux))
}
