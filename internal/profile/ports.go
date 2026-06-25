package profile

import "context"

// ProfileRepository persists one profile per user. Writes are an idempotent upsert keyed by
// user_id; reads are scoped to the given user_id and return ErrProfileNotFound when absent
// (SPEC-101 FR-1016, BR-1011/BR-1012). The userID always originates from the authenticated
// context — the repository never derives identity itself.
type ProfileRepository interface {
	UpsertProfile(ctx context.Context, p Profile) error
	GetProfileByUserID(ctx context.Context, userID string) (Profile, error) // ErrProfileNotFound when absent
}

// ProfileReader is the consumer-facing read port (SPEC-101 FR-1015): the seam through which
// the Insight Engine, Rebalancing Assistant, and Health Score (SPEC-104/105/106) read a
// user's profile without coupling to HTTP or SQL. The Service satisfies it.
type ProfileReader interface {
	GetProfile(ctx context.Context, userID string) (Profile, error) // ErrProfileNotFound when absent
}
