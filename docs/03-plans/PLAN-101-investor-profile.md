# Implementation Plan

## 1. Document Information

| Field           | Value                                                        |
| --------------- | ------------------------------------------------------------ |
| Plan Name       | Investor Profile (risk profile, objectives, horizon)         |
| Related Feature | SPEC-101 — the first user-facing feature; first consumer of the auth identity seam |
| Related Spec    | [SPEC-101](../02-specs/SPEC-101-investor-profile.md)         |
| Version         | 0.1.0                                                        |
| Status          | Done (decisions D1–D5 resolved, D1 = jsonb)                  |
| Author          | Gabigol                                                      |
| Last Updated    | 2026-06-24                                                   |

---

## 2. Objective

### Goal

Persist an authenticated investor's risk profile, objectives, and horizon (FR-003), and
expose them through a read port for the goal-aware analysis features that follow.

### Expected Outcome

`PUT /profile` upserts the caller's profile and `GET /profile` returns it (or `404`), both
per-user-isolated from the session context. A `ProfileReader` port lets SPEC-104/105/106
read the profile without touching HTTP/DB. Nothing analyses the profile yet — this spec
lands the record + the port.

---

## 3. Scope

### Included

- `internal/profile`: domain (`Profile`) + value objects (`RiskProfile`, `Objective`,
  `Horizon`), sentinels, the service, the repository port, and the consumer `ProfileReader`
  port.
- `internal/profile/postgres` repository; migration `0004_profiles` (paired up/down,
  embedded, manual).
- `GET /profile` + `PUT /profile` handlers (DTOs, edge validation), wired into the router
  behind the existing deny-by-default auth middleware; `cmd/api` wiring.
- Observability (route spans already provided by `otelhttp`); tests; closeout.

### Excluded (SPEC-101 §2)

- Consuming the profile in any analysis/AI feature (SPEC-104/105/106).
- Onboarding/UI, multi-profile, profile history/versioning, `PATCH` partial edits.
- Any AI output (FR-013/FR-014 N/A) and any money (no `int64`/bps here).

---

## 4. Dependencies

### Technical Dependencies

- SPEC-003 (auth): `auth.UserID(ctx)`, the deny-by-default middleware, the `users` table
  (FK target). SPEC-002 (DB, migration runner). SPEC-004 (`otelhttp` route spans). The
  `Clock` port for timestamps. The `transport/http` router `Deps` + `writeJSON` envelope.

### New Dependencies

- **None.** Pure stdlib + the existing `database/sql` + pgx stack.

### Blocking Decisions (SPEC-101 §14 — all resolved)

- **D2** `404` when unset · **D3** `user_id` as PK · **D4** `PUT` full-replace upsert · **D5**
  horizon int 1–50 — all low-risk.
- **D1 (resolved) — objectives storage = `jsonb` array.** Stores the closed-enum objectives as
  a JSON array: `json.Marshal` to write, `[]byte` + `json.Unmarshal` to read — stdlib-clean
  with `database/sql` + pgx (no array-scan friction), still queryable
  (`objectives @> '["retirement"]'`). Chosen over `text[]` (driver friction) and a normalized
  child table (overkill for a fixed 4-value set).

---

## 5. Architecture Impact

### Existing Components Affected

| Component | Impact |
| --------- | ------ |
| `internal/transport/http` | New `profile.go` handlers + `Deps.Profile`; register `GET/PUT /profile` (auto-protected — not on the public allowlist) |
| `cmd/api` | Wire the profile service (repo + clock) into `Deps` |
| `migrations/` | New `0004_profiles` up/down (manual) |
| `README` | Endpoints table gains `/profile` |

### New Components

| Component | Purpose |
| --------- | ------- |
| `internal/profile` | Domain, value objects, sentinels, service, repository + `ProfileReader` ports |
| `internal/profile/postgres` | `ProfileRepository` adapter (upsert + scoped read) |

---

## 6. Implementation Strategy

### Approach

Bottom-up and layered. The domain core stays pure (no SQL/HTTP); the Postgres adapter and
the HTTP handlers sit at the edges. Security is the throughline: **identity comes from
`auth.UserID(ctx)`, never the request body**, and every query is `WHERE user_id = $1`
(SPEC-003 seam). Conventions enforced throughout: `RiskProfile`/`Objective`/`Horizon` are
parse-don't-validate value objects; errors wrap `%w` with sentinels; `Clock` over
`time.Now()`; `ctx` first; reads named `GetProfileByUserID`; DTOs separate from domain;
`testify/require` + hand-written fakes; test files mirror their source; doc comments cite
SPEC/BR. No money and no AI output in this spec.

### Rollout Method

Incremental and additive. New endpoints behind existing auth; migration `0004` applied
manually with a tested down. No existing behavior changes.

### Rollback Strategy

Drop the endpoints / revert wiring; `0004` down removes `profiles`. No data migration.

---

## 7. Implementation Phases

### Phase 1 — Domain & Ports

#### Tasks

- [ ] Value objects with constructors: `RiskProfile` (conservative|moderate|aggressive),
      `Objective` (4-value closed enum), `Horizon` (int years, 1–50) — each returns a
      sentinel error on invalid input.
- [ ] `Profile` entity (UserID, RiskProfile, Objectives set, Horizon, timestamps);
      objective-set dedupe + non-empty rule (BR-1014); sentinels `ErrProfileNotFound`,
      `ErrInvalidRiskProfile`, `ErrInvalidObjective`, `ErrInvalidHorizon`,
      `ErrNoObjectives`.
- [ ] Ports: `ProfileRepository` (`UpsertProfile`, `GetProfileByUserID`) and the consumer
      `ProfileReader` (`GetProfile(ctx, userID)`).

#### Deliverables

