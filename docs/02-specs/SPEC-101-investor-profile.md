# Feature Specification (SPEC)

## 1. Document Information

| Field        | Value                                                  |
| ------------ | ------------------------------------------------------ |
| Feature Name | Investor Profile (risk profile, objectives, horizon)   |
| Feature ID   | SPEC-101 (feature)                                     |
| Related PRD  | [PRD.md](../01-product/PRD.md) — FR-003, Epic 2, §13 Dependencies |
| Related ADRs | [ADR-0002](../04-architecture/adr/ADR-0002-tech-stack-and-layering.md) (layering) |
| Version      | 0.1.0                                                  |
| Status       | Done                                                   |
| Plan         | [PLAN-101](../03-plans/PLAN-101-investor-profile.md)   |

---

## 2. Overview

### Purpose

Let an authenticated investor define and persist their **risk profile**, **objectives**, and
**investment horizon** (FR-003), and expose that profile — through a read port — to every
goal-aware analysis component that comes later (Insight Engine SPEC-104, Rebalancing
SPEC-105, Health Score SPEC-106). It is the first user-facing **feature** spec and the first
to consume the SPEC-003 identity-from-context seam.

### Business Value

YieldForge's whole premise is *goal-oriented* analysis (PRD §4): insights must consider who
the investor is, not just what they hold. The profile is the small, stable record that makes
"is my portfolio aligned with my objectives?" answerable. Without it, the AI features can
only give generic, ungrounded output.

### Scope

**In scope**

- A `profile` feature package: the `Profile` domain + value objects (`RiskProfile`,
  `Objective`, `Horizon`), the service, the repository port + Postgres adapter, and a
  consumer-facing read port for later specs.
- HTTP endpoints `GET /profile` and `PUT /profile` (create-or-update), behind the
  deny-by-default auth middleware, with per-user isolation from context.
- Migration `0004_profiles`; observability; tests; the working-agreement closeout.

**Out of scope**

- Consuming the profile in any analysis/AI feature (SPEC-104/105/106) — this spec stores it
  and exposes the read port only.
