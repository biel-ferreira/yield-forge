# YieldForge — Investment Copilot

An AI-powered personal investment platform that helps Brazilian retail investors
**understand, monitor, and optimize** a portfolio of FIIs and fixed income through
explainable, data-driven insights.

> It **assists** decisions and **never** gives buy/sell financial advice. Every
> AI-generated insight is explainable.

Built with **Spec-Driven Development** — see [`docs/`](docs/) for the PRD, specs,
plans, and architecture. Start at the [PRD](docs/01-product/PRD.md).

**Status:** early development. Four foundational specs are complete:
[SPEC-001](docs/02-specs/SPEC-001-project-scaffolding-and-layering.md) (runnable Go
skeleton — config, structured logging, health endpoints, graceful shutdown, Docker),
[SPEC-002](docs/02-specs/SPEC-002-persistence-baseline-and-migrations.md)
(PostgreSQL connection pool, `golang-migrate` migrations, DB-aware `/readyz`),
[SPEC-003](docs/02-specs/SPEC-003-authentication-and-per-user-isolation.md)
(email+password auth, server-side sessions, deny-by-default per-user isolation), and
[SPEC-004](docs/02-specs/SPEC-004-observability-baseline.md)
(OpenTelemetry traces + metrics + log correlation, no-op without a backend).

---

## Tech stack

Go · PostgreSQL · **Next.js** (the [`web/`](web/) client) · free/local LLM behind a swappable port · Docker ·
OpenTelemetry. The whole stack targets **zero cost** (free tiers /
free-forever / local) — see [ADR-0003](docs/04-architecture/adr/ADR-0003-zero-cost-and-pluggable-llm.md).

## Prerequisites

