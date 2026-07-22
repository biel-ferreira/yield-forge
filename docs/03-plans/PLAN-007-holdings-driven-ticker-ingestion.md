# PLAN-007 — Holdings-Driven FII Ticker Ingestion

## 1. Document Information

| Field           | Value                                   |
| --------------- | ---------------------------------------- |
| Plan Name       | Holdings-Driven FII Ticker Ingestion      |
| Related Feature | Holdings-Driven FII Ticker Ingestion      |
| Related Spec    | [SPEC-007](../02-specs/SPEC-007-holdings-driven-ticker-ingestion.md) (Done) |
| Version         | 0.1.0                                      |
| Status          | Done                                        |
| Author          | Gabigol                                   |
| Last Updated    | 2026-07-15                                 |

---

## 2. Objective

### Goal

Make the SPEC-006 ingestion worker discover the FII tickers to price **from what users actually
hold**, instead of only from the static `MARKETDATA_WATCHLIST` env var — closing the diagnosed gap
where 9 of a user's 10 held FIIs never got a quote because the watchlist was empty.

### Expected Outcome

Every ingestion run fetches a quote for every distinct FII ticker held by any user, with
`MARKETDATA_WATCHLIST` demoted to an optional seed (still backward-compatible, unioned in). No
migration, no HTTP surface, no change to how a quote is fetched or stored — this is purely a new
*source* for the ticker set the worker already knew how to consume.

---

## 3. Scope

### Included

- `portfolio.SystemReader` — a new, narrow, system-scoped read port (`DistinctFIITickers`).
- Its Postgres adapter — one parameterless `SELECT DISTINCT ticker FROM fii_holdings`.
- A holdings-backed `marketdata.TickerSource` adapter at the ingestion composition edge.
- A composite/union `TickerSource` that dedupes `holdings ∪ watchlist` and degrades per-source.
- Wiring into `ingest.New` (`factory.go`) — the worker itself (`worker.go`) is unchanged.
- `.env.example` + README reframing of `MARKETDATA_WATCHLIST` as an optional seed.
- Unit tests (hand-written fakes) + one gated integration test for the distinct query.

### Excluded

- Any change to quote fetching/storage, provider adapters, or the macro ingestion path.
- Any new HTTP endpoint, DTO, or `api/openapi.yaml` change.
- Any new migration or table (reuses `fii_holdings` as-is).
- Caching or incremental diffing of the ticker set (FR-075 — deferred, not built).
- Fixed-income tickers (out of scope — FI holdings aren't exchange-quoted).

---

## 4. Dependencies

### Technical Dependencies

- `internal/marketdata` — `TickerSource` port, `Watchlist` (`watchlist.go`), the ingestion worker
  and its factory (`internal/marketdata/ingest/{worker.go,factory.go}`), `ParseTicker`.
- `internal/portfolio` — `FIIHolding.Ticker` (already-validated on write, SPEC-102), the existing
  `ports.go`/`postgres/postgres.go` pair this plan extends with one new port + method.

### External Dependencies

None new.

### Blocking Decisions

SPEC-007 §14 already resolves its five design decisions (D1–D5: adapter location, separate
system port vs. `Repository` method, seed-not-replace, no cache, no new index) — this plan inherits
them as-is rather than re-litigating. The one open item is procedural, not technical:

| # | Decision | Resolution |
|---|----------|------------|
| P1 | SPEC-007 is Draft, not Approved | Flip to **Approved** before `/spec-implement 007` starts Phase 1 — no code should land against an unapproved spec (working agreement). |

---

## 5. Architecture Impact

### Existing Components Affected

| Component | Impact |
| --------- | ------ |
| `internal/portfolio/ports.go` | Add `SystemReader` port (new interface, additive) |
| `internal/portfolio/postgres/postgres.go` | Add `DistinctFIITickers` method on the existing `Repository` struct (same concrete type satisfies both `Repository` and the new narrow `SystemReader` — no new struct) |
| `internal/marketdata/watchlist.go` | Doc comment updated to cite SPEC-007 (done) instead of "arrives with SPEC-102"; behavior unchanged — it becomes one input to the new composite source, not the sole source |
| `internal/marketdata/ports.go` | `TickerSource`'s doc comment updated the same way |
| `internal/marketdata/ingest/factory.go` | `New` composes the composite `TickerSource` (holdings + optional watchlist seed) instead of injecting `Watchlist` directly |
| `.env.example` | `MARKETDATA_WATCHLIST` comment reframed as an optional seed; empty is documented as the normal case |
| `README.md` | Market Data section notes ingestion is holdings-driven |

### New Components

| Component | Purpose |
| --------- | ------- |
| `portfolio.SystemReader` (port, `ports.go`) | `DistinctFIITickers(ctx) ([]string, error)` — system-scoped, no `user_id` (BR-071) |
| Holdings-backed `TickerSource` (new file, `internal/marketdata/ingest`) | Implements `marketdata.TickerSource` over `portfolio.SystemReader` + `ParseTicker` |
| Composite/union `TickerSource` (new file, `internal/marketdata/ingest`) | Dedupes N sources (holdings + watchlist seed), degrades independently — mirrors the existing `combined` pattern already in `factory.go` for `MarketDataProvider` |

---

## 6. Implementation Strategy

### Approach

Bottom-up, same order as the SPEC's own phase framing: the new port → its Postgres adapter → the
two new ingestion-edge adapters (holdings source, composite/union) → wire into the factory → tests
→ docs. Each phase keeps `task vet` + `task test:short` green. No phase touches `worker.go` itself
— the whole point is that the worker's `w.tickers.Tickers(ctx)` call (already
`marketdata.TickerSource`-typed) doesn't need to change at all.

