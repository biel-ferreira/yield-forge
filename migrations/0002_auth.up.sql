-- 0002_auth — authentication baseline (SPEC-003 FR-301..FR-305).
--
-- Introduces the first feature-adjacent tables, following the 0001 conventions:
-- UUID primary keys (gen_random_uuid), timestamptz/UTC, snake_case.

CREATE TABLE users (
    id            uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    email         text        NOT NULL,
    password_hash text        NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

-- Uniqueness is case-insensitive because the application stores the email already
-- normalized (trimmed + lowercased, auth.NormalizeEmail), so a plain unique index
-- suffices (SPEC-003 FR-301).
CREATE UNIQUE INDEX users_email_key ON users (email);

CREATE TABLE sessions (
    id         uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash text        NOT NULL,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- Look up sessions by their token hash (never the raw token, SPEC-003 BR-303).
CREATE UNIQUE INDEX sessions_token_hash_key ON sessions (token_hash);
-- Support cascade deletes and per-user session queries.
CREATE INDEX sessions_user_id_idx ON sessions (user_id);
