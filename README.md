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

Go · PostgreSQL · Next.js (later) · free/local LLM behind a swappable port · Docker ·
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
