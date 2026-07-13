# SPEC-211 — Portfolio Management Screens

## 1. Document Information

| Field        | Value                                                                 |
| ------------ | --------------------------------------------------------------------- |
| Feature Name | Portfolio Management Screens                                          |
| Feature ID   | SPEC-211                                                              |
| Version      | 0.1.0                                                                 |
| Status       | Draft                                                                 |
| Author       | Gabigol                                                              |
| Last Updated | 2026-07-02                                                            |
| Related PRD  | [Epic 1 / FR-001, FR-002](../01-product/PRD.md)                       |
| Consumes     | [SPEC-102](SPEC-102-portfolio-management.md) + [SPEC-109](SPEC-109-fixed-income-rate-indexers.md) (backend) over the [OpenAPI contract](../../api/openapi.yaml); built on [SPEC-200](SPEC-200-app-foundation.md); stack [ADR-0006](../04-architecture/adr/ADR-0006-frontend-ui-stack-and-design-system.md) |
| Blocked on   | ~~[SPEC-109](SPEC-109-fixed-income-rate-indexers.md)~~ — **unblocked** (SPEC-109 shipped Done 2026-07-02): the fixed-income rate indexer (% do CDI / IPCA+) and `GET /market/indicators` have landed; FR-2116/FR-2120 below can now be implemented against the real contract |

---

## 2. Overview

### Purpose

The **Carteira** screen — turning the `/portfolio` stub into the frontend face of the SPEC-102
backend. The investor **registers, edits, and removes** their FII and fixed-income holdings: the
system of record for what they own. It reads/writes the 8 SPEC-102 endpoints (`/holdings/fii` and
`/holdings/fixed-income`, each with list/create/update/delete) through the SPEC-200 typed client.

### Business Value

Every downstream feature — the Dashboard (SPEC-212), Insights/Rebalancing/Health (SPEC-213),
Projections (SPEC-214) — computes over these holdings. Without a working Carteira screen, nothing
else in the product has real data to reason about. Both personas need this directly: Rafael logs
his monthly FII purchases here; Carla maintains her fixed-income ladder alongside her FIIs.

### Success Criteria

- The investor can **add, edit, and delete** an FII holding (ticker, quantity, average price) and
  a fixed-income holding (name, institution, invested amount, annual rate, maturity, liquidity).
- Invalid input (zero/negative quantity, a past maturity date on a new at-maturity holding, a
  missing required field) is rejected **at the edge** with a clear message, mirroring the contract.
- Money/rate values are **entered and displayed in pt-BR** but never cross the wire as anything but
  integer centavos/basis points — including the new **input-parsing** direction (not just display).
- No new endpoint; **no `api/openapi.yaml` change**.

---

## 3. Functional Requirements

### FR-2111 — List FII holdings

On entering `/portfolio`, fetch `GET /holdings/fii`. An empty list shows a clear "nenhum FII
cadastrado" state with a call to add one; loading and transient errors use the shared shell
patterns (SPEC-200).

#### Acceptance Criteria

- [ ] Populated → a table (ticker, quantity, average price, actions) sorted by ticker.
- [ ] Empty → an empty state with an "Adicionar FII" call to action, not an error.
- [ ] Loading → skeleton; a transient error → retry (never a blank/crash).

### FR-2112 — Add an FII holding

A form (ticker, quantity, average price) that builds an `FIIHoldingRequest` and calls
`POST /holdings/fii`.

#### Acceptance Criteria

- [ ] Ticker required (normalized uppercase); quantity a positive whole number (**≥1**); average
      price a pt-BR currency input **parsed to integer centavos** (**≥0**) before the request.
- [ ] `400` (e.g. malformed ticker, zero/negative quantity) surfaces the server message inline;
      input is preserved for correction.
- [ ] On success the new holding appears in the list without a full page reload.

### FR-2113 — Edit an FII holding

Prefilled from the selected row; submits `PUT /holdings/fii/{id}` with the same shape/validation as add.

#### Acceptance Criteria

- [ ] The edit form prefills ticker, quantity, average price from the row.
- [ ] Same validation as FR-2112; a successful save updates the row in place.
- [ ] A `404` (already deleted / not owned — BR-2111) is treated as "this holding no longer
      exists": the list is refreshed, not a scary error.

