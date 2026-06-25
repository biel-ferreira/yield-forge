package profile

import (
	"context"
	"fmt"

	"github.com/biel-ferreira/yield-forge/internal/platform/clock"
)

// Service is the profile application logic (SPEC-101 FR-1012/FR-1013). It depends only on
// the repository port and the Clock, so it is pure and unit-testable with hand-written
// fakes. It satisfies ProfileReader, the consumer port later analysis specs read through.
type Service struct {
	repo  ProfileRepository
	clock clock.Clock
}

var _ ProfileReader = (*Service)(nil)

// NewService builds a Service over the repository and clock.
func NewService(repo ProfileRepository, clk clock.Clock) *Service {
	return &Service{repo: repo, clock: clk}
}

// SetProfileInput is the raw, edge-validated input a handler passes in. The service parses
// it into value objects (parse-don't-validate) — the userID is supplied separately, always
// from the authenticated context (BR-1012), never from this struct.
type SetProfileInput struct {
	RiskProfile  string
	Objectives   []string
	HorizonYears int
}

// SetProfile validates the input, upserts the profile under userID, and returns the stored
// row (re-read so created_at reflects first creation on an update). A validation failure
// returns the relevant sentinel (Err*RiskProfile / Err*Objective / Err*Horizon) and writes
// nothing.
func (s *Service) SetProfile(ctx context.Context, userID string, in SetProfileInput) (Profile, error) {
	risk, err := ParseRiskProfile(in.RiskProfile)
	if err != nil {
		return Profile{}, err
	}
	objectives, err := ParseObjectives(in.Objectives)
	if err != nil {
		return Profile{}, err
	}
	horizon, err := ParseHorizon(in.HorizonYears)
	if err != nil {
		return Profile{}, err
	}

	now := s.clock.Now()
	p := Profile{
		UserID:     userID,
		Risk:       risk,
		Objectives: objectives,
		Horizon:    horizon,
		CreatedAt:  now, // ignored on update — the store preserves the original (BR-1011)
		UpdatedAt:  now,
	}
	if err := s.repo.UpsertProfile(ctx, p); err != nil {
		return Profile{}, fmt.Errorf("set profile: %w", err)
	}
	return s.repo.GetProfileByUserID(ctx, userID)
}

// GetProfile returns the user's profile or ErrProfileNotFound (ProfileReader).
func (s *Service) GetProfile(ctx context.Context, userID string) (Profile, error) {
	return s.repo.GetProfileByUserID(ctx, userID)
}
