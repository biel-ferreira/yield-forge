-- Reverse of 0001_init. Drops the pgcrypto extension enabled by the up migration.
-- IF EXISTS keeps the down idempotent if run against a database that never applied up.
DROP EXTENSION IF EXISTS pgcrypto;
