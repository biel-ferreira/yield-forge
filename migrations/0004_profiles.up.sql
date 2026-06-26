-- 0004_profiles — investor profile (SPEC-101 FR-003).
--
-- One profile per user (user_id is the PK, BR-1011), scoped to and owned by the
-- authenticated user (BR-1012). Follows the 0001/0002 conventions: timestamptz/UTC,
-- snake_case. Objectives are a jsonb array of the closed enum (D1) — the application
-- validates each value and the non-empty rule. No money here.
CREATE TABLE profiles (
    user_id       uuid        PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    risk_profile  text        NOT NULL,
    objectives    jsonb       NOT NULL,
    horizon_years integer     NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);
