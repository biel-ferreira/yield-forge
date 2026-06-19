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

## 2. Architecture & conventions lens
Invoke the **hexagonal-reviewer** subagent on the diff. It checks dependency direction,
the binding product guards (explainability FR-013 / non-advice FR-014), identity-from-
context, and the code conventions (money `int64` centavos, `%w` errors, `testify/require`
+ hand fakes, `Clock` port, doc comments citing SPEC/BR). Surface its verdict verbatim.

## 3. Correctness & robustness lens
Invoke the **go-correctness-reviewer** subagent on the diff. It hunts real bugs — nil
derefs, unchecked errors, concurrency/races, goroutine & resource leaks, SQL injection,
missing validation, edge cases, panics, money-as-float — plus Go best practices. Surface
its verdict verbatim and treat its blocking bugs as merge-blockers.

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
