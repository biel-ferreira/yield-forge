# Implementation Plan

## 1. Document Information

| Field           | Value                                                        |
| --------------- | ------------------------------------------------------------ |
| Plan Name       | Project Scaffolding & Hexagonal Layering                     |
| Related Feature | Foundational — running Go skeleton                           |
| Related Spec    | [SPEC-001](../02-specs/SPEC-001-project-scaffolding-and-layering.md) |
| Version         | 0.1.0                                                        |
| Status          | Draft                                                        |
| Author          | Gabigol                                                      |
| Last Updated    | 2026-06-17                                                   |

---

## 2. Objective

### Goal

Deliver the runnable Go skeleton and the package-oriented (by-feature) hexagonal
layout specified in SPEC-001: an app that builds, loads config, logs in structured
form, serves health/version endpoints, shuts down gracefully, and runs in Docker —
with **no business logic**.

### Expected Outcome

`make run` (or `go run ./cmd/api`) starts the server; `GET /healthz` returns
`200 {"status":"ok"}`; `docker compose up` does the same in a container. The
package layout, dependency rules, config contract, logging baseline, and developer
workflow are in place for SPEC-002+ to build on. `CHANGELOG.md` tracks the change.

---

## 3. Scope

### Included

- Go module init + package-oriented layout with documented placeholder packages.
- `config.Load()` (env-driven, defaults, fail-fast on required, optional `.env`).
- Structured logging baseline (`log/slog`) + request-logging middleware.
- HTTP server (stdlib `net/http` ServeMux) with `/healthz`, `/readyz`, `/version`.
- Graceful shutdown on `SIGINT`/`SIGTERM`.
- Multi-stage `Dockerfile` (non-root) + `docker-compose.yml` (app only).
- `Makefile` (`run`/`build`/`test`/`lint`/`docker-up` + help).
- `CHANGELOG.md` (already created; kept current).
- Unit + integration tests for the above.

### Excluded (owned by later specs)

- Database, migrations, repositories → SPEC-002.
- Authentication → SPEC-003.
- OpenTelemetry tracing/metrics → SPEC-004 (logging only here).
- `Insighter` / `MarketDataProvider` ports + adapters → SPEC-005 / SPEC-006.
- Any domain entity, service, or business handler → feature specs (SPEC-1xx).

---

## 4. Dependencies

### Technical Dependencies

- **Go** — latest stable (pin in `go.mod`; **≥ 1.22** required for `ServeMux`
  pattern routing and `log/slog`).
- **Docker** + Docker Compose (local run).
- **golangci-lint** — dev tool for `make lint` (not a runtime/module dependency).

### External Dependencies

- None at runtime. The MVP target is the standard library only (ADR-0003,
  BR-005). The single optional convenience is a tiny **in-house `.env` loader**
  (dev only, no third-party dependency); `godotenv` is an acceptable swap if
  preferred later.

### Blocking Decisions (resolved)

- **Module path:** `github.com/biel-ferreira/yield-forge` (adjust if the GitHub
  repo path differs).
- **HTTP router:** stdlib `net/http` `ServeMux` (no web framework) — per SPEC-001
  FR-003.
- **Invalid `LOG_LEVEL`:** fall back to `info` and emit a warning (do **not** fail
  fast — logging config is non-critical). Resolves SPEC-001 §9.
- **`Clock` interface:** **deferred** — not introduced in SPEC-001 (no consumer of
  time yet); the first feature needing time establishes it. Resolves SPEC-001 §6.

---

## 5. Architecture Impact

### Existing Components Affected

| Component | Impact |
| --------- | ------ |
| `docs/`   | Update — flip SPEC-001 status; keep CHANGELOG current |
| `.gitignore` / `.gitattributes` | Already present; unchanged |

### New Components

