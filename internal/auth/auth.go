// Package auth provides authentication and per-user identity for YieldForge
// (SPEC-003). It owns the User and Session domain, the ports it depends on
// (ports.go), and the request-scoped current-user value that feature repositories
// scope their queries by (context.go, FR-306).
//
// Per the hexagonal rules (SPEC-001 BR-001/002, SPEC-002 BR-202, SPEC-003 BR-306),
// this core imports no SQL, HTTP, or vendor-SDK types: bcrypt lives in the bcrypt
// adapter subpackage and SQL in the postgres adapter; the HTTP middleware lives in
// transport/http and calls the service here.
package auth

import (
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"
)

// Domain errors. These are sentinels — check with errors.Is.
//
// ErrInvalidCredentials is deliberately generic: login returns it whether the email
// is unknown or the password is wrong, so failures don't enable account enumeration
// (SPEC-003 BR-305).
var (
	ErrEmailTaken         = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidEmail       = errors.New("invalid email")
	ErrWeakPassword       = errors.New("password too short")
	ErrPasswordTooLong    = errors.New("password too long")
	ErrSessionNotFound    = errors.New("session not found or expired")
	ErrUserNotFound       = errors.New("user not found")
)

const (
	// MinPasswordLength is the minimum accepted password length (SPEC-003 FR-301).
	MinPasswordLength = 8
	// MaxPasswordLength caps the password at bcrypt's 72-byte limit: bytes beyond it
	// are silently ignored by bcrypt, so without this two long passwords sharing a
	// 72-byte prefix would be interchangeable. Reject them at the boundary instead.
	MaxPasswordLength = 72
)

// User is an authenticated account. PasswordHash never crosses the API boundary —
// it is exposed only between the service and its persistence adapter (BR-302).
type User struct {
	ID           string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Session is a server-side login session. Only TokenHash is persisted; the raw
// token lives solely on the client (SPEC-003 BR-303).
type Session struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
}

// NormalizeEmail trims surrounding whitespace and lowercases an email so that
// uniqueness and lookups are case-insensitive (SPEC-003 FR-301).
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// ValidateEmail normalizes and syntactically validates an email, returning the
// normalized form or ErrInvalidEmail.
func ValidateEmail(email string) (string, error) {
	normalized := NormalizeEmail(email)
	if normalized == "" {
		return "", fmt.Errorf("%w: empty", ErrInvalidEmail)
	}
	if _, err := mail.ParseAddress(normalized); err != nil {
		return "", fmt.Errorf("%w: %q", ErrInvalidEmail, email)
	}
	return normalized, nil
}

// ValidatePassword enforces the password policy (SPEC-003 FR-301): at least
// MinPasswordLength and at most MaxPasswordLength bytes.
func ValidatePassword(password string) error {
	if len(password) < MinPasswordLength {
		return fmt.Errorf("%w: need at least %d characters", ErrWeakPassword, MinPasswordLength)
	}
	if len(password) > MaxPasswordLength {
		return fmt.Errorf("%w: at most %d bytes", ErrPasswordTooLong, MaxPasswordLength)
	}
	return nil
}
