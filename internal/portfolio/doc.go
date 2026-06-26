// Package portfolio is the Portfolio Management feature (SPEC-102): an authenticated
// investor's FII holdings (FR-001) and Fixed Income holdings (FR-002) — the system of
// record for what the user owns. It owns the two holding entities, their value objects
// (Quantity, LiquidityType; the FII Ticker is reused from marketdata, D1), the service,
// and the ports — Repository (persistence) and Reader (the consumer seam the dashboard,
// Fact Builder, and projections read, SPEC-103/104/107).
//
// The core is pure (hexagonal): no SQL, HTTP, or vendor SDK — the Postgres adapter lives
// in the postgres subpackage and the handlers in transport/http. Money is int64 centavos
// and rates integer basis points, never float (BR-1022). Every holding is scoped to a
// user_id from the authenticated context, never request input, and mutations are
// double-scoped by (id, user_id) so one user can never touch another's holding (BR-1021).
package portfolio
