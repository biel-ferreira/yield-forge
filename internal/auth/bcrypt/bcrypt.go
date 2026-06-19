// Package bcrypt provides a bcrypt-backed implementation of auth.PasswordHasher
// (SPEC-003 D2 / BR-302). It is an adapter: it depends on the auth core (for the
// port and sentinel errors), never the reverse. The hashing algorithm is isolated
// here so it can be swapped (e.g. for argon2id) without touching the service.
package bcrypt

import (
	"errors"
	"fmt"

	bcryptlib "golang.org/x/crypto/bcrypt"

	"github.com/biel-ferreira/yield-forge/internal/auth"
)

// DefaultCost is the bcrypt cost factor. 12 is a sensible default: slow enough to
// resist offline cracking, fast enough for interactive login (~tens to hundreds of ms).
const DefaultCost = 12

// Hasher implements auth.PasswordHasher using bcrypt.
type Hasher struct {
	cost int
}

// New returns a Hasher at the default cost.
func New() Hasher { return Hasher{cost: DefaultCost} }

// Hash returns the bcrypt hash of password (salt is generated and embedded by bcrypt).
func (h Hasher) Hash(password string) (string, error) {
	cost := h.cost
	if cost == 0 {
		cost = DefaultCost
	}
	b, err := bcryptlib.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(b), nil
}

// Compare returns nil when password matches hash, auth.ErrInvalidCredentials when it
// does not (the generic anti-enumeration error, BR-305), and a wrapped error for any
// other failure such as a malformed hash. The comparison is constant-time.
func (h Hasher) Compare(hash, password string) error {
	err := bcryptlib.CompareHashAndPassword([]byte(hash), []byte(password))
	if errors.Is(err, bcryptlib.ErrMismatchedHashAndPassword) {
		return auth.ErrInvalidCredentials
	}
	if err != nil {
		return fmt.Errorf("compare password: %w", err)
	}
	return nil
}
