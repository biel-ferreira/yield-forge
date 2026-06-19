package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/biel-ferreira/yield-forge/internal/platform/clock"
)

// Service orchestrates the authentication use cases (SPEC-003 FR-301..FR-305). It
// depends only on ports (repositories, the password hasher) and the Clock, so it is
// pure application logic — unit-testable with hand-written fakes, no DB or HTTP.
type Service struct {
	users     UserRepository
	sessions  SessionRepository
	hasher    PasswordHasher
	clock     clock.Clock
	ttl       time.Duration
	dummyHash string // a real hash, compared against on unknown-email logins (BR-305)
}

// NewService builds a Service. ttl is the session lifetime (config.SessionTTL).
func NewService(users UserRepository, sessions SessionRepository, hasher PasswordHasher, clk clock.Clock, ttl time.Duration) *Service {
	s := &Service{users: users, sessions: sessions, hasher: hasher, clock: clk, ttl: ttl}
	// Precompute a dummy hash so a login for an unknown email still performs a real
	// Compare — keeping timing comparable to a wrong-password login and denying an
	// account-enumeration oracle (SPEC-003 BR-305). A failure here is non-fatal: the
	// dummy compare simply does less work.
	if h, err := hasher.Hash("yieldforge-constant-time-dummy"); err == nil {
		s.dummyHash = h
	}
	return s
}

// Register validates and creates a new user, storing only the password hash
// (SPEC-003 FR-301 / BR-302). It returns ErrEmailTaken (wrapped) on a duplicate.
func (s *Service) Register(ctx context.Context, email, password string) (User, error) {
	normalized, err := ValidateEmail(email)
	if err != nil {
		return User{}, err
	}
	if err := ValidatePassword(password); err != nil {
		return User{}, err
	}

	hash, err := s.hasher.Hash(password)
	if err != nil {
		return User{}, fmt.Errorf("register: %w", err)
	}

	user, err := s.users.CreateUser(ctx, normalized, hash)
	if err != nil {
		return User{}, fmt.Errorf("register: %w", err)
	}
	return user, nil
}

// Login verifies credentials and, on success, issues a session — returning the user
// and the RAW session token (only its hash is persisted, BR-303). Every failure path
// returns the same generic ErrInvalidCredentials, with comparable timing, so failures
// don't reveal whether an email exists (SPEC-003 FR-302 / BR-305).
func (s *Service) Login(ctx context.Context, email, password string) (User, string, error) {
	normalized := NormalizeEmail(email)

	user, err := s.users.UserByEmail(ctx, normalized)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			_ = s.hasher.Compare(s.dummyHash, password) // burn comparable time
			return User{}, "", ErrInvalidCredentials
		}
		return User{}, "", fmt.Errorf("login: %w", err)
	}

	if err := s.hasher.Compare(user.PasswordHash, password); err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			return User{}, "", ErrInvalidCredentials
		}
		return User{}, "", fmt.Errorf("login: %w", err)
	}

	token, err := NewSessionToken()
	if err != nil {
		return User{}, "", fmt.Errorf("login: %w", err)
	}
	expiresAt := s.clock.Now().Add(s.ttl)
	if _, err := s.sessions.CreateSession(ctx, user.ID, HashToken(token), expiresAt); err != nil {
		return User{}, "", fmt.Errorf("login: %w", err)
	}
	return user, token, nil
}

// Logout revokes the session identified by the raw token (hard DELETE, SPEC-003
// FR-303). An empty token or an already-gone session is a no-op.
func (s *Service) Logout(ctx context.Context, rawToken string) error {
	if rawToken == "" {
		return nil
	}
	if err := s.sessions.DeleteSession(ctx, HashToken(rawToken)); err != nil {
		return fmt.Errorf("logout: %w", err)
	}
	return nil
}

// Authenticate resolves a raw session token to its user, enforcing expiry against the
// Clock (SPEC-003 BR-307). A missing, unknown, or expired token returns
// ErrSessionNotFound; an expired session is also deleted lazily.
func (s *Service) Authenticate(ctx context.Context, rawToken string) (User, error) {
	if rawToken == "" {
		return User{}, ErrSessionNotFound
	}

	session, err := s.sessions.SessionByTokenHash(ctx, HashToken(rawToken))
	if err != nil {
		return User{}, err
	}

	if !session.ExpiresAt.After(s.clock.Now()) {
		_ = s.sessions.DeleteSession(ctx, session.TokenHash) // lazy cleanup of an expired session
		return User{}, ErrSessionNotFound
	}

	user, err := s.users.UserByID(ctx, session.UserID)
	if err != nil {
		return User{}, fmt.Errorf("authenticate: %w", err)
	}
	return user, nil
}
