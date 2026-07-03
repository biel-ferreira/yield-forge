-- 0007_fixed_income_indexer — a rate indexer for fixed-income holdings (SPEC-109 FR-1091).
--
-- Additive, backward-compatible: every row written before this migration defaults to
-- 'prefixado' (the flat-rate behavior that already existed), so annual_rate_bps's meaning is
-- UNCHANGED for them. The CHECK constraint mirrors the app-level closed enum (parse-don't-
-- validate at both layers, per convention). annual_rate_bps itself is untouched — SPEC-109
-- reinterprets its meaning per indexer_type rather than adding new rate columns.

ALTER TABLE fixed_income_holdings
    ADD COLUMN indexer_type text NOT NULL DEFAULT 'prefixado'
        CHECK (indexer_type IN ('prefixado', 'cdi_percentual', 'ipca_spread'));
