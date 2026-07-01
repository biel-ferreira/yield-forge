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

- **Frontend harness** — the review/verification layer for the `SPEC-2xx` track, mirroring the Go
  one. Two subagents: **`react-correctness-reviewer`** (hooks/effects + setState-in-effect,
  client/server boundaries, hydration, listener/stream leaks, async races, unsafe TS) and
  **`frontend-reviewer`** (conventions + the client guards, contract-from-OpenAPI, money-no-float,
  identity-from-server, tokens-as-code, a11y). A product-focused **`frontend-lesson-writer`**
  (teaches the *product*, not React — the backend-contract seam in place of the hexagonal bridge).
  Two hooks: **`prettier-edited`** (PostToolUse format-on-edit, the mirror of `gofmt-edited`; skips
  the generated `schema.ts`) and **`on-stop-web`** (Stop reminder — run the `web/` gate, update
  CHANGELOG, regen types). `/spec-implement` and `/pr-review` are now **track-aware** (backend Go
  vs frontend `web/`). Recorded in PLAN-200 (Phase 6) and the `.claude/README.md` inventory.
- **Frontend track kickoff (SDD).** With the backend MVP complete (SPEC-001…108), the
  frontend is opened as a first-class SDD track: **ADR-0006 — Frontend UI Stack & Design
  System** (*Accepted*) decides the layer inside Next.js that [ADR-0004](docs/04-architecture/adr/ADR-0004-frontend-repository-strategy.md)
  deferred — Next.js App Router + React + **TypeScript strict**, **Tailwind + shadcn/ui**
  (copy-in, no lock-in), a **typed API client generated from `api/openapi.yaml`**
  (`openapi-typescript` + `openapi-fetch` + TanStack Query, with a client-side drift check
  mirroring the backend's), **Recharts** for the data-dense views, and the reusable component
  library authored via **Claude Design** (`/design-sync`). The two binding guards become
  first-class UI primitives — an `InsightCard` whose explanation slot is non-optional (FR-013)
  and a `NonAdviceDisclaimer` (FR-014) — and money/rates stay **integer centavos/bps on the
  client**, formatted to a `pt-BR` string only at the render edge (the `float64` ban extends
  into the UI). Introduces the **`SPEC-2xx` frontend spec tier** (foundational `SPEC-20x`,
  feature `SPEC-21x`, matching `PLAN-2xx`), registered in the specs index; refines ADR-0004's
  original `SPEC-1xx` numbering (now occupied by backend features 101–108). First frontend
  spec **SPEC-200 (App Foundation)** drafted.
- **YieldForge design system — "Aurora"** (`docs/05-design/design-system.md`, *alpha*) — a
  token-driven, **dark-first** (optional light theme) system for the web client, chosen after a
  three-concept exploration (calm/emerald · modern/indigo · aurora). Aurora: a near-black canvas
  washed by soft **aurora glows**, one warm **gold `#e9a94c`** accent (often a glowing outline),
  **glass cards** with a colored ambient shadow, and a signature **spectrum gradient** repurposed
  as the allocation-by-sector bar. Type is **Fraunces** (display serif) + **Inter** + **IBM Plex
  Mono** numbers (all SIL Open Font License). Semantic gain/loss/caution/info stay reserved (figure
  colors, never brand or a fill); the glow is decorative and never colors data. pt-BR money/rate
  formatting at the render edge (integer centavos/bps, no float), and the binding guards as
  **first-class components** — an `InsightCard` with a non-optional explanation slot (FR-013) and a
  required `NonAdviceDisclaimer` (FR-014), with Buy/Sell/order affordances deliberately omitted.
  Seeded into the `yieldforge-aurora` Claude Design project (ADR-0006); preview sources under
  `docs/05-design/ds/`.
- **SPEC-200 (App Foundation) approved + PLAN-200 drafted.** The first frontend spec is flipped to
  **Approved**; the copilot is reframed as a **global floating widget** (the shell owns an overlay
  slot; there is **no Chat route** — SPEC-215 implements the widget). **PLAN-200** lays out the build
  in 8 frontend-adapted phases (scaffold → tokens-as-code → typed API client + `money.ts` + SSE
  transport → auth/session → shell + copilot slot → quality gates → tests → docs), resolving five
  decisions: a same-origin **Next.js proxy** for the SPEC-003 cookie, CSR-by-default, Vitest + RTL +
  Playwright, login/register inside SPEC-200, and building the streaming transport up front. Adds the
  full **Dashboard page mockup** (`docs/05-design/ds/pages/dashboard.html`) — app shell + summary +
  allocation + health + insights + the floating copilot — which doubles as the shell spec.

- **SPEC-107 — Projections (Income & Net Worth)**: two deterministic, reproducible forward-looking
  views over the current portfolio — a **passive-income projection** (monthly/annual across
  pessimistic / base / optimistic, from FII dividends + fixed-income rates, base ±200 bps of yield)
  and a **net-worth projection** (value over a configurable horizon, compounding reinvested income +
  a configurable monthly contribution, emitted as yearly `{year, value}` points for charting). Like
  the dashboard, it is **pure computation, no LLM** — same `(holdings, market, contribution, horizon)`
  → same figures + series; integer centavos/bps, half-up monthly compounding, no float. Each scenario
  exposes its assumptions; the output is a labelled non-guaranteed estimate (FR-014). New
  `GET /projections?monthly_contribution_centavos=&horizon_years=` (auth-scoped; integer query params,
  defaults 0/10, horizon 1–40; bad params → 400). Adds the `internal/projection` engine; no new
  tables; documented in `api/openapi.yaml`; PT-BR lesson `docs/lessons/SPEC-107-aula.html`.
- **SPEC-106 — Portfolio Health Score**: a reproducible **0–100 score** with a per-factor breakdown
  (diversification, concentration, liquidity, goal alignment, risk exposure). Unlike the Insight
  Engine and Rebalancing Assistant, the **score and breakdown are computed, not LLM-generated** — the
  PRD reproducibility metric ("same inputs → same score + identical explanation") and the binding
  "LLM never invents numbers" rule both demand it. The score is **market-aware** because macro
  (SELIC) is an *input* to the goal-alignment/risk factors via a modest, documented, bounded tilt —
  so it adjusts with conditions yet stays reproducible given `(portfolio, profile, macro)`. An
  **optional gated "professor" narrative** (the SPEC-005 Insighter, `health_score` task) explains the
  computed result using the live market — grounded, gated (FR-013/014), degradable, and it **never
  changes the number** (test-enforced). New `GET /health-score` (auth-scoped); empty portfolio →
  `score 0`; LLM outage → `narrative_available:false` with the score + breakdown intact. Adds
  `money.WeightedMeanBps`; no new tables; documented in `api/openapi.yaml`; PT-BR lesson
  `docs/lessons/SPEC-106-aula.html`.
- **SPEC-105 — AI Rebalancing Assistant**: given a contribution amount, explainable guidance on
  where to focus the new money — suggested areas, each with a **deterministically computed share**
  of the contribution (`suggested_share_bps`, half-up, Σ = 10 000 via the new `money.AllocateBps`),
  plus **grounded named FII candidates** nested in the FII area. It is the second consumer of the
  published SPEC-104 Fact Builder seam (`BuildFacts`, reused — no second dashboard read), augmenting
  the facts with the contribution and the live FII universe (new `ListFIIUniverse` read). Two
  "computed, not generated" guards: the % split is computed **before** the LLM (which only explains
  it), and a **grounding guard** drops any candidate naming a ticker the system does not know (no
  hallucinated tickers). All guidance is emitted only through the gated `Insighter`, so
  explainability (FR-013) and non-advice (FR-014) hold by construction — a `%` split across areas is
  a consideration, never a transaction order. New `POST /rebalancing` (auth-scoped; `> 0` integer
  centavos, never a float; optional `include_asset_shares` opts into an illustrative per-candidate
  share); empty portfolio still guides; full LLM outage degrades to `200 available:false`. No new
  tables; documented in `api/openapi.yaml`; PT-BR lesson `docs/lessons/SPEC-105-aula.html`.
- **SPEC-104 — AI Insight Engine**: the first AI feature reaching users. A deterministic
  **Fact Builder** (`engine.BuildFacts`) composes a structured snapshot from the dashboard
  (SPEC-103), profile (SPEC-101), and macro (SPEC-006) seams — money as `int64` centavos and
  rates as integer basis points, never a float — and the engine calls the gated `Insighter`
  (SPEC-005) once per category (portfolio / allocation / market_context), so explainability
  (FR-013) and non-advice (FR-014) hold **by construction**: user-facing AI text is emitted
  only through the port. New `GET /insights` (auth-scoped) returns category-tagged insights —
  each carrying an `explanation` — plus the non-advice `disclaimer`; an empty portfolio
  short-circuits with no LLM call, and a full LLM outage degrades to `200 available:false`
  (a partial result when only some categories fail). The Fact Builder is a **published,
  reusable seam** the Conversational Copilot (SPEC-108) and SPEC-105/106 consume. No new
  tables; documented in `api/openapi.yaml`; PT-BR lesson `docs/lessons/SPEC-104-aula.html`.
- **SPEC-108 — Conversational Copilot (Chat)**: the capstone — a multi-turn, fact-grounded chat
  where the investor asks free-form questions and every reply is grounded in computed facts and
  emitted **only** through the gated `Insighter` (SPEC-005), so explainability (FR-013) and
  non-advice (FR-014) hold turn by turn. It invents no new engine — a deterministic **intent
  classifier** routes each turn to the matching engine's facts (general → SPEC-104 `BuildFacts`;
  "tenho R$X pra aportar" → SPEC-105's computed split; "daqui a N anos" → SPEC-107 projections),
  **grounding from the deterministic data with no second LLM call** (new `rebalancing.
  BuildContributionFacts` + `projection.BuildProjectionFacts`, both LLM-free). Prior assistant text
  is dialogue context, never a source of figures. Persists **bounded, clearable** per-user
  threads/messages (rolling eviction, new migration `0006_chat`); a degraded/gate-rejected turn
  returns a safe reply and is not persisted (the thread stays readable). New `POST /chat/messages`
  (create/continue a thread, content ≤ 2000 chars) + `GET/DELETE /chat/threads[/{id}]` — all
  auth-scoped, double-scoped (`ErrThreadNotFound` → 404, no existence oracle), spans/logs carry ids
  only (never message content). Adds a `chat` `insight.Task` (gates unchanged); the deliberate bridge
  into the Phase-2 multi-agent CIO + Phase-3 MCP (ADR-0005). Documented in `api/openapi.yaml`; PT-BR
  lesson `docs/lessons/SPEC-108-aula.html`. **Completes the Phase-1 backend (SPEC-001…108).**
- **ADR-0005 — Conversational Copilot Orchestration** (Proposed): ground each chat turn with a
  pre-built deterministic fact snapshot and emit only through the `Insighter` port; keep live agentic
  MCP tool-calling out of the MVP, behind the same seam, so the multi-agent CIO lands later as an
  adapter without major redesign.
- **PRD — Conversational Copilot**: new **Epic 10**, functional requirements **FR-023** (multi-turn
  chat), **FR-024** (grounded conversation / orchestration seam), and **FR-025** (bounded, clearable
  conversation memory); added to §5 In Scope, §15 as the Phase 1 capstone + Phase 2 bridge, the
  success metrics, and §16 product-level acceptance. Updated the architecture overview (chat
  orchestration seam, §5b) and the specs/plans indexes.
- **OpenAPI 3.1 API documentation** — a hand-maintained spec at [`api/openapi.yaml`](api/openapi.yaml),
  embedded into the binary and served as interactive **Swagger UI at `GET /docs`** plus the
  raw contract at `GET /openapi.yaml` (both public). The spec is served locally; the Swagger UI
  rendering assets load from a pinned CDN build (`swagger-ui-dist@5.17.14`), so there is **no new
  Go dependency** (ADR-0003). All 17 endpoints documented with request/response schemas, status
  codes, the cookie security scheme, and the centavos/bps money convention.
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

#### SPEC-103 implementation (dashboard)

- `internal/dashboard` — a new read-only **compute** feature package (no tables, no writes): it
  reads holdings (SPEC-102) and FII quotes (SPEC-006) through consumer-defined ports
  (`HoldingsReader`, `QuoteSource`) and derives the figures with a **pure, deterministic
  `Compute` function** — the binding "facts are computed, not generated" constraint applied to
  the read model (BR-1031). The first feature to read *across* features.
- **Portfolio summary** (FR-004): total invested (cost basis), **current value = the full
  patrimony / net worth**, monthly passive income (Σ FII `last_dividend × qty`), and growth
  (centavos + bps). **Allocation** (FR-005) by asset class (`fii`/`fixed_income`/`stocks`/`etfs`,
  the last two 0 in MVP) and **FII sector exposure**, each as a share in basis points.
- **Money is `int64` centavos / integer bps end to end** — domain, computation, DTOs, and the
  OpenAPI schema — with every division **half-up** (new `money.ShareBps`) and the FI accrual
  computed via `big.Int` to avoid overflow (new `money.AccrueSimpleInterest`, simple interest
  to today per PRD A4, D2). No `float64` anywhere. Figures **reconcile** (Σ slices = totals).
- **Graceful degradation:** a held FII with no stored quote is valued at **cost basis** and
  listed in `stale_tickers` (FR-1036), so the total still reconciles.
- HTTP `GET /dashboard` (auth-protected, identity from context); registered in the `routeTable`
  and documented in `api/openapi.yaml` (the drift test stays green). No AI output (FR-013/014
  N/A) — these deterministic facts are the substrate the Fact Builder (SPEC-104) will reuse.
- Tests: the pure computation (table-driven — **reconciliation, determinism, stale fallback,
  loss/negative-growth, empty, divide-by-zero guards**), the money helpers (incl. an overflow
  case), the service (stale-not-error, hard-error-propagates), the handler (money round-trip,
  span carries no figures), and a **real-Postgres end-to-end integration** seeding holdings +
  quotes and asserting the computed figures reconcile across SPEC-102 + SPEC-006.

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

- **Fake market-data provider SELIC scale**: `marketdata.Fake.FetchMacroIndicator` returned the
  policy rate as `10_500` bps (≈105%) while labelling it "10,50%"; the real BCB adapter correctly
  stores 10.50% as `1050` bps (1% = 100 bps). Corrected the fake to `1_050` so dev/CI macro data
  matches production — surfaced while building the SPEC-106 market-aware health-score tilt.
- `.env` loader now strips inline `# comments` from values (respecting quotes), so
  `.env.example`'s annotated lines (`DB_MAX_OPEN_CONNS=10   # default: 10`) can be copied
  verbatim into a working `.env` instead of being parsed as invalid values.
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
