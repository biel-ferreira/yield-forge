# Feature Specification (SPEC)

## 1. Document Information

| Field        | Value                                                  |
| ------------ | ------------------------------------------------------ |
| Feature Name | `MarketDataProvider` Port & Ingestion Worker (FII + macro) |
| Feature ID   | SPEC-006 (foundational)                                |
| Related PRD  | [PRD.md](../01-product/PRD.md) — FR-006, FR-007, Epic 4, §10 NFR (Reliability/Scalability/Cost), §13 Dependencies, §14 Risks |
| Related ADRs | [ADR-0003](../04-architecture/adr/ADR-0003-zero-cost-and-pluggable-llm.md) (zero-cost & pluggable provider), [ADR-0002](../04-architecture/adr/ADR-0002-tech-stack-and-layering.md) (layering) |
| Version      | 0.1.0                                                  |
| Status       | Approved                                               |
| Plan         | PLAN-006 (authored next via /plan-new 006)             |

---

## 2. Overview

### Purpose

Provide the second external-provider seam of YieldForge: a `MarketDataProvider` port and a
scheduled **ingestion worker** that periodically fetches and stores (a) per-FII market data
— current price, dividend yield, P/VP, sector, last dividend (FR-006) — and (b) Brazilian
macro indicators — SELIC, IPCA, CDI, IFIX (FR-007). The data is the deterministic,
last-known-good reference the dashboard (SPEC-103), the Fact Builder (SPEC-104), and the
projections (SPEC-107) read — **never** generated, always computed/ingested.

### Business Value

The dashboard's current value, allocation, and passive-income figures and every
market-context insight are only as trustworthy as the data behind them. Putting ingestion
behind a port (like the `Insighter`) keeps the exact free/public providers swappable —
mitigating the PRD's top risk ("no reliable/free FII data source") — and keeps facts
deterministic so the same inputs always produce the same Health Score (PRD §6).

### Scope

**In scope**

- A `marketdata` feature package: domain (`FIIQuote`, `MacroIndicator`, `Sector`,
  `Indicator`, `Ticker`), the provider port(s), the repository ports, and the ingestion
  service/worker.
- Provider adapters behind the port: a free FII source (default real adapter) and the Banco
  Central **SGS** macro source, plus an IFIX source; a deterministic `Fake`.
- A scheduled, request-decoupled ingestion worker (Clock-driven) and a `cmd/ingest`
  one-shot runner for cron/manual use.
- Idempotent, transactional upserts that preserve last-known-good data on failure; a new
  migration `0003_market_data` (paired up/down, embedded, applied manually).
- Repository reads for downstream specs; freshness timestamping; AI/ingestion observability;
  `MARKETDATA_*` configuration.

**Out of scope**

- Consuming the data in a UI/feature (dashboard SPEC-103, Fact Builder/insights SPEC-104,
  projections SPEC-107). This spec ingests, stores, and exposes repository reads only.
- Brokerage / B3 statement import (PRD A3) and intraday data (PRD A5 — daily freshness is
  sufficient); fixed-income mark-to-market (PRD A4).
- A historical time-series of FII quotes (MVP keeps a current snapshot per ticker); macro is
  stored as a series because the source already returns one.
- Per-user data. Market data is **global reference data** — there is no `user_id` here.

---

## 3. Functional Requirements

### FR-601 — The `MarketDataProvider` Port

A provider-neutral port the ingestion worker depends on; vendor HTTP lives only in adapter
subpackages (FR-018 / ADR-0003).

#### Acceptance Criteria

- [ ] `marketdata` defines the port(s) for reading a single FII quote and a single macro
      indicator from an external source.
- [ ] The worker, repositories, and all callers depend on the interface, never on a vendor
      SDK or `net/http`.
- [ ] The active provider is selected by `MARKETDATA_PROVIDER` with no code change.

### FR-602 — FII Data Provider Adapter (+ Fake)

The MVP source is a **composite** of two free, no-key sources behind the port (D4): the
**Fundamentus** bulk FII table (one request → price, dividend yield, P/VP, segment for every
FII) merged with **Yahoo Finance** (`.SA`) for the last-dividend amount + date. Both sit
behind `MarketDataProvider`, so a cleaner/licensed source (e.g. brapi Pro) is a later config
swap — never a rewrite (BR-601).

#### Acceptance Criteria

- [ ] The adapter yields, per ticker: current **price** (`int64` centavos), **dividend
      yield** (bps), **P/VP** (bps), **sector/segment**, and **last dividend** (centavos,
      with its date when available).
