# Feature Specification (SPEC)

## 1. Document Information

| Field        | Value                                                  |
| ------------ | ------------------------------------------------------ |
| Feature Name | Project Scaffolding & Hexagonal Layering               |
| Feature ID   | SPEC-001 (foundational)                                |
| Related PRD  | [PRD.md](../01-product/PRD.md) — §12 Constraints, §10 NFR |
| Related ADRs | [ADR-0002](../04-architecture/adr/ADR-0002-tech-stack-and-layering.md), [ADR-0003](../04-architecture/adr/ADR-0003-zero-cost-and-pluggable-llm.md) |
| Version      | 0.1.0                                                  |
| Status       | Draft                                                  |
| Plan         | [PLAN-001](../03-plans/) (to be authored)              |

---

## 2. Overview

### Purpose

Establish the **runnable Go skeleton** and the **hexagonal (ports-and-adapters)
package layout** that every subsequent spec builds on. This spec produces an
application that compiles, starts, serves a health endpoint, loads configuration
from the environment, logs in structured form, shuts down gracefully, and runs
locally in Docker — with **no business logic yet**.

It pins the directory structure, dependency-direction rules, configuration
contract, and developer workflow so that feature specs (SPEC-1xx) and the other
foundational specs (SPEC-002…006) all slot into a known shape.

### Business Value

- **De-risks everything downstream.** The layering and dependency rules defined
  here are what make the "no major redesign" success criterion (PRD) achievable —
  the `Insighter` and `MarketDataProvider` ports later swap implementations without
  touching domain or application code.
- **Momentum.** Produces something that actually runs on day one (`go run ./cmd/api`
  → a live health endpoint), which is the best fuel for a solo learning project.
- **Zero cost.** Defaults to the Go standard library (`net/http`, `log/slog`) to
  minimise dependencies and lock-in (ADR-0003).

### Scope

**In scope:** Go module init, package/directory layout, config loading, HTTP
server bootstrap, health/readiness endpoints, structured logging baseline, graceful
shutdown, Dockerfile + docker-compose (app only), Makefile, `.env.example`.

**Out of scope (owned by later foundational specs):**
- Database connection, migrations, repositories → **SPEC-002**
- Authentication → **SPEC-003**
- OpenTelemetry tracing/metrics → **SPEC-004** (this spec sets only the structured
  **logging** baseline)
- `Insighter` LLM port + adapter → **SPEC-005**
- `MarketDataProvider` port + ingestion → **SPEC-006**
- Any domain entity or use case → feature specs (SPEC-1xx)

---

## 3. Functional Requirements

> These requirements are **spec-scoped**. SPEC-001 implements no PRD functional
> requirement directly — it is the foundation that enables all of them (ADR-0002).

### FR-001 — Go Module & Package Layout

The repository contains an initialised Go module and the package-oriented layout in
§3a. The hexagonal dependency rule holds: a feature's **core** (its domain types +
`service` + port interfaces) imports **no** transport, SQL, or vendor-SDK types and
**no** adapter subpackage; adapters and `transport/http` depend **on** the core,
never the reverse. Features do not import each other's adapters — cross-feature
needs go through a port interface.

**Acceptance Criteria**
- [ ] `go.mod` exists with a sensible module path and the project's Go version.
- [ ] The package layout in §3a exists, each package with a short doc comment.
- [ ] A feature core (e.g. `internal/portfolio` core files) imports nothing from
      `transport`, its own `postgres/`/`gemini/` adapters, or any SQL/HTTP/SDK type.
- [ ] `go build ./...` and `go vet ./...` succeed.

### FR-002 — Configuration Loading

A typed `Config` is loaded from environment variables with sensible defaults; in
local development an optional `.env` file may populate them. Missing **required**
config fails fast with a clear error at startup.

**Acceptance Criteria**
- [ ] `config.Load()` returns a populated `Config` or an explanatory error.
- [ ] Defaults exist for non-secret values (e.g. `APP_PORT=8080`, `APP_ENV=dev`,
      `LOG_LEVEL=info`).
- [ ] Secrets are read only from the environment — never hardcoded or committed.
- [ ] `.env.example` documents every variable with placeholder values.

### FR-003 — HTTP Server Bootstrap & Health Endpoints

An HTTP server starts on the configured port using the standard library
`net/http` and its method+path routing, and exposes liveness/readiness endpoints.

**Acceptance Criteria**
- [ ] `GET /healthz` returns `200` with `{"status":"ok"}` (liveness — always ok if
      the process is up).
- [ ] `GET /readyz` returns `200 {"status":"ready"}` (readiness — SPEC-002 will
      extend it to check dependencies; in SPEC-001 it is always ready).
