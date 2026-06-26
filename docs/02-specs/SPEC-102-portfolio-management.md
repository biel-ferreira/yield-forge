# Feature Specification (SPEC)

## 1. Document Information

| Field        | Value                                                  |
| ------------ | ------------------------------------------------------ |
| Feature Name | Portfolio Management (FII + Fixed Income holdings)     |
| Feature ID   | SPEC-102 (feature)                                     |
| Related PRD  | [PRD.md](../01-product/PRD.md) — FR-001, FR-002, Epic 1, §13 Dependencies |
| Related ADRs | [ADR-0002](../04-architecture/adr/ADR-0002-tech-stack-and-layering.md) (layering) |
| Version      | 0.1.0                                                  |
| Status       | Done                                                   |
| Plan         | [PLAN-102](../03-plans/PLAN-102-portfolio-management.md) |

---

## 2. Overview

### Purpose

Let an authenticated investor **register, edit, and delete** their FII holdings (FR-001) and
Fixed Income holdings (FR-002), persisted and scoped per user, and expose them through a read
port for the dashboard (SPEC-103), the Fact Builder (SPEC-104), and projections (SPEC-107).
It is the **system of record for what the user owns** — the deterministic substrate every
downstream computation reconciles against.

### Business Value

"A single, accurate view of everything I own" (Epic 1) is the foundation of the product: the
portfolio summary, allocation breakdown, health score, and every AI insight are only as
correct as the holdings beneath them. This is also the **first feature that handles money** —
so it sets the money discipline (int64 centavos, integer basis points, never float, on the
wire too).

### Scope

**In scope**

- A `portfolio` feature package: the `FIIHolding` and `FixedIncomeHolding` domain + value
  objects (`Quantity`, `LiquidityType`, money as centavos), the service, the repository
  port, and a consumer `Reader` port.
- HTTP CRUD for both holding types, behind the deny-by-default auth middleware, per-user
  isolated and ownership-enforced.
- Migration `0005_holdings`; observability; tests; the working-agreement closeout.

**Out of scope**

- Computing **current value**, allocation, passive income, or growth (that is the dashboard
  SPEC-103 / projections SPEC-107 — they read holdings via the port). This spec stores the
  *cost-basis* facts (`average_price`, `invested_amount`), not market-derived values.
- Stocks/ETFs as managed asset classes (PRD §scope — FII + FI only for MVP).
- Transactions/lots history, corporate actions, brokerage import (manual entry, PRD A3).
- Any AI output (FR-013/FR-014 do not apply) — but money rules **do** apply throughout.

---

## 3. Functional Requirements

### FR-1021 — Holding Domain & Value Objects

#### Acceptance Criteria

- [ ] `FIIHolding` carries `ticker`, `quantity`, `average_price` (centavos); a B3 `Ticker`
      and a positive whole-number `Quantity` are parse-don't-validate value objects.
- [ ] `FixedIncomeHolding` carries `name`, `institution`, `invested_amount` (centavos),
      `annual_rate` (bps), `maturity_date`, and a `LiquidityType` closed enum.
- [ ] All money is `int64` **centavos** and rates integer **basis points** — never `float64`,
      including across the JSON boundary (FR-1029).

### FR-1022 — FII Holding CRUD

#### Acceptance Criteria

- [ ] An authenticated user can **create, list, update, and delete** their FII holdings.
- [ ] Create/update validate the ticker (B3 format), a **positive** quantity, and a
      non-negative average price; invalid input → `400` with the generic envelope, no write.
- [ ] List returns only the caller's holdings; update/delete act only on a holding the caller
      owns (else `404`).

### FR-1023 — Fixed Income Holding CRUD

#### Acceptance Criteria

- [ ] An authenticated user can **create, list, update, and delete** their fixed-income
      holdings.
- [ ] Validation: non-empty `name`/`institution`, positive `invested_amount`, non-negative
      `annual_rate`, a valid `liquidity_type`, and — for an at-maturity instrument — a
      `maturity_date` that is **not in the past at creation** (PRD Epic 1).
- [ ] List/update/delete are caller-scoped and ownership-enforced (else `404`).

### FR-1024 — Per-User Isolation & Ownership

#### Acceptance Criteria

- [ ] `user_id` comes from the authenticated context (`auth.UserID(ctx)`), **never** from
      the request payload or path.