### FR-2114 — Delete an FII holding

A confirmation step, then `DELETE /holdings/fii/{id}`.

#### Acceptance Criteria

- [ ] Delete requires an explicit confirmation (no accidental one-click destroy).
- [ ] On success the row is removed from the list; a `404` (already gone) is treated as success
      (the end state — "not in the list" — is already achieved).

### FR-2115 — List fixed-income holdings

Mirrors FR-2111 for `GET /holdings/fixed-income`: table (name, institution, invested amount,
annual rate, liquidity, maturity), empty/loading/error states.

#### Acceptance Criteria

- [ ] Populated → a table with all `FixedIncomeResponse` fields, human-readable liquidity label
      (Diária / No vencimento) and pt-BR-formatted maturity date (or "—" for daily).
- [ ] Empty → an empty state with an "Adicionar renda fixa" call to action.
- [ ] Loading → skeleton; a transient error → retry.

### FR-2116 — Add a fixed-income holding (with a rate indexer)

A form (name, institution, invested amount, **rate indexer**, rate value, liquidity type, maturity
date) that builds a `FixedIncomeRequest` (SPEC-109 shape) and calls `POST /holdings/fixed-income`.
The rate is entered the way the investor's bank statement actually shows it, not forced into a
flat annual rate.

#### Acceptance Criteria

- [ ] Name and institution required (non-empty); invested amount a pt-BR currency input **parsed
      to integer centavos** (**≥1**).
- [ ] **Indexer** is a single-select: **Prefixado** (flat rate) | **% do CDI** | **IPCA + spread**.
      The rate-value input's label and unit adapt to the selection (a flat annual % / a % of CDI /
      a spread in %), but in every case it's a pt-BR percentage input **parsed to integer basis
      points** (**≥0**, FR-2119) before the request — the wire field is the single
      `annual_rate_bps` whose *meaning* the `indexer_type` selection determines (SPEC-109).
- [ ] Liquidity is a single-select (**Diária | No vencimento**). Choosing **Diária** clears/disables
      the maturity field (sent as `null`); choosing **No vencimento** requires a maturity date.
- [ ] A maturity date **in the past is rejected at the edge** for a new at-maturity holding,
      mirroring the contract's create-time rule (PRD Epic 1).
- [ ] `400` surfaces the server message inline; input is preserved for correction.

### FR-2117 — Edit a fixed-income holding

Prefilled from the selected row; submits `PUT /holdings/fixed-income/{id}` with the same
shape/validation as add.

#### Acceptance Criteria

- [ ] The edit form prefills all fields, including the correct **indexer selection** (Prefixado /
      % do CDI / IPCA+), rate value, liquidity selection, and maturity.
- [ ] Same validation as FR-2116; a successful save updates the row in place.
- [ ] A `404` (already deleted / not owned) refreshes the list rather than erroring.

### FR-2118 — Delete a fixed-income holding

Mirrors FR-2114: confirm, then `DELETE /holdings/fixed-income/{id}`.

#### Acceptance Criteria

- [ ] Requires explicit confirmation.
- [ ] Success removes the row; a `404` (already gone) is treated as success.

### FR-2119 — pt-BR money/rate input parsing at the edge

A shared parsing helper converts a pt-BR-formatted currency/percentage input string to integer
centavos/basis points — the **input-side** counterpart to `lib/money.ts`'s display formatters.

#### Acceptance Criteria

- [ ] `"1.234,56"` → `123456` centavos; `"10,5"` → `1050` bps — exact, table-tested, including the
      round-trip with the existing `formatCentavos`/`formatBps`.
- [ ] Malformed input (non-numeric, empty when required) is rejected at the edge before any
      request is sent — never coerced to `0` silently.
- [ ] Parsing lives in **one place** (`lib/money.ts` or a sibling), never duplicated per form.

### FR-2120 — Live reference indicator, inline with the rate indexer

When **% do CDI** or **IPCA + spread** is selected (FR-2116/2117), show the corresponding live
reference value fetched from `GET /market/indicators` (SPEC-109) — the indicator's current value
and the date it was fetched — right next to the picker, and the **resolved effective annual rate**
once the row exists (`FixedIncomeResponse.effective_annual_rate_bps`) in the list/edit view.

#### Acceptance Criteria

