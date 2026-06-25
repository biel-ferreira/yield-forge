# Implementation Plan

## 1. Document Information

| Field           | Value                                                        |
| --------------- | ------------------------------------------------------------ |
| Plan Name       | `MarketDataProvider` Port & Ingestion Worker (FII + macro)   |
| Related Feature | Foundational — the market-data seam + scheduled ingestion    |
| Related Spec    | [SPEC-006](../02-specs/SPEC-006-marketdata-port-and-ingestion-worker.md) |
| Version         | 0.1.0                                                        |
| Status          | Approved (decisions D1–D6 resolved, incl. D4)                |
| Author          | Gabigol                                                      |
| Last Updated    | 2026-06-24                                                   |

---

## 2. Objective

### Goal

Deliver the `MarketDataProvider` port and a scheduled, request-decoupled ingestion worker
that fetches and stores per-FII market data (FR-006) and Brazilian macro indicators (FR-007)
as deterministic, last-known-good reference data behind a config-swappable provider.

### Expected Outcome

Running `cmd/ingest` (or the in-process scheduler) populates `fii_quotes` and
`macro_indicators` from free/public sources; downstream specs (dashboard SPEC-103, Fact
Builder SPEC-104, projections SPEC-107) read it through repository ports. Nothing consumes
the data yet — this spec lands the seam, the storage, and the worker.

---

## 3. Scope

### Included

- `internal/marketdata`: domain (`FIIQuote`, `MacroIndicator`, `Sector`, `Indicator`,
  `Unit`, `Ticker`), the provider + repository + `TickerSource` ports, sentinels, the
  ingestion service/worker, and a deterministic `Fake` provider.
- Provider adapters: a free FII source (default real adapter) + the BCB **SGS** macro source
  (+ an IFIX source), HTTP confined to subpackages.
- `internal/marketdata/postgres` repositories; migration `0003_market_data` (paired
  up/down, embedded, applied manually); idempotent, transactional, last-known-good-safe
  upserts.
- The Clock-driven worker + optional in-process scheduler + `cmd/ingest` one-shot.
- `MARKETDATA_*` configuration; AI/ingestion observability (spans, success-rate + freshness
  metrics); README/CHANGELOG/.env.example/lesson on close.

### Excluded (later specs / future — SPEC-006 §2)

- Consuming the data in a UI/feature (SPEC-103/104/107).
- Brokerage/B3 import (A3), intraday data (A5), fixed-income mark-to-market (A4).
- FII quote history (snapshot only); macro is a series because the source returns one.
- Any per-user scoping (market data is global reference data — BR-603).

---

## 4. Dependencies

### Technical Dependencies

- SPEC-002 (persistence, migration runner, `database/sql` + pgx), SPEC-003 (`Clock` port),
  SPEC-004 (`observability.Tracer()` / `Meter()` seam), `internal/platform/money` (bps/centavos
  rounding), config loader + `slog.LogValuer` (extended in SPEC-005).

### New Dependencies

- HTTP/JSON are stdlib (`net/http`, `encoding/json`). Parsing the Fundamentus HTML table
  wants a tolerant tokenizer: **`golang.org/x/net/html`** — already present transitively in
  the module graph (OTel/gRPC), so promoting it to a direct dependency adds no new download
  and keeps the zero-cost / stdlib-first posture (ADR-0003). (Alternative: a narrow
  `regexp`/`strings` parser, stdlib-only but more brittle — decide in Phase 3.)

### Blocking Decisions (SPEC-006 §14 — all resolved)

- **D1** one port (FII+macro, FII **batched**) · **D2** `cmd/ingest` primary + optional
  scheduler · **D3** configured watchlist `TickerSource` (holdings-backed later) · **D5**
  macro as a series · **D6** P/VP as ratio-bps.
- **D4 (resolved)** — FII via a **composite of Fundamentus (one bulk request:
  price/DY/P-VP/segment) + Yahoo Finance `.SA` (last dividend)**, both free + no-key, behind
  the port and **swappable** later to a licensed source (brapi Pro). Rationale: brapi's free
  tier no longer exposes P/VP or FII segment, and Yahoo alone can't give them — only the
  scrape-based Fundamentus does, at zero cost. Macro via **BCB SGS** (free/public). Residual
  risk is scraping brittleness, mitigated by the port + `Fake` default + fixture tests +
  graceful degradation (a parse failure keeps last-known-good).

