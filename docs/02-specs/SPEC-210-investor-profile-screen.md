# SPEC-210 — Investor Profile Screen

## 1. Document Information

| Field        | Value                                                                 |
| ------------ | --------------------------------------------------------------------- |
| Feature Name | Investor Profile Screen                                               |
| Feature ID   | SPEC-210                                                              |
| Version      | 1.0.0                                                                 |
| Status       | Done                                                                  |
| Author       | Gabigol                                                              |
| Last Updated | 2026-07-01                                                            |
| Related PRD  | [Epic 2 / FR-003](../01-product/PRD.md)                               |
| Consumes     | [SPEC-101](SPEC-101-investor-profile.md) (backend) over the [OpenAPI contract](../../api/openapi.yaml); built on [SPEC-200](SPEC-200-app-foundation.md); stack [ADR-0006](../04-architecture/adr/ADR-0006-frontend-ui-stack-and-design-system.md) |

---

## 2. Overview

### Purpose

The **Perfil** screen — the first `SPEC-21x` feature — lets the investor **view and set their
risk profile, objectives, and investment horizon** (FR-003, Epic 2). It is the frontend face of
the SPEC-101 backend: it reads `GET /profile` and writes `PUT /profile` through the SPEC-200 typed
client, rendered with the Aurora design system inside the authenticated shell (the `/profile`
route, currently a stub).

### Business Value

Profile is the **personalization seam**: the risk/objectives/horizon it captures are what the
Insight Engine, Rebalancing Assistant, and Health Score reason over (per PRD). Both personas need
it — Carla to confirm alignment with her long-term goals, Rafael to tailor where to direct
contributions. Without a profile, the AI features have no per-user grounding.

### Success Criteria

- A returning user sees their saved profile prefilled; a first-run user sees an empty form with a
  clear call to set it up (from the SPEC-101 `404` when unset).
- The user can set risk profile (single), objectives (one or more of the four), and a horizon
  (1–50 years), and **save** it — persisted via `PUT /profile` and reflected on reload.
- Client validation mirrors the contract (≥1 objective, horizon 1–50) so an invalid submit is
  blocked at the edge; the server stays the authority.
- No new endpoint; **no `api/openapi.yaml` change**. Types come from the generated client.

---

## 3. Functional Requirements

### FR-2101 — Load & display the current profile

On entering `/profile`, fetch `GET /profile` via the typed client (TanStack Query). A `200`
prefills the form; a `404` (no profile yet) shows a first-run empty state; loading and error use
the shared shell patterns (SPEC-200).

#### Acceptance Criteria

- [ ] `200` → the form is prefilled with the saved `risk_profile`, `objectives`, `horizon_years`.
- [ ] `404` → a first-run empty state ("defina seu perfil") with an empty, ready-to-fill form; **not** an error.
- [ ] Loading → skeleton/placeholder; a transient error → retry (never a blank or a crash).

### FR-2102 — Set the risk profile

A single-select control for `Conservador | Moderado | Agressivo` (the `conservative | moderate |
aggressive` enum), using the design system's segmented control.

#### Acceptance Criteria

- [ ] Exactly one risk profile is selectable; the current value is visibly active.
- [ ] Required — save is blocked until one is chosen.

### FR-2103 — Set the objectives (one or more)

A multi-select of the four objectives (`Aposentadoria | Renda Passiva | Preservação de Patrimônio |
Crescimento de Longo Prazo`), using the objective chips.

#### Acceptance Criteria

- [ ] Zero-or-more toggle, but **≥1 required** to save (mirrors the contract `minItems: 1`).
- [ ] Duplicate selection is impossible (a set, not a list); order irrelevant.

### FR-2104 — Set the investment horizon

A **slider** for whole years, **1–50**, with the selected value shown numerically beside it.

#### Acceptance Criteria

- [ ] The slider selects a whole-year value in **1–50**; the current value is displayed as a number.
- [ ] The slider is **keyboard-accessible** (arrow keys) with an accessible label; out-of-range is
      impossible by construction.
- [ ] First-run defaults to a sensible starting point (e.g. 10), but the user must **explicitly save**
      — no value is silently written.

### FR-2105 — Save (create-or-update)

Submit builds the `ProfileRequest` and calls `PUT /profile`; on success it surfaces confirmation
and invalidates the profile query; on error it surfaces the `{"error":"..."}` envelope.

#### Acceptance Criteria

- [ ] The PUT body is exactly `{ risk_profile, objectives, horizon_years }` — **no `user_id`** (BR-2101).
- [ ] `200` → success feedback; the profile query is invalidated so the UI reflects the saved state.
- [ ] `400` → the server message is surfaced inline; the form keeps the user's input (no data loss).
- [ ] The save control is disabled while pending and while the form is invalid.

### FR-2106 — pt-BR labels ↔ contract enums

All labels are pt-BR; the API's snake_case enum values are mapped to display labels in one place.

#### Acceptance Criteria

- [ ] Enum ↔ label mapping is centralized (no scattered string literals) and covers all values.
- [ ] The wire always carries the enum values; the UI always shows the pt-BR labels.

---

## 4. User Flows

### Main Flow (set / update profile)

