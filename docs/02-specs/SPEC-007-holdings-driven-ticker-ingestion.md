# SPEC-007 — Holdings-Driven FII Ticker Ingestion

## 1. Document Information

| Field        | Value                                                    |
| ------------ | -------------------------------------------------------- |
| Feature Name | Holdings-Driven FII Ticker Ingestion                     |
| Feature ID   | SPEC-007 (foundational)                                  |
| Version      | 0.1.0                                                    |
| Status       | Approved                                                 |
| Author       | Gabigol                                                  |
| Last Updated | 2026-07-14                                               |
| Related PRD  | [PRD.md](../01-product/PRD.md) — FR-006 (per-FII market data), §10 NFR (Reliability), §14 Risks — refines *how* the ingestion worker learns which FII tickers to fetch |
| Governing    | Refines [SPEC-006](SPEC-006-marketdata-port-and-ingestion-worker.md) (`TickerSource` / ingestion worker, resolving its **D3** and the `SPEC-102` TODO in `watchlist.go` / `ports.go`); consumes [SPEC-102](SPEC-102-portfolio-management.md) (FII holdings); fixes an observed data gap surfaced through [SPEC-103](SPEC-103-dashboard.md) (Dashboard) |
| Related ADRs | [ADR-0002](../04-architecture/adr/ADR-0002-tech-stack-and-layering.md) (layering), [ADR-0003](../04-architecture/adr/ADR-0003-zero-cost-and-pluggable-llm.md) (zero-cost & pluggable) |

> **Numbering note (like SPEC-109).** This is a *refinement*, not a new PRD capability: it
> resolves SPEC-006's own deferred decision **D3** ("Configured `MARKETDATA_WATCHLIST` now via
> `TickerSource`; swap to a holdings-backed source when SPEC-102 lands") and the two in-code TODOs
> that name SPEC-102 (`internal/marketdata/watchlist.go`, `internal/marketdata/ports.go`). Because
> the seam it refines is **foundational** (SPEC-006, the ingestion worker — no user-facing screen),
> it takes the next free **foundational** number (SPEC-007), the mirror of how SPEC-109 — refining
> the *feature* spec SPEC-102 — took the next free *feature* number. It introduces **no new FR** in
> the PRD; it makes FR-006 actually cover the tickers users own.

---

## 2. Overview

### Purpose

Make the SPEC-006 ingestion worker **discover the FII tickers to refresh from what users actually
hold**, instead of from a hand-maintained static list (`MARKETDATA_WATCHLIST`). Each run, the worker
resolves the set of **distinct FII tickers across all users' holdings** and fetches a quote for every
one — so a ticker a user owns is always a ticker the worker knows to price.

### Business Value

This closes a **real, diagnosed data gap** (production/dev investigation, not hypothetical): a user
with **10 FIIs** in their portfolio had a quote for **only 1** of them in `fii_quotes` — the other 9
were never fetched — because `MARKETDATA_WATCHLIST` was empty and the worker had no way to know those
tickers existed. The visible fallout was all **correct dashboard behavior over incomplete data**:

