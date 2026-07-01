---
description: Review a GitHub PR (or the current branch's PR) against YieldForge's architecture, binding guards, conventions, and the SDD closeout — delegating the architecture lens to the hexagonal-reviewer subagent.
argument-hint: [pr-number (optional — defaults to the current branch's PR)]
allowed-tools: Bash(gh *), Bash(git *), Read, Grep, Glob
---

Review the pull request as the final gate before merge. Target PR: **$ARGUMENTS** (if
empty, resolve the current branch's PR with `gh pr view --json number,title,headRefName`).

## 1. Gather the PR
- `gh pr view $ARGUMENTS --json number,title,body,headRefName,baseRefName,files`
- `gh pr diff $ARGUMENTS` for the full diff. Note which spec this PR closes (from the
  branch name / title / body).

**Track — pick the review lenses by what the PR touches.** A **backend** PR (Go under
`internal/`, `cmd/`) uses the **hexagonal-reviewer** + **go-correctness-reviewer** pair
(§2–§3 as written). A **frontend** PR (`SPEC-2xx`, changes under `web/`) instead uses
**frontend-reviewer** + **react-correctness-reviewer**, and its closeout (§4) expects the
`web/` gate (typecheck / lint / check:api / build) rather than Go build+tests, with **no**
`api/openapi.yaml` or migration changes.

## 2. Architecture & conventions lens
Invoke the **hexagonal-reviewer** subagent on the diff. It checks dependency direction,
the binding product guards (explainability FR-013 / non-advice FR-014), identity-from-
context, and the code conventions (money `int64` centavos, `%w` errors, `testify/require`
+ hand fakes, `Clock` port, doc comments citing SPEC/BR). Surface its verdict verbatim.

> **Frontend PR:** invoke **frontend-reviewer** instead — contract-from-OpenAPI (no
> hand-written DTOs), money-no-float/format-at-edge, the guards on the client (`InsightCard`
> explanation + `NonAdviceDisclaimer`, no order affordances), identity-from-server, tokens-as-
> code, and accessibility.

## 3. Correctness & robustness lens
Invoke the **go-correctness-reviewer** subagent on the diff. It hunts real bugs — nil
derefs, unchecked errors, concurrency/races, goroutine & resource leaks, SQL injection,
missing validation, edge cases, panics, money-as-float — plus Go best practices. Surface
its verdict verbatim and treat its blocking bugs as merge-blockers.

> **Frontend PR:** invoke **react-correctness-reviewer** instead — hook rules, effect/deps +
> setState-in-effect, client/server component boundaries, hydration mismatches, listener/
> subscription/stream leaks, async races, and unsafe types.

For auth/security-sensitive PRs, also recommend the user run `/security-review` (secrets,
plaintext passwords/tokens, authz gaps).

## 4. SDD closeout gate (working agreement)
Verify the PR satisfies the Definition of Done:
- `CHANGELOG.md` `[Unreleased]` updated in the same PR.
- `README.md` / `.env.example` updated if endpoints or env changed.
- The SPEC and PLAN flipped to **Done** in their docs and indexes.
- The PT-BR lesson `docs/lessons/SPEC-0NN-aula.html` exists.
- No edits to committed migrations or accepted ADRs (superseded, not edited).

## 5. Verdict
Summarize: **APPROVE / CHANGES REQUESTED**, blocking issues first (each `file:line —
problem — fix`), then non-blocking notes, then the closeout checklist status.

If the user passed a request to post (e.g. `--comment`), post the summary with
`gh pr comment $ARGUMENTS --body ...`; otherwise just report it here and let the user decide.
