# CLAUDE.md — YieldForge

AI-powered Investment Copilot for Brazilian retail investors (FIIs + Fixed Income).
Go backend, hexagonal/ports-and-adapters, PostgreSQL, free/local LLM behind a
swappable port. Built with **Spec-Driven Development**. Single source of truth lives
in [`docs/`](docs/) — start at the [PRD](docs/01-product/PRD.md).

## Binding product constraints (NEVER violate)

These flow down from PRD §6 into every spec, line of code, and user-facing string:

1. **Explainability first** — every AI insight/score/suggestion MUST carry a
   human-readable explanation. No black-box output. (FR-013 is the enforcement gate.)
2. **AI as copilot, never advisor** — NEVER emit specific buy/sell orders, tickers to
   buy, quantities, or price targets. Output is _areas/considerations_ + a non-advice
   disclaimer. (FR-014 is the enforcement gate.)
3. **Facts are computed, not generated** — the LLM reasons _over_ deterministic
   portfolio/market facts (the Fact Builder); it never invents numbers.
4. **Zero cost** — free tiers / free-forever / local only. Every external provider
   (LLM, market data, DB host) sits behind a port and is config-swappable.

## SDD working agreement

- A feature is not built until it has a **SPEC**; a SPEC is not built until it has a
  matching **PLAN** (same number). Flow: PRD → SPEC → PLAN → CODE.
- Numbering: foundational `SPEC-0NN` / feature `SPEC-1NN`; `PLAN-<same-number>`.
  Templates in [`templates/`](templates/). Build order in [docs/02-specs/README.md](docs/02-specs/README.md).
- Update [`CHANGELOG.md`](CHANGELOG.md) `[Unreleased]` in the **same change** as any
  notable work (Keep a Changelog format, SemVer headings).
- On closing a spec: flip its SPEC + PLAN status to Done, update `README.md` if
  endpoints/env changed, and produce a **PT-BR HTML lesson** at
  `docs/lessons/SPEC-0NN-aula.html`.
- ADRs are immutable once accepted — supersede, never edit.

## Architecture rules

Package-oriented hexagonal (see [architecture-overview.md](docs/04-architecture/architecture-overview.md)
and [SPEC-001](docs/02-specs/SPEC-001-project-scaffolding-and-layering.md)):

- **Dependency direction:** domain core imports NO SQL, HTTP, or framework types.
  Ports (interfaces) live with the feature; adapters implement them at the edge.
- Each feature package (`portfolio/`, `profile/`, `marketdata/`, `insight/`,
  `projection/`) owns its domain + service + ports; adapters sit beside it.
- Cross-cutting code lives in `internal/platform/` (config, logging, httpserver,
  database, buildinfo). HTTP router/handlers/middleware in `internal/transport/http/`.
- Identity comes from the authenticated session/context, **never** from a request
  payload (no client-supplied `user_id` is trusted). Per-user scoping is `WHERE
user_id = $1` with the ID from context.
- All external providers behind ports: `Insighter` (LLM), `MarketDataProvider`,
  repositories, `Clock`. This is the seam for the future multi-agent CIO + MCP — keep
  it intact.

## Code conventions

Deliberately chosen (not accidental). Apply to all new Go code:

- **Money is never `float64`.** Monetary amounts are `int64` **minor units (centavos)**.
  Rates/percentages (dividend yield, SELIC, projection rates) are integers in **basis
  points** (1 bp = 0.01%), never floats. All money math + rounding lives in a small
  helper (`internal/platform/money`) with **one documented rounding rule** (half-up) so
  results are deterministic and reproducible (PRD: same inputs → same Health Score).
- **Errors:** wrap with `%w` and a lowercase action prefix —
  `fmt.Errorf("create user: %w", err)`. Domain errors are sentinels
  (`var ErrInvalidTicker = errors.New("...")`); check with `errors.Is`/`errors.As`.
- **Parse, don't validate.** Value objects (`Ticker`, `Money`, `Email`) validate in
  their constructor and return an error — an invalid instance must be unrepresentable.
