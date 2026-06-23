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

### Fixed

- `.gitignore` had been overwritten with a literal PowerShell here-string, leaving all
  ignore rules inert; restored to plain patterns so `.env`, `/bin/`, and build
  artifacts are ignored again.
- Integration tests run serialized (`go test ./... -p 1` in `task test:integration`):
  they share one database, and the migration round-trip drops every table, so package
  test binaries must not run concurrently.

[Unreleased]: https://github.com/biel-ferreira/yield-forge/commits/main