- [ ] Every read is scoped `WHERE user_id = $1`; every update/delete is
      `WHERE id = $1 AND user_id = $2`, so one user can never read, edit, or delete another's
      holding (a mismatched owner is indistinguishable from "not found").

### FR-1025 — Consumer Read Port

#### Acceptance Criteria

- [ ] A `Reader` port exposes the caller's holdings (FII + fixed income) to later specs
      (dashboard/facts/projections) without coupling them to HTTP/DB.
- [ ] It returns the cost-basis facts only; market-derived values are computed downstream.

### FR-1026 — Persistence

#### Acceptance Criteria

- [ ] Migration `0005_holdings` (paired up/down, embedded, manual) creates `fii_holdings`
      and `fixed_income_holdings`, each keyed by a UUID `id`, with `user_id` FK → `users`
      `ON DELETE CASCADE` and an index on `user_id`; money columns are `bigint` centavos,
      rate columns `integer` bps — no floats.
- [ ] The Postgres repository implements the port with per-user-scoped, ownership-checked SQL.

### FR-1027 — API Contract & Money on the Wire

#### Acceptance Criteria

- [ ] Request/response DTOs are separate from domain; money fields cross JSON as **integer
      centavos** (`average_price_centavos`, `invested_amount_centavos`) and rates as integer
      bps (`annual_rate_bps`) — never a float.
- [ ] All `/holdings/*` routes require authentication (not on the public allowlist); errors
      use the `{"error":"..."}` envelope.

### FR-1028 — Observability

#### Acceptance Criteria

- [ ] Endpoints inherit the `otelhttp` route spans (`POST /holdings/fii`, …); DB writes/reads
      appear as child query spans; no PII beyond `user_id`.

### FR-1029 — Documentation

#### Acceptance Criteria

- [ ] `README` (endpoints) + `CHANGELOG` updated; the PT-BR lesson
      `docs/lessons/SPEC-102-aula.html` produced on close.

---

## 4. User Flows

### Main Flow — register and view

1. The user `POST /holdings/fii` (or `/holdings/fixed-income`) with the holding fields.
2. The service validates via value objects, inserts under the context `user_id`, returns the
   stored holding (money as integer centavos).
3. `GET /holdings/fii` lists the caller's holdings; downstream specs read them via `Reader`.

### Alternative Flow — edit/delete someone else's holding

1. User B issues `PUT/DELETE /holdings/fii/{id}` for an id owned by user A.
2. The scoped `WHERE id = $1 AND user_id = $2` matches no row → `404 {"error":"holding not found"}`.

---

## 5. Business Rules

- **BR-1021 — Identity & ownership from context.** `user_id` is from the session, never a
  request field; reads are `WHERE user_id = $1` and mutations `WHERE id = $1 AND user_id = $2`
  (no client-supplied identity; cross-user access is "not found", not "forbidden").
- **BR-1022 — Money is never `float64`.** `average_price`/`invested_amount` are `int64`
  centavos, `annual_rate` integer bps; all parsing/rounding via `internal/platform/money`
  (half-up). The ban extends to the JSON wire (FR-1027).
- **BR-1023 — Parse, don't validate.** `Ticker`, `Quantity`, `LiquidityType` validate in
  their constructors and return sentinels; an invalid holding is unrepresentable.
- **BR-1024 — Cost basis, not market value.** Holdings store what the user paid
  (`average_price`, `invested_amount`); current value, yield, and allocation are computed
  downstream from market data (SPEC-103/107) — this spec never invents market numbers.
- **BR-1025 — No AI output.** No LLM here, so FR-013/FR-014 do not apply; the holdings are
  deterministic facts the Fact Builder (SPEC-104) later hands the Insighter.
- **BR-1026 — Conventions.** Errors `%w` + lowercase prefix; sentinels via `errors.Is`;
  `Clock` over `time.Now()`; `ctx` first; reads named `Get*By*`/`List*By*`; DTOs separate
  from domain; doc comments cite SPEC/BR; no package-name stutter (`portfolio.Repository`).

---

## 6. Domain Model

### Entity: FIIHolding

| Field                | Type      | Description                          |
| -------------------- | --------- | ------------------------------------ |
| id                   | UUID      | Surrogate key                        |
| user_id              | UUID      | Owner (from context)                 |
| ticker               | Ticker    | B3 FII ticker (value object)         |
| quantity             | Quantity  | Whole number of cotas (> 0)          |
| average_price_centavos | int64   | Average purchase price, minor units  |
| created_at/updated_at | Timestamp | UTC                                 |

