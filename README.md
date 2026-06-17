# YieldForge — Investment Copilot

An AI-powered personal investment platform that helps Brazilian retail investors
**understand, monitor, and optimize** a portfolio of FIIs and fixed income through
explainable, data-driven insights.

> It **assists** decisions and **never** gives buy/sell financial advice. Every
> AI-generated insight is explainable.

Built with **Spec-Driven Development** — see [`docs/`](docs/) for the PRD, specs,
plans, and architecture. Start at the [PRD](docs/01-product/PRD.md).

**Status:** early development. The foundational scaffolding ([SPEC-001](docs/02-specs/SPEC-001-project-scaffolding-and-layering.md))
is complete — a runnable Go skeleton with config, structured logging, health
endpoints, graceful shutdown, Docker, and tests.

---

## Tech stack

Go · PostgreSQL (SPEC-002) · Next.js (later) · free/local LLM behind a swappable
port · Docker · OpenTelemetry (SPEC-004). The whole stack targets **zero cost**
(free tiers / free-forever / local) — see [ADR-0003](docs/04-architecture/adr/ADR-0003-zero-cost-and-pluggable-llm.md).

## Prerequisites

- **Go** ≥ 1.22 (1.23 recommended)
- **Docker** (optional — for the containerised run)
- **[Task](https://taskfile.dev)** (optional — convenience task runner). Without it,
  use the raw `go` commands shown below.

## Quickstart

```bash
# Run the API locally  →  http://localhost:8080
task run            # or: go run ./cmd/api

# In another terminal:
curl http://localhost:8080/healthz   # {"status":"ok"}
curl http://localhost:8080/version   # {"version":"dev","commit":"none",...}
```

### Common tasks

| Task            | Raw command                                              |
| --------------- | -------------------------------------------------------- |
| `task run`      | `go run ./cmd/api`                                       |
| `task build`    | `go build -o bin/yield-forge ./cmd/api`                  |
| `task test`     | `go test ./... -cover`                                   |
| `task test:short` | `go test ./... -short` (skips socket integration tests) |
| `task lint`     | `go vet ./...`                                           |
| `task docker-up`| `docker compose -f deploy/docker-compose.yml up --build` |

> On Windows, `task` works in any shell; the `Makefile` is kept for Unix/CI.

## Configuration

Environment-driven (12-factor). Copy [`.env.example`](.env.example) to `.env` for
local development — real environment variables always take precedence. See the file
for every variable and its default.

## Endpoints

| Method | Path        | Purpose                                  |
| ------ | ----------- | ---------------------------------------- |
| GET    | `/healthz`  | Liveness — `200 {"status":"ok"}`         |
| GET    | `/readyz`   | Readiness (dependency checks in SPEC-002)|
| GET    | `/version`  | Build metadata (`version`/`commit`/`built_at`) |

## Project layout

Package-oriented hexagonal layout — each feature owns its domain, service, and
ports; adapters sit beside them. Full tree and rules in
[SPEC-001 §3a](docs/02-specs/SPEC-001-project-scaffolding-and-layering.md).

```
cmd/api/              entrypoint (config → logger → server)
internal/
  platform/           config, logging, httpserver, buildinfo (cross-cutting)
  transport/http/     router, handlers, DTOs, middleware
  portfolio/ profile/ marketdata/ insight/ projection/   feature packages
deploy/               Dockerfile, docker-compose.yml
docs/                 SDD: PRD, specs, plans, architecture
```

## Changelog

See [`CHANGELOG.md`](CHANGELOG.md).
