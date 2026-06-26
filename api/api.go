// Package api holds the YieldForge HTTP API contract: the hand-maintained OpenAPI
// 3.1 specification (openapi.yaml), embedded so the running server can serve it at
// GET /openapi.yaml and render it at GET /docs (Swagger UI).
//
// The spec is the source of truth for the API surface. A drift test
// (internal/transport/http/openapi_test.go) fails the build if a route is added,
// removed, or re-pathed without updating the spec — enforcing the CLAUDE.md working
// agreement that the OpenAPI document stays in lockstep with the router.
package api

import _ "embed"

// OpenAPISpec is the embedded OpenAPI 3.1 document (YAML), served verbatim at
// /openapi.yaml and consumed by the drift test.
//
//go:embed openapi.yaml
var OpenAPISpec []byte
