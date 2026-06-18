# YieldForge — Investment Copilot

An AI-powered personal investment platform that helps Brazilian retail investors
**understand, monitor, and optimize** a portfolio of FIIs and fixed income through
explainable, data-driven insights.

> It **assists** decisions and **never** gives buy/sell financial advice. Every
> AI-generated insight is explainable.

Built with **Spec-Driven Development** — see [`docs/`](docs/) for the PRD, specs,
plans, and architecture. Start at the [PRD](docs/01-product/PRD.md).

**Status:** early development. Two foundational specs are complete:
[SPEC-001](docs/02-specs/SPEC-001-project-scaffolding-and-layering.md) (runnable Go
skeleton — config, structured logging, health endpoints, graceful shutdown, Docker)
and [SPEC-002](docs/02-specs/SPEC-002-persistence-baseline-and-migrations.md)
(PostgreSQL connection pool, `golang-migrate` migrations, DB-aware `/readyz`).

---

## Tech stack

Go · PostgreSQL · Next.js (later) · free/local LLM behind a swappable port · Docker ·
OpenTelemetry (SPEC-004). The whole stack targets **zero cost** (free tiers /
free-forever / local) — see [ADR-0003](docs/04-architecture/adr/ADR-0003-zero-cost-and-pluggable-llm.md).

## Prerequisites

- **Go** ≥ 1.24 (raised from 1.23 by the pgx / golang-migrate dependencies)
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

| Method | Path        | Purpose                                  |
| ------ | ----------- | ---------------------------------------- |
| GET    | `/healthz`  | Liveness — `200 {"status":"ok"}` (always, if the process is up) |
| GET    | `/readyz`   | Readiness — `200`/`503` reflecting the database (`checks.db`) |
| GET    | `/version`  | Build metadata (`version`/`commit`/`built_at`) |

## Project layout

Package-oriented hexagonal layout — each feature owns its domain, service, and
ports; adapters sit beside them. Full tree and rules in
[SPEC-001 §3a](docs/02-specs/SPEC-001-project-scaffolding-and-layering.md).

```
cmd/api/              entrypoint (config → logger → db → server)
cmd/migrate/          migration runner (up / down / status / create)
internal/
  platform/           config, logging, httpserver, database, buildinfo (cross-cutting)
  transport/http/     router, handlers, DTOs, middleware
  portfolio/ profile/ marketdata/ insight/ projection/   feature packages
migrations/           versioned SQL migrations (embedded via go:embed)
deploy/               Dockerfile, docker-compose.yml
docs/                 SDD: PRD, specs, plans, architecture
```

## Changelog

See [`CHANGELOG.md`](CHANGELOG.md).
