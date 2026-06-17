// Package httpserver bootstraps and runs the HTTP server.
//
// It owns the *http.Server lifecycle: starting it on the configured port and
// shutting it down gracefully on SIGINT/SIGTERM within a bounded timeout. It is
// transport-agnostic about routes — the handler is supplied by the caller.
//
// Implemented in SPEC-001 phase 4.
package httpserver
