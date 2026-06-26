# Changelog

All notable changes to **YieldForge** are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

> **Convention:** update the `[Unreleased]` section in the **same pull request** as
> the change. On release, rename `[Unreleased]` to the new version + date and start
> a fresh `[Unreleased]` on top. Entry types: `Added`, `Changed`, `Deprecated`,
> `Removed`, `Fixed`, `Security`.

## [Unreleased]

### Added

- **OpenAPI 3.1 API documentation** — a hand-maintained spec at [`api/openapi.yaml`](api/openapi.yaml),
  embedded into the binary and served as interactive **Swagger UI at `GET /docs`** plus the
  raw contract at `GET /openapi.yaml` (both public; Swagger UI from a pinned CDN, so **no new
  Go dependency** and no vendored asset bundle — ADR-0003). All 17 endpoints documented with
  request/response schemas, status codes, the cookie security scheme, and the centavos/bps
  money convention.
- **OpenAPI drift guard** — the HTTP route surface is now declared once in a `routeTable`
  (`internal/transport/http/routes.go`), and a dependency-free unit test (`openapi_test.go`)
  fails the build if a registered route is undocumented **or** a documented route no longer
  exists, keeping the spec in lockstep with the router.
- Code conventions / SDD working agreement (`CLAUDE.md`): a binding rule to update
  `api/openapi.yaml` in the same change as any endpoint addition/change, and to refresh it on
  spec closeout.
- Harness: `block-layering` PostToolUse hook — deterministic gate enforcing the core
  architecture rule (a feature core package must not import SQL/HTTP/vendor SDKs),
  promoting it from subjective `hexagonal-reviewer` checks to a hard `exit 2` block.
- Code conventions (`CLAUDE.md`): closed-enum idiom (typed `string` + `ParseX`),
  money across the JSON boundary as integer centavos (never float), concurrency
  (owned + `ctx`-cancellable goroutines, `errgroup`), structured `log/slog`, and
  avoid package-name stutter.
- `docs/TECH-DEBT.md` backlog, with TD-001 (rename stutter in core types).
- Spec-Driven Development (SDD) workspace under `docs/` — README/process guide,
  product, specs, plans, and architecture folders.
- Product Requirements Document (PRD) for YieldForge — Investment Copilot: vision,
  scope, personas, user stories, functional requirements (FR-001…FR-018),
  non-functional requirements, success metrics, and phased release strategy.
- Passive-income projection and net-worth projection features (FR-016, FR-017).
- Zero-cost constraint (G12) and pluggable free/local LLM strategy (FR-018).
- Architecture overview — C4 context & containers, hexagonal + package-oriented
  layering, the explainable AI insight pipeline, and multi-agent / MCP readiness.
- ADR-0001 (record architecture decisions), ADR-0002 (tech stack & backend
  layering), ADR-0003 (zero-cost infrastructure & pluggable LLM provider).
- ADR-0004 (frontend repository strategy — mono-repo: Next.js under `web/`,
  path-scoped CI, OpenAPI contract; *Proposed*).
- Two-tier SPEC/PLAN structure — foundational (`0xx`) and feature (`1xx`).
- SPEC-001 — Project Scaffolding & Hexagonal Layering, with a package-oriented
  (by-feature) hybrid layout; FR-008 requires this changelog.
- PLAN-001 — implementation plan for SPEC-001 (phases, risks, DoD).
- SPEC-002 — Persistence Baseline & Migrations, and PLAN-002 (resolved decisions:
  DB mandatory, `database/sql` + pgx, `golang-migrate`, manual migrations).
- SPEC-003 — Authentication & Per-User Isolation, and PLAN-003 (resolved decisions:
  email+password + server-side sessions, bcrypt, HttpOnly cookie, app-level scoping).
- SPEC-004 — Observability Baseline (OpenTelemetry), and PLAN-004 (resolved decisions:
  traces + metrics + log-correlation, OTLP/HTTP exporter disabled-by-default, otelhttp,
  slog trace-correlation, otelsql).
- Repository setup: `.gitignore` and `.gitattributes` (LF line-ending normalisation).
- This `CHANGELOG.md` for change traceability.

#### SPEC-001 implementation (running Go skeleton)

- Go module `github.com/biel-ferreira/yield-forge` and the package-oriented
  hexagonal layout (`cmd/api`, `internal/{platform,transport,portfolio,profile,
  marketdata,insight,projection}`).
