-- 0003_market_data — market data baseline (SPEC-006 FR-006/FR-007).
--
-- Two tables of GLOBAL reference data — there is NO user_id here (BR-603), unlike the
-- auth tables. Follows the 0001/0002 conventions: timestamptz/UTC, snake_case. Money is
-- bigint minor units (centavos) and rates are integer basis points — never floating
-- point (BR-604), so the same inputs always reproduce the same figures (PRD §6).

-- Current snapshot per FII; upserted by ticker. A failed/empty fetch never reaches here,
-- so the row is always the last-known-good (BR-602). p_vp_bps is the price/book ratio
-- ×10000 (e.g. 0.95 -> 9500). last_dividend_date is nullable (the source may omit it).
CREATE TABLE fii_quotes (
    ticker                 text        PRIMARY KEY,
    price_centavos         bigint      NOT NULL,
    dividend_yield_bps     integer     NOT NULL,
    p_vp_bps               integer     NOT NULL,
    sector                 text        NOT NULL,
    last_dividend_centavos bigint      NOT NULL DEFAULT 0,
    last_dividend_date     date,
    source                 text        NOT NULL,
    observed_at            timestamptz NOT NULL,
    fetched_at             timestamptz NOT NULL
);

-- Macro indicators as an idempotent time series; upserted by (indicator, reference_date).
-- A GetLatest read takes the newest reference_date per indicator (the PK btree serves the
-- backward scan, so no extra index is needed). value is in `unit` (bps for rates, points
-- for IFIX).
CREATE TABLE macro_indicators (
    indicator      text        NOT NULL,
    value          bigint      NOT NULL,
    unit           text        NOT NULL,
    reference_date date        NOT NULL,
    source         text        NOT NULL,
    fetched_at     timestamptz NOT NULL,
    PRIMARY KEY (indicator, reference_date)
);
