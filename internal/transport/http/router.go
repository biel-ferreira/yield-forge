package http

import (
	"log/slog"
	"net/http"

	"github.com/biel-ferreira/yield-forge/internal/platform/buildinfo"
)

// NewRouter builds the application's HTTP handler: the route table plus the
// middleware chain (request id, then request logging). It owns transport concerns
// only — handlers delegate to feature services (none yet in SPEC-001).
func NewRouter(logger *slog.Logger, build buildinfo.Info) http.Handler {
	h := apiHandler{build: build}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", h.healthz)
	mux.HandleFunc("GET /readyz", h.readyz)
	mux.HandleFunc("GET /version", h.version)
	mux.HandleFunc("/", h.notFound) // catch-all → JSON 404

	// Middleware applied outermost-first: requestID → logRequests → mux.
	return requestID(logRequests(logger)(mux))
}