| Component | Purpose |
| --------- | ------- |
| `cmd/api/main.go` | Entrypoint: config → logger → server → run → graceful shutdown |
| `internal/platform/config` | `config.Load()` → typed `Config` |
| `internal/platform/logging` | `slog` logger construction from config |
| `internal/platform/httpserver` | Server bootstrap, middleware, graceful shutdown |
| `internal/transport/http` | Router + health/version handlers + request middleware |
| `internal/{portfolio,profile,marketdata,insight,projection}` | Empty, documented placeholder feature packages |
| `migrations/` | Empty placeholder dir (SPEC-002) |
| `deploy/Dockerfile`, `deploy/docker-compose.yml` | Containerised local run |
| `Makefile`, `.env.example` | Developer workflow + config contract |

---

## 6. Implementation Strategy

### Approach

Build the skeleton bottom-up so each phase compiles and is independently testable:
config → logging → HTTP/health → shutdown → container → tests → docs. Keep the
feature packages as documented empty placeholders so the layout (SPEC-001 §3a) is
fully present without any business logic. Honour the dependency rules (BR-001…006)
from the first commit.

### Rollout Method

**Incremental** — one PR for SPEC-001 (small enough to review in one pass). May be
split into two PRs if convenient (skeleton+config+logging, then HTTP+docker), but a
single PR is expected.

### Rollback Strategy

Greenfield, not deployed anywhere — rollback is simply reverting the PR. No data,
no migrations, no external state to undo.

---

## 7. Implementation Phases

> The template's domain/persistence/application phases don't apply to a foundational
> scaffolding spec; phases below are adapted to the actual work.

### Phase 1 — Repo & Module Bootstrap

#### Tasks
- [ ] `go mod init github.com/biel-ferreira/yield-forge`.
- [ ] Create the package layout from SPEC-001 §3a; add a doc comment to each
      package (incl. empty feature placeholders).
- [ ] Add a `.go` doc file (e.g. `doc.go`) to placeholder packages so they compile.
- [ ] Scaffold the `Makefile` (`run`/`build`/`test`/`lint`/`docker-up`, default help).

#### Deliverables
- Module + full directory tree; `go build ./...` succeeds; `make` prints help.

---

### Phase 2 — Configuration

#### Tasks
- [ ] Define `Config` struct (`AppEnv`, `AppPort`, `LogLevel`, `LogFormat`, …).
- [ ] `config.Load()` — read env, apply defaults, validate required, return typed
      `Config` or descriptive error.
- [ ] Optional dev `.env` loading (in-house minimal parser; only when present).
- [ ] `LOG_LEVEL` invalid → default `info` + warning (per §4 decision).
- [ ] Write `.env.example` documenting every variable.

#### Deliverables
- Config loads with defaults; missing-required errors clearly; `.env.example` ready.

---

### Phase 3 — Logging Baseline

#### Tasks
- [ ] `logging.New(cfg)` → `*slog.Logger` (level from config; JSON in non-dev,
      text in dev).
- [ ] Inject the logger (no global); pass into server/handlers.

#### Deliverables
- Structured logger constructed from config and injected.

---

### Phase 4 — HTTP Server, Health Endpoints & Shutdown

#### Tasks
- [ ] `transport/http` router (`ServeMux`) + handlers: `/healthz`, `/readyz`,
      `/version` (build info via linker flags).
- [ ] Request-logging + request-id middleware (method, path, status, duration_ms).
- [ ] JSON `404` for unknown routes (no HTML default).
- [ ] `httpserver` bootstrap: start on `APP_PORT`; `Shutdown` on `SIGINT`/`SIGTERM`
      with a bounded timeout via `signal.NotifyContext`.
- [ ] Wire everything in `cmd/api/main.go`.

#### Deliverables
- Server serves health/version; graceful shutdown works; requests are logged.

---

### Phase 5 — Containerisation

#### Tasks
- [ ] Multi-stage `Dockerfile` (build on `golang`, run on distroless/alpine,
      non-root user).
- [ ] `docker-compose.yml` with the `app` service (structured so SPEC-002 can add
      `postgres` without restructuring).
- [ ] `make docker-up` wired.

