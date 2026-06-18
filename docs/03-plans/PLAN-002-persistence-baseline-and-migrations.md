# Implementation Plan

## 1. Document Information

| Field           | Value                                                        |
| --------------- | ------------------------------------------------------------ |
| Plan Name       | Persistence Baseline & Migrations                            |
| Related Feature | Foundational — database connection + schema-change process   |
| Related Spec    | [SPEC-002](../02-specs/SPEC-002-persistence-baseline-and-migrations.md) |
| Version         | 0.1.0                                                        |
| Status          | Done                                                         |
| Author          | Gabigol                                                      |
| Last Updated    | 2026-06-17                                                   |

---

## 2. Objective

### Goal

Wire PostgreSQL into the SPEC-001 skeleton: a pooled `*sql.DB` owned by `platform`,
a `golang-migrate`-driven `migrations/` workflow, a `/readyz` that checks the real
database, local Postgres in `docker-compose`, and the documented repository seam —
with **no feature tables and no business logic**.

### Expected Outcome

With local Postgres running and `0001_init` applied: `go run ./cmd/api` connects
(ping-on-boot), `GET /readyz` returns `200 {"status":"ready","checks":{"db":"up"}}`,
and `GET /healthz` stays liveness-only. `task migrate:up|down|create|status` manage
schema versions. `docker compose up` brings up app + Postgres together. The
persistence seam is ready for SPEC-101/102 to plug feature repositories into.

---

## 3. Scope

### Included

- `internal/platform/database`: `Connect(ctx, cfg)` → pooled `*sql.DB` (pgx stdlib
  driver), ping-on-connect, pool tuning, graceful close.
- `Config` extension: `DATABASE_URL` (required secret) + pool knobs + connect timeout.
- `golang-migrate` wired as a library (`iofs` + `go:embed`) and via Task targets.
- `migrations/0001_init` (extensions + conventions; **no feature tables**) with a
  tested `down`.
- `/readyz` real DB check (bounded) → `200`/`503`; `/healthz` unchanged.
- Repository seam documented + a minimal **non-feature** DB probe to exercise it.
- `docker-compose` Postgres service (healthcheck + `depends_on: service_healthy`).
- `.env.example`, `CHANGELOG.md`, `README.md` updates.
- Unit tests (no DB) + env-gated integration tests (real Postgres).

### Excluded (owned by later specs)

- Any feature table / repository (`holdings`, `investor_profile`, …) → SPEC-101/102/…
- `users` table + per-user row isolation (FR-015) → SPEC-003 (conventions reserved now).
- Query-level tracing / pool metrics → SPEC-004.
- Auto-migrate-on-startup (`MIGRATE_ON_START`) → out of scope (D3 = manual only).
- ORM / query builder (`sqlc`) → deferred (BR-204, §15 open question).

---

## 4. Dependencies

### Technical Dependencies

- **Go** — pinned `1.23.x` (unchanged from SPEC-001).
- **PostgreSQL 16** — `postgres:16-alpine` locally; managed free-tier later (ADR-0003).
- **Docker + Compose** — local DB + parity runs.

### External Dependencies (new Go modules — first runtime deps in the project)

- `github.com/jackc/pgx/v5` (+ `/stdlib` driver) — Postgres driver behind `database/sql`.
- `github.com/golang-migrate/migrate/v4` — migration engine (with `postgres` + `iofs`
  drivers; build-tag the unused sources to keep the dependency surface small).

> These are the **first third-party runtime dependencies**, deliberately accepted:
> Go has no stdlib Postgres driver, and both are best-in-class, free, and
> widely-used (ADR-0003 "stdlib first, add a dep only with clear need").

### Blocking Decisions (resolved — SPEC-002 §14)

- **D1 — DB mandatory:** no `DATABASE_URL` → fail fast; no dev escape hatch.
- **D2 — driver/tool:** `database/sql` + `pgx/v5/stdlib`; migrations via `golang-migrate`.
- **D3 — apply migrations:** manual `task migrate:up` only; never auto on startup.
- **D4 — integration tests:** env-gated real Postgres + `testing.Short()` skip; no
  testcontainers.

---

## 5. Architecture Impact

### Existing Components Affected