- Environment-driven configuration (`config.Load`): typed `Config` with defaults,
  validation, non-fatal warnings (invalid `LOG_LEVEL`/`LOG_FORMAT` fall back), and
  optional `.env` seeding; documented in `.env.example`.
- Structured logging baseline (`log/slog`) — JSON or human-readable text by
  environment.
- HTTP API: `GET /healthz`, `/readyz`, `/version`; request-id and request-logging
  middleware; JSON 404; graceful shutdown on SIGINT/SIGTERM.
- Multi-stage `Dockerfile` (static binary on distroless, non-root) and
  `docker-compose.yml` (Postgres service shape staged for SPEC-002).
- Unit and integration test suite (config, logging, HTTP handlers, server
  graceful-shutdown drain) using stdlib `testing` + `httptest`.
- Root `README.md` quickstart.
- `Taskfile.yml` — cross-platform task runner (`task run|build|test|lint|docker-up`),
  alongside the `Makefile`.

#### SPEC-002 implementation (persistence baseline)

- PostgreSQL connection pool in `internal/platform/database` (`database/sql` with the
  pgx stdlib driver): ping-on-connect (fail fast), configurable pool sizing, graceful
  close after the HTTP server drains.
- Database configuration: required `DATABASE_URL` secret plus pool knobs
  (`DB_MAX_OPEN_CONNS`, `DB_MAX_IDLE_CONNS`, `DB_CONN_MAX_LIFETIME`,
  `DB_CONN_MAX_IDLE_TIME`, `DB_CONNECT_TIMEOUT`); redacted-DSN logging that never
  prints the password; `.env.example` updated.
- Schema migrations via `golang-migrate`, embedded with `go:embed`. Baseline
  `migrations/0001_init` enables `pgcrypto` and documents the table conventions
  (UUID PKs, `timestamptz`/UTC, money-never-float, `user_id` shape). `cmd/migrate`
  runner + `task migrate:up|down|status|create`.
- `/readyz` now performs a bounded database health check —
  `200 {"checks":{"db":"up"}}` / `503 {"checks":{"db":"down"}}`; `/healthz` unchanged.
- `docker-compose` Postgres service (`postgres:16-alpine`, `pg_isready` healthcheck,
  `depends_on: service_healthy`, host port **5433**); `api` wired with `DATABASE_URL`.
- Env-gated integration tests (real Postgres via `TEST_DATABASE_URL`): connect,
  unreachable-fails-fast, migration up→down→up round-trip, and live `/readyz`.

#### SPEC-003 implementation (authentication & per-user isolation)

- `internal/auth` feature package: `User`/`Session` domain, the `UserRepository`,
  `SessionRepository`, and `PasswordHasher` ports, the auth `Service`
  (register/login/logout/authenticate), session-token generation, and the
  `auth.UserID(ctx)` per-user-isolation seam.
- bcrypt password hashing (`internal/auth/bcrypt`) behind the `PasswordHasher` port;
  passwords stored only as hashes. Session tokens from `crypto/rand`; only
  `sha256(token)` persisted (a DB leak yields no usable sessions).
- Migration `0002_auth` (users + sessions, FK `ON DELETE CASCADE`) with a tested down;
  Postgres repository adapters (`internal/auth/postgres`).
- Auth endpoints — `POST /auth/register`, `POST /auth/login`, `POST /auth/logout`,
  `GET /auth/me` — with generic auth errors (anti-enumeration) and a hardened session
  cookie (`HttpOnly` + `SameSite=Lax`, `Secure` outside dev).
- Deny-by-default auth middleware: every route requires a valid session except the
  public allowlist (`/healthz`, `/readyz`, `/version`, `/auth/register`,
  `/auth/login`); resolves and injects the authenticated `user_id` into the context
  and the request log.
- `Clock` port (`internal/platform/clock`) for deterministic session expiry;
  configuration `SESSION_TTL` + `AUTH_COOKIE_NAME`; `.env.example` updated.
- Unit tests (hand-written fakes) for the service, handlers, and middleware; env-gated
  integration tests over real Postgres: full register→login→me→logout flow,
  no-plaintext-stored assertion, and the per-user isolation seam.

#### SPEC-004 implementation (observability baseline)

