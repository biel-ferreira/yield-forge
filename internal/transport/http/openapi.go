package http

import (
	"fmt"
	"net/http"

	"github.com/biel-ferreira/yield-forge/api"
)

// swaggerUIVersion pins the Swagger UI dist build served from the CDN. Bump it
// deliberately (it is the only external asset version in the project).
const swaggerUIVersion = "5.17.14"

// swaggerUITemplate renders the embedded OpenAPI spec with Swagger UI loaded from a
// pinned CDN build — no Go dependency and no vendored multi-megabyte asset bundle
// (ADR-0003 zero-cost / stdlib-first). The spec itself is served locally at
// /openapi.yaml, so the API contract never leaves the deployment; only the rendering
// UI is fetched from the CDN.
const swaggerUITemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>YieldForge API — Swagger UI</title>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@%[1]s/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@%[1]s/swagger-ui-bundle.js" crossorigin></script>
  <script>
    window.ui = SwaggerUIBundle({
      url: "/openapi.yaml",
      dom_id: "#swagger-ui",
      deepLinking: true,
    });
  </script>
</body>
</html>
`

// swaggerUIPage is the rendered HTML, built once at init.
var swaggerUIPage = []byte(fmt.Sprintf(swaggerUITemplate, swaggerUIVersion))

// serveOpenAPISpec serves the embedded OpenAPI 3.1 document. It is a public meta-route
// (it exposes only the schema, never data) so tooling can fetch the contract without a
// session.
func serveOpenAPISpec(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(api.OpenAPISpec)
}

// serveSwaggerUI serves the Swagger UI host page that renders /openapi.yaml.
func serveSwaggerUI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(swaggerUIPage)
}