### Entity: FixedIncomeHolding

| Field                   | Type          | Description                              |
| ----------------------- | ------------- | ---------------------------------------- |
| id                      | UUID          | Surrogate key                            |
| user_id                 | UUID          | Owner (from context)                     |
| name                    | string        | Instrument name (non-empty)              |
| institution             | string        | Issuer/broker (non-empty)                |
| invested_amount_centavos | int64        | Amount invested, minor units             |
| annual_rate_bps         | int           | Contracted annual rate, basis points     |
| maturity_date           | date?         | Required for at-maturity; null for daily |
| liquidity_type          | LiquidityType | daily \| at_maturity                     |
| created_at/updated_at   | Timestamp     | UTC                                      |

Value objects: `Ticker` (**reused from `marketdata`** — one B3-ticker definition, D1),
`Quantity` (positive integer), `LiquidityType` (closed enum) — all parse-don't-validate.

---

## 7. Ports

```go
// portfolio.Repository — persistence for both holding types, per-user scoped and
// ownership-checked. (Reads Get*/List*; writes Create/Update/Delete.)
type Repository interface {
    CreateFIIHolding(ctx, h FIIHolding) (FIIHolding, error)
    ListFIIHoldingsByUserID(ctx, userID string) ([]FIIHolding, error)
    UpdateFIIHolding(ctx, h FIIHolding) (FIIHolding, error)        // scoped by id + user_id; ErrHoldingNotFound
    DeleteFIIHolding(ctx, userID, id string) error                 // scoped; ErrHoldingNotFound
    // … the same four for FixedIncomeHolding …
}

// portfolio.Reader — the consumer seam SPEC-103/104/107 read through.
type Reader interface {
    ListHoldings(ctx, userID string) (Holdings, error) // { FII []FIIHolding; FixedIncome []FixedIncomeHolding }
}
```

> Sentinels: `ErrHoldingNotFound` (absent or not owned), plus the validation sentinels on the
> value objects. Identity is never part of a method's *input data* — only the scoping `userID`.

---

## 8. Data Model

`migrations/0005_holdings.up.sql` / `.down.sql` (paired, embedded, manual):

```
fii_holdings
  id                     uuid        PRIMARY KEY DEFAULT gen_random_uuid()
  user_id                uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE
  ticker                 text        NOT NULL
  quantity               integer     NOT NULL          -- > 0 (app-enforced)
  average_price_centavos bigint      NOT NULL
  created_at/updated_at  timestamptz NOT NULL DEFAULT now()
  INDEX (user_id)

fixed_income_holdings
  id                       uuid        PRIMARY KEY DEFAULT gen_random_uuid()
  user_id                  uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE
  name, institution        text        NOT NULL
  invested_amount_centavos bigint      NOT NULL
  annual_rate_bps          integer     NOT NULL
  maturity_date            date        NULL
  liquidity_type           text        NOT NULL
  created_at/updated_at    timestamptz NOT NULL DEFAULT now()
  INDEX (user_id)
```

Money/rate columns are `bigint`/`integer` only — no floating-point. `user_id` is indexed for
the per-user list/scope queries.

---

## 9. Edge Cases

| Scenario | Expected behavior |
| -------- | ----------------- |
| Negative/zero quantity, malformed ticker | `400`, no write (value-object rejection). |
| Empty name/institution, negative amount/rate, bad liquidity_type | `400`, no write. |
| New at-maturity FI with a past `maturity_date` | `400` (PRD Epic 1). |
| Update/delete an id the caller doesn't own | `404 {"error":"holding not found"}` (scoped, no leak). |
| Update/delete a non-existent id | `404`. |
| Owning user deleted | Holdings removed by `ON DELETE CASCADE`. |
| List with no holdings | `200 []` (empty array, not 404). |
| Daily-liquidity FI with no maturity | Allowed (`maturity_date` null). |

---

## 10. Security Considerations

- **Isolation & ownership** — every operation scoped to the context `user_id`; mutations
  double-scoped by `id + user_id`. A user can never read/edit/delete another's holding; a
  cross-user id is "not found", never "forbidden" (no existence oracle).
