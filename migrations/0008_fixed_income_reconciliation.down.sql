-- Reverse of 0008_fixed_income_reconciliation.
ALTER TABLE fixed_income_holdings
    DROP COLUMN total_contributed_centavos,
    DROP COLUMN last_reconciled_at;