---

## 5. Architecture Impact

### Existing Components Affected

| Component | Impact |
| --------- | ------ |
| `internal/platform/config` | Add `MARKETDATA_*` fields (provider, base URLs, interval, timeout, watchlist) + http(s)/positive validation. MVP sources need **no API key**; the `slog.LogValuer` still masks any future provider token |
| `cmd/api` | Optionally start/stop the in-process scheduler goroutine in the lifecycle (flag-gated), flushing before shutdown |
| `migrations/` | New `0003_market_data` up/down (manual) |
| `Taskfile.yml` / `Makefile` / `README` | `task ingest` target + Market Data docs |
| `internal/platform/observability` | Reuse the Tracer/Meter seam (no change) |

### New Components

| Component | Purpose |
| --------- | ------- |
| `internal/marketdata` | Domain, ports, sentinels, ingestion service/worker, `Fake` |
| `internal/marketdata/fundamentus` | Bulk FII fundamentals adapter (price/DY/P-VP/segment) |
| `internal/marketdata/yahoo` | FII last-dividend adapter (`.SA`); composed with Fundamentus |
| `internal/marketdata/bcb` | BCB-SGS macro adapter (+ IFIX source) |
| `internal/marketdata/postgres` | `FIIQuoteRepository` + `MacroRepository` adapters |
| `cmd/ingest` | One-shot ingestion runner (cron/manual) |

---

## 6. Implementation Strategy

### Approach

Bottom-up and layered, mirroring SPEC-005: a pure core (domain + ports, no HTTP/SQL/OTel),
then persistence, then the two provider adapters at the edge, then the worker that composes
them, then observability/cmd, then tests and docs. Each phase keeps `go build` + unit tests
green and is independently reviewable (the per-phase cadence). Conventions are enforced as
we go: money `int64` centavos / rates integer **bps** via `internal/platform/money`; errors
`%w` + lowercase prefix with sentinels; `Clock` over `time.Now()`; `ctx` first; **no
`user_id`** (global data, BR-603); doc comments cite SPEC/BR; `testify/require` + hand-written
fakes; test files mirror their source (`foo.go` → `foo_test.go` / `foo_integration_test.go`).

### Rollout Method

Incremental, behind a port and config. The provider is selected by `MARKETDATA_PROVIDER`
(default to the deterministic `Fake` so the zero-config app and CI never hit the network);
the scheduler is flag-gated. No data is consumed by any feature yet, so there is no
user-facing rollout.

### Rollback Strategy

Pure additive change. Disable by leaving the scheduler off / not running `cmd/ingest`; the
migration `0003` has a tested down. No existing behavior is modified.

---

## 7. Implementation Phases

### Phase 1 — Domain, Ports & Config

#### Tasks

