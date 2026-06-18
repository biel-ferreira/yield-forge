# Feature Specification (SPEC)

## 1. Document Information

| Field        | Value                                                  |
| ------------ | ------------------------------------------------------ |
| Feature Name | Persistence Baseline & Migrations                      |
| Feature ID   | SPEC-002 (foundational)                                |
| Related PRD  | [PRD.md](../01-product/PRD.md) — §7 Data, §10 NFR (Security/Reliability), §12 Constraints |
| Related ADRs | [ADR-0002](../04-architecture/adr/ADR-0002-tech-stack-and-layering.md), [ADR-0003](../04-architecture/adr/ADR-0003-zero-cost-and-pluggable-llm.md) |
| Version      | 0.1.0                                                  |
| Status       | Done                                                   |
| Plan         | [PLAN-002](../03-plans/PLAN-002-persistence-baseline-and-migrations.md) |

---

## 2. Overview

### Purpose

Give the application a **real database connection and a repeatable schema-change
process**, without yet modelling any feature. SPEC-002 wires PostgreSQL into the
skeleton from SPEC-001: a pooled connection owned by `platform`, a **migration
tool + `migrations/` convention**, a `/readyz` that actually checks the database,
local Postgres in `docker-compose`, and the **repository seam** every feature spec
will plug its tables into.

After this spec, the app still has **no business logic** — but `go run ./cmd/api`
connects to Postgres, `/readyz` reports the DB's real state, and a developer can
create/apply/roll back migrations with one Task command.

### Business Value

- **Unblocks every feature spec.** SPEC-101 (Profile), SPEC-102 (Portfolio) and the
  rest cannot store anything until this baseline exists. It is the second foundation
  stone after the HTTP skeleton.
- **Safe, reviewable schema evolution.** Versioned, paired up/down migrations make
  every schema change traceable and reversible — the data equivalent of the
  CHANGELOG (FR-008).
- **Honest readiness.** `/readyz` checking the DB means orchestrators (compose,
  later a cloud host) only route traffic when the app can actually serve it.
- **Zero cost (ADR-0003).** Local Postgres in Docker for dev; a managed **free-tier**
  Postgres (Neon / Supabase) for any hosted environment — selected purely by
  `DATABASE_URL`, no code change.

### Scope

**In scope:** DB driver + pooled connection in `platform`; configuration
(`DATABASE_URL` + pool tuning); migration tooling and the `migrations/` convention;
a baseline migration (extensions + shared conventions, **no feature tables**);
`/readyz` DB health check; graceful pool close on shutdown; Postgres service in
`docker-compose`; Task targets for migrations; `.env.example` updates; integration
tests against a real Postgres.

**Out of scope (owned by later specs):**
- Any feature table (`holdings`, `investor_profile`, `quotes`, …) and its repository
  → the owning feature spec (SPEC-101/102/106/…). SPEC-002 only proves the *pattern*.
- A `users` table and per-user row isolation (FR-015) → **SPEC-003** (Auth). SPEC-002
  establishes the conventions (UUID PKs, `user_id` column shape) it will use.
- Tracing/metrics around queries → **SPEC-004** (Observability).
- Caching of insights / market data → **SPEC-005 / SPEC-006**.
- An ORM. We use `database/sql` + hand-written SQL deliberately (see §5 BR-204).

---

## 3. Functional Requirements

> Like SPEC-001, these are **spec-scoped** foundational requirements. SPEC-002
> implements no PRD *feature* FR directly; it enables the §7 Data and the
> reliability/security NFRs the feature specs depend on.

### FR-201 — Pooled Database Connection in `platform`

A single, pooled database handle is constructed at startup from config and injected
(never global), living in a cross-cutting `platform` package — not in any feature.

**Acceptance Criteria**
- [ ] `internal/platform/database` exposes `Connect(ctx, cfg) (*sql.DB, error)` (or
      equivalent pool type) configured with max open/idle connections and lifetimes
      from config.
- [ ] The handle is created once in `cmd/api/main.go` and passed to whatever needs
      it; no package-level/global DB variable.
- [ ] `Connect` verifies connectivity (a `PingContext` with a bounded timeout)
      before returning success, so a bad `DATABASE_URL` fails fast at startup.
