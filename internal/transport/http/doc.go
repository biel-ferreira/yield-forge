// Package http is the driving HTTP adapter: the router, handlers, and DTOs that
// translate HTTP/JSON to and from the feature services. It owns transport
// concerns only — no business logic lives here.
//
// SPEC-001 phase 4 adds the router plus the health, readiness, and version
// handlers and request-logging middleware. Feature handlers (e.g. portfolio)
// arrive with their own feature specs.
package http
