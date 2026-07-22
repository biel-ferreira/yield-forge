-- 0008_fixed_income_reconciliation — separates lifetime contributions from accrued interest on
-- fixed-income holdings, and gives each holding its own accrual clock (SPEC-110 FR-1101).
--
-- Additive, backward-compatible: every row written before this migration backfills
-- total_contributed_centavos = invested_amount_centavos and last_reconciled_at = created_at —
-- correct by construction, since no reconciliation could have happened yet (BR-1103). Existing
-- Dashboard/Projections output is unchanged until a holding's first reconciliation.

ALTER TABLE fixed_income_holdings
    ADD COLUMN total_contributed_centavos bigint NOT NULL DEFAULT 0,
    ADD COLUMN last_reconciled_at timestamptz;

UPDATE fixed_income_holdings
    SET total_contributed_centavos = invested_amount_centavos,
        last_reconciled_at = created_at;

ALTER TABLE fixed_income_holdings
    ALTER COLUMN last_reconciled_at SET NOT NULL;