- Compiling pure core; value-object + dedupe unit tests green.

---

### Phase 2 — Persistence (migration + repository)

#### Tasks

- [ ] `migrations/0004_profiles.up.sql`/`.down.sql`: `profiles` keyed by `user_id`
      (FK → `users` `ON DELETE CASCADE`), `risk_profile text`, `objectives jsonb`,
      `horizon_years integer`, timestamps; tested down.
- [ ] `internal/profile/postgres`: idempotent `UpsertProfile` (`INSERT … ON CONFLICT
      (user_id) DO UPDATE`, preserving `created_at`) and `GetProfileByUserID` (→
      `ErrProfileNotFound`); objectives via `json.Marshal`/`Unmarshal` (jsonb); compile-time
      port assertion.

#### Deliverables

- Persistence with idempotent upsert + scoped read; gated integration scaffold ready.

---

### Phase 3 — Application (service)

#### Tasks

- [ ] `profile.Service`: `SetProfile(ctx, userID, input)` (validate → dedupe → upsert) and
      `GetProfile(ctx, userID)` (satisfies `ProfileReader`); identity passed in from the
      handler's context, never trusted from input.
- [ ] Hand-written fake repo for unit tests.

#### Deliverables

- Service with set/get; service unit tests (replace-on-upsert, not-found, validation) green.

---

### Phase 4 — API (transport)

#### Tasks

- [ ] `internal/transport/http/profile.go`: `GET /profile` (200/404) and `PUT /profile`
      (200, 400 on invalid), request/response DTOs separate from domain, `writeJSON`
      envelope; **`user_id` from `auth.UserID(ctx)` — a body `user_id` is ignored**.
- [ ] Add `Deps.Profile`; register routes (auto-protected); wire in `cmd/api`.

#### Deliverables

- Working endpoints behind auth; handler unit tests (DTO validation, identity-from-context,
  401 when unauthenticated) green.

---

### Phase 5 — Observability

#### Tasks

- [ ] Confirm the `otelhttp` route spans name `GET /profile` / `PUT /profile`; DB calls
      appear as child query spans (no argument values). Optional `profile.updates` counter
      by outcome. No PII beyond `user_id`.

#### Deliverables

- Endpoints traced via the existing seam; no new telemetry config.

---

### Phase 6 — Testing

#### Unit Tests

- [ ] Value objects (valid/invalid/dedupe); service (set/get/replace/not-found); handlers
      (validation, identity-from-context, 400/404/401).

#### Integration Tests (gated)

- [ ] Real Postgres (`TEST_DATABASE_URL`, `-p 1`): upsert + read round-trip, `0004` up/down,
      **per-user isolation** (user A never sees user B), and cascade-on-user-delete.

#### Deliverables

- Full suite green; quality gate clean.

---

### Phase 7 — Documentation & Lesson

#### Tasks

- [ ] `README` endpoints table + `CHANGELOG` entry.
- [ ] Flip SPEC-101 + PLAN-101 to **Done**; update indexes; `CLAUDE.md` status.
- [ ] lesson-writer → `docs/lessons/SPEC-101-aula.html` (PT-BR).

#### Deliverables

- Working-agreement closeout complete.

---

## 8. Risks

| Risk | Impact | Mitigation |
| ---- | ------ | ---------- |
| Objectives serialization round-trip (jsonb ↔ []Objective) | Low | Marshal/Unmarshal the closed-enum strings; validate each on read; covered by the integration round-trip test. |
| Identity leak — a handler trusts a body `user_id` | High | Always `auth.UserID(ctx)`; an explicit test asserts a body `user_id` is ignored (SPEC-003 BR). |
| Migration `0004` FK/cascade wrong | Medium | FK to `users` `ON DELETE CASCADE`; integration test proves cascade + a tested down. |
| Over-/under-validation of objectives (empty / unknown / dupes) | Low | Closed enum + non-empty + dedupe rules with a unit corpus. |

---

## 9. Validation Checklist

### Functional Validation

- [ ] FR-1011…FR-1019 implemented; BR-1011…BR-1017 respected; acceptance criteria met.
- [ ] Per-user isolation proven; not-set returns `404`; upsert replaces.

### Technical Validation

- [ ] Hexagonal layering (domain pure; SQL in adapter; HTTP in transport); identity from
      context; conventions (`%w`, `Clock`, `ctx` first, doc comments, test-file naming).

### Quality Validation

- [ ] `task vet` + `task test:short` green; gated integration green with a DB; gofmt clean;
      hexagonal-reviewer + go-correctness-reviewer pass.

---

## 10. Definition of Done

- [ ] All phases complete; acceptance criteria satisfied; quality gate clean.
- [ ] `0004_profiles` up/down + per-user isolation proven against a real Postgres at least once.
- [ ] CHANGELOG + README updated; SPEC-101 + PLAN-101 flipped to **Done**; indexes +
      `CLAUDE.md` status updated; PT-BR lesson produced.
- [ ] PR opened and `/pr-review` run as the pre-merge gate.

---

## 11. Deliverables

### Code Deliverables

- `internal/profile` (domain + value objects + service + ports), `internal/profile/postgres`,
  `internal/transport/http/profile.go`, `cmd/api` wiring.

### Infrastructure Deliverables

- Migration `0004_profiles` (up/down).

### Documentation Deliverables

- README endpoints, CHANGELOG entry, `SPEC-101-aula.html`.

---

## 12. Post-Implementation Tasks

### Monitoring

- Watch `profile.updates` (if added) once analysis features consume the profile.

### Future Improvements

- `PATCH` partial edits; profile history/versioning; gating other features on a set profile
  (a product decision).

### Technical Debt

- Revisit objectives storage if the set grows or needs relational querying.
