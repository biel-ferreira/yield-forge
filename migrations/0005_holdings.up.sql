-- 0005_holdings — portfolio holdings (SPEC-102 FR-001/FR-002).
--
-- Two tables, one per holding type (D2), each owned by a user. Follows the 0001/0002
-- conventions: UUID PKs, timestamptz/UTC, snake_case, FK ON DELETE CASCADE. Money columns
-- are bigint minor units (centavos) and the rate column integer basis points — never
-- floating point (BR-1022), so balances are exact. user_id is indexed for the per-user
-- list/scope queries; mutations are additionally scoped by (id, user_id) at the app layer.

CREATE TABLE fii_holdings (
    id                     uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    ticker                 text        NOT NULL,
    quantity               integer     NOT NULL,        -- > 0 (app-enforced)
    average_price_centavos bigint      NOT NULL,
    created_at             timestamptz NOT NULL DEFAULT now(),
    updated_at             timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX fii_holdings_user_id_idx ON fii_holdings (user_id);

CREATE TABLE fixed_income_holdings (
    id                       uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                  uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name                     text        NOT NULL,
    institution              text        NOT NULL,
    invested_amount_centavos bigint      NOT NULL,
    annual_rate_bps          integer     NOT NULL,
    maturity_date            date,                        -- NULL for daily-liquidity holdings
    liquidity_type           text        NOT NULL,
    created_at               timestamptz NOT NULL DEFAULT now(),
    updated_at               timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX fixed_income_holdings_user_id_idx ON fixed_income_holdings (user_id);
