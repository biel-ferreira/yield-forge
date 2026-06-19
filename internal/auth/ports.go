package auth

import (
	"context"
	"time"
)

// UserRepository persists and retrieves users. Implemented by an adapter
// (internal/auth/postgres); the auth core depends only on this interface
// (SPEC-001 BR-003 — ports defined by their consumer).
type UserRepository interface {
	// CreateUser inserts a new user and returns it (with DB-generated id/timestamps).
	// It returns ErrEmailTaken if the email already exists.
	CreateUser(ctx context.Context, email, passwordHash string) (User, error)
	// UserByEmail returns the user with the given (already normalized) email, or
	// ErrUserNotFound.
	UserByEmail(ctx context.Context, email string) (User, error)
	// UserByID returns the user with the given id, or ErrUserNotFound.
	UserByID(ctx context.Context, id string) (User, error)
}

// SessionRepository persists and retrieves sessions by their token hash. Expiry is
// decided by the service against the Clock (SPEC-003 BR-307), so lookups here are
// pure CRUD and do not filter on time.
type SessionRepository interface {
	// CreateSession stores a session and returns it (with DB-generated id/timestamps).
	CreateSession(ctx context.Context, userID, tokenHash string, expiresAt time.Time) (Session, error)
	// SessionByTokenHash returns the session for a token hash, or ErrSessionNotFound.
	SessionByTokenHash(ctx context.Context, tokenHash string) (Session, error)
	// DeleteSession removes the session with the given token hash (logout). Deleting a
	// session that does not exist is not an error.
	DeleteSession(ctx context.Context, tokenHash string) error
}

// PasswordHasher hashes and verifies passwords (SPEC-003 BR-302). The algorithm is
// swappable behind this port (bcrypt now; argon2id later) with no ripple into the
// service. Compare must run in constant time.
type PasswordHasher interface {
	Hash(password string) (string, error)
	// Compare returns nil when password matches hash, ErrInvalidCredentials otherwise.
	Compare(hash, password string) error
}
