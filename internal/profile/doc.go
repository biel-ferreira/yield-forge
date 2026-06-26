// Package profile is the Investor Profile feature (SPEC-101): an authenticated user's
// risk profile, objectives, and investment horizon (FR-003). It owns the Profile domain,
// its value objects (RiskProfile, Objective, Horizon), the service, and the ports —
// ProfileRepository (persistence) and ProfileReader (the consumer port that the Insight
// Engine, Rebalancing, and Health Score read, SPEC-104/105/106).
//
// The core is pure (hexagonal): no SQL, HTTP, or vendor SDK — the Postgres adapter lives in
// the postgres subpackage and the handlers in transport/http. Every profile is scoped to a
// user_id that comes from the authenticated context, never from request input (BR-1012).
package profile
