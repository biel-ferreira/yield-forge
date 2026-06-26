# `.claude/` — the YieldForge harness

This directory is the **development harness**: the system around the model (Claude) that
makes it efficient and safe at building this project. Harness engineering = engineering the
**context, tools, memory, control flow, and verification** around the model — not the model
itself.

> Full lesson (PT-BR): [`docs/lessons/harness-engineering-aula.html`](../docs/lessons/harness-engineering-aula.html).

---

## Mental model — 4 buckets

Every primitive serves one of these purposes:

```
CONTEXT       →  what I know          →  CLAUDE.md (+ rules)
AUTOMATION    →  what runs on its own →  hooks
FLOW SHORTCUTS→  what YOU trigger     →  commands / skills
WORKERS       →  who I delegate to    →  agents (subagents)
```

## What each primitive is

| Primitive | Where it lives | Who triggers it | In one line |
| --------- | -------------- | --------------- | ----------- |
| **CLAUDE.md** | repo root | automatic, every session | Project memory: facts and rules I must always know |
| **Rules** | *not a folder* | — | A **concept**, not a file. Soft in `CLAUDE.md`, hard in hooks/permissions |
| **Hooks** | `.claude/hooks/` + `settings.json` | the harness, on events | Deterministic script on `PreToolUse`/`PostToolUse`/`Stop`. Guaranteed automation |
| **Commands** | `.claude/commands/` | **you**, via `/name` | Template for a repeatable flow, run in the main context |
| **Agents** | `.claude/agents/` | me (delegation) or you (`@name`) | Specialized "worker" with isolated context and its own tools |
| **Skills** | `.claude/skills/<n>/SKILL.md` | **me**, when the task matches | Knowledge/procedure I pull in on my own when relevant (can bundle scripts) |

### The two that get confused

- **Rules** is not a folder (that's Cursor/Windsurf). It is the *concept* "a project rule".
  Where it lives depends on strength: a rule I *should* follow → `CLAUDE.md` (soft); a rule
  that must *never* fail → hook/permission (hard).
- **Command vs Skill** = who decides to use it. **Command**: you type `/...`. **Skill**: I
  recognize the task and pull it in myself. In current Claude Code they are nearly the same
  machine — a command is a skill only you invoke. Think **"who triggers it"**, not the word.

---

## When to add/improve the harness

The practical trigger for each piece:

| Add a… | When the signal is… |
| ------ | ------------------- |
| **Rule** (`CLAUDE.md`) | "I had to correct/explain the same thing twice" — a convention/preference being missed |
| **Hook** | "It's not enough that I *should* — it must happen **every time**, and is machine-checkable" |
| **Command** | "I've typed this same step-by-step 3+ times" — a repeatable procedure |
| **Agent** | "I want an isolated-context specialist (review/research) that won't clutter the chat, and I'll reuse it" |
| **Skill** | "There's knowledge/a procedure I should apply *on my own* when the topic comes up" |

### 4 principles that govern the "when"

1. **Rule of three** — automate on the *third* repetition, not the first. Earlier is speculation.
2. **Lowest level that enforces it** — prose (`CLAUDE.md`) for "should"; a hook for "must".
3. **Don't build speculatively** — the harness grows from *real pain*, not "might be useful". An extra piece = extra maintenance.
4. **Improve on failure** — a bug slipped that a reviewer should have caught? Strengthen the reviewer. A convention got violated? Make it a rule or a hook. The harness evolves by reacting to concrete failures.

> Caveat: the rule of three is about avoiding *speculation*. An **already-established**
> convention (e.g. "all docs in English") gets codified the **first** time you see it broken,
> not after three misses.

> Summary: **add it when the pain hits the 3rd time, and place it at the lowest level that fixes it.**

---

## Current inventory

**Memory & config** (root + here)
- [`../CLAUDE.md`](../CLAUDE.md) — project memory (binding constraints, layering, conventions, commands).
- `settings.json` — versioned (shared) permissions + hook registration. **Committed.**
- `settings.local.json` — your machine's local allowlist (Read paths + recurring local
  Bash/WebFetch you've approved). Machine-specific; prune one-off/debug entries periodically. **Gitignored.**

**Hooks** (`hooks/`) — the 3 modes
- `block-immutable.ps1` — `PreToolUse`: **blocks** editing a committed migration/ADR (`exit 2`).
- `block-layering.ps1` — `PostToolUse`: **blocks** (`exit 2`) a feature *core* file
  (`internal/<feature>/<file>.go`) that imports SQL/HTTP/vendor — the #1 architecture rule
  promoted from the `hexagonal-reviewer` (subjective) to a deterministic gate. Adapter
  subpackages, `platform/`, and `transport/` are skipped.
- `gofmt-edited.ps1` — `PostToolUse`: **acts**, runs `gofmt` on the just-edited `.go` file.
- `on-stop.ps1` — `Stop`: **warns** (non-blocking) with `go vet` + a CHANGELOG reminder when `.go` changed.

**Agents** (`agents/`) — isolated-context subagents
- `hexagonal-reviewer` — architecture/layering, explainability/non-advice guards, conventions.
- `go-correctness-reviewer` — nil derefs, unchecked errors, concurrency/races, leaks, SQL, edge cases.
- `lesson-writer` — produces the PT-BR HTML lesson (with Harness + AI Engineering sections).

**Commands** (`commands/`) — the SDD loop
- `/spec-new` — draft a SPEC from the template, grounded in the PRD.
- `/plan-new` — draft the PLAN mirroring the SPEC's number.
- `/spec-implement` — implement phase by phase (`phased`) or straight through (`auto`); close with review + docs + lesson.
- `/pr-review` — final PR gate: 2 reviewers + SDD closeout (needs `gh`).

**Skills** (`skills/`) — *not created yet.* The "I-apply-it-myself" / bundled-scripts kind.
Future candidate: something in Tier 3 (the product), e.g. a grounding/evals helper for the
`Insighter` (SPEC-005).

---

## Operational notes

- Changes to `settings.json`/hooks/agents/commands are read at **session start** — restart
  for them to take effect.
- Commit everything under `.claude/` **except** `settings.local.json` (gitignored).
- The same concepts here reappear in the **product harness** (`Insighter` port, FR-013/014
  gates, multi-agent CIO, MCP) — see the Harness Engineering lesson.