#### Deliverables
- `docker compose up` serves `/healthz` from the host.

---

### Phase 6 — Testing

#### Unit Tests
- [ ] `config.Load`: defaults, required-missing error, env override, invalid
      `LOG_LEVEL` fallback.
- [ ] `logging.New`: correct level/format per config.
- [ ] Health/version handlers: status + JSON body (`httptest`).

#### Integration Tests
- [ ] Boot server on ephemeral port; assert `/healthz`, `/readyz`, `/version`.
- [ ] Graceful shutdown: cancel context mid-request; in-flight completes, server
      stops.

#### End-to-End Tests
- [ ] Manual: `make run` + `curl /healthz`; `docker compose up` + health from host.

#### Deliverables
- `go test ./...` green; `go vet` + `gofmt`/`goimports` clean.

---

### Phase 7 — Documentation

#### Tasks
- [ ] Update `CHANGELOG.md` `[Unreleased]` with the scaffolding work.
- [ ] Add a root `README.md` quickstart (run, test, docker) — or note as follow-up.
- [ ] Flip SPEC-001 status to Approved/Done in the specs index.

#### Deliverables
- Docs current; SPEC-001 closed.

---

## 8. Risks

| Risk | Impact | Mitigation |
| ---- | ------ | ---------- |
| Over-engineering the skeleton (premature abstractions) | Medium | Strict scope (§3 Excluded); placeholders stay empty; defer `Clock`. |
| In-house `.env` parser edge cases (quotes/comments) | Low | Keep dev-only + minimal; swap to `godotenv` if it bites. |
| Dependency-direction drift from day one | Medium | Encode BR-001…006 in review; consider an import-lint rule later. |
| Go/router version mismatch (ServeMux routing needs ≥1.22) | Low | Pin Go version in `go.mod`; document minimum. |
| Docker image bloat / running as root | Low | Multi-stage build; distroless/alpine; explicit non-root user. |

---

## 9. Validation Checklist

### Functional Validation
- [ ] FR-001…FR-008 acceptance criteria satisfied.
- [ ] `/healthz`, `/readyz`, `/version` behave as specified.
- [ ] Graceful shutdown drains in-flight requests.

### Technical Validation
- [ ] Layout matches SPEC-001 §3a; dependency rules (BR-001…006) hold.
- [ ] Secrets only from env; `.env` git-ignored; `.env.example` complete.
- [ ] Docker image runs non-root; no build toolchain in runtime image.

### Quality Validation
- [ ] Unit + integration tests pass.
- [ ] `go build`, `go vet`, `gofmt`/`goimports`, `golangci-lint` clean.
- [ ] Code reviewed; CHANGELOG updated in the same PR.

---

## 10. Definition of Done

- [ ] All phases complete; SPEC-001 acceptance criteria met.
- [ ] `make run` and `docker compose up` both serve `/healthz`.
- [ ] Tests pass; lint/vet/fmt clean.
- [ ] `CHANGELOG.md` updated; SPEC-001 marked Approved in the index.
- [ ] PR reviewed and merged to `main`; tagged as the scaffolding baseline.

---

## 11. Deliverables

### Code Deliverables
- `cmd/api/main.go`, `internal/platform/*`, `internal/transport/http/*`, documented
  placeholder feature packages.

### Infrastructure Deliverables
- `deploy/Dockerfile`, `deploy/docker-compose.yml`, `Makefile`, `.env.example`.

### Documentation Deliverables
- Updated `CHANGELOG.md`; (optional) root `README.md` quickstart; SPEC-001 status
  flipped.

---

## 12. Post-Implementation Tasks

### Monitoring
- None yet (OTel is SPEC-004). Confirm health endpoints are suitable for future
  container/orchestration probes.

### Future Improvements
- Add an import-direction lint rule to enforce BR-001…006 automatically.
- Add a root `README.md` if deferred.

### Technical Debt
- In-house `.env` loader is intentionally minimal — revisit if config grows.