- [ ] The pool is **closed on graceful shutdown** (SPEC-001's shutdown path), after
      the HTTP server has drained.

### FR-202 — Database Configuration

Connection and pool settings are environment-driven (12-factor), extending the
existing `Config`.

**Acceptance Criteria**
- [ ] `DATABASE_URL` is read from the environment (e.g.
      `postgres://user:pass@host:5432/db?sslmode=...`). It is a **secret** — no
      default, never committed.
- [ ] Pool knobs have sensible defaults and are overridable:
      `DB_MAX_OPEN_CONNS`, `DB_MAX_IDLE_CONNS`, `DB_CONN_MAX_LIFETIME`,
      `DB_CONN_MAX_IDLE_TIME`, plus a `DB_CONNECT_TIMEOUT`.
- [ ] Whether the DB is **required** is explicit: if `DATABASE_URL` is unset the app
      fails fast with a clear error (DB is mandatory once SPEC-002 lands), **unless**
      a documented escape hatch is chosen in §14-D1.
- [ ] `.env.example` documents every new variable with placeholder values, and the
      `sslmode` expectation differs dev (`disable`) vs hosted (`require`).

### FR-203 — Migration Tooling & `migrations/` Convention

Schema changes are expressed as **versioned, paired up/down SQL migrations** applied
by a single documented tool; the order and naming are fixed so any contributor (or
CI) applies them deterministically.

**Acceptance Criteria**
- [ ] `migrations/` holds files named `NNNN_short_name.up.sql` /
      `NNNN_short_name.down.sql` (zero-padded, monotonically increasing), one concern
      per migration.
- [ ] A single tool (see §14-D2) applies/rolls back migrations and records applied
      versions in a tracking table; partially-applied state is detectable.
- [ ] Task targets exist: `migrate:up`, `migrate:down` (one step),
      `migrate:create -- <name>`, and `migrate:status`/`version`.
- [ ] Every `*.up.sql` has a corresponding `*.down.sql` that cleanly reverses it
      (the down is tested, not just present).
- [ ] Migrations are **idempotent-safe to re-run** at the tool level (re-running
      `migrate:up` on an up-to-date DB is a no-op, not an error).

### FR-204 — Baseline Migration (`0001`)

The first migration establishes shared, feature-neutral foundations only.

**Acceptance Criteria**
- [ ] `0001_init.up.sql` enables required extensions (e.g. `pgcrypto` for
      `gen_random_uuid()`) and sets project-wide conventions — **no feature tables**.
- [ ] `0001_init.down.sql` reverses it.
- [ ] The conventions feature tables must follow are documented (UUID v4 primary
      keys via `gen_random_uuid()`; `created_at`/`updated_at timestamptz default now()`;
      monetary values stored as `numeric`/integer minor units per §5 BR-203; a
      `user_id uuid` column shape reserved for SPEC-003 isolation).

### FR-205 — Readiness Reflects the Database

`/readyz` reports `200` only when the database is reachable, and `503` otherwise —
turning SPEC-001's always-ready stub into a real dependency check.

**Acceptance Criteria**
- [ ] `GET /readyz` runs a short, bounded DB check (`PingContext` or `SELECT 1`)
      per request (or a recent cached result) and returns
      `200 {"status":"ready","checks":{"db":"up"}}` when healthy.
- [ ] When the DB is unreachable it returns
      `503 {"status":"not_ready","checks":{"db":"down"}}` (matches SPEC-001 §7's
      forecast contract).
- [ ] `GET /healthz` (liveness) is **unchanged** — it stays `200` regardless of DB
      state (the process is alive even if the DB is down).
- [ ] The readiness check has its own short timeout so a hung DB can't hang the probe.

### FR-206 — Repository Seam (pattern, not a feature)

SPEC-002 proves the persistence pattern end-to-end without owning a feature, so
SPEC-101/102 inherit a known shape.

**Acceptance Criteria**
- [ ] The pattern is documented: a feature declares a `Repository` **port interface**
      in its core (BR-003 from SPEC-001); a `postgres/` subpackage implements it using
      the injected `*sql.DB`; wiring happens in `cmd/api/main.go`.
- [ ] A minimal, **non-feature** repository demonstrates it against the baseline schema
      (e.g. a `platform/database` health/`SELECT 1` probe, or a tiny throwaway
      `ping`-style query) — enough to exercise the seam in an integration test without
      introducing domain types that belong to a feature spec.
- [ ] No feature domain type (`Holding`, `InvestorProfile`, …) is introduced here.

### FR-207 — Local Postgres in Docker Compose

`docker compose up` brings up the app **and** a Postgres it can talk to, with no
manual DB setup.

**Acceptance Criteria**
- [ ] The `db` service (commented in SPEC-001's compose) is enabled: `postgres:16-alpine`,
      named volume for data, healthcheck via `pg_isready`.
- [ ] The `api` service gets `DATABASE_URL` pointing at `db` and `depends_on` it with
      `condition: service_healthy`.
- [ ] After `docker compose up`, `GET /readyz` returns `200` (DB up) once migrations
      have been applied; the migration step is documented (manual Task target or the
      §14-D3 startup option).

### FR-208 — CHANGELOG & Docs Updated

Per FR-008, the persistence work is recorded.

**Acceptance Criteria**
- [ ] `CHANGELOG.md` `[Unreleased]` gains `Added` entries for the DB connection,
      migrations tooling, `/readyz` DB check, and compose Postgres.
- [ ] `README.md` documents the DB prerequisites, how to run migrations, and the new
      env vars.
- [ ] The specs index (SPEC-002 row) is flipped to ✅ Done at close.

---

## 4. User Flows

> The "user" of SPEC-002 is still the **developer**.

### Flow 1 — First run with a database
1. Developer copies `.env.example` → `.env`, sets `DATABASE_URL` to local Postgres.
2. Starts Postgres (`task docker-up` or a local instance).
3. Runs `task migrate:up` → baseline migration applied; tracking table records `0001`.
4. `go run ./cmd/api` → pool connects, `PingContext` succeeds, server starts.
5. `GET /readyz` → `200 {"status":"ready","checks":{"db":"up"}}`.

### Flow 2 — Create a new migration (future feature author)
1. `task migrate:create -- add_holdings` → generates
   `migrations/0002_add_holdings.up.sql` + `.down.sql` stubs.
2. Author writes the SQL; runs `task migrate:up`, verifies, then `task migrate:down`
   to confirm the down works; re-applies.

### Flow 3 — Database down (error path)
1. Postgres is stopped while the app runs.
2. `GET /readyz` → `503 {"status":"not_ready","checks":{"db":"down"}}`.
3. `GET /healthz` still → `200` (process alive). No crash; the next successful check
   flips readiness back to `200`.

### Flow 4 — Bad `DATABASE_URL` at startup (fail fast)
1. `DATABASE_URL` is malformed or unreachable.
2. `Connect`'s `PingContext` fails within `DB_CONNECT_TIMEOUT`.
3. `main` logs an explanatory error and exits non-zero **before** serving.

---

## 5. Business Rules (Architectural)

- **BR-201 — The DB handle lives in `platform`, not a feature.** The pool is
  cross-cutting infrastructure; features receive it (or a repository built from it)
  via injection, honoring SPEC-001's BR-001 dependency direction.
- **BR-202 — Repositories are adapters behind a feature-owned port.** SQL lives only
  in `*/postgres/` adapter subpackages; a feature's core never imports `database/sql`
  or any driver type (SPEC-001 BR-002/BR-003 extended to persistence).
- **BR-203 — Money is never a float.** Monetary amounts are stored as `numeric` (or
  integer minor units / centavos) — never `float`/`double` — to avoid rounding
  errors in financial math. The exact representation is fixed here and reused by all
  feature tables.
- **BR-204 — SQL-first, no ORM.** We use `database/sql` and hand-written SQL for
  transparency, learning value, and zero hidden cost (ADR-0003). A query builder may
  be reconsidered later but is out of scope.
- **BR-205 — Migrations are forward-only in spirit, reversible in practice.** Applied
  migrations are never edited after merge; a correction is a new migration. Every
  migration ships a tested `down` for local/dev rollback.
- **BR-206 — Secrets only from the environment.** `DATABASE_URL` (with credentials)
  is read from the env, git-ignored in `.env`, placeholdered in `.env.example`
  (SPEC-001 BR-004).
- **BR-207 — UTC + `timestamptz`.** All timestamps are `timestamptz` and reasoned
  about in UTC; presentation-layer localization (America/Sao_Paulo) is a concern of
  later feature/transport code, never the stored value.

---

## 6. Domain Model

No feature domain entities are introduced (same as SPEC-001). SPEC-002 introduces
only the **persistence conventions** that future entities must follow:

| Convention            | Rule                                                        |
| --------------------- | ----------------------------------------------------------- |
| Primary key           | `id uuid primary key default gen_random_uuid()`             |
| Timestamps            | `created_at timestamptz not null default now()`, `updated_at` same |
| Ownership (SPEC-003)  | feature rows carry `user_id uuid not null` (FK added with auth) |
| Money                 | `numeric(18,2)` **or** integer minor units — never float (BR-203) |
| Naming                | snake_case tables (plural) and columns                      |

The only Go type added is the infrastructure pool wrapper in
`internal/platform/database` — not a domain type.

---

## 7. API Specification

Only `/readyz` changes; `/healthz` and `/version` are unchanged from SPEC-001.

### Readiness (now dependency-aware)
```
GET /readyz
200 OK
{ "status": "ready", "checks": { "db": "up" } }

503 Service Unavailable
{ "status": "not_ready", "checks": { "db": "down" } }
```

### Liveness (unchanged)
```
GET /healthz
200 OK
{ "status": "ok" }
```

All responses remain `application/json`; health/readiness stay public (no auth).

---

## 8. Data Storage

### Engine
PostgreSQL 16 (local: `postgres:16-alpine` in compose; hosted: managed free-tier
Neon/Supabase). Access via `database/sql` with a Postgres driver (§14-D2).

### Schema after SPEC-002
| Object                | Origin                | Notes                                  |
| --------------------- | --------------------- | -------------------------------------- |
| `schema_migrations` (or tool equivalent) | migration tool | Tracks applied versions. |
| `pgcrypto` extension  | `0001_init`           | Provides `gen_random_uuid()`.          |
| *(no feature tables)* | —                     | Added by feature specs.                |

### Indexes
None beyond what the migration tool needs. Feature specs add their own indexes
(e.g. `idx_holdings_user_id`) following the conventions in §6.

---

## 9. Edge Cases

| Scenario | Expected behaviour |
| -------- | ------------------ |
| `DATABASE_URL` unset | Fail fast at startup with a clear error (DB required), unless §14-D1 escape hatch chosen. |
| `DATABASE_URL` malformed / DB unreachable at boot | `Connect` ping fails within timeout; `main` logs and exits non-zero before serving. |
| DB goes down while running | `/readyz` → `503`; `/healthz` stays `200`; no crash; recovers when DB returns. |
| Migration applied twice | Tool no-ops (already at that version); not an error (FR-203). |
| Migration fails mid-way | Tool leaves a recorded/"dirty" state; `migrate:status` surfaces it; documented recovery (force version / fix + re-run). |
| Pool exhausted under load | Queries block up to context timeout then error; no goroutine leak; surfaced in logs. |
| Slow/hung DB during `/readyz` | Readiness check's own timeout fires → `503`, probe never hangs. |
| Shutdown with open connections | Pool closed after HTTP drain; in-flight queries finish or are cancelled by context. |

---

## 10. Security Considerations

- `DATABASE_URL` carries credentials — **environment only**, git-ignored, placeholder
  in `.env.example` (BR-206).
- **TLS in transit:** hosted environments use `sslmode=require` (or stricter); only
  local dev uses `sslmode=disable` (PRD §10 "encryption in transit"). Documented in
  `.env.example`.
- **Encryption at rest** is provided by the managed Postgres host (PRD §10); noted as
  a deployment requirement, not app code.
- SQL is parameterized exclusively (`$1,$2,…`) — **no string concatenation** of user
  input — even though SPEC-002 has no user input yet, the pattern is set now.
- The readiness endpoint exposes only `"up"/"down"`, never connection strings, DB
  versions, or error internals.
- Per-user data isolation (FR-015) is **not** implemented here — it is SPEC-003 — but
  the `user_id` convention (§6) reserves its shape.

---

## 11. Observability

- **Logs:** startup logs a redacted DB target (host/db name only, **never** the
  password), the chosen pool sizes, and migration apply/rollback events. A failed
  readiness check logs at `warn` with the reason.
- **Metrics / Traces:** query-level tracing and pool metrics are **SPEC-004**. SPEC-002
  only ensures the pool/handle is injected so SPEC-004 can wrap it without rework.
- **Readiness signal:** `/readyz` is the externally observable DB-health signal until
  SPEC-004 adds richer telemetry.

---

## 12. Testing Strategy

### Unit Tests
- Config: new DB vars parse, defaults applied, `DATABASE_URL` required-missing errors
  (or escape-hatch behavior), pool knobs parse and validate.
- Readiness handler: returns `200`/`503` for a stubbed/faked DB checker (inject a
  small `Pinger` interface so the handler is testable without a real DB).

### Integration Tests (real Postgres, gated)
- Gated by `testing.Short()` **and** presence of a test `DATABASE_URL` (consistent
  with SPEC-001's socket tests): skip cleanly when no DB is available, run in CI/dev
  when one is.
- Apply all migrations up, then all the way down, then up again — asserting a clean
  round-trip (proves every `down` works — BR-205).
- `Connect` + `PingContext` against the real DB; `/readyz` returns `200`.
- Stop/deny the DB → `/readyz` returns `503` (or simulate via a closed pool).
- Decision §14-D4 fixes whether the DB is provided by compose/env or testcontainers.

### End-to-End / Manual
- `task docker-up` → `task migrate:up` → `curl /readyz` → `200`.
- Stop the `db` container → `curl /readyz` → `503`, `curl /healthz` → `200`.

### Quality gate
- `go build ./...`, `go vet ./...`, `gofmt`/`goimports` clean; unit tests pass
  without a DB; integration tests pass with one; dependency-direction rule (BR-201/202)
  holds (feature cores import no `database/sql`).

---

## 13. Definition of Done

- [ ] `internal/platform/database` connect + pool + ping + graceful close implemented.
- [ ] `Config` extended with `DATABASE_URL` + pool knobs; `.env.example` updated;
      required/escape-hatch behavior (§14-D1) implemented and documented.
- [ ] Migration tool wired; `migrations/` convention + Task targets
      (`up`/`down`/`create`/`status`) working.
- [ ] `0001_init` baseline migration (extensions + conventions) with a tested `down`.
- [ ] `/readyz` performs a bounded DB check → `200`/`503`; `/healthz` unchanged; both
      tested.
- [ ] Repository seam documented + a minimal non-feature probe exercised in tests
      (FR-206).
- [ ] `docker-compose` Postgres service enabled with healthcheck + `depends_on`;
      `docker compose up` → `/readyz` `200` after migrations.
- [ ] Unit tests pass with no DB; integration tests pass against a real Postgres
      (up/down/up round-trip green).
- [ ] `go build`, `go vet`, `gofmt` clean; BR-201/202 dependency rule verified.
- [ ] `CHANGELOG.md` + `README.md` updated; specs index row flipped to Done.
- [ ] PLAN-002 followed; PR reviewed and merged.

---

## 14. Decisions (resolved)

> Confirmed with the project owner before PLAN-002. These are now binding.

### D1 — Database is **mandatory** ✅
No escape hatch. If `DATABASE_URL` is unset, the app **fails fast** at startup with a
clear error. Local dev always runs Postgres in Docker (`task docker-up`). The DB is a
foundation from SPEC-002 onward.

### D2 — `database/sql` + `pgx` stdlib driver, migrated by `golang-migrate` ✅
- **Driver:** `database/sql` with the `pgx` stdlib driver (`github.com/jackc/pgx/v5/stdlib`)
  — the portable stdlib interface over the best-maintained Postgres driver. (Native
  `pgxpool` rejected for now: faster but a Postgres-specific API with less transferable
  learning value.)
- **Migration tool:** `golang-migrate` — plain `.up.sql`/`.down.sql` files, CLI **and**
  Go library (`iofs` + `go:embed` to ship migrations in the binary). (`goose` and a
  hand-rolled runner rejected.)

### D3 — Migrations applied **manually** via Task ✅
`task migrate:up` is explicit and deliberate; **never auto-migrate** (no startup
auto-run, in any environment, in SPEC-002). The compose workflow documents the
migrate step. (A `MIGRATE_ON_START` dev convenience may be added in a later spec if
needed — out of scope here.)

### D4 — Integration tests run against a real Postgres, **env-gated** ✅
Gated by `testing.Short()` **and** the presence of a test `DATABASE_URL`, mirroring
SPEC-001's socket tests — skip cleanly with no DB, run in CI/dev when one is provided.
No new test dependency. (`testcontainers-go` rejected: clean but heavy and forces a
Docker requirement onto otherwise-unit-ish runs.)

---

## 15. Open Questions (deferred, not blocking)

- Concrete managed free-tier host (Neon vs Supabase) — decided at deploy time
  (ADR-0003 open item); SPEC-002 only requires "a `DATABASE_URL` that points at one."
- Connection-pool sizing for the free tier's connection caps (Neon pooling / pgBouncer)
  — tuned when a host is chosen; defaults are conservative now.
- Whether to add a query builder (e.g. `sqlc` for type-safe generated code) — revisit
  when the first real repository (SPEC-102) is built; SQL-first for now (BR-204).