- `internal/platform/observability`: `Setup` builds the OpenTelemetry Tracer/Meter
  providers, a service `Resource`, the W3C propagator, and a configurable exporter —
  **disabled by default** (no endpoint ⇒ no-op, the app runs identically at zero cost).
  OTLP/HTTP + stdout exporters; graceful flush-on-shutdown wired into `cmd/api`
  (flushes last, after the HTTP server drains and the DB pool closes).
- HTTP instrumentation via `otelhttp` — one server span per request named by the
  **matched route** (`GET /auth/me`, low-cardinality) + request duration/count metrics;
  liveness/readiness probes filtered out of traces.
- Database instrumentation via `otelsql` at the pool — child query spans under the
  request span; records the parameterised statement only, never argument values
  (no PII/secrets); repositories untouched.
- Log↔trace correlation: a `slog` handler adds `trace_id`/`span_id` to context-aware
  records (alongside `request_id`); auth handlers/middleware log via `*Context`.
- The `observability.Tracer()` / `Meter()` seam + documented conventions for feature
  instrumentation (SPEC-005 AI spans/cost, SPEC-006 ingestion metrics).
- Configuration `OTEL_SERVICE_NAME`, `OTEL_EXPORTER_KIND`, `OTEL_EXPORTER_OTLP_ENDPOINT`,
  `OTEL_EXPORTER_OTLP_HEADERS` (secret), `OTEL_TRACE_SAMPLE_RATIO`; `.env.example` updated.
- Unit tests (in-memory exporter): no-op-safe `Setup`, route-named span, probe
  filtering, incoming-trace continuation, log correlation; gated integration test
  proving a request produces a parent HTTP span with a child DB span.

#### SPEC-005 implementation (Insighter port & free/local LLM adapter)

- `internal/insight` core (pure — no HTTP/SDK/OTel): the `Insighter` and `Cache` ports,
  the `Facts`/`Insight`/`InsightRequest`/`InsightResult` domain, sentinel errors, and a
  provider-neutral prompt layer (`BuildPrompt`/`ParseResult`/`GenerateWithReask`) with an
  English instruction prompt that emits structured JSON in pt-BR.
- **The binding-guard gate** (`Gated` decorator) — enforced for every provider via the
  factory: rejects any insight missing an **explanation** (FR-013) and any output that
  reads as a **transaction order** (FR-014, bilingual order/price/guaranteed-return
  detector), fails closed, and attaches the non-advice disclaimer only on the pass path.
- Two provider adapters behind the port, zero new dependencies (stdlib `net/http`):
  `ollama` (local/dev, JSON mode — facts stay on-device) and `groq` (hosted,
  OpenAI-compatible, Bearer key). Both share one re-ask-once-then-degrade policy: a
  malformed reply is re-asked once; any other failure (incl. a 429) degrades to
  `ErrInsightsUnavailable` with **no charge-incurring retry**.
- `internal/insight/factory` composition root — `observed(cached(gated(provider)))`:
  an in-memory **LRU+TTL cache** (keyed by `sha256(user · task · facts)`, user-scoped,
  stores only gated results, never caches errors; TTL via the `Clock` port), and an AI
  **observability** decorator (`insight.generate` span + generation counter by outcome,
  recording provider/model/outcome/cache_hit/cost only — **never prompt, facts, or
  generated text**). A deterministic `Fake` provider for CI and the AI-off mode.
- Configuration `INSIGHTER_PROVIDER` (ollama|groq|fake), `INSIGHTER_OLLAMA_BASE_URL`,
  `INSIGHTER_OLLAMA_MODEL`, `INSIGHTER_GROQ_BASE_URL`, `INSIGHTER_GROQ_API_KEY` (secret,
  required iff provider=groq), `INSIGHTER_GROQ_MODEL`, `INSIGHTER_TIMEOUT` (>0),
  `INSIGHTER_CACHE_TTL` (>0), `INSIGHTER_CACHE_SIZE` (≥1); `.env.example` updated.
- Tests: a non-advice corpus (order phrasings caught vs holdings/considerations passed),
  adapter `httptest` suites (success, malformed→re-ask, error/429/401→degrade, key never
  in errors), cache hit/miss/TTL/per-user/no-cache-on-error, a full `New(fake)` chain
  test, and an in-memory-exporter assertion that **no facts/content reach a span**
  (BR-505). Gated live-Ollama integration test skips cleanly without `TEST_OLLAMA_URL`.

