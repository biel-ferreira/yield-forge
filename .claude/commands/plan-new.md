---
description: Scaffold the PLAN for an approved SPEC from the template, mirroring the spec's number, and register it in the plans index.
argument-hint: [spec-number, e.g. 003]
---

Draft the implementation PLAN for **SPEC-$ARGUMENTS**.

## Steps
1. Read the SPEC (`docs/02-specs/SPEC-$ARGUMENTS-*.md`) in full, plus
   `templates/plan-template.md`, `CLAUDE.md`, and the architecture overview. The SPEC
   should be **Approved**; if it is still Draft, note that the plan is provisional.
2. Create `docs/03-plans/PLAN-$ARGUMENTS-<same-short-name>.md` from the template
   (Status: **Draft**), mirroring the spec's number and name.
3. Define an execution strategy that follows the layered phase order — Domain → Persistence
   → Application → API → Observability → Testing → Documentation — adapted to this spec.
   For each phase: concrete tasks and deliverables that keep the build green and are
   independently reviewable (the per-phase review cadence).
4. Capture: scope (in/out), dependencies + blocking decisions (resolve or flag them),
   architecture impact (components affected / new), risks + mitigations, validation
   checklist, and a Definition of Done that includes the working-agreement closeout
   (CHANGELOG, README, flip to Done, PT-BR lesson).
5. Reflect the conventions in the tasks (money `int64` centavos, `testify/require` + hand
   fakes, errors with `%w`, identity from context, doc comments citing SPEC/BR).
6. Add the plan to `docs/03-plans/README.md`.

Pause and present the draft PLAN for the user to review. Once approved, it can be built
with `/spec-implement $ARGUMENTS`.
