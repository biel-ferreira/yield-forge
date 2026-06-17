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
- Two-tier SPEC/PLAN structure — foundational (`0xx`) and feature (`1xx`).
- SPEC-001 — Project Scaffolding & Hexagonal Layering, with a package-oriented
  (by-feature) hybrid layout; FR-008 requires this changelog.
- PLAN-001 — implementation plan for SPEC-001 (phases, risks, DoD).
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

### Changed

- Adopted package-oriented (by-feature) organisation over package-by-layer while
  keeping hexagonal principles; clarified in ADR-0002 and SPEC-001 §3a.
- `httpserver.Run` accepts a `context.Context` so shutdown can be triggered by
  cancellation (tests) as well as by OS signals (production).

[Unreleased]: https://github.com/biel-ferreira/yield-forge/commits/main
