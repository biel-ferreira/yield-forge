# PLAN-210 — Investor Profile Screen

## 1. Document Information

| Field           | Value                                   |
| --------------- | --------------------------------------- |
| Plan Name       | Investor Profile Screen                 |
| Related Feature | Investor Profile Screen                 |
| Related Spec    | [SPEC-210](../02-specs/SPEC-210-investor-profile-screen.md) (Done) |
| Version         | 1.0.0                                    |
| Status          | Completed                               |
| Author          | Gabigol                                 |
| Last Updated    | 2026-07-01                              |

> **Phase-order note.** Frontend spec — the template's backend phase order is mapped to its
> frontend analogue (data → components → screen → tests → docs). This feature **builds on the
> SPEC-200 foundation** (typed client, design system, shell, auth gate, Vitest/Playwright, the
> track-aware harness), so it is small: no new infra, only the profile data + form + screen.

---

## 2. Objective

### Goal

Turn the `/profile` stub into the real **Perfil** screen (SPEC-210): load the investor profile
from `GET /profile`, let the user set risk profile / objectives / horizon, and save via
`PUT /profile` — using the SPEC-200 typed client and the Aurora design system.

### Expected Outcome

A returning user sees their profile prefilled; a first-run user sees an empty form; both can save
a valid profile (risk + ≥1 objective + horizon 1–50) that persists and reflects on reload. No new
endpoint, **no `api/openapi.yaml` change**.

---

## 3. Scope

### Included

- Profile **data hooks** (`useProfile` over `GET /profile`, `useSaveProfile` over `PUT /profile`)
  and the **enum ↔ pt-BR label** mapping.
- Reusable **form controls**: a segmented single-select, a multi-select chip group, and a
  keyboard-accessible **slider** (native range), all token-styled.
- The **Perfil screen** (`app/(app)/profile/page.tsx`): load (200 prefill / 404 first-run /
  loading / error), edge validation, explicit **Salvar**, success + error feedback.
- Tests (Vitest/RTL + integration + a Playwright E2E) and the SDD closeout.

### Excluded