### Rollout Method

**Incremental**, zero-risk by construction: no migration, no schema change, no endpoint. The
watchlist seed keeps every existing `.env`-driven environment working unchanged (BR-073); an empty
watchlist — the new normal case — simply means the effective set equals the holdings-derived one.

### Rollback Strategy

Revert the `factory.go` wiring to inject `Watchlist` directly instead of the composite source — a
one-line change, since the worker's dependency stays `marketdata.TickerSource` throughout. No data
to roll back (no migration, no persisted state beyond the existing `fii_holdings`/`fii_quotes`).

---

## 7. Implementation Phases

### Phase 1 — Domain Layer

#### Tasks

- [x] `internal/portfolio/ports.go`: add `SystemReader` interface with `DistinctFIITickers(ctx
      context.Context) ([]string, error)`, doc comment citing SPEC-007 FR-071/BR-071 and explicitly
      stating it is a **system read** (no `user_id`), reachable only from the ingestion edge.
- [x] No new value object needed — `marketdata.Ticker` (existing) is reused for parsing at the
      ingestion edge (Phase 3), not here; `portfolio.SystemReader` returns raw strings on purpose
      (FR-071 AC3), keeping `portfolio` free of `marketdata` ticker semantics.

#### Deliverables

- `SystemReader` port defined, documented, zero behavior yet (interface only).

---

### Phase 2 — Persistence Layer

#### Tasks

- [x] `internal/portfolio/postgres/postgres.go`: add `DistinctFIITickers(ctx) ([]string, error)` on
      the existing `Repository` struct — `SELECT DISTINCT ticker FROM fii_holdings ORDER BY
      ticker`, no parameters, no `user_id` (BR-071); `%w`-wrapped error prefix `"list distinct fii
      tickers: %w"`, no sentinel (empty result is a valid empty slice).
- [x] No migration — reuses the existing `fii_holdings` table/index as-is (FR-075/D5).

#### Deliverables

- [x] Gated integration test (`TEST_DATABASE_URL`, `testing.Short()` skip): seed FII holdings for
      two users with one overlapping ticker; assert the deduped, alphabetically-ordered result.

---

### Phase 3 — Application Layer

#### Tasks

- [x] New file in `internal/marketdata/ingest` (e.g. `holdings_source.go`): a holdings-backed
      adapter satisfying `marketdata.TickerSource`, depending only on `portfolio.SystemReader` +
      `marketdata.ParseTicker` — never on `portfolio`'s SQL adapter or per-user `Repository`
      (BR-072). Each returned string is parsed via `ParseTicker`; a malformed entry is skipped +
      logged, never aborts the call (BR-075) — defensive, since holdings validate on write.
