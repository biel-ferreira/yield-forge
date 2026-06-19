---
description: Implement a SPEC by following its PLAN and the SDD working agreement, phase by phase, closing with review + docs + PT-BR lesson.
argument-hint: [spec-number, e.g. 003]
---

Implement **SPEC-$ARGUMENTS** end to end, following the matching PLAN and the YieldForge
working agreement. The spec number is `$ARGUMENTS`.

## 0. Preconditions (stop if unmet)
- Read `CLAUDE.md` (conventions + binding constraints), then the SPEC
  (`docs/02-specs/SPEC-$ARGUMENTS-*.md`) and its PLAN (`docs/03-plans/PLAN-$ARGUMENTS-*.md`)
  **in full**.
- The working agreement: a SPEC is not built without a matching PLAN. If the PLAN is
  missing, stop and run `/plan-new $ARGUMENTS` first.
- Confirm the SPEC status is **Approved** (or ask the user before proceeding if Draft).

## 1. Implement phase by phase
Follow the PLAN's phases in order (bottom-up: domain → persistence → application → API →
tests → docs). After **each** phase:
- Keep the code compiling and the phase's tests passing.
- Run the quality gate: `task vet` and `task test:short` (raw: `go vet ./...`;
  `go test ./... -short`). gofmt runs automatically via hook. When a phase adds DB or HTTP
  integration tests and `TEST_DATABASE_URL` is set, also run `task test:integration`.
- **Pause and summarize the phase for the user to review before continuing** — this is
  the established cadence (a phase, you review, continue). Do not steamroll all phases.

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

For security-sensitive specs (e.g. auth), also suggest the user run `/security-review`.

## 3. Close the spec (working agreement)
- Update `CHANGELOG.md` `[Unreleased]` in the same change (Keep a Changelog format).
- Update `README.md` if endpoints or env vars changed; update `.env.example` if config changed.
- Flip the SPEC and PLAN **Status to Done**, and update the indexes
  (`docs/02-specs/README.md`, `docs/03-plans/README.md`).
- Invoke the **lesson-writer** subagent to produce `docs/lessons/SPEC-$ARGUMENTS-aula.html`.
- Final gate: `task vet`, `task test:short`, and `go build ./...` clean. Then the
  **integration tests**: if `TEST_DATABASE_URL` is set (or the compose Postgres on host
  port 5433 is up), run `task test:integration` and require it green. If no DB is
  available, say so explicitly — do **not** silently skip: the spec is not Done until the
  integration tests have passed against a real Postgres at least once.
- Once the change is pushed and a PR is opened, run **`/pr-review`** as the final
  pre-merge gate (architecture + correctness + SDD closeout).

Report what was built, the review verdict, and the closing checklist status.