- **Go** ≥ 1.25 (raised by the OpenTelemetry ecosystem in SPEC-004)
- **Docker** — for the local PostgreSQL (and the containerised run)
- **[Task](https://taskfile.dev)** (optional — convenience task runner). Without it,
  use the raw `go` commands shown below.

## Quickstart

The app **requires a database** (`DATABASE_URL`) and fails fast without one. The
fastest path uses Docker for Postgres:

```bash
cp .env.example .env        # DATABASE_URL points at the compose db on localhost:5433

task docker-up              # starts the API (8080) + PostgreSQL (host port 5433)
task migrate:up             # applies migrations (run from a second terminal)

curl http://localhost:8080/healthz   # {"status":"ok"}
curl http://localhost:8080/readyz    # {"status":"ready","checks":{"db":"up"}}
curl http://localhost:8080/version   # {"version":"dev","commit":"none",...}
```

To run the API on the host instead of in a container (Postgres still in Docker):
`docker compose -f deploy/docker-compose.yml up -d db` → `task migrate:up` →
`task run`.

### Common tasks

| Task                  | Raw command                                              |
| --------------------- | -------------------------------------------------------- |
| `task run`            | `go run ./cmd/api`                                       |
| `task build`          | `go build -o bin/yield-forge ./cmd/api`                  |
| `task test`           | `go test ./... -cover`                                   |
| `task test:short`     | `go test ./... -short` (unit only — skips integration)  |
| `task test:integration` | `go test ./... -count=1` (needs `TEST_DATABASE_URL`)  |
| `task migrate:up`     | `go run ./cmd/migrate up`                                |
| `task migrate:down`   | `go run ./cmd/migrate down [n]`                          |
| `task migrate:status` | `go run ./cmd/migrate status`                            |
| `task migrate:create` | `go run ./cmd/migrate create <name>`                     |
| `task lint`           | `go vet ./...`                                           |
| `task docker-up`      | `docker compose -f deploy/docker-compose.yml up --build` |

> On Windows, `task` works in any shell; the `Makefile` is kept for Unix/CI.

## Configuration

Environment-driven (12-factor). Copy [`.env.example`](.env.example) to `.env` for
local development — real environment variables always take precedence. See the file
for every variable and its default. **`DATABASE_URL` is required.**

## Database & migrations

PostgreSQL, accessed through `database/sql` + the pgx driver (no ORM). Schema changes
are versioned, paired up/down SQL files in [`migrations/`](migrations/), applied with
[`golang-migrate`](https://github.com/golang-migrate/migrate) via the `cmd/migrate`
runner:

```bash
task migrate:up                 # apply all pending migrations
task migrate:status             # show the current schema version
task migrate:down -- 1          # roll back one migration
task migrate:create -- add_xyz  # scaffold migrations/NNNN_add_xyz.up/down.sql
```

Migrations are applied **manually** (never auto-run). Local Postgres is published on
host port **5433** (chosen to coexist with a native Postgres on 5432); inside the
compose network the API reaches it as `db:5432`. Inspect it with any client (e.g.
DBeaver) using the credentials in `.env.example`.

> **Integration tests** run against a real Postgres when `TEST_DATABASE_URL` is set
> (point it at a disposable database — the migration round-trip test rolls schema
> changes back and forth). Without it, they skip cleanly.

## AI insights (Insighter)

AI insights are produced by the **`Insighter` port** (SPEC-005), so the LLM provider is
config-swappable and the binding product guards are enforced regardless of provider:

- **Explainability (FR-013):** every insight must carry a human-readable explanation, or
  it is rejected.
- **Non-advice (FR-014):** output that reads as a buy/sell order (or price target /
  guaranteed return) is rejected; passing output gets a non-advice disclaimer. The LLM
  reasons only over **computed facts** — it never invents numbers.

The provider is selected by `INSIGHTER_PROVIDER`:

| Value | Use | Notes |
| ----- | --- | ----- |
| `ollama` (default) | Local dev | Talks to a local [Ollama](https://ollama.com) (`INSIGHTER_OLLAMA_BASE_URL`, `INSIGHTER_OLLAMA_MODEL`); facts stay on-device. |
| `groq` | Hosted | OpenAI-compatible free tier; needs `INSIGHTER_GROQ_API_KEY` (secret). |
| `fake` | CI / AI-off | Deterministic, no network — used by tests and when AI is disabled. |

Results are cached in-memory (`INSIGHTER_CACHE_SIZE`, `INSIGHTER_CACHE_TTL`) and every
call is bounded by `INSIGHTER_TIMEOUT`; on any provider failure the Insighter degrades
gracefully rather than erroring. AI telemetry records only metadata (provider, model,
outcome, latency, cache hit) — never prompts, facts, or generated text. See
[`.env.example`](.env.example) for all `INSIGHTER_*` variables.

> The returned `Insighter` isn't wired into an endpoint yet — the AI feature engine
> (SPEC-104) consumes it with the Fact Builder. SPEC-005 ships the port + guards + adapters.

## Market data (ingestion)

Market data is ingested by a background **worker** behind the **`MarketDataProvider` port**
(SPEC-006) and stored as **global, last-known-good reference data** (no per-user scoping):

- **FII quotes** (FR-006) — price, dividend yield, P/VP, sector, last dividend.
- **Macro indicators** (FR-007) — SELIC, CDI, IPCA (IFIX is a documented gap; see below).

All money is `int64` **centavos** and all rates integer **basis points** — never `float64`
(via `internal/platform/money`, half-up). A failed/malformed fetch **never overwrites** good
data; upserts are idempotent.

The provider is selected by `MARKETDATA_PROVIDER`:

| Value | Use | Sources |
| ----- | --- | ------- |
| `fake` (default) | Dev / CI | Deterministic, no network |
| `live` | Real data | **Fundamentus** (FII fundamentals, one bulk request) + **Yahoo** `.SA` (last dividend) + **BCB-SGS** (macro) — all free, no API key |

Run it two ways (both call the same pass):

```bash
task ingest                          # one-shot (cron-friendly); raw: go run ./cmd/ingest
# or: the in-process scheduler runs inside the API when MARKETDATA_SCHEDULER_ENABLED=true
```

Set `MARKETDATA_SCHEDULER_ENABLED=false` for multi-replica deploys and drive `cmd/ingest`
from cron, to avoid duplicate ingestion. See [`.env.example`](.env.example) for all
`MARKETDATA_*` variables. Ingestion telemetry records only metadata (provider, outcome,
counts, freshness) — no payloads.

> **Known gaps:** the FII source is **scraped** (no official free API), so it is treated
> defensively (header-keyed parsing, degrade-to-last-known-good) and is config-swappable to
> a licensed source later. **IFIX** has no free source yet, so it degrades gracefully and is
> a tracked follow-up. The stored data isn't surfaced in an endpoint yet — the dashboard
> (SPEC-103) and Fact Builder (SPEC-104) consume it.

## Endpoints

| Method | Path             | Auth | Purpose                                  |
| ------ | ---------------- | ---- | ---------------------------------------- |
| GET    | `/healthz`       | public | Liveness — `200 {"status":"ok"}` (always, if the process is up) |
| GET    | `/readyz`        | public | Readiness — `200`/`503` reflecting the database (`checks.db`) |
| GET    | `/version`       | public | Build metadata (`version`/`commit`/`built_at`) |
| POST   | `/auth/register` | public | Create an account — `{email,password}` → `201 {id,email}` |
| POST   | `/auth/login`    | public | Start a session — sets an `HttpOnly` session cookie |
| POST   | `/auth/logout`   | session | Revoke the current session (`204`) |
| GET    | `/auth/me`       | session | The authenticated caller's `{id,email}` |
| GET    | `/profile`       | session | The caller's investor profile — `200` or `404 {"error":"profile not set"}` |
| PUT    | `/profile`       | session | Create/replace the profile — `{risk_profile, objectives[], horizon_years}` (SPEC-101) |
| POST   | `/holdings/fii`  | session | Create an FII holding — `{ticker, quantity, average_price_centavos}` → `201` (SPEC-102) |
| GET    | `/holdings/fii`  | session | List the caller's FII holdings |
| PUT    | `/holdings/fii/{id}`    | session | Update an owned FII holding (`404` if not owned) |
| DELETE | `/holdings/fii/{id}`    | session | Delete an owned FII holding (`204` / `404`) |
| POST   | `/holdings/fixed-income` | session | Create a fixed-income holding — `{name, institution, invested_amount_centavos, annual_rate_bps, maturity_date, liquidity_type}` → `201` |
| GET    | `/holdings/fixed-income` | session | List the caller's fixed-income holdings |
| PUT    | `/holdings/fixed-income/{id}` | session | Update an owned fixed-income holding |
| DELETE | `/holdings/fixed-income/{id}` | session | Delete an owned fixed-income holding |
| GET    | `/dashboard`     | session | Computed summary + allocation + FII sector exposure (full patrimony; money as `*_centavos`, shares as `*_bps`); SPEC-103 |
| GET    | `/insights`      | session | Explainable AI insights (portfolio / allocation / market_context) grounded in computed facts; every item carries an `explanation` (FR-013) and the response a non-advice `disclaimer` (FR-014); `available:false` on a full LLM outage; SPEC-104 |
| POST   | `/rebalancing`   | session | Contribution guidance — `{contribution_centavos, include_asset_shares?}` → suggested areas with a **computed** `suggested_share_bps` (Σ 10000) + grounded named FII candidates nested in the FII area; every item explained (FR-013), non-advice `disclaimer` (FR-014); SPEC-105 |
| GET    | `/health-score`  | session | Reproducible 0–100 **computed** Portfolio Health Score + per-factor breakdown (diversification / concentration / liquidity / goal_alignment / risk_exposure, weights as `*_bps`); market-aware (macro is an input); an optional gated AI `narrative` explains it (`narrative_available:false` on outage) but never changes the number; SPEC-106 |
| GET    | `/projections`   | session | Deterministic income + net-worth projections (pessimistic / base / optimistic), query `?monthly_contribution_centavos=&horizon_years=` (int, defaults 0/10, horizon 1–40); net-worth as yearly `{year, value_centavos}` points; computed (not LLM), assumptions + estimate `disclaimer` shown (FR-014); SPEC-107 |
| POST   | `/chat/messages` | session | Conversational copilot turn — `{thread_id?, content}` → gated assistant reply (`explanation` + `disclaimer`); grounds each turn in computed facts (general / "tenho R$X" → SPEC-105 / "daqui a N anos" → SPEC-107), routed by intent; `available:false` on LLM outage; SPEC-108 |
| GET    | `/chat/threads`  | session | List the caller's conversation threads (most-recent first) |
| GET    | `/chat/threads/{id}` | session | Read a thread + its ordered messages (`404` if not owned) |
| DELETE | `/chat/threads/{id}` | session | Delete a thread (`204`) |
| DELETE | `/chat/threads`  | session | Clear all conversation history (`204`); threads are bounded + clearable (FR-025) |

> Money crosses the wire as **integer centavos** (`*_centavos`) and rates as integer basis
> points (`*_bps`) — never a float. `maturity_date` is a `YYYY-MM-DD` string (null for
> daily-liquidity holdings). All `/holdings/*` routes are per-user and ownership-scoped.

### API docs (OpenAPI / Swagger)

The full contract is the hand-maintained OpenAPI 3.1 spec at [`api/openapi.yaml`](api/openapi.yaml),
embedded into the binary and served two public ways:

| Method | Path            | Purpose                                              |
| ------ | --------------- | ---------------------------------------------------- |
| GET    | `/docs`         | Interactive **Swagger UI** (renders the spec)        |
| GET    | `/openapi.yaml` | The raw OpenAPI 3.1 document                          |

The spec is served locally (embedded); `/docs` loads the Swagger UI rendering assets from a
pinned CDN build, so there is no extra Go dependency. (The `/docs` page therefore needs
internet to render; the spec at `/openapi.yaml` is fully local.)

The spec is kept in lockstep with the router by a build-failing drift test
(`internal/transport/http/openapi_test.go`): every registered route must be documented, and
vice-versa. Adding or changing an endpoint **requires** updating `api/openapi.yaml` in the
same change (see `CLAUDE.md`).

## Authentication

Email + password with **server-side sessions** (SPEC-003). Passwords are stored only
as bcrypt hashes; the session token lives in an `HttpOnly` + `SameSite=Lax` cookie
(`Secure` outside dev), and only its `sha256` is stored. Routes are **deny-by-default**:
everything requires a valid session except the five public routes above. Identity comes
from the session, never from request input — the seam feature endpoints scope data by.

```bash
curl -c jar -X POST localhost:8080/auth/register -d '{"email":"me@x.com","password":"supersecret"}'
curl -c jar -X POST localhost:8080/auth/login    -d '{"email":"me@x.com","password":"supersecret"}'
curl -b jar localhost:8080/auth/me     # {"id":"...","email":"me@x.com"}
```

Config: `SESSION_TTL` (default `168h`) and `AUTH_COOKIE_NAME` (default `yf_session`).

## Observability

OpenTelemetry **traces + metrics + log correlation** (SPEC-004), wired as cross-cutting
infrastructure so every endpoint is observable: route-named HTTP spans (`GET /auth/me`)
→ child DB query spans, request-latency metrics, and `trace_id`/`span_id` on the logs.

It is **off by default and never required to run** — with no exporter configured the
pipeline is a no-op (zero cost). Point it at any OTLP backend to turn it on:

```bash
# e.g. a local Jaeger all-in-one exposing OTLP/HTTP on :4318
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 task run
# or just print spans/metrics to the console:
OTEL_EXPORTER_KIND=stdout task run
```

Config: `OTEL_EXPORTER_OTLP_ENDPOINT` (empty ⇒ disabled), `OTEL_EXPORTER_KIND`
(`otlp`/`stdout`/`none`), `OTEL_EXPORTER_OTLP_HEADERS` (secret, e.g. a backend API key),
`OTEL_SERVICE_NAME`, `OTEL_TRACE_SAMPLE_RATIO`. Telemetry never carries secrets/PII
(no passwords, tokens, raw emails, or SQL argument values).

## Project layout

Package-oriented hexagonal layout — each feature owns its domain, service, and
ports; adapters sit beside them. Full tree and rules in
[SPEC-001 §3a](docs/02-specs/SPEC-001-project-scaffolding-and-layering.md).

```
cmd/api/              entrypoint (config → logger → db → server)
cmd/migrate/          migration runner (up / down / status / create)
internal/
  platform/           config, logging, httpserver, database, clock, observability, buildinfo (cross-cutting)
  transport/http/     router, handlers, DTOs, middleware (incl. auth middleware)
  auth/               authentication feature — domain, service, ports + bcrypt/postgres adapters
  portfolio/ profile/ marketdata/ insight/ projection/   feature packages
migrations/           versioned SQL migrations (embedded via go:embed)
deploy/               Dockerfile, docker-compose.yml
docs/                 SDD: PRD, specs, plans, architecture
```

## Changelog

See [`CHANGELOG.md`](CHANGELOG.md).