- [x] New file in `internal/marketdata/ingest` (e.g. `composite_source.go`): a small unexported
      type unioning N `TickerSource`s (mirrors the existing `combined` pattern in `factory.go` for
      `MarketDataProvider`) — calls each source, dedupes into a deterministic (sorted) set, and
      degrades **per source**: a failing holdings read still yields the watchlist seed and vice
      versa (BR-074), never a hard failure unless every source fails.
- [x] `internal/marketdata/ingest/factory.go`: `New` builds the holdings-backed source from the
      `*sql.DB` it already receives (via a `portfolio/postgres.Repository` instance, typed narrowly
      as `portfolio.SystemReader`), unions it with `Watchlist` (built exactly as today from
      `cfg.MarketDataWatchlist`, still fails fast on an invalid entry — SPEC-006 FR-604 unchanged),
      and injects the composite into `newWorker`. `worker.go` itself: **no changes**.
- [x] Update `.env.example`'s `MARKETDATA_WATCHLIST` comment (optional seed, empty is normal) and
      the two in-code TODOs (`watchlist.go`, `ports.go`) to cite SPEC-007 as done rather than
      "arrives with SPEC-102".

#### Deliverables

- [x] `cmd/ingest` (one-shot runner) and the in-process scheduler both pick up the holdings-backed
      source automatically, with no change to their own code (FR-074 AC3) — verified by running
      `task run` locally against the dev DB with `MARKETDATA_WATCHLIST` unset and confirming the
      worker's log line reports the holdings-derived ticker count.

---

### Phase 4 — API Layer

**N/A.** SPEC-007 adds no HTTP endpoint, DTO, or status code (§7 of the spec). No
`api/openapi.yaml` change; the drift test (`openapi_test.go`) is expected to stay green untouched
— confirmed, not modified, in Phase 6.

---

### Phase 5 — Observability

#### Tasks

- [x] Log the resolved ticker count at the start of the FII ingestion pass (and, optionally, the
      holdings-vs-seed split) — no `user_id`, no PII (BR-071, mirrors SPEC-006 §11).
- [x] Confirm (no new code expected) that a holdings-read failure flows through the worker's
      existing `ingestFII` error path — logged, metered `ingestion_items_total{kind=fii,
      outcome=source_error}` via `recordItem`, keeps last-known-good — exactly the path already at
      `worker.go:141-147`, now also reachable when the *holdings* read (not just a watchlist parse)
      fails.

#### Deliverables

- [x] A short manual/log-inspection check (not a new metric) that the ticker-count log line appears
      on a local run; no new span, no new metric added (SPEC-007 §11 — rides existing telemetry).

---

### Phase 6 — Testing

#### Unit Tests

- [x] Holdings-backed adapter, hand-written fake `SystemReader` (table-driven, `testify/require`):
      distinct passthrough → parsed `Ticker`s; malformed entry skipped, others still returned;
      empty result → empty slice; reader error surfaced as the adapter's own error.
- [x] Composite/union source: `distinct(holdings ∪ watchlist)`; per-source degradation (holdings
      source errors → watchlist tickers still returned; watchlist empty → holdings-only result).
