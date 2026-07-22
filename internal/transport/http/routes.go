package http

import "net/http"

// route is one registered application endpoint: the HTTP method, the ServeMux pattern
// (Go 1.22+ syntax, e.g. "/holdings/fii/{id}"), and the handler.
type route struct {
	method  string
	pattern string
	handler http.HandlerFunc
}

// routeTable declares every documented application endpoint in one place — the single
// source of truth for the API surface. NewRouter registers each route from it, and the
// OpenAPI drift test (openapi_test.go) asserts every (method, pattern) pair appears in
// api/openapi.yaml and vice-versa. Add or change an endpoint here and the test fails
// until the spec is updated to match (CLAUDE.md: OpenAPI stays in lockstep).
//
// The doc-serving meta-routes (/docs, /openapi.yaml) and the catch-all 404 are
// deliberately NOT in this table — they are transport plumbing, not part of the
// documented API surface, and are registered separately in NewRouter.
func routeTable(api apiHandler, authH authHandler, profileH profileHandler, holdingsH holdingsHandler, dashboardH dashboardHandler, insightsH insightsHandler, rebalancingH rebalancingHandler, healthH healthHandler, projectionsH projectionsHandler, chatH chatHandler, marketH marketHandler) []route {
	return []route{
		{http.MethodGet, "/healthz", api.healthz},
		{http.MethodGet, "/readyz", api.readyz},
		{http.MethodGet, "/version", api.version},
		{http.MethodPost, "/auth/register", authH.register},
		{http.MethodPost, "/auth/login", authH.login},
		{http.MethodPost, "/auth/logout", authH.logout},
		{http.MethodGet, "/auth/me", authH.me},
		{http.MethodGet, "/profile", profileH.getProfile},
		{http.MethodPut, "/profile", profileH.putProfile},
		{http.MethodPost, "/holdings/fii", holdingsH.createFIIHolding},
		{http.MethodGet, "/holdings/fii", holdingsH.listFIIHoldings},
		{http.MethodPut, "/holdings/fii/{id}", holdingsH.updateFIIHolding},
		{http.MethodDelete, "/holdings/fii/{id}", holdingsH.deleteFIIHolding},
		{http.MethodPost, "/holdings/fixed-income", holdingsH.createFixedIncomeHolding},
		{http.MethodGet, "/holdings/fixed-income", holdingsH.listFixedIncomeHoldings},
		{http.MethodPut, "/holdings/fixed-income/{id}", holdingsH.updateFixedIncomeHolding},
		{http.MethodDelete, "/holdings/fixed-income/{id}", holdingsH.deleteFixedIncomeHolding},
		{http.MethodPost, "/holdings/fixed-income/{id}/reconcile", holdingsH.reconcileFixedIncomeHolding},
		{http.MethodGet, "/dashboard", dashboardH.getDashboard},
		{http.MethodGet, "/insights", insightsH.getInsights},
		{http.MethodPost, "/rebalancing", rebalancingH.postRebalancing},
		{http.MethodGet, "/health-score", healthH.getHealthScore},
		{http.MethodGet, "/projections", projectionsH.getProjections},
		{http.MethodPost, "/chat/messages", chatH.postMessage},
		{http.MethodGet, "/chat/threads", chatH.listThreads},
		{http.MethodGet, "/chat/threads/{id}", chatH.getThread},
		{http.MethodDelete, "/chat/threads/{id}", chatH.deleteThread},
		{http.MethodDelete, "/chat/threads", chatH.clearThreads},
		{http.MethodGet, "/market/indicators", marketH.getMarketIndicators},
	}
}