- [ ] Selecting **% do CDI** shows e.g. *"CDI atual: 10,50% a.a. (ref. 01/07/2026)"*; **IPCA +**
      shows the equivalent for IPCA. **Prefixado** shows nothing extra (no indexer to reference).
- [ ] The fixed-income table shows the **resolved effective annual rate** for indexed holdings
      (e.g. *"120% do CDI · ≈12,60% a.a. (ref. 01/07/2026)"*), not just the raw stored percentage —
      read from the response, **never recomputed client-side** (BR-2117). The reference date comes
      from `GET /market/indicators`'s `reference_date` for that holding's indicator, shown
      **plainly, with no computed "stale" threshold** (MVP simplicity — SELIC/CDI update roughly
      daily, IPCA monthly, so a one-size threshold would misjudge one or the other; the date alone
      lets the investor judge freshness themselves).
- [ ] If the reference indicator is unavailable (loading, or the backend degrades per SPEC-109
      BR-1094), show a clear "indisponível no momento" state — never a blank, never a stale-looking
      silent zero.
- [ ] **Never-ingested cross-reference (SPEC-109 D3):** `FixedIncomeResponse.effective_annual_rate_bps`
      carries no flag distinguishing a genuine live resolution from a silent fallback to the raw
      stored rate (the backend degrades this way on purpose, BR-1094 — see PLAN-109 D3). For an
      indexed holding (`indexer_type` ≠ `prefixado`) whose reference indicator (CDI for
      `cdi_percentual`, IPCA for `ipca_spread`) is **entirely absent** from `GET /market/indicators`
      (never ingested, or a transient fetch error), show "sem referência disponível" instead of a
      reference date next to the effective rate — a plain presence/absence check, no staleness
      threshold. This is a client-side **presentation** decision over two already-server-sourced
      values; it does not compute or override the rate, so it does not violate BR-2117.
- [ ] This is **read-only reference data**; it never overrides or auto-fills the rate-value input
      the user is entering (the user still explicitly types "120", not the raw CDI number).

---

## 4. User Flows

### Main Flow (add a holding)

1. User opens **Carteira**; sees existing FII and fixed-income holdings (or empty states).
2. User opens "Adicionar FII" (or "Adicionar renda fixa"), fills the form, submits.
3. Valid → the holding is created and appears in its table. Invalid → inline error, input kept.

### Alternative Flow (edit / delete)

1. User selects an existing holding to edit → prefilled form → save → row updates in place.
2. User deletes a holding → confirms → the row is removed.
3. A stale edit/delete (the row was already removed elsewhere) resolves gracefully via a `404`
   → the list refreshes instead of showing an alarming error.

---

## 5. Business Rules