- [ ] Value objects with constructors (parse-don't-validate): `Ticker`, `Sector`,
      `Indicator`, `Unit`; closed-enum validation; unknown sector → `Other` (raw retained).
- [ ] Entities `FIIQuote`, `MacroIndicator` (money `int64` centavos, rates integer bps,
      `observed_at`/`fetched_at` UTC); sentinels `ErrFIIQuoteNotFound`, `ErrMacroNotFound`,
      `ErrProviderUnavailable`.
- [ ] Ports: `MarketDataProvider`, `FIIQuoteRepository`, `MacroRepository`, `TickerSource`.
- [ ] `MARKETDATA_*` config (provider, base URLs, token (secret), interval, timeout,
      scheduler-enabled, watchlist) with defaults + validation (http(s) URL, positive
      interval/timeout); extend `Config.LogValue` to mask the token; `.env.example` stub.

#### Deliverables

- A compiling pure core (no HTTP/SQL/OTel) + config; value-object + config unit tests green.

---

### Phase 2 — Persistence (migration + repositories)

#### Tasks

- [ ] `migrations/0003_market_data.up.sql`/`.down.sql`: `fii_quotes` (PK `ticker`, bigint
      centavos, int bps, no floats, no `user_id`) and `macro_indicators` (PK
      `indicator, reference_date`); tested down.
- [ ] `internal/marketdata/postgres`: `UpsertFIIQuote` / `GetFIIQuoteByTicker` and
      `UpsertMacroIndicator` / `GetLatestMacroIndicator`; **idempotent** `INSERT … ON
      CONFLICT … DO UPDATE`; transactional; `scan*` maps no-rows → the `…NotFound` sentinel.
- [ ] Compile-time `var _ marketdata.FIIQuoteRepository = …` assertions.

#### Deliverables

- Persistence layer with idempotent, last-known-good-safe upserts; gated integration test
  scaffold (round-trip) ready for Phase 6.

---

### Phase 3 — FII Provider Adapter: Fundamentus + Yahoo composite (+ Fake)

#### Tasks

- [ ] Capture a real Fundamentus `fii_resultado.php` response + a Yahoo `.SA` dividend
      response as **test fixtures**; confirm the exact columns / JSON shape.
- [ ] `internal/marketdata/fundamentus`: one bulk GET → parse the HTML table (x/net/html or a
      narrow stdlib parser) → `map[Ticker]FIIQuote` with price/DY/P-VP/segment (bps/centavos
      via `money`); descriptive `User-Agent`; `io.LimitReader` body cap; a layout change /
      empty parse → `ErrProviderUnavailable` (degrade, no garbage written).
- [ ] `internal/marketdata/yahoo`: per-ticker last-dividend (amount + date) from the
      `.SA` dividend data; **optional** — a failure yields a quote without last-dividend.
- [ ] A `composite` provider implementing `FetchFIIQuotes` that merges the two; unknown
      sector string → `Other` (raw retained).
- [ ] Deterministic `Fake` provider (valid fixed data) — the **default** `MARKETDATA_PROVIDER`.

#### Deliverables

- A working composite FII adapter behind the port, fixture/`httptest`-tested (bulk parse,
  partial row, Yahoo-missing→partial, layout-change→degrade, 429, body-cap).

---

### Phase 4 — Macro Provider Adapter (BCB SGS + IFIX)

#### Tasks

- [ ] `internal/marketdata/bcb`: fetch SELIC/CDI/IPCA from BCB-SGS by series code (codes as
      documented consts), parse → `MacroIndicator` (bps, `reference_date`); IFIX from its
      configured source (index `points`).
- [ ] Same degradation contract (timeout, body cap, outage → `ErrProviderUnavailable`).

#### Deliverables

- A macro adapter behind the port, `httptest`-tested (series parse, publish-lag
  `reference_date`, outage→degrade).

---

### Phase 5 — Ingestion Worker, Scheduler, Observability & `cmd/ingest`

#### Tasks

- [ ] `Service.RunOnce(ctx)`: resolve tickers via `TickerSource` (watchlist) + the fixed
      macro set; per-item fetch→validate→upsert; **isolate per-item failures** (skip, keep
      last-known-good); return a run summary.
- [ ] In-process scheduler (Clock-driven interval, flag-gated) wired into `cmd/api`
      lifecycle with graceful stop; `cmd/ingest` one-shot (exit non-zero only on fatal
      config/DB error).
- [ ] Observability: run span + child span per provider call (`provider`/`kind`/`outcome`,
      **no token**); metrics `ingestion_runs_total{outcome}`,
      `ingestion_items_total{kind,outcome}`, `marketdata_freshness_seconds{kind}`; structured
      per-item logs (no secrets).
- [ ] `Taskfile`/`Makefile` `ingest` target.

#### Deliverables

- End-to-end ingestion (Fake provider → real repo) runnable via `cmd/ingest`; worker
  unit-tested with fakes (idempotency, last-known-good, partial success).

---

### Phase 6 — Testing

#### Unit Tests (no network/DB)

- [ ] Value objects, money/bps mapping; both adapters via `httptest`; worker with fakes
      (idempotency, last-known-good-on-failure, partial-run); freshness/staleness with a fake
      `Clock`; config (defaults, invalid URL, non-positive interval/timeout, token masked).

#### Integration Tests (gated)

- [ ] Real Postgres (`TEST_DATABASE_URL`, `-p 1`): upsert idempotency + `0003` up/down
      round-trip.
- [ ] Optional live BCB-SGS / FII fetch behind an env flag (skips cleanly in CI, like the
      live-Ollama test).

#### Deliverables

- Full suite green; quality gate clean.

---

### Phase 7 — Documentation & Lesson

#### Tasks

- [ ] `README` Market Data section (providers, `MARKETDATA_*`, `task ingest`);
      `.env.example` finalized; `CHANGELOG` `[Unreleased]` entry.
- [ ] Flip SPEC-006 + PLAN-006 to **Done**; update spec/plan indexes; `CLAUDE.md` status.
- [ ] lesson-writer → `docs/lessons/SPEC-006-aula.html` (PT-BR).

#### Deliverables

- Working-agreement closeout complete.

---

## 8. Risks

| Risk | Impact | Mitigation |
| ---- | ------ | ---------- |
| Fundamentus HTML layout changes break parsing (D4 source is scraped) | High | Fixture-based parser tests; degrade to last-known-good on parse failure; port makes a swap to a licensed source (brapi Pro) a config change; `Fake` keeps dev/CI green. |
| Yahoo `.SA` unofficial endpoint shape changes / blocks | Medium | Last-dividend is optional (partial quote, not a failed row); descriptive `User-Agent`, low daily cadence; swappable behind the port. |
| Duplicate ingestion when API is horizontally scaled | Medium | `cmd/ingest` (cron) is the decoupled primary; in-process scheduler is flag-gated (off for multi-replica). |
| A bad fetch corrupts last-known-good data | High | Idempotent, transactional upserts; skip-on-failure; explicit last-known-good tests. |
| Free FII source rate-limit / 429 | Medium | Per-request timeout + backoff; degrade with no charge (FR-610); daily cadence stays within free limits (A7). |
| IFIX free source uncertain | Medium | Configurable source; degrade gracefully if absent — SELIC/CDI/IPCA still ingest. |
| BCB-SGS series-code / publish-lag confusion | Low | Codes as documented consts; `reference_date` ≠ `fetched_at`; series PK dedupes. |

---

## 9. Validation Checklist

### Functional Validation

- [ ] FR-601…FR-612 implemented; BR-601…BR-608 respected; acceptance criteria met.
- [ ] Last-known-good preserved on per-item failure; upserts idempotent.

### Technical Validation

- [ ] Hexagonal layering (port in core, HTTP in adapters, OTel only at the edge/worker);
      no `user_id` anywhere; money/bps via `money`; `Clock` over `time.Now()`.
- [ ] Secrets masked / never in telemetry; base URLs validated; body reads capped.

### Quality Validation

- [ ] `task vet` + `task test:short` green; gated integration green when a DB is present;
      gofmt clean; hexagonal-reviewer + go-correctness-reviewer pass.

---

## 10. Definition of Done

- [ ] All phases complete; acceptance criteria satisfied; quality gate clean.
- [ ] `0003_market_data` up/down proven against a real Postgres at least once.
- [ ] CHANGELOG + README + `.env.example` updated; SPEC-006 + PLAN-006 flipped to **Done**;
      spec/plan indexes + `CLAUDE.md` status updated; PT-BR lesson produced.
- [ ] PR opened and `/pr-review` run as the pre-merge gate.

---

## 11. Deliverables

### Code Deliverables

- `internal/marketdata` (core + service/worker + `Fake`), the FII + BCB adapters, the
  Postgres repositories, `cmd/ingest`, `MARKETDATA_*` config.

### Infrastructure Deliverables

- Migration `0003_market_data` (up/down); `task ingest` target.

### Documentation Deliverables

- README Market Data section, `.env.example`, CHANGELOG entry, `SPEC-006-aula.html`.

---

## 12. Post-Implementation Tasks

### Monitoring

- Watch ingestion success-rate + freshness metrics once a real provider is wired.

### Future Improvements

- Holdings-backed `TickerSource` (with SPEC-102); FII quote history for projections
  (SPEC-107); an optional authenticated on-demand ingestion endpoint.

### Technical Debt

- Confirm the FII source's long-term free terms; keep ≥1 alternative free FII source behind
  the port (PRD Risk row).