- Any **backend change** — SPEC-101 is Done; this consumes it. No new endpoint, no `openapi.yaml` edit.
- A **first-run onboarding flow** (SPEC-210 §14 decision: the Perfil screen only for the MVP).
- Consuming the profile in AI features (that's SPEC-213) — this screen only *captures* it.

---

## 4. Dependencies

### Technical Dependencies

- **SPEC-101** backend (`GET /profile` 200/404, `PUT /profile`) — Done and running.
- **SPEC-200** foundation: the typed client (`ProfileRequest`/`ProfileResponse` in `lib/api/schema.ts`),
  TanStack Query, the `(app)` shell + `RequireAuth`, the design system, the Vitest/Playwright setup.

### External Dependencies

- None new. Zero new runtime deps (ADR-0003): the slider is a native `<input type="range">`.

### Blocking Decisions

| # | Decision | Resolution (this plan) |
|---|----------|------------------------|
| D1 | Where the form controls live | Generic primitives in **`components/ui/`** (`segmented`, `chip-toggle`, `slider`); the Perfil screen composes them. Reusable by later screens. |
| D2 | Profile data hooks | **`lib/profile/`** — `useProfile` + `useSaveProfile`, mirroring `lib/auth/session.ts`. |
| D3 | `GET /profile` `404` handling | Treat **404 as `null`** ("no profile yet") in the query fn — the analogue of `useSession`'s 401→null — so the screen shows a first-run empty state, not an error. |
| D4 | Enum ↔ label mapping | Centralized in **`lib/profile/labels.ts`**; the wire always carries the API enums, the UI the pt-BR labels. |
| D5 | Horizon slider | **Native `<input type="range">`** (zero-dep, keyboard-accessible), styled with tokens; value shown numerically. |

---

## 5. Architecture Impact

### Existing Components Affected

| Component | Impact |
| --------- | ------ |
| `app/(app)/profile/page.tsx` | Stub → the real Perfil screen |
| `docs/05-design/design-system.md` | Optional: note the `slider` control alongside the existing `risk-profile-segmented` / `objective-chip` specs |

### New Components

| Component | Purpose |
| --------- | ------- |
| `lib/profile/profile.ts` | `useProfile` (GET, 404→null) + `useSaveProfile` (PUT, invalidates) |
| `lib/profile/labels.ts` | risk/objective enum ↔ pt-BR label maps |
| `components/ui/segmented.tsx` | single-select segmented control |
| `components/ui/chip-toggle.tsx` | multi-select toggle chip (group) |
| `components/ui/slider.tsx` | accessible range slider + value |
| `app/(app)/profile/` form pieces | the Perfil form composition |

---

## 6. Implementation Strategy

### Approach

**Incremental**, bottom-up: data + mapping first (typed, testable in isolation), then the reusable
controls, then the screen that composes them, then tests, then closeout. Each phase leaves the
`web/` gate green (`typecheck`/`lint`/`check:api`/`test`/`build`) and is independently reviewable.

### Rollout Method

**Incremental.** The screen replaces a stub behind the existing auth gate; no launch, no flag.

### Rollback Strategy

Revert the `web/**` changes — no backend, no data, no migration involved.

---

## 7. Implementation Phases

### Phase 1 — Profile data & label mapping  *(≈ persistence/data)*

#### Tasks
- [ ] `lib/profile/labels.ts` — `RiskProfile` and `Objective` enum ↔ pt-BR label maps (all values),
      typed from `components["schemas"]` so a contract change surfaces in TS.
- [ ] `lib/profile/profile.ts` — `useProfile()` (`GET /profile`; **404 → null**; typed) and
      `useSaveProfile()` (`PUT /profile`; on success invalidate the profile query); surface the
      `{"error":"..."}` message on failure. No hand-written DTOs (BR-2104).

#### Deliverables
- Typed hooks + mapping, unit-testable; the gate stays green.

---

### Phase 2 — Form controls  *(≈ domain/UI vocabulary)*

#### Tasks
- [ ] `components/ui/segmented.tsx` — single-select segmented control (design-system tokens; a11y roles).
- [ ] `components/ui/chip-toggle.tsx` — a multi-select chip group (selected uses `objective-chip-selected`).
- [ ] `components/ui/slider.tsx` — native range slider, **1–50**, keyboard-accessible, labelled, value shown.
- [ ] Token-styled per the Aurora design system; no raw hex; reduced-motion respected.

#### Deliverables
- Three reusable, tested controls; render in the styleguide for a visual check.

---

### Phase 3 — The Perfil screen  *(≈ application/edge)*

#### Tasks
- [ ] `app/(app)/profile/page.tsx` — compose the form (risk segmented, objective chips, horizon slider).
- [ ] Load states: `200` prefill · `404` first-run empty state ("defina seu perfil") · loading skeleton ·
      transient error → retry (FR-2101).
- [ ] **Edge validation** (risk required, ≥1 objective, horizon 1–50) gating an explicit **Salvar**
      (disabled while invalid/pending); `PUT` body is exactly `{ risk_profile, objectives, horizon_years }`
      — **no `user_id`** (BR-2101).
- [ ] Success feedback + query invalidation; `400` surfaced inline with input preserved (FR-2105).

#### Deliverables
- A working, navigable Perfil screen wired to the live backend.

---

### Phase 4 — Testing

#### Unit / Component (Vitest + RTL)
- [ ] Enum ↔ pt-BR label mapping (all values).
- [ ] Validation: Salvar blocked with no objective / out-of-range horizon; enabled when valid.
- [ ] `404` → first-run empty state; `200` → prefill.
- [ ] Submit builds the exact `ProfileRequest` (no `user_id`) and calls `PUT /profile` (mocked client).

#### Integration
- [ ] Against a running backend: `GET` → save → re-`GET` reflects the change.

#### End-to-End (Playwright)
- [ ] Fill the profile → Salvar → reload → values persist. Gated to skip without a backend.

#### Deliverables
- Green unit/component (in CI) + integration + a gated E2E.

---

### Phase 5 — Documentation & Closeout

#### Tasks
- [ ] **CHANGELOG** `[Unreleased]` entry.
- [ ] **No `api/openapi.yaml` change** — assert it (consumes SPEC-101; adds no endpoint).
- [ ] Flip **SPEC-210 + PLAN-210 → Done**; update the specs/plans indexes.
- [ ] Optional: note the `slider` control in `design-system.md`.
- [ ] **Review** with **frontend-reviewer** + **react-correctness-reviewer**; fix blockers.
- [ ] **PT-BR lesson** `docs/lessons/SPEC-210-aula.html` via **frontend-lesson-writer** (product-focused).

#### Deliverables
- Docs updated, spec closed, lesson published.

---

## 8. Risks

| Risk | Impact | Mitigation |
| ---- | ------ | ---------- |
| `GET /profile` `404` mishandled as an error | Medium | D3: treat 404 as `null` in the query fn → first-run empty state (mirrors `useSession` 401→null) |
| Enum ↔ label drift / missed value | Low/Med | Centralize the map (`labels.ts`), type it from the contract enums, and test it covers all values |
| Slider accessibility (keyboard, label) | Low | Native `<input type="range">` + explicit label + numeric value; verified by the frontend-reviewer a11y check |
| Losing user input on a `400`/error | Medium | Keep form state on error; surface the envelope message; never clear on failure (FR-2105) |
| Scope creep (onboarding, AI consumption) | Low | Hard rule: Perfil screen only; consumption is SPEC-213 |

---

## 9. Validation Checklist

### Functional Validation
- [ ] FR-2101…FR-2106 implemented; Epic 2 acceptance criteria satisfied.
- [ ] BR-2101 (identity from session, no `user_id`), BR-2102 (edge validation, server authoritative),
      BR-2103 (no money/AI — guards N/A), BR-2104 (types from the contract) respected.

### Technical Validation
- [ ] Consumes SPEC-101 only; **no `api/openapi.yaml` change**; drift check green.
- [ ] `404`→empty and `401`→login handled; no `user_id` on the wire; no float, no order affordance.

### Quality Validation
- [ ] Vitest/RTL + integration + gated E2E passing.
- [ ] a11y (slider keyboard + labels; AA contrast); reduced-motion respected.
- [ ] Reviewed by **frontend-reviewer** + **react-correctness-reviewer**; docs updated.

---

## 10. Definition of Done

- [ ] All phases complete; SPEC-210 acceptance criteria satisfied.
- [ ] Unit/component + integration + gated E2E green in the `web/` CI gate.
- [ ] **CHANGELOG** updated; **`api/openapi.yaml` unchanged** (asserted).
- [ ] **SPEC-210 + PLAN-210 flipped to Done**; specs/plans indexes updated.
- [ ] **PT-BR lesson** `docs/lessons/SPEC-210-aula.html` produced (via **frontend-lesson-writer**).
- [ ] Reviewed by the frontend review agents.
- [ ] Pull Request approved.

---

## 11. Deliverables

### Code Deliverables
- `lib/profile/` (hooks + labels), `components/ui/{segmented,chip-toggle,slider}.tsx`, the Perfil
  screen, and their tests.

### Documentation Deliverables
- CHANGELOG entry, PT-BR lesson, specs/plans index updates; optional `design-system.md` slider note.

---

## 12. Post-Implementation Tasks

### Future Improvements
- A first-run **onboarding** flow that includes profile setup (deferred by SPEC-210 §14).
- Surfacing the profile in the AI features (SPEC-213) and reflecting "profile incomplete" prompts elsewhere.

### Technical Debt
- If the horizon needs presets (5/10/20) later, extend the slider with snap points.
