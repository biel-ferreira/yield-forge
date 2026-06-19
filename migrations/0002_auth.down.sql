-- Reverse of 0002_auth. Drop sessions first (it references users).
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;
