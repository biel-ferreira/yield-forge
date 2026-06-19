package auth

import "context"

// ctxKey is an unexported context-key type, so keys here cannot collide with keys
// set by other packages (Go idiom for context values).
type ctxKey int

const userIDKey ctxKey = iota

// WithUserID returns a child context carrying the authenticated user's ID. The auth
// middleware sets it after resolving a valid session (SPEC-003 FR-305); handlers and
// repositories read it via UserID — never from request input (BR-304).
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// UserID returns the authenticated user's ID from ctx and whether one is present.
// A false second return means the request is unauthenticated; callers must treat
// that as an error, never as an empty-but-valid identity (SPEC-003 BR-304 / FR-306).
func UserID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(userIDKey).(string)
	if !ok || id == "" {
		return "", false
	}
	return id, true
}