### BR-2111 — Identity & ownership from the session
Identity comes from the SPEC-003 session (BR-2001); every mutation is implicitly scoped to the
caller. A cross-user or already-deleted id resolves as **`404`, never a distinguishable
"forbidden"** (mirrors the backend's BR-1021) — the frontend treats a mutation `404` as "this
holding no longer exists for you" and refreshes, not as a hard failure.

### BR-2112 — Money & rates are integers, both directions
Display formatting (`formatCentavos`/`formatBps`) already converts integer → pt-BR string at the
render edge (SPEC-200 BR-2003). This spec adds the **inverse**: user-entered pt-BR strings are
**parsed to integers** before ever reaching a request body (FR-2119). No float, in either
direction, at any point.

### BR-2113 — Cost basis only, not market value
Holdings show what the user paid (`average_price_centavos` / `invested_amount_centavos`) — never a
computed **current value** or unrealized gain/loss. That computation is the **Dashboard's** job
(SPEC-212, reading holdings + market data together), mirroring the backend's BR-1024. The one
exception is the **effective annual rate** for indexed fixed-income holdings (FR-2120) — that is a
*rate*, not a valuation, is resolved server-side (SPEC-109), and is shown for transparency about
what the indexer means today, not as a computed net-worth figure.

### BR-2114 — Validate at the edge, mirroring the contract; server stays authoritative
Client validation enforces the `FIIHoldingRequest`/`FixedIncomeRequest` shapes (ticker present,
quantity ≥1, average price ≥0, invested amount ≥1, annual rate ≥0, maturity required iff
at-maturity and not in the past) for fast feedback, but a `400` from the server is always handled,
never assumed away.

### BR-2115 — No money-output guards, no AI output on this screen
This screen has no LLM-generated text, so FR-013/FR-014 do not apply (mirrors the backend's
BR-1025 and SPEC-210's BR-2103). Its job is to *record* facts the AI features later reason over.

### BR-2116 — Types from the generated contract
Request/response types come from `lib/api/schema.ts` (`FIIHoldingRequest/Response`,
`FixedIncomeRequest/Response`, `MarketIndicatorResponse`) — no hand-written DTOs.

### BR-2117 — Reference rates are read from the server, never computed or hardcoded client-side
The live SELIC/CDI/IPCA values (FR-2120) and the resolved `effective_annual_rate_bps` are always
read from the backend (SPEC-109) — the client never recomputes the indexer math, caches a value
past its fetch, or hardcodes a "current" rate as a fallback.

---

## 6. Domain Model

Not applicable — the screen holds no domain model of its own. It renders the SPEC-102/SPEC-109
`FIIHoldingResponse` / `FixedIncomeResponse` / `MarketIndicatorResponse` and submits their
`*Request` counterparts, all typed from the contract. Local state is form/UI state only.

---

## 7. API Contract

**Consumes the existing SPEC-102 + SPEC-109 endpoints — declares none, changes none.** No
`api/openapi.yaml` edit belongs to this spec (SPEC-109 owns that contract change).

- `GET /holdings/fii` → `200 FIIHoldingResponse[]`
- `POST /holdings/fii` (body `FIIHoldingRequest` `{ticker, quantity, average_price_centavos}`) → `201`
- `PUT /holdings/fii/{id}` (same body) → `200` · `404` if not owned/found
- `DELETE /holdings/fii/{id}` → `204` · `404` if not owned/found
- `GET /holdings/fixed-income` → `200 FixedIncomeResponse[]` (SPEC-109 shape: includes
  `indexer_type` and the computed, read-only `effective_annual_rate_bps`)
- `POST /holdings/fixed-income` (body `FixedIncomeRequest`
  `{name, institution, invested_amount_centavos, annual_rate_bps, indexer_type, maturity_date, liquidity_type}`) → `201`
- `PUT /holdings/fixed-income/{id}` (same body) → `200` · `404` if not owned/found
- `DELETE /holdings/fixed-income/{id}` → `204` · `404` if not owned/found
- `GET /market/indicators` (SPEC-109) → `200 MarketIndicatorResponse[]`
  `{indicator, value_bps, reference_date}` for SELIC/CDI/IPCA — read for FR-2120.

---

## 8. Data Model

Not applicable — no new tables, no client persistence. Holdings live in the SPEC-102 Postgres
tables; the client caches list responses ephemerally via TanStack Query, invalidated on mutation.

---

## 9. Edge Cases

### Empty portfolio
Both FII and fixed-income sections show their own empty state with a clear call to action —
not a combined blank page.

### Invalid input
Zero/negative quantity, malformed ticker, a past maturity date on a new at-maturity holding, or a
missing required field → blocked/rejected at the edge with an inline message; on a `400` from the
server, the same message is surfaced and input is preserved.

### Live reference indicator unavailable
`GET /market/indicators` is loading, errors, or SPEC-109 degrades (no ingested CDI/IPCA yet) → the
inline reference (FR-2120) shows a clear "indisponível" state. The form is **not** blocked — the
user can still enter and save "120% do CDI" (the raw rate value) even if today's live CDI can't be
shown right now; only the reference *display*, not the save path, is affected.

### An existing indexed holding's effective rate may be a silent fallback
The same indicator absence that triggers the above also means any already-saved `cdi_percentual`/
`ipca_spread` holding's `effective_annual_rate_bps` in the table could be a silent fallback to the
raw stored rate (SPEC-109 D3), not a fresh resolution. MVP treatment is deliberately simple (no
computed "stale" flag or per-indicator threshold): the table always shows the reference date
alongside the resolved rate when the indicator is present in `GET /market/indicators`, and "sem
referência disponível" when it's entirely absent — the investor judges freshness from the date
itself, the same way they would reading a bank statement.

### Editing/deleting a holding that's already gone
A `404` on `PUT`/`DELETE` (deleted in another tab, or an id that was never the caller's) is treated
as **"already achieved the desired end state"** — refresh the list silently rather than alarm the user.

### Switching liquidity type mid-edit
Toggling **No vencimento → Diária** clears the maturity date (sent as `null`); toggling back
requires re-entering a valid, non-past date before the form is valid.

### Transient failure loading/saving
A network/5xx blip on list load → retry state; on save → error message + the form intact (no data loss).

### Session expired mid-edit
A `401` on any call → the SPEC-200 auth handling clears state and routes to login.

---

## 10. Security Requirements

### Authentication
Screen is behind the `(app)` `RequireAuth` gate (SPEC-200); all calls carry the session cookie.

### Authorization
The backend scopes every read/write to the session user (double-scoped update/delete); the client
never sends or trusts a `user_id`, and never distinguishes "not mine" from "doesn't exist" (both `404`).

### Data Protection
No secrets; holdings data is the user's own financial records — nothing logged to the console.

---

## 11. Observability

### Metrics / Logs / Traces
No new client instrumentation in the MVP; the backend already traces the holdings endpoints
(SPEC-004). Client error surfacing is UI-level (the shared error pattern).

---

## 12. Testing Strategy

### Unit / Component (Vitest + RTL)
- Money/rate **input parsing** (FR-2119): pt-BR string → integer centavos/bps, including
  malformed-input rejection and the round-trip with `formatCentavos`/`formatBps`.
- Validation gating for both forms (FII: ticker/quantity/price; FI: required fields, liquidity ↔
  maturity interaction, past-date rejection).
- Empty vs. populated list rendering for both sections.
- Submit builds the exact request body for each resource (no `user_id`); a `404` on edit/delete is
  handled as list-refresh / already-removed, not an error toast.
- Indexer selection (FR-2116/2117): the rate-value label/unit adapts per indexer; edit prefills the
  correct indexer + value; the live-reference display (FR-2120) renders the fetched value/date,
  and degrades to "indisponível" without blocking the save path when unavailable.
- Effective-rate reference date (FR-2120): given a `cdi_percentual` holding, the table shows the
  `reference_date` next to the resolved rate when CDI is present in the `GET /market/indicators`
  fixture, and "sem referência disponível" when CDI is absent — table-tested against both
  indicators (CDI/IPCA) and `prefixado` (never shows a reference, no indexer to reference).

### Integration
- Against a running backend: create → list reflects it → edit → list reflects it → delete → list
  no longer has it, for both FII and fixed-income.

### End-to-End (Playwright)
- Add an FII holding and a fixed-income holding, see both appear; delete one, confirm it's gone.
  Gated to skip without a backend.

---

## 13. Definition of Done

- [ ] FR-2111…FR-2120 implemented; Epic 1 acceptance criteria satisfied for both holding types.
- [ ] BR-2111…BR-2117 respected (ownership-as-404, money bidirectional-integer, cost-basis-only,
      edge validation, no AI guards needed, generated types, reference rates never client-computed).
- [ ] Consumes SPEC-102 + SPEC-109 only; **no `api/openapi.yaml` change**.
- [ ] Vitest/RTL + integration + gated E2E green in the `web/` CI gate.
- [ ] Reviewed by **frontend-reviewer** + **react-correctness-reviewer**.
- [ ] CHANGELOG updated; SPEC-211 + PLAN-211 flipped to Done; indexes updated.
- [ ] PT-BR lesson `docs/lessons/SPEC-211-aula.html` via **frontend-lesson-writer**.

---

## 14. Open Questions

1. **Add/edit UI pattern** — a modal dialog vs. an inline expanding row vs. a separate route per
   form? A modal keeps the table clean and is the most common pattern for this shape of CRUD;
   needs your call before PLAN-211.
2. **Delete confirmation UX** — a confirm dialog vs. an inline "tem certeza?" replacing the action,
   vs. an undo-toast? A simple confirm dialog is the safest default for the MVP.
3. **Ticker input** — free-text (uppercase-normalized, no live validation against a known-tickers
   list) vs. an autocomplete against the marketdata universe? The backend does not validate ticker
   against a known list at creation (SPEC-102), so free-text is the natural MVP default;
   autocomplete would need a new/reused lookup endpoint and is likely out of scope here.
