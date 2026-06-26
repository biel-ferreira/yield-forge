package http

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/biel-ferreira/yield-forge/internal/auth"
	"github.com/biel-ferreira/yield-forge/internal/profile"
)

// ProfileService is the slice of the profile service the transport needs. Consumer-defined
// here so handlers stay testable with a small fake; *profile.Service satisfies it (SPEC-101).
type ProfileService interface {
	SetProfile(ctx context.Context, userID string, in profile.SetProfileInput) (profile.Profile, error)
	GetProfile(ctx context.Context, userID string) (profile.Profile, error)
}

// profileHandler serves the /profile endpoints.
type profileHandler struct {
	service ProfileService
	logger  *slog.Logger
}

// profileRequest is the PUT body. There is deliberately NO user_id field — identity comes
// from the authenticated session (BR-1012), and decodeJSON's DisallowUnknownFields rejects a
// stray user_id, so a client cannot supply an identity.
type profileRequest struct {
	RiskProfile  string   `json:"risk_profile"`
	Objectives   []string `json:"objectives"`
	HorizonYears int      `json:"horizon_years"`
}

// profileResponse is the public view of a profile.
type profileResponse struct {
	RiskProfile  string    `json:"risk_profile"`
	Objectives   []string  `json:"objectives"`
	HorizonYears int       `json:"horizon_years"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func toProfileResponse(p profile.Profile) profileResponse {
	objectives := make([]string, len(p.Objectives))
	for i, o := range p.Objectives {
		objectives[i] = string(o)
	}
	return profileResponse{
		RiskProfile:  string(p.Risk),
		Objectives:   objectives,
		HorizonYears: p.Horizon.Years(),
		CreatedAt:    p.CreatedAt,
		UpdatedAt:    p.UpdatedAt,
	}
}

// getProfile returns the authenticated caller's profile (SPEC-101 FR-1013). Identity comes
// from the context the middleware set, never from request input (BR-1012).
func (h profileHandler) getProfile(w http.ResponseWriter, r *http.Request) {
	id, ok := auth.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	p, err := h.service.GetProfile(r.Context(), id)
	if errors.Is(err, profile.ErrProfileNotFound) {
		writeError(w, http.StatusNotFound, "profile not set")
		return
	}
	if err != nil {
		h.logger.ErrorContext(r.Context(), "get profile failed", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, toProfileResponse(p))
}

// putProfile creates or replaces the authenticated caller's profile (SPEC-101 FR-1012).
func (h profileHandler) putProfile(w http.ResponseWriter, r *http.Request) {
	id, ok := auth.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	var req profileRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	p, err := h.service.SetProfile(r.Context(), id, profile.SetProfileInput{
		RiskProfile:  req.RiskProfile,
		Objectives:   req.Objectives,
		HorizonYears: req.HorizonYears,
	})
	switch {
	case errors.Is(err, profile.ErrInvalidRiskProfile):
		writeError(w, http.StatusBadRequest, "risk_profile must be conservative, moderate, or aggressive")
		return
	case errors.Is(err, profile.ErrNoObjectives):
		writeError(w, http.StatusBadRequest, "at least one objective is required")
		return
	case errors.Is(err, profile.ErrInvalidObjective):
		writeError(w, http.StatusBadRequest, "objectives must be retirement, passive_income, wealth_preservation, or long_term_growth")
		return
	case errors.Is(err, profile.ErrInvalidHorizon):
		writeError(w, http.StatusBadRequest,
			fmt.Sprintf("horizon_years must be between %d and %d", profile.MinHorizonYears, profile.MaxHorizonYears))
		return
	case err != nil:
		h.logger.ErrorContext(r.Context(), "set profile failed", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, toProfileResponse(p))
}