- **Passive income / month understated** — a held FII with no quote contributes **zero**
  `last_dividend` to `monthlyIncome` ([`internal/dashboard/compute.go:43`](../../internal/dashboard/compute.go#L43)).
- **"Valorização" (appreciation) concentrated in one ticker** — the 9 unpriced FIIs fall back to
  **cost basis** (`current = invested`), so only the 1 priced FII shows any gain/loss
  ([`compute.go:44-48`](../../internal/dashboard/compute.go#L44-L48)).
- **90%+ of sector exposure collapses into "Outros / sem cotação"** — a missing quote is bucketed as
  `SectorOther` ([`compute.go:44-47`](../../internal/dashboard/compute.go#L44-L47)).

None of that is a Dashboard bug — the Dashboard is faithfully computing over the data it is handed.
The **data itself is incomplete**, because ingestion never learned those tickers. Sourcing tickers
from holdings makes the deterministic facts the Dashboard, Fact Builder (SPEC-104), and Projections
(SPEC-107) read **complete by construction**, which is the entire promise of SPEC-006.

### Success Criteria

- Each ingestion run fetches a quote for **every distinct FII ticker held by any user**, with no
  operator action and no per-ticker configuration.
- With `MARKETDATA_WATCHLIST` **empty** (the now-normal case), a fresh user's newly-added FIIs are
  priced on the **next run** — the 10-of-10 case, not 1-of-10.
- `MARKETDATA_WATCHLIST` stays **backward-compatible**: a configured list still works, now as an
  **optional seed** unioned with the holdings-derived set (deduped), so existing `.env`-driven
  environments and local/dev-without-holdings keep working unchanged.
- The hexagonal boundary is **intact**: neither the `marketdata` core nor the `portfolio` core imports
  the other; the holdings→ticker adaptation lives at the ingestion **composition edge**.
- The refinement adds **no HTTP surface** and **no migration** — it reuses the existing
  `fii_holdings` table via one `DISTINCT` read.

### Scope

**In scope**

- A **holdings-backed `TickerSource`** implementing `marketdata.TickerSource`, wired into the worker
  via `factory.New`.
- A narrow, **system-scoped** read port on the `portfolio` side that lists the distinct FII tickers
  across all holdings (no user scoping — see BR-071), plus its Postgres adapter (one `SELECT DISTINCT`).
- Reframing `MARKETDATA_WATCHLIST` from *the* ticker source to an **optional seed/complement** unioned
  with the holdings set.
- Unit tests (fake holdings source) + a gated integration test for the real distinct query.

**Out of scope**

- Any change to *how* a quote is fetched or stored, to the provider adapters, or to the macro path
  (all unchanged from SPEC-006).
- A new HTTP endpoint, DTO, or OpenAPI change (this is an internal ingestion seam).
- A new migration or table (reuses `fii_holdings`).
- Caching / incremental diffing of the ticker set (a single `DISTINCT` per run is sufficient for MVP
  volume — see FR-075).
- Fixed-income tickers (FI holdings are not exchange-quoted; they resolve rates via SPEC-109 macro
  data, not `fii_quotes`).

---

## 3. Functional Requirements

### FR-071 — System-scoped distinct FII ticker read port (portfolio side)

`portfolio` exposes a **narrow, system-scoped** read port that returns the set of distinct FII tickers
present across **all** users' holdings, for the internal ingestion job. It is deliberately **separate**
from the per-user `Repository` / `Reader` (which are `WHERE user_id = $1`): this read is not made on
behalf of a user, returns no user-identifying data, and needs no scoping (BR-071).

#### Acceptance Criteria

- [ ] A new port (e.g. `portfolio.SystemReader` with `DistinctFIITickers(ctx) ([]string, error)`)
      is defined next to the existing ports, documenting that it is a **system read**, not per-user.
- [ ] The Postgres adapter implements it with a single `SELECT DISTINCT ticker FROM fii_holdings
      ORDER BY ticker` — parameterless, no `user_id` (BR-071), deterministic order.
- [ ] The method returns raw ticker **strings**; validation/parsing into `marketdata.Ticker` happens
      at the marketdata edge (FR-072), keeping `portfolio` free of `marketdata`-ticker semantics.
- [ ] It follows repository conventions: `%w`-wrapped error prefix (`"list distinct fii tickers: %w"`),
      no sentinel needed (an empty result is a valid empty slice, not "not found").

### FR-072 — Holdings-backed `TickerSource` adapter (marketdata ingestion edge)

A new adapter in the ingestion package implements `marketdata.TickerSource` by calling the FR-071
system reader and parsing each string through `ParseTicker` (parse-don't-validate). This is where the
two feature cores meet — at the **composition edge**, so neither core imports the other (BR-072).

#### Acceptance Criteria

- [ ] The adapter satisfies `marketdata.TickerSource` (`Tickers(ctx) ([]Ticker, error)`) and depends
      only on a narrow `portfolio` read port + `marketdata`'s own `ParseTicker` — never on
      `portfolio`'s SQL adapter or the per-user `Repository`.
- [ ] Each returned string is parsed via `ParseTicker`; a malformed stored ticker is **skipped and
      logged**, never aborting the run (defensive — holdings validate tickers on write, so this is a
      guard, not an expected path) (BR-075).
- [ ] A read error from the system reader is returned as the source error; the worker's existing
      `ingestFII` handling logs it, meters `source_error`, and **keeps last-known-good** (BR-074) —
      exactly the path already at [`worker.go:141-147`](../../internal/marketdata/ingest/worker.go#L141-L147).

### FR-073 — `MARKETDATA_WATCHLIST` becomes an optional seed (union, not replacement)

The effective ticker set for a run is the **deduplicated union** of (a) the holdings-derived tickers
(FR-072) and (b) the configured `MARKETDATA_WATCHLIST` seed. The watchlist is thus retained for
backward compatibility and for **seeding local/dev or CI environments that have no holdings yet**, but
it is no longer required for a held ticker to be priced.

#### Acceptance Criteria

- [ ] With `MARKETDATA_WATCHLIST` empty (the new default posture), the effective set equals the
      holdings-derived set.
- [ ] With `MARKETDATA_WATCHLIST` non-empty, the effective set is `distinct(holdings ∪ watchlist)` —
      a ticker held **and** listed appears exactly once (no duplicate provider work).
- [ ] The union is composed via a small **composite `TickerSource`** so ordering is deterministic and
      each underlying source degrades independently (a failing holdings read still yields the watchlist
      seed, and vice-versa) (BR-074).
- [ ] `.env.example` and its comment are updated to describe the watchlist as an **optional seed**, not
      the primary source; an empty value is documented as valid and expected.

### FR-074 — Wire the holdings-backed source into the worker

`factory.New` composes the worker's `TickerSource` from the new holdings-backed source (+ the watchlist
seed per FR-073), using the `*sql.DB` it already receives — so a live deployment fetches quotes for
every owned ticker each run with no further wiring.

#### Acceptance Criteria

- [ ] `ingest.New` builds the composite `TickerSource` (holdings + optional watchlist seed) and injects
      it into `newWorker`; the worker code (`ingestFII`) is **unchanged** — it still depends only on
      `marketdata.TickerSource`.
- [ ] An invalid `MARKETDATA_WATCHLIST` entry still **fails fast at startup** (unchanged from SPEC-006
      FR-604), before the worker runs.
- [ ] `cmd/ingest` (the one-shot runner) and the in-process scheduler both pick up the holdings-backed
      source with no change to their own code.

### FR-075 — MVP performance & scale posture

At MVP volume, a single `SELECT DISTINCT ticker FROM fii_holdings` per run is sufficient; **no cache,
no incremental diffing** is added.

#### Acceptance Criteria

- [ ] Exactly one distinct-ticker query is issued per ingestion run (not one-per-user, not per-ticker).
- [ ] The downstream provider fetch stays a **single bulk request** for the whole set (SPEC-006 FR-602,
      Fundamentus bulk) — the ticker count grows the result map, not the request count.
- [ ] The zero-cost posture holds: the query hits the existing indexed `fii_holdings` table; any future
      cache is explicitly deferred (documented in Open Questions), not built now.

### FR-076 — Test coverage

#### Acceptance Criteria

- [ ] **Unit** — the holdings-backed adapter is tested with a **hand-written fake** system reader
      (table-driven, `testify/require`, no `gomock`/`mockery`): distinct passthrough, malformed-ticker
      skip, empty result, and a reader error surfaced correctly.
- [ ] **Unit** — the composite/union `TickerSource` dedupes holdings ∪ watchlist and degrades per source.
- [ ] **Integration** — gated by `testing.Short()` + `TEST_DATABASE_URL`: seed FII holdings for two
      users (with an overlapping ticker), assert `DistinctFIITickers` returns the deduped, ordered set;
      skips cleanly with no DB.
- [ ] Quality gate green: `task vet` + `task test:short`, `gofmt`-clean.

### FR-077 — Documentation & closeout

#### Acceptance Criteria

- [ ] `README.md` Market Data section notes that ingestion is **holdings-driven**, with the watchlist
      as an optional seed.
- [ ] `CHANGELOG.md` `[Unreleased]` updated; the two in-code TODOs (`watchlist.go`, `ports.go` on
      `TickerSource`) are updated to reference **SPEC-007 (done)** rather than "arrives with SPEC-102".
- [ ] SPEC-007 + PLAN-007 flipped to **Done**; `docs/02-specs/README.md` foundational table updated;
      PT-BR lesson `docs/lessons/SPEC-007-aula.html` produced via **lesson-writer** (backend track).
- [ ] No `api/openapi.yaml` change (no endpoint added/changed); the drift test stays green untouched.

---

## 4. User Flows

> "User" here is the **system** (Epic 4): ingestion is a background actor. The end-user impact is
> indirect — their Dashboard becomes complete.

### Flow 1 — Holdings-driven ingestion (happy path)

1. The scheduler fires (or `cmd/ingest` runs).
2. The worker calls `TickerSource.Tickers(ctx)`; the composite source issues one
   `SELECT DISTINCT ticker FROM fii_holdings`, unions the (possibly empty) watchlist seed, dedupes.
3. The worker fetches quotes for the whole set in one bulk provider request and upserts each (SPEC-006
   FR-605 unchanged).
4. Every held FII now has a fresh row in `fii_quotes`; the next Dashboard read prices all of them.

### Flow 2 — Fresh user adds FIIs (the diagnosed case, now fixed)

1. A user adds 10 FIIs via SPEC-102; `MARKETDATA_WATCHLIST` is empty.
2. On the next run, all 10 tickers are discovered from `fii_holdings` and priced.
3. The Dashboard shows real passive income, per-ticker appreciation, and true sector exposure — **10 of
   10 priced**, not 1 of 10.

### Flow 3 — Holdings read fails (degradation)

1. The distinct-ticker query errors (DB blip).
2. The composite source still returns the watchlist seed (if any); a total failure surfaces as the
   source error the worker already handles — logged, metered `source_error`, **last-known-good kept**,
   run continues, next cycle retries (SPEC-006 FR-610).

---

## 5. Business Rules

### BR-071 — Ingestion ticker discovery is a system read, not per-user (extends BR-603)

The distinct-ticker read spans **all** users' holdings on behalf of an **internal worker**, not an
authenticated request — so the "identity-from-context, `WHERE user_id = $1`" rule (SPEC-003) **does not
apply**: there is no request user to scope to. It is safe because it returns **only public B3 tickers**
— no `user_id`, no amounts, no PII, nothing user-identifying crosses the boundary. This mirrors
SPEC-006 **BR-603** ("market data is global reference data, no `user_id`"): the *set of tickers to
price* is likewise global reference data derived from holdings, not a per-user view of them. The port
is named and documented as a **system read** and is reachable **only** from the ingestion edge, never
from an HTTP handler.

### BR-072 — Hexagonal boundary preserved (no core-to-core import)

The `marketdata` core must not import `portfolio`, and the `portfolio` core must not import
`marketdata`'s `TickerSource`. The holdings→ticker adaptation lives at the **ingestion composition
edge** (`internal/marketdata/ingest`, which already imports both `marketdata` and the portfolio
Postgres adapter), consuming a **narrow** `portfolio` read port and producing a `marketdata.TickerSource`.
Ports stay with their feature; the adapter sits at the edge (ADR-0002 dependency direction).

### BR-073 — Backward compatibility is non-negotiable

A configured `MARKETDATA_WATCHLIST` keeps working (now as a seed, unioned + deduped); an **empty**
watchlist is valid and becomes the normal case. No env var is removed or renamed. An invalid watchlist
entry still fails fast at startup (SPEC-006 FR-604 unchanged).

### BR-074 — Graceful degradation, per source (extends BR-602 / FR-610)

A failed holdings read or a malformed stored ticker **never aborts** ingestion and **never overwrites**
last-known-good `fii_quotes`. Within the composite source, each underlying source degrades independently
— a holdings-read failure still yields the watchlist seed, and an unparseable ticker is skipped, not
fatal. This is the same last-known-good posture SPEC-006 already guarantees.

### BR-075 — Parse-don't-validate at the edge

Each distinct ticker string is turned into a `marketdata.Ticker` through its constructor `ParseTicker`;
an invalid instance is unrepresentable. Because holdings already validate the ticker on write (SPEC-102),
an unparseable value here is a defensive guard (skip + log), not an expected code path.

### BR-076 — Zero-cost / MVP scale

One `DISTINCT` query per run over the indexed `fii_holdings` table is the whole cost; no cache, no
per-user fan-out, no extra provider requests (the bulk fetch already serves the full set). Any future
optimization is deferred, not pre-built (ADR-0003 posture).

---

## 6. Domain Model

No new domain entity. This spec adds **ports and an adapter**, and reuses existing value objects:

- **`marketdata.Ticker`** — the validated B3 ticker value object (SPEC-006), reused to parse the
  distinct strings from holdings.
- **`portfolio.FIIHolding.Ticker`** — the source of truth for "held tickers"; already validated on write.

### Port: `portfolio.SystemReader` (new, system-scoped)

| Method                | Signature                                | Description                                              |
| --------------------- | ---------------------------------------- | ------------------------------------------------------- |
| `DistinctFIITickers`  | `(ctx) ([]string, error)`                | Distinct FII tickers across **all** holdings; no scoping (BR-071) |

### Adapter: holdings-backed `TickerSource` (new, ingestion edge)

Implements `marketdata.TickerSource.Tickers(ctx)` by calling `SystemReader.DistinctFIITickers`, parsing
each via `ParseTicker` (skip+log invalid), returning `[]marketdata.Ticker`.

### Composite `TickerSource` (new, ingestion edge)

Unions N `TickerSource`s (holdings + watchlist seed), dedupes, deterministic order; each underlying
source degrades independently (BR-074).

---

## 7. API Contract

**None.** This is an internal ingestion seam: no HTTP endpoint, request/response DTO, or status code is
added or changed. Consequently **no `api/openapi.yaml` change** is required and the OpenAPI drift test is
untouched (contrast SPEC-109, which *did* add `GET /market/indicators`).

---

## 8. Data Model

**No new table, no migration.** The distinct-ticker read reuses the existing `fii_holdings` table
(SPEC-102). The query is:

```sql
SELECT DISTINCT ticker FROM fii_holdings ORDER BY ticker;
```

`fii_holdings` is already indexed on `user_id`; `ticker` cardinality is low and the table small at MVP
volume, so a plain distinct scan is adequate (FR-075). Whether a dedicated index on `ticker` is worth
adding later is deferred (Open Questions) — not added now, to keep the change migration-free.

---

## 9. Edge Cases

| Scenario | Expected behavior |
| -------- | ----------------- |
| No holdings anywhere **and** empty watchlist | `Tickers` returns empty; the worker's FII loop is a clean no-op; **macro still ingests** (unchanged SPEC-006 behavior, `worker.go:148-150`). |
| Same ticker held by many users | `DISTINCT` collapses it to one; one fetch, one upsert. |
| Ticker both held and in the watchlist seed | Union dedupes to one entry — no duplicated provider work (FR-073). |
| Malformed ticker somehow stored in `fii_holdings` | `ParseTicker` fails → **skip + log**, run continues with the rest (BR-075). |
| Distinct-ticker query fails (DB blip) | Composite still returns the watchlist seed; a total failure is the source error the worker already handles → log + meter `source_error`, keep last-known-good, retry next cycle (BR-074). |
| Very large held-ticker set | Still **one** bulk provider request (SPEC-006 FR-602); the set grows the result map, not the request count (FR-075). |
| A held ticker has no provider quote (delisted/unknown) | Unchanged SPEC-006 per-item miss: skip upsert, keep last-known-good; the Dashboard's cost-basis + stale fallback applies (this spec makes the ticker *known*, not the quote *guaranteed*). |

---

## 10. Security Requirements

### Authentication / Authorization

No new auth surface. The system read is reachable **only** from the internal ingestion edge (worker /
`cmd/ingest`), never from an HTTP handler; if an on-demand HTTP trigger is ever added it must sit behind
the deny-by-default auth middleware (SPEC-003), unchanged from SPEC-006 §10.

### Data Protection

The cross-user read returns **only public B3 tickers** — no `user_id`, amounts, or PII leaves the
`portfolio` boundary (BR-071). Only public tickers reach the provider (unchanged SPEC-006 egress rule:
no user data is ever sent to an external source).

### Input Validation

Tickers are parsed as value objects (`ParseTicker`) before they reach any provider URL — request-forgery
via a crafted identifier stays prevented (SPEC-006 §10), and a malformed stored value is skipped.

---

## 11. Observability

No new metric or span is required — the refinement rides the existing SPEC-006 ingestion telemetry:

- **Metrics** — a holdings-read failure meters the existing `ingestion_items_total{kind=fii,
  outcome=source_error}` (via `recordItem`); success continues to meter per upserted quote.
- **Traces** — the existing `marketdata.ingest` → `marketdata.fetch_fii` spans are unchanged; the
  ticker set is resolved before the provider span, as today.
- **Logs** — the resolved **ticker count** (and, optionally, how many came from holdings vs. the seed)
  is logged at the start of the FII pass for debuggability; **no `user_id`, no PII** is logged
  (BR-071, SPEC-006 §11).

---

## 12. Testing Strategy

### Unit Tests

- Holdings-backed adapter with a **hand-written fake** `SystemReader` (table-driven, `testify/require`):
  distinct passthrough → parsed `Ticker`s; malformed ticker skipped + others returned; empty result →
  empty slice; reader error surfaced.
- Composite/union `TickerSource`: `distinct(holdings ∪ watchlist)`; per-source degradation (holdings
  fails → watchlist still returned).
- Worker regression with the fake: a `source_error` from the holdings read keeps last-known-good and
  continues the run (reuses SPEC-006 worker-test scaffolding).

### Integration Tests (gated)

- Real Postgres (`TEST_DATABASE_URL`, `testing.Short()` skip): seed FII holdings for two users with one
  overlapping ticker; assert `DistinctFIITickers` returns the **deduped, ordered** set. Skips cleanly
  with no DB.

### Quality Gate

`task vet`, `task test:short`, `gofmt`-clean; integration serialized (`-p 1`) when a DB is present.

---

## 13. Definition of Done

- [ ] FR-071…FR-077 implemented; BR-071…BR-076 respected.
- [ ] Hexagonal boundary proven intact: no `marketdata`↔`portfolio` core import; the adapter lives at
      the ingestion edge (hexagonal-reviewer green).
- [ ] Holdings-backed + watchlist-seed composite `TickerSource` wired via `factory.New`; the worker is
      unchanged.
- [ ] The two in-code TODOs (`watchlist.go`, `ports.go` `TickerSource`) updated to cite SPEC-007.
- [ ] Unit + gated integration tests green; `task vet` + `task test:short` clean, `gofmt`-clean.
- [ ] go-correctness-reviewer + hexagonal-reviewer pass; blocking findings fixed.
- [ ] `README.md` + `.env.example` updated (watchlist reframed as optional seed); `CHANGELOG.md`
      `[Unreleased]` updated.
- [ ] **No** `api/openapi.yaml` change; drift test still green.
- [ ] SPEC-007 + PLAN-007 flipped to **Done**; `docs/02-specs/README.md` foundational table updated;
      PT-BR lesson `docs/lessons/SPEC-007-aula.html` produced (lesson-writer, backend track).

---

## 14. Decisions (proposed — confirm in review)

| # | Decision | Recommendation |
| - | -------- | -------------- |
| D1 | Where does the holdings→ticker adapter live? | **`internal/marketdata/ingest`** (the composition edge that already imports both `marketdata` and the portfolio Postgres adapter), consuming a **narrow `portfolio.SystemReader`** and producing a `marketdata.TickerSource`. Keeps both cores import-clean (BR-072). |
| D2 | New method on `portfolio.Repository` vs a separate system port? | **Separate `portfolio.SystemReader`** — the existing `Repository`/`Reader` are per-user (`WHERE user_id = $1`); a cross-user, unscoped read does not belong on them and would blur the identity rule. A distinct, clearly-documented system port keeps the scoping contract honest (BR-071). |
| D3 | Replace, complement, or seed the watchlist? | **Seed/complement** — effective set = `distinct(holdings ∪ watchlist)`. Preserves `.env` backward compat and gives local/dev/CI a way to price tickers before any holdings exist, at zero extra provider cost (FR-073). |
| D4 | Cache the distinct set between runs? | **No** — one `DISTINCT` per run is trivial at MVP volume; a cache is premature (FR-075). Deferred to Open Questions. |
| D5 | Add an index on `fii_holdings.ticker`? | **Not now** — keeps the change migration-free; low volume makes a distinct scan fine. Revisit if the table grows (Open Questions). |

---

## 15. Open Questions (deferred, not blocking)

- **Ticker-set caching / change-detection** — if run cadence or holdings volume grows, cache the
  distinct set (or refresh it on holding create/delete) instead of querying every run. Deferred.
- **Index on `fii_holdings.ticker`** — worth a small migration if the distinct scan ever shows up in
  query stats; skipped for MVP to stay migration-free (D5).
- **Retire the watchlist entirely?** — once every environment has holdings, `MARKETDATA_WATCHLIST`
  could be dropped. Kept for now for backward compat + dev seeding (BR-073); revisit post-MVP.
- **Pre-warming quotes at holding-create time** — a future optimization could fetch a just-added
  ticker's quote synchronously so the first Dashboard render is never cost-basis-only; out of scope
  here (belongs with SPEC-102/103), noted for later.