1. User opens **Perfil**. The app fetches `GET /profile`.
2. Saved profile → form prefilled; first-run (`404`) → empty form + setup prompt.
3. User picks a risk profile, toggles ≥1 objective, sets a horizon (1–50).
4. User saves → `PUT /profile` → success confirmation; the profile is now persisted.

### Alternative Flow (validation / error)

1. User tries to save with no objective (or an out-of-range horizon) → save is blocked with an inline message.
2. Server returns `400` → the envelope message is shown; input is preserved for correction.

---

## 5. Business Rules

### BR-2101 — Identity from the session, never the client
The authenticated user comes from the SPEC-003 session (BR-2001); the `PUT /profile` body carries
**no `user_id`** (the contract omits it). One user can only read/write their own profile.

### BR-2102 — Validate at the edge, mirroring the contract
Client validation enforces the `ProfileRequest` shape (risk-profile enum, ≥1 objective, horizon
1–50) **before** the PUT, for fast feedback — but the **server remains the authority** (a `400` is
always handled, never assumed-away).

### BR-2103 — No money, no AI output on this screen
Profile has no monetary values and produces no AI text, so the explainability (FR-013), non-advice
(FR-014), and money-format conventions **do not apply here** (mirrors SPEC-101 BR-1016). This
screen's job is to *feed* the AI features, not to render AI output.

### BR-2104 — Types from the generated contract
Request/response types come from the generated `lib/api/schema.ts` (`ProfileRequest` /
`ProfileResponse`) — **no hand-written DTOs**. Enum values are the API's; pt-BR is display-only.

---

## 6. Domain Model

Not applicable — the screen holds **no domain model of its own**. It renders the SPEC-101
`ProfileResponse` and submits a `ProfileRequest`, both typed from the contract. Local state is
form/UI state only.

---

## 7. API Contract

**Consumes the existing SPEC-101 endpoints — declares none, changes none.** No `api/openapi.yaml`
edit belongs to this spec.

- `GET /profile` → `200 ProfileResponse` `{ risk_profile, objectives, horizon_years, created_at, updated_at }` · `404` when unset.
- `PUT /profile` (body `ProfileRequest` `{ risk_profile, objectives, horizon_years }`) → `200` the saved profile · `400` on invalid input.

---

## 8. Data Model

Not applicable — no new tables, no client persistence. The profile lives in the SPEC-101 Postgres
table; the client caches the response ephemerally via TanStack Query.

---

## 9. Edge Cases

### First-run (no profile)
`GET /profile` `404` → empty form + "defina seu perfil" prompt, not an error state.

### Invalid input
No objective selected / horizon out of 1–50 → save blocked at the edge with an inline message.

### Server rejects the save (`400`)
Surface the `{"error":"..."}` message; preserve the user's input for correction.

### Transient failure loading/saving
A network/5xx blip on load → retry state (not a redirect); on save → error message + the form intact.

### Session expired mid-edit
A `401` on save → the SPEC-200 auth handling clears state and routes to login (input not silently lost beyond the session boundary).

---

## 10. Security Requirements

### Authentication
Screen is behind the `(app)` `RequireAuth` gate (SPEC-200); all calls carry the session cookie.

### Authorization
The backend scopes the profile to the session user; the client never sends or trusts a `user_id`.

### Data Protection
No secrets, no PII beyond the profile fields; nothing logged to the console.

---

## 11. Observability

### Metrics / Logs / Traces
No new client instrumentation in the MVP; the backend already traces `GET`/`PUT /profile`
(SPEC-004). Client error surfacing is UI-level (the shared error pattern).

---

## 12. Testing Strategy

### Unit / Component (Vitest + RTL)
- Enum ↔ pt-BR label mapping (all values).
- Validation: save blocked with no objective / out-of-range horizon; enabled when valid.
- First-run empty state on `404`; prefill on `200`.
- Submit builds the exact `ProfileRequest` (no `user_id`) and calls `PUT /profile`.

### Integration
- Load + save against a running backend (`GET` then `PUT` then re-`GET` reflects the change).

### E2E (Playwright)
- Fill the profile → save → reload → values persist. Gated to skip without a backend.

---

## 13. Definition of Done

- [ ] FR-2101…FR-2106 implemented; Epic 2 acceptance criteria satisfied.
- [ ] BR-2101…BR-2104 respected (identity from session, edge validation, no AI/money, generated types).
- [ ] Consumes SPEC-101 only; **no `api/openapi.yaml` change**.
- [ ] Vitest/RTL + integration + E2E (gated) green in the `web/` CI gate.
- [ ] Reviewed by **frontend-reviewer** + **react-correctness-reviewer**.
- [ ] CHANGELOG updated; SPEC-210 + PLAN-210 flipped to Done; indexes updated.
- [ ] PT-BR lesson `docs/lessons/SPEC-210-aula.html` via **frontend-lesson-writer**.

---

## 14. Resolved Decisions

1. **Horizon control → a slider** (1–50 whole years; value shown numerically; keyboard-accessible).
   See FR-2104.
2. **Save → an explicit "Salvar" button** (no auto-save — avoids partial writes; disabled while the
   form is invalid or a save is pending). See FR-2105.
3. **Onboarding → the Perfil screen only** for the MVP; a dedicated first-run onboarding flow is a
   later concern (not in this spec's scope).
