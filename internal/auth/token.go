package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

// tokenBytes is the entropy of a session token: 32 bytes = 256 bits.
const tokenBytes = 32

// NewSessionToken returns a new high-entropy session token (base64url, unpadded),
// drawn from crypto/rand. The raw token is handed to the client; only its hash
// (HashToken) is ever persisted, so a database leak yields no usable sessions
// (SPEC-003 BR-303).
func NewSessionToken() (string, error) {
	b := make([]byte, tokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate session token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// HashToken returns the SHA-256 (hex) of a raw session token. This is the value the
// server stores and looks sessions up by (SPEC-003 BR-303). SHA-256 (not bcrypt) is
// appropriate here: the token is already high-entropy, so the lookup must be fast and
// deterministic — there is nothing to brute-force.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