- [ ] `GET /version` returns build metadata (`version`, `commit`, `built_at`)
      injected at build time via linker flags (placeholder values acceptable in dev).
- [ ] The router is the stdlib `http.ServeMux` (Go 1.22+ pattern routing); no web
      framework dependency is introduced.

### FR-004 — Structured Logging Baseline

The application logs in structured form using the standard library `log/slog`,
with level and format driven by config.

**Acceptance Criteria**
- [ ] A configured `*slog.Logger` is created at startup and injected (not global)
      into the server/handlers.
- [ ] Log level is set from `LOG_LEVEL`; format is JSON in non-dev, human-readable
      in dev.
- [ ] Each HTTP request is logged with method, path, status, and duration.
- [ ] No tracing/metrics here — that is SPEC-004.

### FR-005 — Graceful Shutdown

The process shuts down cleanly on `SIGINT`/`SIGTERM`, draining in-flight requests
within a timeout.

**Acceptance Criteria**
- [ ] `SIGINT`/`SIGTERM` triggers `http.Server.Shutdown` with a bounded timeout.
- [ ] In-flight requests are allowed to finish (within the timeout); new ones are
      refused once shutdown begins.
- [ ] A shutdown log line is emitted; the process exits `0` on clean shutdown.

### FR-006 — Containerised Local Run

The app builds and runs in Docker, with a compose file for local parity.

**Acceptance Criteria**
- [ ] A multi-stage `Dockerfile` produces a small runtime image (e.g. distroless
      or alpine) running a non-root user.
- [ ] `docker compose up` starts the app and `GET /healthz` responds.
- [ ] The compose file is structured so SPEC-002 can add a `postgres` service
      without restructuring.

### FR-007 — Developer Workflow (Makefile)

Common tasks are one command each.

**Acceptance Criteria**
- [ ] `make run`, `make build`, `make test`, `make lint`, `make docker-up` exist
      and work (or print a clear "not yet wired" message where a later spec owns it).
- [ ] `make` with no target prints help.

### FR-008 — Change Traceability (CHANGELOG)

A root `CHANGELOG.md` records notable changes to the project, providing a
human-readable history independent of git log, updated as part of each change.

