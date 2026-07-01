---
description: Implement a SPEC by following its PLAN and the SDD working agreement, closing with review + docs + PT-BR lesson. Phased (pause per phase) or auto (run through, one review at the end).
argument-hint: [spec-number, e.g. 003] [mode: phased|auto — default phased]
---

Implement a SPEC end to end, following its PLAN and the YieldForge working agreement.

**Arguments** (`$ARGUMENTS`): the first token is the **spec number** `<NNN>` (e.g. `003`);
an optional second token is the **mode** — `phased` (default) or `auto`. Use only `<NNN>`
when resolving the file paths below.

## Track — backend (Go) vs frontend (`web/`)

YieldForge is a mono-repo with two tracks. Detect the SPEC's track and substitute the right
tooling throughout this flow:

- **Backend** (`SPEC-0xx` / `SPEC-1xx`; Go under `internal/`, `cmd/`): the gate is `task vet`
  + `task test:short`; the review lenses are **hexagonal-reviewer** + **go-correctness-reviewer**;
  the lesson is written by **lesson-writer**. (This is what the steps below describe by default.)
- **Frontend** (`SPEC-2xx`; TypeScript/React under `web/`): also read **`web/CLAUDE.md`**; the
  gate is `npm run typecheck` + `npm run lint` + `npm run build` (from `web/`, Node ≥ 20) —
  **not** `task vet`/`go test`; the review lenses are **frontend-reviewer** +
  **react-correctness-reviewer**; the lesson is written by **frontend-lesson-writer**. A frontend
  spec adds **no** `api/openapi.yaml` change and has no Postgres integration gate (its integration
  check is the auth/API flow against the running backend). Everything else — the phased cadence,
  the SDD closeout — is identical.

## 0. Preconditions (stop if unmet)
- Read `CLAUDE.md` (conventions + binding constraints), then the SPEC
  (`docs/02-specs/SPEC-<NNN>-*.md`) and its PLAN (`docs/03-plans/PLAN-<NNN>-*.md`)
  **in full**.
- The working agreement: a SPEC is not built without a matching PLAN. If the PLAN is
  missing, stop and run `/plan-new <NNN>` first.
- Confirm the SPEC status is **Approved** (or ask the user before proceeding if Draft).

## 1. Implement the phases
Follow the PLAN's phases in order (bottom-up: domain → persistence → application → API →
tests → docs). After **every** phase, regardless of mode, keep the build green:
- Keep the code compiling and the phase's tests passing.
- Run the quality gate: `task vet` and `task test:short` (raw: `go vet ./...`;
  `go test ./... -short`). gofmt runs automatically via hook. When a phase adds DB or HTTP
  integration tests and `TEST_DATABASE_URL` is set, also run `task test:integration`.

**Cadence depends on the mode:**
- **`phased`** (default) — after each phase, pause and summarize it for the user to review
  before continuing. Do not steamroll. Best for learning and for foundational/security specs.
- **`auto`** — run all phases end to end without pausing (still gating build/tests per
  phase). Give a one-line running note per phase, but save the full review for step 2 at
  the end. `auto` skips the *review* pause, not your judgment: if a phase's gate fails or a
  real ambiguity/design decision arises, still stop and ask.

Honor the architecture + conventions while coding (the hexagonal-reviewer will check):
dependency direction, money as `int64` centavos, errors with `%w`, `testify/require` +
hand-written fakes, `Clock` port over `time.Now()`, identity from context, doc comments
citing the SPEC/BR. Do **not** edit committed migrations or accepted ADRs (create new
ones — the PreToolUse hook enforces this).

## 2. Review before closing
Run two review lenses on the finished change and fix blocking findings before proceeding:
- **hexagonal-reviewer** subagent — architecture/layering, binding guards (explainability
  / non-advice), conventions, identity-from-context.
- **go-correctness-reviewer** subagent — nil derefs, unchecked errors, concurrency/races,
  resource leaks, SQL safety, edge cases, best practices.

> **Frontend track (`SPEC-2xx`):** use **frontend-reviewer** (conventions, the guards on the
> client, money-no-float, contract-from-OpenAPI, identity-from-server, tokens-as-code, a11y) +
> **react-correctness-reviewer** (hooks/effects + setState-in-effect, client/server boundaries,
> hydration, listener/stream leaks, async races, unsafe TS) instead of the Go pair.

For security-sensitive specs (e.g. auth), also suggest the user run `/security-review`.

## 3. Close the spec (working agreement)
- Update `CHANGELOG.md` `[Unreleased]` in the same change (Keep a Changelog format).
- Update `README.md` if endpoints or env vars changed; update `.env.example` if config changed.
- Flip the SPEC and PLAN **Status to Done**, and update the indexes
  (`docs/02-specs/README.md`, `docs/03-plans/README.md`).
- Invoke the **lesson-writer** subagent to produce `docs/lessons/SPEC-<NNN>-aula.html`
  (frontend spec → invoke **frontend-lesson-writer** instead; same output path).
- Final gate: `task vet`, `task test:short`, and `go build ./...` clean. Then the
  **integration tests**: if `TEST_DATABASE_URL` is set (or the compose Postgres on host
  port 5433 is up), run `task test:integration` and require it green. If no DB is
  available, say so explicitly — do **not** silently skip: the spec is not Done until the
  integration tests have passed against a real Postgres at least once.
  - **Frontend track:** the final gate is `npm run typecheck` + `npm run lint` +
    `npm run check:api` + `npm run build` (from `web/`) clean, plus the auth/API flow verified
    against the running backend (Go API + Postgres up); there is no Go build or Postgres gate.
- Once the change is pushed and a PR is opened, run **`/pr-review`** as the final
  pre-merge gate (architecture + correctness + SDD closeout).

Report what was built, the review verdict, and the closing checklist status.
