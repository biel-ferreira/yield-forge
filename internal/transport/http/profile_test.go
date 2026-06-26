package http

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/biel-ferreira/yield-forge/internal/auth"
	"github.com/biel-ferreira/yield-forge/internal/platform/buildinfo"
	"github.com/biel-ferreira/yield-forge/internal/profile"
)

// fakeProfileService records the userID it was called with so we can assert that identity
// comes from the context, not the request body.
type fakeProfileService struct {
	gotUserID string
	setResult profile.Profile
	setErr    error
	getResult profile.Profile
	getErr    error
}

func (f *fakeProfileService) SetProfile(_ context.Context, userID string, _ profile.SetProfileInput) (profile.Profile, error) {
	f.gotUserID = userID
	return f.setResult, f.setErr
}

func (f *fakeProfileService) GetProfile(_ context.Context, userID string) (profile.Profile, error) {
	f.gotUserID = userID
	return f.getResult, f.getErr
}

func newProfileHandler(svc ProfileService) profileHandler {
	return profileHandler{service: svc, logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
}

func sampleProfile(userID string) profile.Profile {
	h, _ := profile.ParseHorizon(10)
	now := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	return profile.Profile{
		UserID:     userID,
		Risk:       profile.RiskModerate,
		Objectives: []profile.Objective{profile.ObjectiveRetirement, profile.ObjectivePassiveIncome},
		Horizon:    h,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// authed returns a request whose context carries an authenticated user (as the middleware
// would set it), so handler tests exercise identity-from-context directly.
func authed(method, target, body, userID string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	return req.WithContext(auth.WithUserID(req.Context(), userID))
}

func TestPutProfile_UsesContextIdentity(t *testing.T) {
	svc := &fakeProfileService{setResult: sampleProfile("u1")}
	h := newProfileHandler(svc)

	body := `{"risk_profile":"moderate","objectives":["retirement","passive_income"],"horizon_years":10}`
	rec := httptest.NewRecorder()
	h.putProfile(rec, authed(http.MethodPut, "/profile", body, "u1"))

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "u1", svc.gotUserID, "the service is called with the context user_id")

	var resp profileResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, "moderate", resp.RiskProfile)
	require.Equal(t, []string{"retirement", "passive_income"}, resp.Objectives)
	require.Equal(t, 10, resp.HorizonYears)
}

func TestPutProfile_BodyUserIDRejected(t *testing.T) {
	svc := &fakeProfileService{setResult: sampleProfile("u1")}
	h := newProfileHandler(svc)

	// A client trying to smuggle an identity in the body: DisallowUnknownFields rejects it,
	// so there is no way to supply user_id (BR-1012).
	body := `{"user_id":"hacker","risk_profile":"moderate","objectives":["retirement"],"horizon_years":10}`
	rec := httptest.NewRecorder()
	h.putProfile(rec, authed(http.MethodPut, "/profile", body, "u1"))

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Empty(t, svc.gotUserID, "the service is never reached when the body is rejected")
}

func TestPutProfile_ValidationErrors(t *testing.T) {
	cases := map[string]struct {
		err  error
		body string
	}{
		"bad risk":      {profile.ErrInvalidRiskProfile, `{"risk_profile":"risky","objectives":["retirement"],"horizon_years":10}`},
		"no objectives": {profile.ErrNoObjectives, `{"risk_profile":"moderate","objectives":[],"horizon_years":10}`},
		"bad objective": {profile.ErrInvalidObjective, `{"risk_profile":"moderate","objectives":["yolo"],"horizon_years":10}`},
		"bad horizon":   {profile.ErrInvalidHorizon, `{"risk_profile":"moderate","objectives":["retirement"],"horizon_years":0}`},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			svc := &fakeProfileService{setErr: c.err}
			h := newProfileHandler(svc)
			rec := httptest.NewRecorder()
			h.putProfile(rec, authed(http.MethodPut, "/profile", c.body, "u1"))
			require.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}
}

func TestGetProfile_FoundAndNotFound(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		svc := &fakeProfileService{getResult: sampleProfile("u1")}
		h := newProfileHandler(svc)
		rec := httptest.NewRecorder()
		h.getProfile(rec, authed(http.MethodGet, "/profile", "", "u1"))
		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "u1", svc.gotUserID)
	})

	t.Run("not set", func(t *testing.T) {
		svc := &fakeProfileService{getErr: profile.ErrProfileNotFound}
		h := newProfileHandler(svc)
		rec := httptest.NewRecorder()
		h.getProfile(rec, authed(http.MethodGet, "/profile", "", "u1"))
		require.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestProfile_Unauthenticated(t *testing.T) {
	h := newProfileHandler(&fakeProfileService{})
	// No auth.WithUserID on the context.
	rec := httptest.NewRecorder()
	h.getProfile(rec, httptest.NewRequest(http.MethodGet, "/profile", nil))
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

// TestHTTP_ProfileSpanRouteNamed exercises auth → profile end to end and verifies the server
// span is named by the matched route and carries no profile values (SPEC-101 FR-1018).
func TestHTTP_ProfileSpanRouteNamed(t *testing.T) {
	exp := spanRecorder(t)
	user := auth.User{ID: "u1", Email: "me@example.com"}
	router := NewRouter(Deps{
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		Build:      buildinfo.Info{},
		Ready:      fakePinger{},
		Auth:       fakeAuth{authUser: user},
		Profile:    &fakeProfileService{setResult: sampleProfile("u1")},
		CookieName: "yf_session",
		SessionTTL: time.Hour,
	})

	body := `{"risk_profile":"moderate","objectives":["retirement"],"horizon_years":10}`
	rr := doReq(router, http.MethodPut, "/profile", body, &http.Cookie{Name: "yf_session", Value: "tok"})
	require.Equal(t, http.StatusOK, rr.Code)

	spans := exp.GetSpans()
	require.Len(t, spans, 1, "one server span per request")
	require.Equal(t, "PUT /profile", spans[0].Name, "named by the matched route, not the raw path")
	for _, kv := range spans[0].Attributes {
		v := kv.Value.Emit()
		require.NotContains(t, v, "moderate", "no profile values on the span")
		require.NotContains(t, v, "retirement", "no profile values on the span")
	}
}