#### SPEC-006 implementation (MarketDataProvider port & ingestion worker)

- `internal/marketdata` core (pure — no HTTP/SQL/OTel/SDK): the `MarketDataProvider`,
  `FIIQuoteRepository`, `MacroRepository`, and `TickerSource` ports; the `FIIQuote` /
  `MacroIndicator` domain and `Ticker` / `Sector` / `Indicator` / `Unit` value objects;
  sentinel errors; a deterministic `Fake`; and a `Watchlist` ticker source. Market data is
  **global reference data — no `user_id` anywhere** (BR-603).
- `internal/platform/money` — the project's single rounding rule (**half-up**):
  `DecimalToMinor` parses Brazilian and plain decimal strings into `int64` minor units
  (centavos) and integer **basis points** (DY, P/VP, SELIC/CDI/IPCA), so no value is ever
  a `float64` (BR-604).
- Provider adapters behind the port (FII source chosen per D4, free + no key): `fundamentus`
  (one bulk request → price/DY/P-VP/segment, HTML table parsed by header keyword via
  `golang.org/x/net/html`), `yahoo` (`.SA` last dividend, best-effort), and `bcb` (BCB-SGS
  SELIC/CDI/IPCA by series code). A `fii` composite merges fundamentals + last dividend.
  Every adapter caps the body (`io.LimitReader`), sends a descriptive `User-Agent`, and
  degrades to `ErrProviderUnavailable` on any failure — **never corrupting last-known-good**.
- Persistence: migration `0003_market_data` (`fii_quotes` snapshot per ticker;
  `macro_indicators` time series; `bigint`/`integer` only, no floats, no `user_id`) and the
  Postgres repositories with **idempotent `ON CONFLICT` upserts** (a re-run/overlap is safe;
  a failed fetch never overwrites a good row — BR-602).
- `internal/marketdata/ingest` (the edge — OTel lives here, core stays pure): the **worker**
  (`RunOnce` with per-item failure isolation + graceful degradation), the Clock-driven
  **scheduler** (flag-gated, wired into `cmd/api` and drained before the DB closes), the
  composition **factory** (`live` = composite + BCB, default `fake`), and ingestion
  **observability** — `marketdata.ingest` span + per-provider-call child spans, `ingestion_runs`
  / `ingestion_items` counters, and a `seconds_since_last_run` freshness gauge, all metadata
  only (no secrets/payload — BR-608).
- `cmd/ingest` one-shot runner (cron-friendly; `task ingest` / `make ingest`), the
  request-decoupled alternative to the in-process scheduler.
- Configuration `MARKETDATA_PROVIDER` (fake|live), `MARKETDATA_FUNDAMENTUS_BASE_URL`,
  `MARKETDATA_YAHOO_BASE_URL`, `MARKETDATA_BCB_BASE_URL` (all http(s)-validated),
  `MARKETDATA_WATCHLIST`, `MARKETDATA_REFRESH_INTERVAL` (>0), `MARKETDATA_TIMEOUT` (>0),
  `MARKETDATA_SCHEDULER_ENABLED`; `.env.example` updated.
- Tests: value objects + money (BR/plain forms, half-up, overflow/lone-sign rejection),
  adapter `httptest` suites (header-keyed parse, layout-change/HTTP-error/429→degrade,
  best-effort dividend, BCB series codes), worker paths (last-known-good on outage,
  store-failure isolation, macro-only), span-metadata-only assertion, and a **real-Postgres
  integration** proving upsert idempotency + the `0003` round-trip.
- New direct dependency: `golang.org/x/net` (HTML table parsing) — already transitive via
  the OTel stack, so no new download (ADR-0003 stdlib-first posture preserved).

#### SPEC-101 implementation (investor profile)

- `internal/profile` core (pure — no SQL/HTTP/SDK): the `Profile` domain and value objects
  (`RiskProfile`, `Objective`, `Horizon`), the service, sentinels, and two ports —
  `ProfileRepository` (persistence) and **`ProfileReader`** (the consumer seam the Insight
  Engine, Rebalancing, and Health Score will read, SPEC-104/105/106). The first user-facing
  feature, and the first to consume the SPEC-003 identity-from-context seam.
