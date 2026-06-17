package http

import (
	"net/http"

	"github.com/biel-ferreira/yield-forge/internal/platform/buildinfo"
)

// apiHandler holds the dependencies the HTTP handlers need. For SPEC-001 that is
// only build metadata; feature handlers will add their services here.
type apiHandler struct {
	build buildinfo.Info
}

// statusResponse is the body for the health, readiness, and not-found responses.
type statusResponse struct {
	Status string `json:"status"`
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

// readyz is the readiness probe. In SPEC-001 the app is always ready; SPEC-002
// extends this to check dependencies (e.g. the database) and return 503 when down.
func (h apiHandler) readyz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, statusResponse{Status: "ready"})
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