- [ ] The Fundamentus fetch is a **single bulk request** serving all requested tickers; the
      Yahoo last-dividend lookup is per-ticker and **optional** (its absence yields a quote
      without last-dividend, not a failed row).
- [ ] An unknown ticker, a partial payload (e.g. missing P/VP), a changed/garbage HTML table,
      and a non-200/429 response are each handled without panicking and without corrupting
      stored data.
- [ ] A deterministic `Fake` adapter returns fixed, valid data for tests and offline/dev,
      and is the **default provider** so the zero-config app and CI never hit the network.

### FR-603 — Macro Data Provider Adapter

#### Acceptance Criteria

- [ ] SELIC, IPCA, and CDI are fetched from the Banco Central **SGS** API (free, public, no
      key) by series code; IFIX from its configured source.
- [ ] Each indicator yields a value in its documented unit (rates in **bps**; IFIX as an
      index level) plus the **reference date** the source attributes it to.
- [ ] A source outage or malformed series response degrades gracefully (see FR-610).

### FR-604 — Ingestion Worker / Scheduler

#### Acceptance Criteria

- [ ] A worker runs the ingestion job **on a schedule, decoupled from request handling**,
      driven by the injected `Clock` (deterministic in tests).
- [ ] One run refreshes the configured set of FII tickers and the fixed set of macro
      indicators; per-item failures are isolated (one bad ticker does not abort the run).
- [ ] A `cmd/ingest` one-shot command performs a single run (cron/manual friendly) and exits
      non-zero only on a fatal (config/DB) error, not on a per-item provider failure.

### FR-605 — Idempotent, Last-Known-Good-Safe Persistence

#### Acceptance Criteria

- [ ] Writes are **idempotent upserts** (FII by ticker; macro by indicator + reference date),
      safe to re-run and safe under an overlapping schedule.
- [ ] A failed or empty fetch **never overwrites** an existing good row (PRD Reliability NFR).
- [ ] Each upsert is transactional; a partial run leaves the store consistent.

### FR-606 — Freshness & Timestamping

#### Acceptance Criteria