**Acceptance Criteria**
- [ ] `CHANGELOG.md` exists at the repo root, following the
      [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) format and
      [Semantic Versioning](https://semver.org/) for version headings.
- [ ] It has an **`[Unreleased]`** section at the top; entries are grouped by type
      (`Added` / `Changed` / `Fixed` / `Removed` / `Deprecated` / `Security`).
- [ ] The convention is "update the CHANGELOG in the **same PR** as the change" —
      stated in the file header and the SDD working agreement.
- [ ] The initial file is seeded with the work done so far (SDD baseline + SPEC-001).

---

## 3a. Package & Directory Layout (central deliverable)

**Style: hexagonal *principles* + package-oriented *organisation*.** Packages are
grouped **by feature/capability** (`portfolio`, `marketdata`, `insight`), not by
technical layer. Each feature package owns its domain types, its use cases, and the
**port interfaces** it depends on; concrete **adapters** live as subpackages right
next to the port they implement. The dependency rule of hexagonal still holds: the
feature core never imports its own adapters or any transport/SQL/SDK type.

```
yield-forge/
├── cmd/
│   └── api/
│       └── main.go            # entrypoint: load config → logger → build adapters → wire services → run → graceful shutdown
├── internal/
│   ├── portfolio/             # FEATURE package — example shape (built in SPEC-102)
│   │   ├── portfolio.go       #   domain: Holding entity, value objects, rules        (no infra imports)
│   │   ├── service.go         #   use cases / application logic (the "service")
│   │   ├── repository.go      #   PORT: Repository interface (defined by its consumer)
│   │   └── postgres/          #   ADAPTER: Postgres impl of Repository                 (SPEC-002)
│   ├── profile/               # FEATURE — InvestorProfile                              (SPEC-101)
│   ├── marketdata/            # FEATURE — market data
│   │   ├── marketdata.go      #   domain: Quote, MacroIndicator
│   │   ├── service.go
│   │   ├── provider.go        #   PORT: MarketDataProvider interface
│   │   └── brapi/  bcb/       #   ADAPTERS: concrete data sources                      (SPEC-006)
│   ├── insight/               # FEATURE — AI insights
│   │   ├── insight.go         #   domain: Insight, explanation, non-advice rules
│   │   ├── service.go         #   use cases + explainability / non-advice GATES (middleware)
│   │   ├── insighter.go       #   PORT: Insighter interface
│   │   └── gemini/  ollama/   #   ADAPTERS: free/local LLM implementations             (SPEC-005)
│   ├── projection/            # FEATURE — income & net-worth projections (no LLM)      (SPEC-107)
│   ├── platform/              # KIT / cross-cutting infra (no business logic)
│   │   ├── config/            #   config.Load() → typed Config
│   │   ├── logging/           #   slog logger construction
│   │   └── httpserver/        #   server bootstrap, middleware, graceful shutdown
│   └── transport/
│       └── http/              # DRIVING adapter: router + handlers (controllers) + DTOs.
│           │                  #   Calls feature services; owns HTTP/JSON, nothing else.
│           ├── router.go      #   route table (ServeMux) — wires paths → handlers
│           ├── health.go      #   /healthz, /readyz, /version handlers   (SPEC-001)
│           ├── middleware.go  #   request logging, request-id            (SPEC-001)
│           └── portfolio.go   #   PortfolioHandler + its DTOs            (SPEC-102, example)
├── migrations/                # SQL migrations                          (SPEC-002)
├── deploy/
│   ├── Dockerfile
│   └── docker-compose.yml
├── docs/                      # SDD docs (exists)
├── CHANGELOG.md               # change traceability (Keep a Changelog + SemVer)
├── Makefile
├── .env.example
├── go.mod
└── go.sum
```

> **What SPEC-001 actually creates now:** `cmd/api`, `internal/platform/*`,
> `internal/transport/http` (health/version/middleware/router only), and **empty,
> documented placeholder packages** for the features (`portfolio`, `marketdata`,
> `insight`, `profile`, `projection`) and the `migrations/` dir — so later specs
> have a home. No domain types or business handlers yet.

**Why `internal/`:** Go's `internal/` prevents these packages from being imported
by anything outside the module — keeping the architecture private to this app.

**Ports placement:** each port interface (`Repository`, `MarketDataProvider`,
`Insighter`, `Clock`) is declared **inside the feature package that consumes it**,
per the Go idiom "accept interfaces, return structs". Its concrete adapter is a
subpackage next to it. This co-location is the seam ADR-0002/0003 depend on — and
keeps everything about one capability in one place.

---

## 4. User Flows

> The "user" of SPEC-001 is the **developer** (the author).

### Flow 1 — Run locally (happy path)
1. Developer copies `.env.example` → `.env` and adjusts values.
2. Runs `make run` (or `go run ./cmd/api`).
3. Config loads; logger initialises; server starts on `APP_PORT`.
4. `GET /healthz` → `200 {"status":"ok"}`.

### Flow 2 — Run in Docker
1. Developer runs `make docker-up` (`docker compose up`).
2. App image builds and starts; health endpoint responds on the mapped port.

### Flow 3 — Missing required config (error path)
1. A required env var is absent.
2. `config.Load()` returns an error naming the missing variable.
3. `main` logs the error and exits non-zero **before** starting the server.

### Flow 4 — Graceful shutdown
1. Developer presses `Ctrl-C` (`SIGINT`).
2. Server stops accepting new requests, drains in-flight ones within the timeout.
3. A shutdown log line is emitted; process exits `0`.

---

## 5. Business Rules (Architectural)

- **BR-001 — Dependency direction points to the core.** Within a feature package,
  `adapters (postgres/, gemini/, …) → core (service + domain types + ports)`, and
  `transport/http → core`. The core never imports adapters or `transport`. Enforced
  by review (and optionally a lint rule later).
- **BR-002 — No framework leakage into the core.** A feature's core files must not
  import HTTP, SQL, or any vendor-SDK types. Those live only in adapter subpackages,
  `transport/http`, and `platform`.
- **BR-003 — Ports are interfaces, defined by their consumer.** Each port lives
  **inside the feature package that uses it** (e.g. `portfolio.Repository`,
  `insight.Insighter`); concrete adapters implement it from a subpackage.
- **BR-006 — Features don't depend on each other's adapters.** If feature A needs
  feature B, it depends on B's port/service interface, not B's concrete adapter —
  wiring happens in `cmd/api/main.go`.
- **BR-004 — Configuration is environment-driven (12-factor).** No environment-
  specific values compiled into the binary; secrets only from the environment.
- **BR-005 — Standard library first.** Prefer stdlib (`net/http`, `log/slog`,
  `context`) over third-party libraries unless a clear need justifies a dependency
  (ADR-0003, zero-cost / low lock-in).

---

## 6. Domain Model

No domain entities are introduced in SPEC-001. The feature packages
(`internal/portfolio`, `internal/profile`, `internal/marketdata`,
`internal/insight`, `internal/projection`) exist only as **empty, documented
placeholders**; their domain types (`Holding`, `InvestorProfile`, …) and ports are
defined by their owning feature specs. The only cross-cutting abstraction this spec
may introduce is a `Clock` interface (in `internal/platform` or a small shared
package) so time is injectable and testable later — optional in SPEC-001, but a
natural place to establish the pattern.

---

## 7. API Specification

### Liveness
```
GET /healthz
200 OK
{ "status": "ok" }
```

### Readiness
```
GET /readyz
200 OK
{ "status": "ready" }
# SPEC-002 extends this to 503 { "status": "not_ready", "checks": { "db": "down" } }
# when a dependency check fails.
```

### Version / build info
```
GET /version
200 OK
{ "version": "0.1.0", "commit": "abc1234", "built_at": "2026-06-16T00:00:00Z" }
```

All responses are `application/json`. No authentication on these endpoints (SPEC-003
introduces auth for business endpoints; health/version remain public for probes).

---

## 8. Data Storage

None in SPEC-001. No database connection, schema, tables, or indexes — all deferred
to **SPEC-002 (Persistence Baseline & Migrations)**. The `migrations/` directory and
the per-feature adapter subpackages (e.g. `internal/portfolio/postgres/`) are created
as empty placeholders so SPEC-002 has a home.

---

## 9. Edge Cases

| Scenario | Expected behaviour |
| -------- | ------------------ |
| Required env var missing | `config.Load()` errors naming the variable; process exits non-zero before serving. |
| `APP_PORT` already in use | Server fails to bind; error logged; process exits non-zero. |
| Invalid `LOG_LEVEL` value | Fall back to `info` with a warning, or fail fast — pick one and document it. |
| `SIGTERM` during an in-flight request | Request completes within the shutdown timeout; then process exits. |
| Shutdown exceeds timeout | Force-close remaining connections; log a warning; exit. |
| Unknown route | `404` JSON response, not an HTML default. |

---

## 10. Security Considerations

- Secrets (DB creds, LLM keys — used by later specs) are read **only** from the
  environment; `.env` is git-ignored; `.env.example` holds placeholders.
- The Docker runtime image runs as a **non-root** user and contains no build
  toolchain (multi-stage build).
- Health/version endpoints expose **no sensitive data** (no secrets, no internal
  config dumps).
- Unknown routes return JSON `404` (no stack traces or framework banners leaked).

---

## 11. Observability

SPEC-001 establishes the **logging** baseline only; full OpenTelemetry tracing and
metrics are **SPEC-004**.

- **Logs:** structured `log/slog`; level from `LOG_LEVEL`; JSON in non-dev. A
  request-logging middleware records `method`, `path`, `status`, `duration_ms`, and
  a correlation/request id (generated per request; trace propagation arrives in
  SPEC-004).
- **Metrics / Traces:** out of scope here (SPEC-004) — but the server bootstrap is
  structured so middleware can be added without rework.

---

## 12. Testing Strategy

### Unit Tests
- `config.Load()`: defaults applied, required-missing errors, override via env.
- Logger construction: correct level/format per config.
- Health/version handlers: status codes and JSON bodies.

### Integration Tests
- Start the server on an ephemeral port; assert `GET /healthz`, `/readyz`,
  `/version` responses.
- Graceful shutdown: send `SIGTERM`-equivalent (cancel context) mid-request; assert
  the in-flight request completes and the server stops.

### End-to-End / Manual
- `make run` then `curl localhost:8080/healthz` → `200`.
- `docker compose up` then hit the health endpoint from the host.

### Quality gate
- `go build ./...`, `go vet ./...`, and `gofmt`/`goimports` clean; tests pass; the
  dependency-direction rule (BR-001) holds.

---

## 13. Definition of Done

- [ ] Go module initialised; package layout (§3a) created with doc comments.
- [ ] `config.Load()` implemented with defaults, required-field validation, and
      `.env.example` documented.
- [ ] HTTP server starts; `/healthz`, `/readyz`, `/version` implemented and tested.
- [ ] Structured `slog` logging + request-logging middleware in place.
- [ ] Graceful shutdown on `SIGINT`/`SIGTERM` implemented and tested.
- [ ] `Dockerfile` (multi-stage, non-root) + `docker-compose.yml` (app service);
      `docker compose up` serves the health endpoint.
- [ ] `Makefile` with `run`/`build`/`test`/`lint`/`docker-up` + help.
- [ ] `CHANGELOG.md` created (Keep a Changelog + SemVer), seeded with work to date.
- [ ] `go build`, `go vet`, `gofmt` clean; unit + integration tests pass.
- [ ] Dependency-direction rule (BR-001) verified.
- [ ] PLAN-001 followed; PR reviewed and merged; tagged as the scaffolding baseline.
- [ ] Documentation updated (this spec marked Approved; specs index status flipped).
