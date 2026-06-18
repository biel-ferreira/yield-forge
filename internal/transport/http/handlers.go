package http

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/biel-ferreira/yield-forge/internal/platform/buildinfo"
)

// readinessCheckTimeout bounds the dependency check behind /readyz so a hung
// database can never hang the probe (SPEC-002 §9).
const readinessCheckTimeout = 2 * time.Second

// Pinger is the minimal readiness dependency: anything that can confirm it is
// reachable within a context deadline. *sql.DB satisfies it, and tests can supply a
// fake — so the handler is unit-testable without a real database (SPEC-002 FR-206).
type Pinger interface {
	PingContext(ctx context.Context) error
}

// apiHandler holds the dependencies the HTTP handlers need: build metadata, the
// readiness dependency, and a logger. Feature handlers will add their services here.
type apiHandler struct {
	build  buildinfo.Info
	ready  Pinger
	logger *slog.Logger
}

// statusResponse is the body for the liveness and not-found responses.
type statusResponse struct {
	Status string `json:"status"`
}

// readinessResponse is the body for /readyz: an overall status plus a per-dependency
// check map (e.g. {"db": "up"}).
type readinessResponse struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
}

// versionResponse is the body for the /version endpoint.
type versionResponse struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	BuiltAt string `json:"built_at"`
}

// healthz is the liveness probe: ok as long as the process is serving.
func (h apiHandler) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, statusResponse{Status: "ok"})
}

// readyz is the readiness probe: 200 only when the database is reachable, 503
// otherwise. Unlike /healthz, it reflects whether the app can actually serve
// requests that need its dependencies (SPEC-002 FR-205).
func (h apiHandler) readyz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), readinessCheckTimeout)
	defer cancel()

	if err := h.ready.PingContext(ctx); err != nil {
		h.logger.Warn("readiness check failed",
			slog.String("check", "db"),
			slog.String("error", err.Error()),
		)
		writeJSON(w, http.StatusServiceUnavailable, readinessResponse{
			Status: "not_ready",
			Checks: map[string]string{"db": "down"},
		})
		return
	}
	writeJSON(w, http.StatusOK, readinessResponse{
		Status: "ready",
		Checks: map[string]string{"db": "up"},
	})
}

// version reports build metadata.
func (h apiHandler) version(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, versionResponse{
		Version: h.build.Version,
		Commit:  h.build.Commit,
		BuiltAt: h.build.BuildTime,
	})
}

// notFound is the JSON catch-all for unmatched routes (no HTML default).
func (h apiHandler) notFound(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusNotFound, statusResponse{Status: "not found"})
}
