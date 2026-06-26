package profile

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// fakeRepo is an in-memory ProfileRepository that mimics the DB's created_at preservation
// on upsert, so the service's update semantics are testable without Postgres.
type fakeRepo struct{ data map[string]Profile }

func newFakeRepo() *fakeRepo { return &fakeRepo{data: map[string]Profile{}} }

func (r *fakeRepo) UpsertProfile(_ context.Context, p Profile) (Profile, error) {
	if existing, ok := r.data[p.UserID]; ok {
		p.CreatedAt = existing.CreatedAt // preserved across upserts (BR-1011)
	}
	r.data[p.UserID] = p
	return p, nil
}

func (r *fakeRepo) GetProfileByUserID(_ context.Context, userID string) (Profile, error) {
	p, ok := r.data[userID]
	if !ok {
		return Profile{}, ErrProfileNotFound
	}
	return p, nil
}

type fakeClock struct{ t time.Time }

func (c *fakeClock) Now() time.Time { return c.t }

func validInput() SetProfileInput {
	return SetProfileInput{
		RiskProfile:  "moderate",
		Objectives:   []string{"retirement", "passive_income", "retirement"}, // dupe on purpose
		HorizonYears: 10,
	}
}

func TestService_SetProfile_ParsesAndDedupes(t *testing.T) {
	clk := &fakeClock{t: time.Unix(1_700_000_000, 0).UTC()}
	svc := NewService(newFakeRepo(), clk)

	got, err := svc.SetProfile(context.Background(), "u1", validInput())
	require.NoError(t, err)
	require.Equal(t, "u1", got.UserID)
	require.Equal(t, RiskModerate, got.Risk)
	require.Equal(t, []Objective{ObjectiveRetirement, ObjectivePassiveIncome}, got.Objectives, "deduped")
	require.Equal(t, 10, got.Horizon.Years())
	require.Equal(t, clk.t, got.CreatedAt)
}

func TestService_SetProfile_InvalidInputWritesNothing(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, &fakeClock{t: time.Unix(1_700_000_000, 0).UTC()})

	cases := map[string]SetProfileInput{
		"bad risk":      {RiskProfile: "risky", Objectives: []string{"retirement"}, HorizonYears: 10},
		"no objectives": {RiskProfile: "moderate", Objectives: nil, HorizonYears: 10},
		"bad objective": {RiskProfile: "moderate", Objectives: []string{"yolo"}, HorizonYears: 10},
		"bad horizon":   {RiskProfile: "moderate", Objectives: []string{"retirement"}, HorizonYears: 0},
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := svc.SetProfile(context.Background(), "u1", in)
			require.Error(t, err)
			require.Empty(t, repo.data, "nothing is written on a validation failure")
		})
	}
}

func TestService_SetProfile_UpdatePreservesCreatedAt(t *testing.T) {
	clk := &fakeClock{t: time.Unix(1_700_000_000, 0).UTC()}
	svc := NewService(newFakeRepo(), clk)

	first, err := svc.SetProfile(context.Background(), "u1", validInput())
	require.NoError(t, err)

	clk.t = clk.t.Add(48 * time.Hour) // time advances before the update
	in := validInput()
	in.RiskProfile = "aggressive"
	second, err := svc.SetProfile(context.Background(), "u1", in)
	require.NoError(t, err)

	require.Equal(t, RiskAggressive, second.Risk)
	require.Equal(t, first.CreatedAt, second.CreatedAt, "created_at preserved across updates")
	require.True(t, second.UpdatedAt.After(first.UpdatedAt), "updated_at advances")
}

func TestService_GetProfile_NotFound(t *testing.T) {
	svc := NewService(newFakeRepo(), &fakeClock{t: time.Unix(1_700_000_000, 0).UTC()})
	_, err := svc.GetProfile(context.Background(), "nobody")
	require.ErrorIs(t, err, ErrProfileNotFound)
}