- **Identity from context, enforced structurally (BR-1012):** the PUT DTO has no `user_id`
  field and `DisallowUnknownFields` rejects a smuggled one, so a client cannot supply or
  override identity; the handler passes `auth.UserID(ctx)` to the service and every query is
  scoped `WHERE user_id = $1`.
- Value objects validate on construction (parse-don't-validate): `RiskProfile`
  (conservative|moderate|aggressive), `Objective` (4-value set — **deduplicated, ≥1
  required**), `Horizon` (1–50 whole years).
- `HTTP`: `GET /profile` (200 / 404 when unset) and `PUT /profile` (create-or-update),
  behind the deny-by-default auth middleware; DTOs separate from domain; the generic
  `{"error":"..."}` envelope. No money and no AI output here, so the explainability/non-advice
  gates do not apply (BR-1016).
- Persistence: migration `0004_profiles` (PK `user_id`, FK → `users` `ON DELETE CASCADE`,
  objectives as **`jsonb`** per D1) and the Postgres repository with an idempotent
  `INSERT … ON CONFLICT (user_id) DO UPDATE … RETURNING` upsert that **preserves `created_at`,
  advances `updated_at`, and returns the authoritative row atomically** (no re-read, no race);
  reads are user-scoped, re-validate stored values through their constructors, and map absence
  to `ErrProfileNotFound`.
- Observability: the endpoints inherit route-named `otelhttp` spans (`GET /profile` /
  `PUT /profile`); a test asserts the span carries no profile values (FR-1018).
- Tests: value objects (dedupe/bounds), service (validation, **created_at preservation**),
  handlers (**context-identity, body-`user_id` rejected**, 400/404/401, span-no-PII), and a
  **real-Postgres integration** proving upsert idempotency, per-user isolation, and FK cascade.

#### SPEC-102 implementation (portfolio management)

- `internal/portfolio` core (pure — no SQL/HTTP/SDK): the `FIIHolding` and
  `FixedIncomeHolding` domain, value objects (`Quantity`, `LiquidityType`; the FII `Ticker`
  is **reused from `marketdata`** — a pure value object from a foundational seam, D1), the
  service, sentinels, and two ports — `Repository` (persistence) and **`Reader`** (the
  consumer seam the dashboard, Fact Builder, and projections read, SPEC-103/104/107). The
  system of record for what the user owns, and the first feature to handle money.
- **Identity AND ownership from context (BR-1021):** no DTO carries a `user_id`
  (`DisallowUnknownFields` rejects a smuggled one); reads are scoped `WHERE user_id = $1` and
  mutations are **double-scoped** `WHERE id = $1 AND user_id = $2`, so a cross-user id is
  "not found", never an existence oracle — one user can never read, edit, or delete another's
  holding.
- **Money is `int64` centavos / integer basis points end to end** — domain, DB columns
  (`bigint`/`integer`, no floats), and the JSON wire (`*_centavos` / `*_bps`, never a
  `float64`) — so balances are exact (BR-1022).
- Value objects (parse-don't-validate): `Quantity` (positive whole cotas, D5), `LiquidityType`
  (`daily`|`at_maturity`, D3). The at-maturity **past-date rule** is enforced on create via the
  `Clock`; a daily-liquidity holding normalizes its maturity to null. Cost basis only —
  current value is computed downstream (BR-1024).
- HTTP CRUD: `POST/GET/PUT/DELETE /holdings/fii` and `/holdings/fixed-income` (8 endpoints),
  behind the deny-by-default auth middleware; per-type DTOs separate from domain; `maturity_date`
  as a `YYYY-MM-DD` wire string; ownership errors → `404`. No AI output (FR-013/014 N/A).
- Persistence: migration `0005_holdings` (two tables, UUID PK, `user_id` FK → `users`
  `ON DELETE CASCADE`, `user_id` index, money `bigint` / rate `integer`) and the Postgres
  repository with `RETURNING` creates/updates (preserving `created_at`) and double-scoped,
  ownership-checked update/delete returning `ErrHoldingNotFound`.
- Observability: route-named `otelhttp` spans; tests assert the `{id}` routes stay
  low-cardinality (`PUT /holdings/fii/{id}`, not the raw UUID) and that no money/ticker leaks
  onto a span (FR-1028).
- Tests: value objects, service (maturity matrix, validation, ownership propagation),
  handlers (**context-identity, body-`user_id` rejected**, money round-trip, ownership 404,
  maturity parsing, 401), and a **real-Postgres integration** proving CRUD, per-user isolation,
  **ownership scoping** (B cannot touch A's row), and FK cascade.

### Changed

- Adopted package-oriented (by-feature) organisation over package-by-layer while
  keeping hexagonal principles; clarified in ADR-0002 and SPEC-001 §3a.
- `httpserver.Run` accepts a `context.Context` so shutdown can be triggered by
  cancellation (tests) as well as by OS signals (production).
- Minimum Go version raised to **1.24** with the project's first third-party runtime
  dependencies (`jackc/pgx/v5`, `golang-migrate/migrate/v4`).
- `DATABASE_URL` is now **required** — the app fails fast at startup if it is unset.
- `/readyz` changed from an always-ready stub to a real database dependency check
  (via an injected `Pinger` seam).
- `cmd/api` restructured into a `run() error` so the database pool is reliably closed
  on graceful shutdown.
- `transport/http` `NewRouter` now takes a `Deps` struct (logger, build, readiness,
  auth service, cookie settings) instead of positional arguments.
- Repository read methods named with a `Get<Entity>By<Attr>` convention (e.g.
  `GetUserByEmail`); documented in `CLAUDE.md` for future feature repositories.
- `testify/require` adopted for new tests (`stretchr/testify` promoted to a direct
  dependency), per the testing convention.
- Minimum Go version raised to **1.25** — the OpenTelemetry ecosystem and the
  `golang.org/x/*` libraries now require it; Dockerfile builder bumped to `golang:1.25`.
- Test files are named after the file under test (`foo.go` → `foo_test.go` /
  `foo_integration_test.go`), never after a concept; documented in `CLAUDE.md`.
- **AI governance posture (PRD §6 + SPEC-005) rebalanced from "areas-only" to
  "intelligent copilot".** The non-advice guard now targets the **transaction-order
  signature** (imperative buy/sell, quantity, price/entry-exit target, guaranteed
  return) instead of any asset mention, so the copilot may surface **named candidate
  assets** as reasoned *considerations*. PRD §6 expanded into a full set of AI
  Governance Principles (explainability, facts-computed, portfolio-centric,
  goal-oriented, intelligence-as-duty, full disclosure, user autonomy, no
  advice/orders/false-certainty); added **FR-019** (suggestion capability),
  **FR-020** (risk/assumption disclosure), **FR-021** (user autonomy & no-guarantee);
  tightened **FR-011/FR-013/FR-014** and the Epic 5/6 acceptance criteria. SPEC-005
  FR-504/BR-506/D3, edge cases, and the test corpus updated to require **true-negative**
  cases (legitimate candidates must pass) and to fail closed only on the order
  signature. Added risks for guard over-blocking and CVM personalized-recommendation
  framing.

### Fixed

- `.gitignore` had been overwritten with a literal PowerShell here-string, leaving all
  ignore rules inert; restored to plain patterns so `.env`, `/bin/`, and build
  artifacts are ignored again.
- Integration tests run serialized (`go test ./... -p 1` in `task test:integration`):
  they share one database, and the migration round-trip drops every table, so package
  test binaries must not run concurrently.

### Security

- SPEC-005 security review hardening (the non-advice guard FR-014 is a binding control):
  - Widened the order-signature detector to catch advisory/infinitive phrasings
    ("recomendo aumentar a posição em HGLG11", "you should add 100 shares", "venda tudo",
    "sell half"), fair-value price anchors, and orders split onto their own line — while
    still letting asset-class/diversification considerations through (FR-019). Expanded
    the must-detect / must-pass corpus accordingly.
  - Capped LLM response bodies with `io.LimitReader` (4 MiB) in both adapters, so a
    malfunctioning or hostile endpoint can't exhaust memory.
  - Validate `INSIGHTER_*_BASE_URL` as http(s) at load; warn when the active Groq
    endpoint is cleartext (portfolio facts would be sent unencrypted).
  - `Config` now implements `slog.LogValuer`, masking the Groq API key and OTLP headers
    and redacting the DSN, so an accidental whole-struct log can't leak a secret.

[Unreleased]: https://github.com/biel-ferreira/yield-forge/commits/main