- **AuthN** — `/holdings/*` require a valid session (absent from the public allowlist).
- **Input validation** — value objects + edge rules at the boundary; generic error envelope.
- **No new secrets, no AI output.** Money is the sensitive invariant: integer centavos
  end-to-end so no rounding/precision bug can corrupt a balance.

---

## 11. Observability

- **Traces** — `otelhttp` route spans (`POST /holdings/fii`, `PUT /holdings/fixed-income/{id}`,
  …); `otelsql` child query spans (statement only, no argument values).
- **Logs** — `user_id` + `request_id` on the request log; never a full holding payload.
- **Metrics** — optional `portfolio.mutations` counter by `{type, op, outcome}`; no PII.

---

## 12. Testing Strategy

### Unit Tests

- Value objects: `Ticker`, `Quantity` (zero/negative rejected), `LiquidityType`; the
  at-maturity past-date rule (with a fake `Clock`).
- Service: CRUD for both types with a hand-written fake repo (create→list→update→delete,
  not-found sentinel, validation).
- Handlers: DTO validation, **identity-from-context** (a body/path `user_id` is ignored),
  money-as-integer-centavos round-trip, ownership `404`.

### Integration Tests (gated)

- Real Postgres (`TEST_DATABASE_URL`, `-p 1`): CRUD round-trip, `0005` up/down, **per-user
  isolation**, **ownership-scoped update/delete** (B cannot touch A's row), cascade-on-delete.

### Quality gate

`task vet`, `task test:short`, gofmt-clean; integration serialized when a DB is present.

---

## 13. Definition of Done

- [ ] FR-1021…FR-1029 implemented; BR-1021…BR-1026 respected; acceptance criteria met.
- [ ] Hexagonal layering (domain pure; SQL in adapter; HTTP in transport); identity/ownership
      from context; money int64 centavos end-to-end incl. the wire; conventions honored.
- [ ] `0005_holdings` up/down + per-user isolation + ownership scoping proven against real Postgres.
- [ ] Unit + gated integration green; quality gate clean; hexagonal + go-correctness reviews pass.
- [ ] Closeout: `CHANGELOG`, `README` (endpoints), SPEC + PLAN → **Done**, indexes updated, PT-BR lesson.

---

## 14. Decisions (resolved)

| # | Decision | Resolution |
| - | -------- | ---------- |
| D1 | FII `Ticker` source | **Reuse `marketdata.Ticker`.** It is a pure value object, so the import keeps `portfolio`'s core pure (the binding dependency-direction rule is about SQL/HTTP/framework, not domain types). `marketdata` is a **foundational** seam, and feature→foundational is the intended direction — the same shape as `profile → auth.UserID(ctx)`; the graph stays acyclic. Reuse also guarantees one definition of "valid B3 ticker" so holdings reconcile against quotes. If a 3rd consumer appears, promote `Ticker` to a shared kernel (`internal/b3`) — Rule of Three. |
| D2 | Holding modeling | **Resolved: two entities + two tables** (`fii_holdings`, `fixed_income_holdings`) — clean, type-safe fields — over one polymorphic table with nullable columns. |
| D3 | `liquidity_type` semantics | **Resolved: a liquidity enum** (`daily` \| `at_maturity`); the instrument name lives in `name`. `maturity_date` is required for `at_maturity`, null for `daily`. (PRD lists CDB/Tesouro/caixinha as instrument examples, not values of this field.) |
| D4 | Endpoint shape | **Resolved: per-type routes** (`/holdings/fii`, `/holdings/fixed-income`) with distinct DTOs/validation, over a single `/holdings` with a `type` discriminator. A unified `Reader` serves the dashboard. |
| D5 | Quantity | **Resolved: whole-number `Quantity` (> 0)** — FII cotas are integral units — over a fractional quantity. |

---

## 15. Open Questions (deferred, not blocking)

- A combined `GET /holdings` (both types in one response) — deferred; the `Reader` already
  serves the aggregate to SPEC-103, and per-type lists cover the CRUD UI.
- Transaction/lot history and average-price recomputation from buys — MVP stores the
  user-entered `average_price` directly (PRD A3, manual entry).
- Whether to validate the FII `ticker` against ingested market data (i.e. reject unknown
  tickers) — deferred; market data is best-effort and may lag, so MVP accepts any valid B3
  format and reconciles in the dashboard.
- Fractional fixed-income rates beyond bps precision (not needed for MVP).
