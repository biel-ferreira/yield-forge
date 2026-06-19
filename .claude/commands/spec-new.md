---
description: Scaffold a new SPEC from the template, grounded in a PRD slice, and register it in the specs index.
argument-hint: [short-name, e.g. portfolio-management]
---

Draft a new feature specification for: **$ARGUMENTS**.

## Steps
1. Read `docs/01-product/PRD.md` (the relevant FRs/epic), `templates/spec-template.md`,
   and `docs/02-specs/README.md` (the index, build order, and naming/numbering rules).
2. **Pick the number + tier.** Foundational cross-cutting seam → `SPEC-0NN`; user-facing
   feature → `SPEC-1NN`. Use the next free number in that tier (check the index).
3. Create `docs/02-specs/SPEC-<NN>-$ARGUMENTS.md` from the template, filled in for this
   feature and grounded in the PRD slice it implements:
   - Document Information (Status: **Draft**; link the PRD and any governing ADRs).
   - Functional requirements with `FR-<NN>x` IDs and concrete, checkable acceptance criteria.
   - Business rules (`BR-<NN>x`), domain model, API contract, data model, edge cases,
     security, observability, testing strategy, Definition of Done.
   - Carry down the binding constraints when AI output is involved (explainability FR-013,
     non-advice FR-014) and the conventions (money, identity-from-context, etc.).
4. Add the spec to the table in `docs/02-specs/README.md` with its PRD refs and status.
5. Do **not** write a PLAN or any code yet (working agreement: SPEC first, then `/plan-new`).

Pause and present the draft for the user to review and approve. Keep it specific to the
PRD — do not invent scope beyond the slice it covers.
