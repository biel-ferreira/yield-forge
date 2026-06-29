package http

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/biel-ferreira/yield-forge/api"
)

// httpMethods is the set of OpenAPI operation keys treated as routes.
var httpMethods = map[string]bool{
	"get": true, "put": true, "post": true, "delete": true,
	"patch": true, "head": true, "options": true,
}

// documentedRoutes parses the `paths:` section of the embedded OpenAPI spec into a set
// of "METHOD /path" keys. The parse is indentation-based and relies on the spec's
// formatting contract (documented at the top of api/openapi.yaml): path keys are
// indented exactly two spaces, operation (method) keys exactly four. This keeps the
// drift guard dependency-free — no YAML library (ADR-0003 stdlib-first).
func documentedRoutes(t *testing.T) map[string]bool {
	t.Helper()
	out := map[string]bool{}
	inPaths := false
	current := ""
	for _, raw := range strings.Split(string(api.OpenAPISpec), "\n") {
		line := strings.TrimRight(raw, "\r")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// A non-indented key starts a new top-level section.
		if !strings.HasPrefix(line, " ") {
			inPaths = strings.TrimRight(line, " ") == "paths:"
			current = ""
			continue
		}
		if !inPaths {
			continue
		}
		// Path key: exactly two-space indent, "/...:".
		if strings.HasPrefix(line, "  /") && !strings.HasPrefix(line, "   ") {
			current = strings.TrimSuffix(strings.TrimSpace(line), ":")
			continue
		}
		// Operation key: exactly four-space indent, a known HTTP method.
		if strings.HasPrefix(line, "    ") && !strings.HasPrefix(line, "     ") {
			method := strings.TrimSuffix(strings.TrimSpace(line), ":")
			if current != "" && httpMethods[method] {
				out[strings.ToUpper(method)+" "+current] = true
			}
		}
	}
	require.NotEmpty(t, out, "parsed no paths from api/openapi.yaml — check the formatting contract")
	return out
}

func registeredRoutes() map[string]bool {
	out := map[string]bool{}
	for _, rt := range routeTable(apiHandler{}, authHandler{}, profileHandler{}, holdingsHandler{}, dashboardHandler{}, insightsHandler{}) {
		out[rt.method+" "+rt.pattern] = true
	}
	return out
}

// TestOpenAPI_DocumentsEveryRoute fails when a router endpoint has no matching entry in
// the OpenAPI spec — the gate that enforces "update the swagger when you add/change an
// endpoint" (CLAUDE.md).
func TestOpenAPI_DocumentsEveryRoute(t *testing.T) {
	documented := documentedRoutes(t)
	for key := range registeredRoutes() {
		require.True(t, documented[key],
			"route %q is registered but not documented in api/openapi.yaml — add it to the spec (CLAUDE.md working agreement)", key)
	}
}

// TestOpenAPI_NoStaleDocumentedRoutes fails when the spec documents an endpoint the
// router no longer serves, so the contract cannot silently drift ahead of the code.
func TestOpenAPI_NoStaleDocumentedRoutes(t *testing.T) {
	registered := registeredRoutes()
	for key := range documentedRoutes(t) {
		require.True(t, registered[key],
			"route %q is documented in api/openapi.yaml but not registered in the router — remove or fix it in the spec", key)
	}
}
