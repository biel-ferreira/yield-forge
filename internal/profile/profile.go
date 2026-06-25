package profile

import (
	"errors"
	"time"
)

// ErrProfileNotFound is returned by a read when the user has no profile yet — a distinct
// signal from a real error, so consumers can treat "not set" gracefully (SPEC-101 FR-1015).
var ErrProfileNotFound = errors.New("profile not set")

// Profile is an investor's risk profile, objectives, and horizon (SPEC-101 FR-003). It is
// keyed by UserID — one profile per user, global to nothing else — and that UserID always
// comes from the authenticated context, never request input (BR-1012). There is no money
// and no AI output here, so the explainability/non-advice gates do not apply (BR-1016).
type Profile struct {
	UserID     string
	Risk       RiskProfile
	Objectives []Objective
	Horizon    Horizon
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