- [ ] Every stored record carries `observed_at` (the source's as-of/reference instant) and
      `fetched_at` (when we ingested it), both UTC.
- [ ] A read exposes enough to compute **staleness** against the `Clock` (freshness ≤ 24h is
      the target; stale data is served but identifiable).

### FR-607 — Repositories & Storage

#### Acceptance Criteria

- [ ] `FIIQuoteRepository` (`UpsertFIIQuote`, `GetFIIQuoteByTicker` → `…NotFound`) and a macro
      repository (`UpsertMacroIndicator`, `GetLatestMacroIndicator`) behind ports.
- [ ] Migration `0003_market_data` creates `fii_quotes` (snapshot per ticker) and
      `macro_indicators` (series), money as `bigint` centavos and rates as integer bps — **no
      floats** — with a tested down migration.

### FR-608 — Configuration

#### Acceptance Criteria

- [ ] `MARKETDATA_*` variables select provider, base URLs, optional API token (secret),
      refresh interval, per-request timeout, scheduler enable, and the MVP watchlist; all
      documented in `.env.example`, all with safe defaults.
- [ ] An invalid base URL or a non-positive interval/timeout fails fast at config load
      (carrying the SPEC-005 hardening).

### FR-609 — Observability (no secrets)

#### Acceptance Criteria

- [ ] An ingestion run emits a span with a child span per provider call (provider, kind,
      outcome — never the API token); ingestion success-rate and data-freshness metrics are
      recorded (PRD §10).
- [ ] No secret (provider token) appears in any log, error, or span attribute.

### FR-610 — Graceful Degradation

#### Acceptance Criteria

- [ ] A provider outage / rate-limit (429) is logged and metered, **incurs no charge and no
      data corruption**, and the run continues; the next cycle retries (with backoff for
      transient failures).
- [ ] Downstream reads keep returning last-known-good data while a provider is down.

### FR-611 — Deterministic Fake & Tests

#### Acceptance Criteria

- [ ] The `Fake` provider + hand-written repository fakes make the worker fully unit-testable
      with no network/DB: idempotency, last-known-good-on-failure, and partial-success paths.

### FR-612 — Documentation

#### Acceptance Criteria

- [ ] `README.md` gains a Market Data section (providers, `MARKETDATA_*`, the `cmd/ingest`
      runner); `CHANGELOG.md` updated; the PT-BR lesson `docs/lessons/SPEC-006-aula.html`
      produced on close.

---

## 4. User Flows

> "User" here is the **system** (Epic 4): ingestion is a background actor, not an end-user
> screen.

### Flow 1 — Scheduled ingestion (happy path)

1. The scheduler fires (interval elapsed per the `Clock`).
2. The worker resolves the ticker set + the fixed macro set.
3. For each item it calls the provider, validates/normalizes the result, and upserts it
   transactionally with fresh `fetched_at`.
4. Run span + success metrics recorded; the store reflects current market data.

### Flow 2 — Per-item provider failure (isolation + last-known-good)

1. A ticker fetch returns 429 / malformed / not-found.
2. The worker logs + meters the item outcome and **skips the upsert** — the prior good row
   stands.
3. The run continues with the remaining items and completes as a partial success.

### Flow 3 — One-shot run (`cmd/ingest`)

1. An operator (or the host's cron) runs `cmd/ingest`.
2. It performs exactly one ingestion run and exits 0 (or non-zero only on a fatal config/DB
   error).

---

## 5. Business Rules (Architectural & Product-Binding)

- **BR-601 — Provider behind the port (FR-018).** Vendor HTTP/SDK lives only in adapter
  subpackages; the `marketdata` core and the worker depend on the interface.
- **BR-602 — Last-known-good integrity.** A failed, empty, or malformed fetch never
  overwrites a good row; upserts are idempotent + transactional (PRD Reliability NFR).
- **BR-603 — Market data is global, not per-user.** No `user_id` scoping here (contrast
  SPEC-003); identity-from-context does not apply to shared reference data.
- **BR-604 — Money is never `float64`.** Price and dividends are `int64` centavos; dividend
  yield, SELIC, CDI, IPCA, and P/VP are integers in **basis points**; all rounding goes
  through `internal/platform/money` (half-up), keeping facts reproducible (PRD §6).
- **BR-605 — Facts are computed/ingested, not generated.** This data is the source of truth
  the Fact Builder (SPEC-104) hands to the `Insighter`; the LLM never invents these numbers
  (upholds SPEC-005 BR-502 upstream).
- **BR-606 — Zero cost & swappable.** Only free/public sources (BCB SGS; a free FII source);
  every provider is config-swappable and **never charges silently** (ADR-0003).
- **BR-607 — Decoupled & deterministic.** Ingestion runs off the request path, on the
  injected `Clock` in UTC, so projections and tests are deterministic.
- **BR-608 — No secrets/PII in telemetry.** The MVP sources need no API key; any future
  provider token is a secret (masked, never logged). No user data is ever sent to a provider
  (only public tickers / series codes).

---

## 6. Domain Model

### Entity: FIIQuote (snapshot per ticker)

| Field                  | Type      | Description                                  |
| ---------------------- | --------- | -------------------------------------------- |
| ticker                 | Ticker    | B3 FII ticker (value object, validated)      |
| price_centavos         | int64     | Current price, minor units                   |
| dividend_yield_bps     | int       | Trailing dividend yield, basis points        |
| p_vp_bps               | int       | Price / book ratio ×10000 (bps of the ratio) |
| sector                 | Sector    | Logistics / Offices / Shopping / Hybrid / Paper / Other |
| last_dividend_centavos | int64     | Last distribution per share, minor units     |
| last_dividend_date     | date?     | Date of that distribution (nullable)         |
| source                 | string    | Provider id that produced the row            |
| observed_at            | timestamp | Source as-of instant (UTC)                   |
| fetched_at             | timestamp | When ingested (UTC)                          |

### Entity: MacroIndicator (time series)

| Field          | Type      | Description                                          |
| -------------- | --------- | ---------------------------------------------------- |
| indicator      | Indicator | SELIC / IPCA / CDI / IFIX                             |
| value          | int64     | Rates in **bps**; IFIX as index level (documented unit) |
| unit           | Unit      | `bps` (rates) or `points` (IFIX)                     |
| reference_date | date      | Date the source attributes the value to              |
| source         | string    | Provider id (e.g. `bcb-sgs`)                         |
| fetched_at     | timestamp | When ingested (UTC)                                  |

### Value objects

- **Ticker** — parses/validates a B3 ticker in its constructor (parse-don't-validate); an
  invalid ticker is unrepresentable. May be promoted to a shared package when SPEC-102 also
  needs it.
- **Sector**, **Indicator**, **Unit** — closed enums validated on construction; an unknown
  provider sector string normalizes to `Other` (raw value retained for debugging).

---

## 7. Provider & Repository Ports

```go
// MarketDataProvider reads current data from an external source (FR-018 seam). FII reads
// are batched because the MVP source (Fundamentus) returns every FII in one request; a
// ticker absent from the returned map is a per-item miss (worker keeps last-known-good).
type MarketDataProvider interface {
    FetchFIIQuotes(ctx context.Context, tickers []Ticker) (map[Ticker]FIIQuote, error)
    FetchMacroIndicator(ctx context.Context, ind Indicator) (MacroIndicator, error)
}

type FIIQuoteRepository interface {
    UpsertFIIQuote(ctx context.Context, q FIIQuote) error
    GetFIIQuoteByTicker(ctx context.Context, t Ticker) (FIIQuote, error) // ErrFIIQuoteNotFound
}

type MacroRepository interface {
    UpsertMacroIndicator(ctx context.Context, m MacroIndicator) error
    GetLatestMacroIndicator(ctx context.Context, ind Indicator) (MacroIndicator, error) // ErrMacroNotFound
}

// TickerSource supplies the tickers to refresh. MVP: a configured watchlist; a
// holdings-backed implementation arrives with SPEC-102.
type TickerSource interface {
    Tickers(ctx context.Context) ([]Ticker, error)
}
```

> Sentinel reads (`ErrFIIQuoteNotFound`, `ErrMacroNotFound`) follow the project's repository
> conventions; identity is **not** part of any key (BR-603).

---

## 8. Data Storage

`migrations/0003_market_data.up.sql` / `.down.sql` (paired, embedded, applied manually):

- **`fii_quotes`** — PK `ticker`; `price_centavos bigint`, `dividend_yield_bps int`,
  `p_vp_bps int`, `sector text`, `last_dividend_centavos bigint`,
  `last_dividend_date date null`, `source text`, `observed_at timestamptz`,
  `fetched_at timestamptz`. Upsert on `ticker`.
- **`macro_indicators`** — PK `(indicator, reference_date)`; `value bigint`, `unit text`,
  `source text`, `fetched_at timestamptz`. Upsert on the PK keeps an idempotent series; a
  `GetLatest` reads the newest `reference_date` per indicator.

No `user_id` columns (BR-603). All money is `bigint` minor units; all rates integer bps — no
floating-point columns.

---

## 9. Edge Cases

| Scenario | Expected behavior |
| -------- | ----------------- |
| Provider returns a partial payload (missing P/VP) | Store valid fields; mark the missing one absent; do not fail the row. |
| Ticker unknown at the provider | Log + meter, skip upsert, keep last-known-good (row may go stale). |
| Provider 429 / rate-limit | Backoff, no charge, no write; degrade; retry next cycle (FR-610). |
| Malformed / oversized response body | Reject (cap with `io.LimitReader`, per SPEC-005 hardening); keep good data. |
| Fundamentus HTML table layout changes | Parser yields no rows / fails validation → degrade, keep last-known-good, log + meter; never write garbage. |
| Yahoo last-dividend lookup fails for a ticker | Store the Fundamentus fundamentals without last-dividend (partial), not a failed row. |
| Unknown sector string | Normalize to `Other`, retain raw for debugging. |
| Overlapping / duplicate runs | Idempotent upserts make a re-run a no-op on data. |
| Macro publish lag (BCB) | `reference_date` ≠ `fetched_at`; series PK dedupes re-fetches of the same date. |
| Empty watchlist (pre-SPEC-102) | Macro still ingests; FII loop is a clean no-op. |

---

## 10. Security Considerations

- **Secrets** — any provider API token comes from env only, never committed; masked by the
  `Config` `slog.LogValuer` and never placed in a span/error (SPEC-005 hardening carried).
- **Egress / privacy** — only public tickers and series codes leave the system; **no user
  data** is sent to any provider.
- **Input validation** — tickers/series codes are value objects validated before they reach a
  URL path, preventing request-forgery via a crafted identifier.
- **Transport** — base URLs are validated as `http(s)` at load; hosted providers use TLS.
- **No new auth surface** — `cmd/ingest` is an operator/cron tool; if an HTTP trigger is ever
  added it must sit behind the deny-by-default auth middleware (SPEC-003).
- **Respectful scraping** — since the FII source is scraped (Fundamentus), the adapter sends a
  descriptive `User-Agent`, runs on a low daily cadence (one bulk request), and caches per
  run; it must not hammer the source. This is an MVP constraint flagged for review if the
  product is ever served beyond the author.

---

## 11. Observability

- **Metrics** — `ingestion_runs_total{outcome}`, `ingestion_items_total{kind,outcome}`,
  `marketdata_freshness_seconds{kind}` (gauge), provider request duration.
- **Traces** — one span per ingestion run → child span per provider call
  (`provider`, `kind`, `outcome`; never the token), via the `observability.Tracer()` seam.
- **Logs** — structured per-item outcome with the trace id; no secrets, no payloads beyond
  ticker/indicator identifiers.

---

## 12. Testing Strategy

### Unit Tests

- Value objects: `Ticker`, `Sector`, `Indicator` parsing (valid/invalid), money/bps mapping.
- Provider adapters via `httptest`: success, partial payload, 404, 429→degrade, malformed,
  body-cap.
- Worker with fakes: idempotency, last-known-good preserved on per-item failure, partial-run
  success, freshness/staleness with a fake `Clock`.

### Integration Tests (gated)

- Real Postgres (`TEST_DATABASE_URL`, `-p 1`): upsert idempotency + `0003` up/down
  round-trip.
- Optional live BCB-SGS / FII fetch behind an env flag (skips cleanly in CI, like the
  live-Ollama test).

### Quality gate

`task vet`, `task test:short`, `gofmt`-clean; integration serialized when a DB is present.

---

## 13. Definition of Done

- [ ] FR-601…FR-612 implemented; acceptance criteria met.
- [ ] Hexagonal layering intact (port in core, HTTP in adapters), money/bps conventions, `%w`
      errors + sentinels, `Clock` over `time.Now()`, doc comments cite SPEC/BR.
- [ ] `0003_market_data` up/down tested; idempotent, last-known-good-safe upserts proven.
- [ ] Unit + gated integration tests green; quality gate clean.
- [ ] hexagonal-reviewer + go-correctness-reviewer pass; blocking findings fixed.
- [ ] Working-agreement closeout: `CHANGELOG` updated, `README` + `.env.example` updated,
      SPEC + PLAN flipped to **Done**, indexes updated, PT-BR lesson produced.

---

## 14. Decisions (proposed — confirm in review)

| # | Decision | Recommendation |
| - | -------- | -------------- |
| D1 | One `MarketDataProvider` port (FII + macro) vs two small ports | **One conceptual port** with two read methods (matches the PRD name); adapters may still differ per source internally. |
| D2 | Scheduling model | **`cmd/ingest` one-shot (cron-driven) as the decoupled primary**, plus an optional in-process interval scheduler toggled by `MARKETDATA_SCHEDULER_ENABLED` (default on for single-instance MVP). Avoids duplicate ingestion when horizontally scaled. |
| D3 | Ticker source before SPEC-102 exists | **Configured `MARKETDATA_WATCHLIST`** now via `TickerSource`; swap to a holdings-backed source when SPEC-102 lands. |
| D4 | FII provider + macro source | **Resolved.** FII via a **composite of Fundamentus (bulk: price/DY/P-VP/segment) + Yahoo Finance `.SA` (last dividend)** — both free, no key — behind the port, **swappable later** to a cleaner/licensed source (brapi Pro ~R$116/mo) without code changes. (brapi's free tier no longer exposes P/VP or FII segment, so it is not the MVP source.) Macro via **BCB SGS** (SELIC/CDI/IPCA, free/public); IFIX source configurable. |
| D5 | Macro storage shape | **Time series** (`indicator, reference_date`) — cheap, useful for projections/charts; expose `GetLatest`. FII stays a current **snapshot**. |
| D6 | P/VP & ratio representation | Integer **basis points of the ratio** (×10000), consistent with the bps convention. |

---

## 15. Open Questions (deferred, not blocking)

- **Scraping robustness** — Fundamentus serves an HTML table (no official API), so a markup
  change can break parsing. Mitigated by the port (swap to a licensed source anytime), the
  `Fake` default, fixture-based parser tests, and graceful degradation (a parse failure keeps
  last-known-good). Confirm the exact Fundamentus columns / Yahoo dividend endpoint shape in
  early Phase 3.
- IFIX free source (B3 vs the FII provider's index endpoint).
- Whether a future HTTP trigger / admin endpoint for on-demand ingestion is worth adding
  (kept to `cmd/ingest` for MVP).
- FII dividend **history** (for projections) — snapshot-only for now; revisit with SPEC-107.
- When to graduate to a **licensed FII source** (brapi Pro) if scraping proves too brittle —
  a config swap, tracked as technical debt.