- [x] Worker regression (reuse SPEC-006's existing worker-test scaffolding): a `source_error` from
      the holdings-backed source keeps last-known-good `fii_quotes` and the run continues.

#### Integration Tests

- [x] Real Postgres (`TEST_DATABASE_URL`, `-p 1` serialized): two users, one overlapping FII ticker
      → `DistinctFIITickers` returns the deduped, ordered set; skips cleanly with no DB configured.

#### Deliverables

- [x] `task vet` + `task test:short` clean, `gofmt`-clean; full integration suite (`go test ./...
      -count=1`) green against a **disposable** Postgres on host port 5434 — **not** the dev
      compose DB (port 5433). A real incident during this implementation (an integration test's
      `TRUNCATE`-based setup wiped real dev data when `TEST_DATABASE_URL` was mistakenly pointed
      at port 5433) produced a hard rule + a `PreToolUse` hook
      (`.claude/hooks/block-dev-db-test.ps1`) blocking this going forward — see CLAUDE.md.
- [x] Confirm `api/openapi.yaml`'s drift test is unaffected (no route added/removed/changed).

---

### Phase 7 — Documentation

#### Tasks

- [x] `README.md` Market Data section: note ingestion is holdings-driven, watchlist is an optional
      seed.
- [x] `CHANGELOG.md` `[Unreleased]` updated.
- [x] Flip **SPEC-007 + PLAN-007 → Done**; update `docs/02-specs/README.md` (foundational table)
      and `docs/03-plans/README.md`.
- [x] PT-BR lesson `docs/lessons/SPEC-007-aula.html` via **lesson-writer** (backend track).

#### Deliverables

- [x] Docs updated, spec + plan closed, lesson published.

---

## 8. Risks

| Risk | Impact | Mitigation |
| ---- | ------ | ---------- |
| A dev/CI environment relying on `MARKETDATA_WATCHLIST` as the *only* source silently changes behavior once holdings exist | Low | Union semantics (not replacement) mean nothing already working stops working — a watchlist ticker is still fetched even if never held (FR-073) |
| Hexagonal boundary slip — `marketdata` reaching into `portfolio`'s SQL, or vice versa | Medium | The adapter lives at the ingestion composition edge, which already legitimately imports both (SPEC-006 precedent); hexagonal-reviewer gate in Phase 6/closeout |
| Malformed ticker stored in `fii_holdings` (shouldn't happen — validated on write) silently drops a holding from pricing | Low | Skip + **log** (not silent) per BR-075; defensive path, exercised by a dedicated unit test |
| `SELECT DISTINCT` cost grows unnoticed as the table grows | Low | Explicitly deferred to Open Questions (index, caching) — not a Phase 1 concern at MVP volume (FR-075/BR-076) |

---

## 9. Validation Checklist

### Functional Validation

- [x] FR-071…FR-077 implemented; SPEC-007 acceptance criteria satisfied.
- [x] BR-071…BR-076 respected (system read scoping, hexagonal boundary, backward compat,
      per-source degradation, parse-don't-validate, zero-cost posture).

### Technical Validation

- [x] No `marketdata`↔`portfolio` core-to-core import; the adapter lives only at
      `internal/marketdata/ingest` (hexagonal-reviewer).
- [x] `worker.go` unchanged (its dependency stays the `marketdata.TickerSource` interface).
- [x] No `api/openapi.yaml` change; drift test still green.

### Quality Validation

- [x] `task vet` + `task test:short` clean; `task test:integration` green against real Postgres.
- [x] Reviewed by **hexagonal-reviewer** + **go-correctness-reviewer**; blocking findings fixed.
- [x] Documentation updated (CHANGELOG, README, `.env.example`, both indexes, PT-BR lesson).

---

## 10. Definition of Done

- [x] All phases complete; SPEC-007 acceptance criteria satisfied.
- [x] Hexagonal boundary proven intact; `worker.go` unchanged.
- [x] Unit + gated integration tests green; `task vet` + `task test:short` clean.
- [x] `README.md` + `.env.example` updated; `CHANGELOG.md` `[Unreleased]` updated.
- [x] **No** `api/openapi.yaml` change; drift test still green.
- [x] SPEC-007 + PLAN-007 flipped to **Done**; both indexes updated; PT-BR lesson published.

---

## 11. Deliverables

### Code Deliverables

- `portfolio.SystemReader` port + Postgres adapter method; holdings-backed + composite/union
  `TickerSource` adapters in `internal/marketdata/ingest`; `factory.go` rewiring.

### Documentation Deliverables

- CHANGELOG entry, PT-BR lesson, `.env.example` + `README.md` updates, specs/plans index updates.

---

## 12. Post-Implementation Tasks

### Future Improvements

- Ticker-set caching / change-detection if ingestion cadence or holdings volume grows (SPEC-007
  Open Questions).
- An index on `fii_holdings.ticker` if the distinct scan ever shows up in query stats.
- Pre-warming a just-added ticker's quote synchronously at holding-create time, so the first
  Dashboard render for a new FII isn't cost-basis-only (belongs with a future SPEC-102/103 touch,
  noted but not scheduled).

### Technical Debt

None anticipated — additive port, no migration, no endpoint; rollback is a one-line factory change.