| Component | Impact |
| --------- | ------ |
| `internal/platform/config` | Add DB fields + parsing/validation (required `DATABASE_URL`, pool knobs). |
| `internal/transport/http` (handlers/router) | `/readyz` becomes dependency-aware via an injected readiness checker. |
| `cmd/api/main.go` | Build the pool, inject it, pass a readiness checker, close pool on shutdown. |
| `internal/platform/httpserver` | Shutdown sequence closes the pool after HTTP drain. |
| `deploy/docker-compose.yml` | Enable `db` service + `depends_on`; give `api` a `DATABASE_URL`. |
| `.env.example` | Document `DATABASE_URL` + pool/timeout vars + `sslmode` guidance. |
| `CHANGELOG.md` / `README.md` / specs index | Update for the persistence baseline. |

### New Components

| Component | Purpose |
| --------- | ------- |
| `internal/platform/database/database.go` | `Connect(ctx, cfg)` → pooled `*sql.DB`, ping, pool config, `Close`. |
| `internal/platform/database/migrate.go` | `golang-migrate` glue (`iofs` source over embedded `migrations/`). |
| `migrations/0001_init.up.sql` / `.down.sql` | Baseline: `pgcrypto` extension + conventions. |
| `migrations/embed.go` | `//go:embed *.sql` to ship migrations in the binary. |
| Readiness checker (small `Pinger` seam) | Injected into the `/readyz` handler; testable without a real DB. |
| Task targets (`Taskfile.yml`) | `migrate:up`, `migrate:down`, `migrate:create`, `migrate:status`. |

---

## 6. Implementation Strategy

### Approach

Build bottom-up so each phase compiles and is independently testable, mirroring
SPEC-001: config → connection → migrations → readiness → compose → tests → docs.
Keep the dependency rule intact — `database/sql`/driver types appear **only** in
`platform/database` and (later) feature `postgres/` adapters, never in a feature core
(BR-201/202). Introduce **no** feature domain types.

### Rollout Method

**Incremental**, one PR for SPEC-002, reviewed phase-by-phase (same cadence as
SPEC-001: I do a phase, you review, we continue).

### Rollback Strategy

Greenfield, not deployed. Rollback = revert the PR. The only stateful artifact is the
local dev database; `0001` ships a tested `down`, and the dev DB is a disposable
Docker volume (`task docker-down -v`). No production data exists.

---

## 7. Implementation Phases

> Adapted from the template's standard order. SPEC-002 is infrastructure, so the
> "domain/application" phases are replaced by connection/migration/readiness work.

### Phase 1 — Configuration (DB settings)

#### Tasks
- [ ] Extend `Config` with `DatabaseURL string` (required secret) and pool knobs:
      `DBMaxOpenConns`, `DBMaxIdleConns`, `DBConnMaxLifetime`, `DBConnMaxIdleTime`,
      `DBConnectTimeout` (sensible defaults; overridable).
- [ ] `Load()`: `DATABASE_URL` missing/empty → **fatal** error (D1). Pool ints/durations
      parsed via existing `getInt`/`getDuration` helpers; validate ranges.
- [ ] Add a redacted-DSN helper (host/db only, never the password) for safe logging.
- [ ] Update `.env.example`: `DATABASE_URL` placeholder + `sslmode` note (dev `disable`
      / hosted `require`) + each pool var.

#### Deliverables
- Config loads DB settings; missing `DATABASE_URL` fails fast with a clear message;
  unit tests cover required-missing + pool parsing/defaults.

---

### Phase 2 — Database Connection (`platform/database`)

