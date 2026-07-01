package http

import (
	"log/slog"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/biel-ferreira/yield-forge/internal/platform/buildinfo"
)

// Deps are the dependencies NewRouter wires into the HTTP handler.
type Deps struct {
	Logger       *slog.Logger
	Build        buildinfo.Info
	Ready        Pinger            // readiness dependency (the DB pool)
	Auth         AuthService       // authentication use cases
	Profile      ProfileService    // investor profile use cases (SPEC-101)
	Portfolio    PortfolioService  // portfolio holdings use cases (SPEC-102)
	Dashboard    DashboardService  // computed dashboard (SPEC-103)
	Insights     InsightsEngine    // AI insight engine (SPEC-104)
	Rebalancing  RebalancingEngine // AI rebalancing assistant (SPEC-105)
	HealthScore  HealthScorer      // portfolio health score (SPEC-106)
	Projections  ProjectionEngine  // income + net-worth projections (SPEC-107)
	Chat         ChatService       // conversational copilot (SPEC-108)
	CookieName   string            // session cookie name
	CookieSecure bool              // set the cookie's Secure flag (off in dev)
	SessionTTL   time.Duration
}

// NewRouter builds the application's HTTP handler: the route table plus the
// middleware chain. Auth is deny-by-default — every route requires a valid session
// except the public allowlist in isPublicRoute (SPEC-003 FR-305).
//
// Chain (outermost first): otelhttp → requestID → logRequests → requireAuth →
// routeNamer → mux. otelhttp is outermost so the server span exists in context for
// the inner middleware (log correlation reads it); routeNamer (innermost) renames the
// span to the matched route. requireAuth stays inside logRequests so failed-auth
// responses are still logged and the resolved user_id reaches the log line
// (SPEC-004 FR-403/FR-405).
func NewRouter(d Deps) http.Handler {
	api := apiHandler{build: d.Build, ready: d.Ready, logger: d.Logger}
	authH := authHandler{
		service:      d.Auth,
		logger:       d.Logger,
		cookieName:   d.CookieName,
		cookieSecure: d.CookieSecure,
		sessionTTL:   d.SessionTTL,
	}
	profileH := profileHandler{service: d.Profile, logger: d.Logger}
	holdingsH := holdingsHandler{service: d.Portfolio, logger: d.Logger}
	dashboardH := dashboardHandler{service: d.Dashboard, logger: d.Logger}
	insightsH := insightsHandler{service: d.Insights, logger: d.Logger}
	rebalancingH := rebalancingHandler{service: d.Rebalancing, logger: d.Logger}
	healthH := healthHandler{service: d.HealthScore, logger: d.Logger}
	projectionsH := projectionsHandler{service: d.Projections, logger: d.Logger}
	chatH := chatHandler{service: d.Chat, logger: d.Logger}

	mux := http.NewServeMux()
	// The application API surface comes from one declared table (routes.go) so the
	// OpenAPI spec can be drift-tested against it (openapi_test.go). Public vs protected
	// is decided by isPublicRoute, not by registration order.
	for _, rt := range routeTable(api, authH, profileH, holdingsH, dashboardH, insightsH, rebalancingH, healthH, projectionsH, chatH) {
		mux.HandleFunc(rt.method+" "+rt.pattern, rt.handler)
	}
	// API documentation meta-routes (public): the embedded OpenAPI spec + Swagger UI.
	// Not part of routeTable — they document the API, they are not part of it.
	mux.HandleFunc("GET /openapi.yaml", serveOpenAPISpec)
	mux.HandleFunc("GET /docs", serveSwaggerUI)
	mux.HandleFunc("/", api.notFound) // catch-all → JSON 404 (when authenticated)

	var handler http.Handler = routeNamer(mux)
	handler = requireAuth(d.Auth, d.CookieName, d.Logger)(handler)
	handler = logRequests(d.Logger)(handler)
	handler = requestID(handler)
	// Outermost: create the server span + HTTP request metrics. Probes are filtered
	// out of tracing to keep the signal low-noise (SPEC-004 FR-403). Metrics come for
	// free from otelhttp via the global MeterProvider (a no-op when telemetry is off).
	return otelhttp.NewHandler(handler, "http.server", otelhttp.WithFilter(traceableRequest))
}

// routeNamer renames the server span to the matched route pattern (e.g. "GET
// /auth/me") and tags it with http.route, keeping span names low-cardinality
// (SPEC-004 FR-403 / BR-406). It resolves the route via the mux; on a no-op span
// (telemetry disabled or a filtered probe) SetName/SetAttributes are harmless no-ops.
func routeNamer(mux *http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, pattern := mux.Handler(r); pattern != "" {
			span := trace.SpanFromContext(r.Context())
			span.SetName(pattern)
			span.SetAttributes(attribute.String("http.route", pattern))
		}
		mux.ServeHTTP(w, r)
	})
}

// traceableRequest keeps liveness/readiness probes out of traces to reduce noise
// (SPEC-004 FR-403).
func traceableRequest(r *http.Request) bool {
	switch r.URL.Path {
	case "/healthz", "/readyz":
		return false
	default:
		return true
	}
}