- Onboarding flow / UI, multi-profile per user, profile history/versioning.
- Any AI output (so the explainability/non-advice gates FR-013/FR-014 do not apply here; the
  profile is deterministic data those gates' features will later read).

---

## 3. Functional Requirements

### FR-1011 — Profile Domain & Value Objects

#### Acceptance Criteria

- [ ] `RiskProfile` is a closed enum: `conservative | moderate | aggressive`.
- [ ] `Objective` is a closed enum: `retirement | passive_income | wealth_preservation |
      long_term_growth`; a profile carries **one or more** (a deduplicated set).
- [ ] `Horizon` is a whole number of years within a sane range (1–50); parse-don't-validate
      constructors make an invalid instance unrepresentable, returning a sentinel error.

### FR-1012 — Set / Update Profile (`PUT /profile`)

#### Acceptance Criteria

- [ ] An authenticated user creates or replaces their profile in one idempotent upsert
      (full replace; one profile per user).
- [ ] An invalid risk profile, an unknown/empty objective set, or an out-of-range horizon
      returns `400` with the generic `{"error":"..."}` envelope; no partial write occurs.
- [ ] Duplicate objectives in the request are deduplicated; `updated_at` advances on update.

### FR-1013 — Get Profile (`GET /profile`)

#### Acceptance Criteria

- [ ] Returns the authenticated user's profile (`200`) or `404` with `{"error":"profile not
      set"}` when none exists.
- [ ] Never returns another user's profile.

### FR-1014 — Per-User Isolation (identity from context)

#### Acceptance Criteria

- [ ] The `user_id` is taken from the authenticated session context (`auth.UserID(ctx)`),
      **never** from the request payload or query.
- [ ] Every repository read/write is scoped `WHERE user_id = $1` with the context ID (BR-202/
      SPEC-003).

### FR-1015 — Consumer Read Port

#### Acceptance Criteria

- [ ] A small read port (e.g. `ProfileReader.GetProfile(ctx, userID)`) exposes the profile to
      later analysis specs without coupling them to the HTTP/DB layer.
- [ ] A `…NotFound` sentinel distinguishes "no profile yet" from a real error.

### FR-1016 — Persistence

#### Acceptance Criteria

- [ ] Migration `0004_profiles` (paired up/down, embedded, applied manually) creates a
      `profiles` table keyed by `user_id` (FK to `users`, `ON DELETE CASCADE`).
- [ ] The Postgres repository implements the port with idempotent upsert and a scoped read.

### FR-1017 — API Contract & Validation

#### Acceptance Criteria

- [ ] Request/response DTOs are separate from domain types; validation happens at the edge;
      errors use the `writeJSON` generic envelope.
- [ ] `/profile` requires authentication (not on the public allowlist).

### FR-1018 — Observability

#### Acceptance Criteria

- [ ] The endpoints are traced by the existing `otelhttp` route spans (`GET /profile`,
      `PUT /profile`); no PII beyond the already-logged `user_id`.

### FR-1019 — Documentation

#### Acceptance Criteria

- [ ] `README` (endpoints) + `CHANGELOG` updated; the PT-BR lesson
      `docs/lessons/SPEC-101-aula.html` produced on close.

---

## 4. User Flows

### Main Flow — Set then read

1. The authenticated user `PUT /profile` with risk profile, objectives, and horizon.
2. The service validates (value objects), upserts under the context `user_id`, returns the
   stored profile.
3. A later `GET /profile` returns it; analysis features read it via `ProfileReader`.

### Alternative Flow — Invalid input

1. `PUT /profile` with an unknown objective or horizon `0`.
2. Validation fails at the edge / in the constructor; `400 {"error":"..."}`; nothing stored.

---

## 5. Business Rules

- **BR-1011 — One profile per user.** `user_id` is the primary key; writes are an idempotent
  upsert (full replace).
- **BR-1012 — Identity from context.** `user_id` comes from the session, never a request
  field; per-user scoping is `WHERE user_id = $1` (SPEC-003 seam, no client-supplied id
  trusted).
- **BR-1013 — Parse, don't validate.** `RiskProfile`/`Objective`/`Horizon` validate in their
  constructors and return sentinel errors; an invalid value object cannot exist.
- **BR-1014 — At least one objective.** Objectives form a non-empty, deduplicated set stored
  in a stable order.
- **BR-1015 — Horizon bounds.** A whole number of years in 1–50 (the PRD's 5/10/20 are
  examples, not a fixed enum).
- **BR-1016 — Profile is a prerequisite for goal-aware AI.** SPEC-104/105/106 read it via the
  port; this spec emits **no AI output**, so FR-013/FR-014 do not apply directly.
- **BR-1017 — Conventions.** Errors wrapped `%w` + lowercase prefix; sentinels via
  `errors.Is`; `ctx` first; timestamps via the `Clock` port; reads named
  `GetProfileByUserID`; DTOs separate from domain; doc comments cite SPEC/BR.

---

## 6. Domain Model

### Entity: Profile

| Field        | Type           | Description                                  |
| ------------ | -------------- | -------------------------------------------- |
| user_id      | UUID           | Owner (PK, from context) — one profile per user |
| risk_profile | RiskProfile    | conservative / moderate / aggressive         |
| objectives   | []Objective    | one or more, deduplicated                    |
| horizon      | Horizon        | whole years, 1–50                            |
| created_at   | Timestamp      | UTC                                          |
| updated_at   | Timestamp      | UTC                                          |

Value objects (`RiskProfile`, `Objective`, `Horizon`) validate on construction (BR-1013).

---

## 7. API Specification

### PUT /profile

#### Request

```json
{ "risk_profile": "moderate", "objectives": ["retirement", "passive_income"], "horizon_years": 10 }
```

#### Response `200`

```json
{
  "risk_profile": "moderate",
  "objectives": ["retirement", "passive_income"],
  "horizon_years": 10,
  "created_at": "2026-06-24T12:00:00Z",
  "updated_at": "2026-06-24T12:00:00Z"
}
```

### GET /profile

`200` — the profile body above. `404 {"error":"profile not set"}` when none exists.
Both endpoints require a valid session (`401` otherwise, via the auth middleware).

---

## 8. Data Model

`migrations/0004_profiles.up.sql` / `.down.sql` (paired, embedded, applied manually):

```
profiles
  user_id       uuid        PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE
  risk_profile  text        NOT NULL
  objectives    jsonb       NOT NULL          -- JSON array of the closed enum; app enforces >= 1
  horizon_years integer     NOT NULL
  created_at    timestamptz NOT NULL DEFAULT now()
  updated_at    timestamptz NOT NULL DEFAULT now()
```

Objectives are stored as a `jsonb` array of the closed enum (D1): `json.Marshal` to write,
`[]byte` + `json.Unmarshal` to read — stdlib-clean with `database/sql` + pgx, still queryable
(`objectives @> '["retirement"]'`), and the app validates each value + the non-empty rule. No
surrogate key — `user_id` is the natural one-to-one key.

---

## 9. Edge Cases

| Scenario | Expected behavior |
| -------- | ----------------- |
| `GET /profile` with no profile set | `404 {"error":"profile not set"}`. |
| `PUT` invalid risk profile / unknown objective / empty objectives / horizon out of range | `400`, generic envelope, nothing written. |
| `PUT` twice | Second is an upsert; `updated_at` advances, `created_at` preserved. |
| Duplicate objectives in the request | Deduplicated before storing. |
| Unauthenticated request | `401` (deny-by-default middleware; `/profile` is not public). |
| Owning user deleted | Profile removed by `ON DELETE CASCADE`. |
| User A reads after User B writes | A sees only A's profile (per-user isolation). |

---

## 10. Security Considerations

- **AuthZ / isolation** — every operation is scoped to the context `user_id`; no
  client-supplied id is trusted (SPEC-003 BR). A user can never read or overwrite another's
  profile.
- **AuthN** — `/profile` requires a valid session; it is absent from the public allowlist.
- **Input validation** — enums + horizon bounds validated at the edge and in constructors;
  the generic error envelope avoids leaking internals.
- **No new secrets**; no money; no AI output (FR-013/FR-014 N/A).

---

## 11. Observability

- **Traces** — the existing `otelhttp` middleware yields route-named spans (`GET /profile`,
  `PUT /profile`); the DB upsert/read appear as child query spans (SPEC-004), arguments not
  recorded.
- **Logs** — request log carries `user_id` + `request_id` (already wired); no profile values
  beyond what the handler needs.
- **Metrics** — optional `profile.updates` counter by outcome; no PII.

---

## 12. Testing Strategy

### Unit Tests

- Value objects: `RiskProfile`/`Objective`/`Horizon` parsing (valid + invalid + dedupe).
- Service: set/get with a hand-written fake repo (upsert replaces; not-found sentinel).
- Handlers: DTO validation, identity-from-context (a body `user_id` is ignored), 400/404 paths.

### Integration Tests (gated)

- Real Postgres (`TEST_DATABASE_URL`, `-p 1`): upsert + read round-trip, the `0004` up/down,
  **per-user isolation** (two users), and cascade-on-user-delete.

### Quality gate

`task vet`, `task test:short`, gofmt-clean; integration serialized when a DB is present.

---

## 13. Definition of Done

- [ ] FR-1011…FR-1019 implemented; BR-1011…BR-1017 respected; acceptance criteria met.
- [ ] Hexagonal layering intact (domain pure; SQL in the adapter; HTTP in transport);
      identity from context; conventions (errors `%w`, `Clock`, `ctx` first, doc comments).
- [ ] `0004_profiles` up/down tested; per-user isolation proven against real Postgres.
- [ ] Unit + gated integration green; quality gate clean; hexagonal + go-correctness reviews pass.
- [ ] Working-agreement closeout: `CHANGELOG`, `README` (endpoints), SPEC + PLAN flipped to
      **Done**, indexes updated, PT-BR lesson produced.

---

## 14. Decisions (proposed — confirm in review)

| # | Decision | Recommendation |
| - | -------- | -------------- |
| D1 | Objectives storage | **Resolved: `jsonb` array** of the closed enum — clean with `database/sql` + pgx (`json.Marshal`/`Unmarshal`, no array-scan friction), still queryable; over `text[]` (driver friction) or a normalized child table (overkill for a fixed 4-value set). |
| D2 | `GET` when not set | **`404` `ErrProfileNotFound`** (clear "not set" signal) over a `200` empty body. |
| D3 | Primary key | **`user_id` as PK** (natural one-to-one) over a surrogate `id`. |
| D4 | Update semantics | **`PUT` full-replace upsert** (the profile is small) over `PATCH` partial updates. |
| D5 | Horizon shape | **Integer years, bounded 1–50** (PRD's 5/10/20 are examples) over a fixed enum. |

---

## 15. Open Questions (deferred, not blocking)

- Whether onboarding should require a profile before other features unlock (a product/UX
  decision; the read port already lets consumers treat "not set" gracefully).
- Profile history/versioning (out of scope for MVP; the table could gain an audit later).
- A future `PATCH` for partial edits if the profile grows beyond a few fields.