#### Tasks
- [ ] `Connect(ctx, cfg) (*sql.DB, error)`: open with `pgx/v5/stdlib`
      (`sql.Open("pgx", cfg.DatabaseURL)`), apply pool settings, `PingContext` within
      `DBConnectTimeout`; return error (don't panic) on failure.
- [ ] Log a **redacted** target + chosen pool sizes at startup.
- [ ] Wire into `cmd/api/main.go`: build pool after config/logger, before the server;
      on failure, log + exit non-zero (fail fast).
- [ ] Close the pool on graceful shutdown — **after** the HTTP server drains
      (extend the SPEC-001 shutdown path).

#### Deliverables
- App connects to a real Postgres on boot or fails fast; pool closed cleanly on
  shutdown. (Verified manually + in Phase 5 integration tests.)

---

### Phase 3 — Migrations (`golang-migrate` + `0001_init`)

#### Tasks
- [ ] `migrations/embed.go` with `//go:embed *.sql`.
- [ ] `platform/database/migrate.go`: build a `migrate.Migrate` from the embedded
      `iofs` source + the `postgres` database driver; expose `Up`, `Down(steps)`,
      `Version`/`Status` helpers usable by both Task targets and tests.
- [ ] `0001_init.up.sql`: `CREATE EXTENSION IF NOT EXISTS pgcrypto;` + a documented
      conventions header comment (UUID PKs, `timestamptz`, money-not-float). **No
      feature tables.**
- [ ] `0001_init.down.sql`: reverse (`DROP EXTENSION IF EXISTS pgcrypto;`).
- [ ] A tiny `cmd/migrate` runner **or** `go run` invocation behind Task targets:
      `migrate:up`, `migrate:down` (one step), `migrate:create -- <name>`,
      `migrate:status`. Document `create` naming (`NNNN_name.up/down.sql`).

#### Deliverables
- `task migrate:up` applies `0001`; `task migrate:status` shows the version;
  `task migrate:down` reverses it; re-running `up` is a no-op (FR-203).

---

### Phase 4 — Readiness Reflects the Database

#### Tasks
- [ ] Define a minimal readiness seam (e.g. `type Pinger interface { PingContext(ctx) error }`,
      satisfied by `*sql.DB`) so the handler is unit-testable with a fake.
- [ ] Rewrite the `/readyz` handler: bounded-timeout DB check →
      `200 {"status":"ready","checks":{"db":"up"}}` or
      `503 {"status":"not_ready","checks":{"db":"down"}}`.
- [ ] Keep `/healthz` unchanged (always `200`). Log failed readiness at `warn`
      (reason only, no DSN).
- [ ] Inject the checker through the router (no global); wire in `main.go`.

#### Deliverables
- `/readyz` returns `200` with DB up, `503` with DB down; `/healthz` unaffected;
  handler unit-tested with a fake pinger.

---

### Phase 5 — Compose & Containerised DB

#### Tasks
- [ ] Enable the `db` service in `docker-compose.yml` (`postgres:16-alpine`, named
      volume, `pg_isready` healthcheck) — uncomment/realise the SPEC-001 shape.
- [ ] Give `api` a `DATABASE_URL` pointing at `db` + `depends_on: { db: { condition:
      service_healthy } }`.
- [ ] Document the run order: `task docker-up` → `task migrate:up` → `curl /readyz`.

#### Deliverables
- `docker compose up` starts app + Postgres; after `migrate:up`, `/readyz` → `200`;
  stopping `db` → `/readyz` → `503` while `/healthz` stays `200`.

---

### Phase 6 — Testing

#### Unit Tests (no DB)
- [ ] Config: `DATABASE_URL` required-missing error; pool knobs defaults + overrides
      + invalid values; redacted-DSN helper hides the password.
- [ ] Readiness handler: `200` when fake pinger succeeds, `503` when it errors,
      respects the timeout.

#### Integration Tests (real Postgres, gated by `testing.Short()` + test `DATABASE_URL`)
- [ ] `Connect` + `PingContext` against a real DB succeeds; bad DSN fails fast.
- [ ] **Migration round-trip:** up → down → up, asserting clean state each way
      (proves every `down` works — BR-205).
- [ ] `/readyz` → `200` against the live DB; closed pool / dead DB → `503`.

#### End-to-End / Manual
- [ ] `task docker-up` → `task migrate:up` → `curl /readyz` `200`; stop `db` → `503`.

#### Deliverables
- `go test ./...` green with **and** without a DB (gated tests skip cleanly when
  `DATABASE_URL`/`-short` dictate); `go vet` + `gofmt`/`goimports` clean.

---

### Phase 7 — Documentation

#### Tasks
- [ ] `CHANGELOG.md` `[Unreleased] → Added`: DB pool, migrations tooling, `/readyz`
      DB check, compose Postgres, new env vars.
- [ ] `README.md`: DB prerequisites, how to run migrations, env-var table, the
      `docker-up → migrate:up → readyz` flow.
- [ ] Flip SPEC-002 status to Done in the specs index; PLAN-002 → Done.

#### Deliverables
- Docs current; SPEC-002 closed.

> After Phase 7 closes, per the standing preference, produce the PT-BR HTML lesson
> `docs/lessons/SPEC-002-aula.html` recapping the persistence work.

---

## 8. Risks

| Risk | Impact | Mitigation |
| ---- | ------ | ---------- |
| First third-party deps enlarge the module / supply-chain surface | Medium | Limit to pgx + golang-migrate; build-tag unused migrate drivers; `go mod tidy`; pin versions. |
| Integration tests flaky/hang against a slow DB | Medium | Bounded contexts everywhere; `testing.Short()` + env gate; healthcheck before app starts (compose). |
| `database/sql` types leaking into a feature core later | Medium | BR-201/202 in review; SQL only in `platform/database` now; revisit an import-lint rule. |
| Migration left "dirty" after a failed apply | Low | `migrate:status` surfaces it; document force-version recovery; tested `down`. |
| Password leaking into logs | Medium | Redacted-DSN helper is the **only** way the DSN is logged; unit-test the redaction. |
| Free-tier Postgres connection caps exceeded by pool | Low | Conservative default pool sizes; revisit when a host is chosen (§15 open question). |
| `sslmode` misconfig (disabled in prod) | Medium | `.env.example` documents dev `disable` vs hosted `require`; call out in README. |

---

## 9. Validation Checklist

### Functional Validation
- [ ] FR-201…FR-208 acceptance criteria satisfied.
- [ ] `/readyz` `200`/`503` tracks DB state; `/healthz` unchanged.
- [ ] `migrate:up/down/create/status` all work; `0001` up+down round-trips.

### Technical Validation
- [ ] DB handle lives in `platform`, injected (no global); pool closed on shutdown.
- [ ] Feature packages import no `database/sql`/driver types (BR-201/202).
- [ ] Money/time conventions documented; secrets only from env; `.env` git-ignored.

### Quality Validation
- [ ] Unit tests pass with no DB; integration tests pass with one.
- [ ] `go build`, `go vet`, `gofmt`/`goimports`, `golangci-lint` clean; `go mod tidy`.
- [ ] Code reviewed; CHANGELOG updated in the same PR.

---

## 10. Definition of Done

- [ ] All phases complete; SPEC-002 acceptance criteria met.
- [ ] `go run ./cmd/api` connects (or fails fast); `/readyz` reflects DB; `/healthz` ok.
- [ ] `docker compose up` + `task migrate:up` → `/readyz` `200`.
- [ ] Migration up/down/up round-trip green; tests + lint/vet/fmt clean.
- [ ] `CHANGELOG.md` + `README.md` updated; SPEC-002 marked Done in the index.
- [ ] PR reviewed and merged to `main`.
- [ ] PT-BR HTML lesson `docs/lessons/SPEC-002-aula.html` produced.

---

## 11. Deliverables

### Code Deliverables
- `internal/platform/database/{database.go,migrate.go}`, readiness seam + updated
  `/readyz` handler, `Config` DB fields, `cmd/api/main.go` wiring + pool close,
  optional `cmd/migrate` runner.

### Infrastructure Deliverables
- `migrations/0001_init.{up,down}.sql` + `embed.go`, `docker-compose.yml` Postgres
  service, `Taskfile.yml` migrate targets, `.env.example` updates.

### Documentation Deliverables
- Updated `CHANGELOG.md`, `README.md`, specs index; PT-BR lesson HTML.

---

## 12. Post-Implementation Tasks

### Monitoring
- None yet (pool metrics + query tracing are SPEC-004). Confirm `/readyz` is suitable
  as the orchestrator readiness probe.

### Future Improvements
- Optional `MIGRATE_ON_START` dev convenience (deferred from D3).
- Evaluate `sqlc` for type-safe queries when the first feature repository lands (SPEC-102).
- Import-direction lint rule to enforce BR-201/202 automatically.

### Technical Debt
- Default pool sizes are placeholders until a hosted free-tier Postgres is chosen.