- **Domain is pure:** no SQL, HTTP, time, or I/O in domain/service core (enforced by
  the layering rules above).
- **Context:** `ctx context.Context` is always the first parameter and is propagated;
  `context.Background()` only in `main` and tests.
- **Time is UTC** and comes from the injected **`Clock`** port, never `time.Now()`
  directly — keeps projections and tests deterministic.
- **Interfaces small + consumer-defined**: "accept interfaces, return structs." Keeps
  ports tiny so fakes stay trivial.
- **Repository methods name their operation:** reads are `Get<Entity>By<Attr>`
  (e.g. `GetUserByEmail`, `GetUserByID`); writes use `Create*` / `Update*` / `Delete*`.
  A read that can be absent returns a `…NotFound` sentinel.
- **Tests:** standard `testing` + **table-driven** structure; **`testify/require`** for
  assertions; **hand-written fakes** for ports — no `gomock`/`mockery`. Integration
  tests gated by `testing.Short()` + `TEST_DATABASE_URL` (skip cleanly without a DB).
- **HTTP:** request/response DTOs are separate from domain types; validate at the edge;
  errors use the generic `{"error":"..."}` envelope via the `writeJSON` helper.
- **Doc comments cite the governing SPEC/BR** they implement (e.g. `(SPEC-002 BR-201)`)
  — keeps SDD traceability from doc to code.
- **Language:** code, docs (PRD/SPEC/PLAN/ADR/README), comments, and commit messages are
  in **English**. The only exception is the **PT-BR HTML lessons**
  (`docs/lessons/*-aula.html`) — deliberately Portuguese teaching material.
- **Commits:** Conventional Commits (`feat:`, `fix:`, `docs:`, `test:`, `refactor:`).
- **Dependencies:** stdlib-first; justify any new dependency (zero-cost posture,
  ADR-0003). `golangci-lint` is the mechanical enforcer once configured.

## Commands

Task runner is [`Task`](https://taskfile.dev) (`Taskfile.yml`); raw `go` fallback shown.

| Need                           | Task                                                | Raw                                                      |
| ------------------------------ | --------------------------------------------------- | -------------------------------------------------------- |
| Quality gate (run before done) | `task vet` + `task test:short`                      | `go vet ./...`; `go test ./... -short`                   |
| Build                          | `task build`                                        | `go build ./...`                                         |
| Run API                        | `task run`                                          | `go run ./cmd/api` (needs `DATABASE_URL`)                |
| Unit tests (no DB)             | `task test:short`                                   | `go test ./... -short`                                   |
| Integration tests (real PG)    | `task test:integration`                             | `go test ./... -count=1` (needs `TEST_DATABASE_URL`)     |
| Migrations                     | `task migrate:up` / `:status` / `:create -- <name>` | `go run ./cmd/migrate <cmd>`                             |
| Docker stack                   | `task docker-up`                                    | `docker compose -f deploy/docker-compose.yml up --build` |

- Always finish a change with `gofmt`-clean, `go vet` clean, unit tests passing.
- Migrations are paired up/down SQL in `migrations/` (embedded via `go:embed`),
  applied **manually** — never auto-run. Local Postgres is on host port **5433**.
- Integration tests skip cleanly without `TEST_DATABASE_URL`; gated by `testing.Short()`.

## Environment notes

- Platform is **Windows + PowerShell** (primary). `Task` works in any shell; the
  `Makefile` is the Unix/CI mirror. Prefer `task ...` over OS-specific commands.
- Go ≥ 1.24. Module: `github.com/biel-ferreira/yield-forge`. No ORM (`database/sql` + pgx).
- Config is 12-factor / env-driven; `.env.example` is the contract. `DATABASE_URL`
  required. Secrets never committed.

## Status

Done: SPEC-001 (skeleton), SPEC-002 (persistence). Approved & next: SPEC-003 (auth).
See [docs/02-specs/README.md](docs/02-specs/README.md) for the live index.
