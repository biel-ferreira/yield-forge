package http

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/biel-ferreira/yield-forge/internal/auth"
	"github.com/biel-ferreira/yield-forge/internal/marketdata"
	"github.com/biel-ferreira/yield-forge/internal/portfolio"
)

// callerID returns the authenticated user id from the request context, writing a 401 and
// returning ok=false when the request is unauthenticated. Identity is only ever read from
// the context the middleware set, never from request input (BR-1021).
func callerID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id, ok := auth.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return "", false
	}
	return id, true
}

// PortfolioService is the slice of the portfolio service the transport needs. Consumer-defined
// here so handlers stay testable with a small fake; *portfolio.Service satisfies it (SPEC-102).
type PortfolioService interface {
	CreateFIIHolding(ctx context.Context, userID string, in portfolio.FIIInput) (portfolio.FIIHolding, error)
	ListFIIHoldings(ctx context.Context, userID string) ([]portfolio.FIIHolding, error)
	UpdateFIIHolding(ctx context.Context, userID, id string, in portfolio.FIIInput) (portfolio.FIIHolding, error)
	DeleteFIIHolding(ctx context.Context, userID, id string) error

	CreateFixedIncomeHolding(ctx context.Context, userID string, in portfolio.FixedIncomeInput) (portfolio.FixedIncomeHolding, error)
	ListFixedIncomeHoldings(ctx context.Context, userID string) ([]portfolio.FixedIncomeHolding, error)
	UpdateFixedIncomeHolding(ctx context.Context, userID, id string, in portfolio.FixedIncomeInput) (portfolio.FixedIncomeHolding, error)
	DeleteFixedIncomeHolding(ctx context.Context, userID, id string) error
	ReconcileFixedIncomeHolding(ctx context.Context, userID, id string, confirmedInterestCentavos, contributionCentavos int64) (portfolio.FixedIncomeHolding, error)
}

// holdingsHandler serves the /holdings endpoints.
type holdingsHandler struct {
	service PortfolioService
	logger  *slog.Logger
}

const dateLayout = "2006-01-02" // date-only wire format for maturity_date

// --- DTOs (money crosses the wire as integer centavos / bps, never a float — FR-1027/BR-1022) ---

type fiiHoldingRequest struct {
	Ticker               string `json:"ticker"`
	Quantity             int    `json:"quantity"`
	AveragePriceCentavos int64  `json:"average_price_centavos"`
}

type fiiHoldingResponse struct {
	ID                   string    `json:"id"`
	Ticker               string    `json:"ticker"`
	Quantity             int       `json:"quantity"`
	AveragePriceCentavos int64     `json:"average_price_centavos"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type fixedIncomeRequest struct {
	Name                   string  `json:"name"`
	Institution            string  `json:"institution"`
	InvestedAmountCentavos int64   `json:"invested_amount_centavos"`
	AnnualRateBps          int     `json:"annual_rate_bps"`
	IndexerType            string  `json:"indexer_type"`  // "" defaults to prefixado (SPEC-109 BR-1093)
	MaturityDate           *string `json:"maturity_date"` // "YYYY-MM-DD" or null
	LiquidityType          string  `json:"liquidity_type"`
}

type fixedIncomeResponse struct {
	ID                     string `json:"id"`
	Name                   string `json:"name"`
	Institution            string `json:"institution"`
	InvestedAmountCentavos int64  `json:"invested_amount_centavos"`
	// TotalContributedCentavos is new (SPEC-110): the lifetime cost basis, distinct from
	// InvestedAmountCentavos once a holding has been reconciled at least once (FR-1101).
	TotalContributedCentavos int64  `json:"total_contributed_centavos"`
	AnnualRateBps            int    `json:"annual_rate_bps"`
	IndexerType              string `json:"indexer_type"`
	// EffectiveAnnualRateBps is computed, never persisted (SPEC-109 FR-1092): the resolved
	// current rate for cdi_percentual/ipca_spread holdings; equal to AnnualRateBps for prefixado.
	EffectiveAnnualRateBps int `json:"effective_annual_rate_bps"`
	// EstimatedInterestCentavos/ReconciliationDue are computed, never persisted (SPEC-110
	// FR-1103/FR-1105) — the pre-fill hint for reconciliation and the staleness signal.
	EstimatedInterestCentavos int64     `json:"estimated_interest_centavos"`
	ReconciliationDue         bool      `json:"reconciliation_due"`
	MaturityDate              *string   `json:"maturity_date"`
	LiquidityType             string    `json:"liquidity_type"`
	LastReconciledAt          time.Time `json:"last_reconciled_at"`
	CreatedAt                 time.Time `json:"created_at"`
	UpdatedAt                 time.Time `json:"updated_at"`
}

// reconcileFixedIncomeRequest is SPEC-110 FR-1103's reconciliation body: both amounts are
// additive (never a replacement), and contribution may be zero (pure interest confirmation).
type reconcileFixedIncomeRequest struct {
	ConfirmedInterestCentavos int64 `json:"confirmed_interest_centavos"`
	ContributionCentavos      int64 `json:"contribution_centavos"`
}

// --- FII handlers ---

func (h holdingsHandler) createFIIHolding(w http.ResponseWriter, r *http.Request) {
	userID, ok := callerID(w, r)
	if !ok {
		return
	}
	var req fiiHoldingRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	holding, err := h.service.CreateFIIHolding(r.Context(), userID, fiiInput(req))
	if h.writeHoldingError(w, r, err) {
		return
	}
	writeJSON(w, http.StatusCreated, toFIIResponse(holding))
}

func (h holdingsHandler) listFIIHoldings(w http.ResponseWriter, r *http.Request) {
	userID, ok := callerID(w, r)
	if !ok {
		return
	}
	holdings, err := h.service.ListFIIHoldings(r.Context(), userID)
	if h.writeHoldingError(w, r, err) {
		return
	}
	out := make([]fiiHoldingResponse, 0, len(holdings))
	for _, hd := range holdings {
		out = append(out, toFIIResponse(hd))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h holdingsHandler) updateFIIHolding(w http.ResponseWriter, r *http.Request) {
	userID, ok := callerID(w, r)
	if !ok {
		return
	}
	var req fiiHoldingRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	holding, err := h.service.UpdateFIIHolding(r.Context(), userID, r.PathValue("id"), fiiInput(req))
	if h.writeHoldingError(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, toFIIResponse(holding))
}

func (h holdingsHandler) deleteFIIHolding(w http.ResponseWriter, r *http.Request) {
	userID, ok := callerID(w, r)
	if !ok {
		return
	}
	if h.writeHoldingError(w, r, h.service.DeleteFIIHolding(r.Context(), userID, r.PathValue("id"))) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Fixed income handlers ---

func (h holdingsHandler) createFixedIncomeHolding(w http.ResponseWriter, r *http.Request) {
	userID, ok := callerID(w, r)
	if !ok {
		return
	}
	var req fixedIncomeRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	in, err := fixedIncomeInput(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid maturity_date (want YYYY-MM-DD)")
		return
	}
	holding, err := h.service.CreateFixedIncomeHolding(r.Context(), userID, in)
	if h.writeHoldingError(w, r, err) {
		return
	}
	writeJSON(w, http.StatusCreated, toFixedIncomeResponse(holding))
}

func (h holdingsHandler) listFixedIncomeHoldings(w http.ResponseWriter, r *http.Request) {
	userID, ok := callerID(w, r)
	if !ok {
		return
	}
	holdings, err := h.service.ListFixedIncomeHoldings(r.Context(), userID)
	if h.writeHoldingError(w, r, err) {
		return
	}
	out := make([]fixedIncomeResponse, 0, len(holdings))
	for _, hd := range holdings {
		out = append(out, toFixedIncomeResponse(hd))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h holdingsHandler) updateFixedIncomeHolding(w http.ResponseWriter, r *http.Request) {
	userID, ok := callerID(w, r)
	if !ok {
		return
	}
	var req fixedIncomeRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	in, err := fixedIncomeInput(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid maturity_date (want YYYY-MM-DD)")
		return
	}
	holding, err := h.service.UpdateFixedIncomeHolding(r.Context(), userID, r.PathValue("id"), in)
	if h.writeHoldingError(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, toFixedIncomeResponse(holding))
}

func (h holdingsHandler) deleteFixedIncomeHolding(w http.ResponseWriter, r *http.Request) {
	userID, ok := callerID(w, r)
	if !ok {
		return
	}
	if h.writeHoldingError(w, r, h.service.DeleteFixedIncomeHolding(r.Context(), userID, r.PathValue("id"))) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// reconcileFixedIncomeHolding confirms interest and/or reports a new contribution for the
// caller's holding (SPEC-110 FR-1103), distinct from the plain-edit updateFixedIncomeHolding —
// keeping "correct a typo" and "confirm this month's interest" semantically separate is exactly
// what fixes the accrual-clock bug this spec exists to resolve (FR-1102, SPEC-110 D4).
func (h holdingsHandler) reconcileFixedIncomeHolding(w http.ResponseWriter, r *http.Request) {
	userID, ok := callerID(w, r)
	if !ok {
		return
	}
	var req reconcileFixedIncomeRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	holding, err := h.service.ReconcileFixedIncomeHolding(r.Context(), userID, r.PathValue("id"),
		req.ConfirmedInterestCentavos, req.ContributionCentavos)
	if h.writeHoldingError(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, toFixedIncomeResponse(holding))
}

// --- mapping + error helpers ---

func fiiInput(req fiiHoldingRequest) portfolio.FIIInput {
	return portfolio.FIIInput{Ticker: req.Ticker, Quantity: req.Quantity, AveragePriceCentavos: req.AveragePriceCentavos}
}

func fixedIncomeInput(req fixedIncomeRequest) (portfolio.FixedIncomeInput, error) {
	maturity, err := parseMaturityDate(req.MaturityDate)
	if err != nil {
		return portfolio.FixedIncomeInput{}, err
	}
	return portfolio.FixedIncomeInput{
		Name: req.Name, Institution: req.Institution,
		InvestedAmountCentavos: req.InvestedAmountCentavos, AnnualRateBps: req.AnnualRateBps,
		IndexerType: req.IndexerType, MaturityDate: maturity, LiquidityType: req.LiquidityType,
	}, nil
}

func toFIIResponse(h portfolio.FIIHolding) fiiHoldingResponse {
	return fiiHoldingResponse{
		ID: h.ID, Ticker: h.Ticker.String(), Quantity: h.Quantity.Value(),
		AveragePriceCentavos: h.AveragePriceCentavos, CreatedAt: h.CreatedAt, UpdatedAt: h.UpdatedAt,
	}
}

func toFixedIncomeResponse(h portfolio.FixedIncomeHolding) fixedIncomeResponse {
	return fixedIncomeResponse{
		ID: h.ID, Name: h.Name, Institution: h.Institution,
		InvestedAmountCentavos: h.InvestedAmountCentavos, TotalContributedCentavos: h.TotalContributedCentavos,
		AnnualRateBps: h.AnnualRateBps, IndexerType: string(h.IndexerType), EffectiveAnnualRateBps: h.EffectiveAnnualRateBps,
		EstimatedInterestCentavos: h.EstimatedInterestCentavos, ReconciliationDue: h.ReconciliationDue,
		MaturityDate: formatDate(h.MaturityDate), LiquidityType: string(h.LiquidityType),
		LastReconciledAt: h.LastReconciledAt, CreatedAt: h.CreatedAt, UpdatedAt: h.UpdatedAt,
	}
}

func parseMaturityDate(s *string) (*time.Time, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	d, err := time.Parse(dateLayout, *s)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func formatDate(d *time.Time) *string {
	if d == nil {
		return nil
	}
	s := d.Format(dateLayout)
	return &s
}

// writeHoldingError maps a service error to its status code and returns true when it wrote a
// response. Validation sentinels → 400; not-found/unowned → 404; anything else → 500.
func (h holdingsHandler) writeHoldingError(w http.ResponseWriter, r *http.Request, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, portfolio.ErrHoldingNotFound):
		writeError(w, http.StatusNotFound, "holding not found")
	case errors.Is(err, marketdata.ErrInvalidTicker):
		writeError(w, http.StatusBadRequest, "invalid ticker")
	case errors.Is(err, portfolio.ErrInvalidQuantity):
		writeError(w, http.StatusBadRequest, "quantity must be a positive whole number")
	case errors.Is(err, portfolio.ErrNegativeAmount):
		// Shared across contexts (FII average_price_centavos, SPEC-110 reconcile amounts) — the
		// message stays field-agnostic rather than assuming which field violated it.
		writeError(w, http.StatusBadRequest, "amount must not be negative")
	case errors.Is(err, portfolio.ErrEmptyField):
		writeError(w, http.StatusBadRequest, "name and institution are required")
	case errors.Is(err, portfolio.ErrInvalidAmount):
		writeError(w, http.StatusBadRequest, "invested_amount_centavos must be positive")
	case errors.Is(err, portfolio.ErrInvalidRate):
		writeError(w, http.StatusBadRequest, "annual_rate_bps must not be negative")
	case errors.Is(err, portfolio.ErrInvalidLiquidityType):
		writeError(w, http.StatusBadRequest, "liquidity_type must be daily or at_maturity")
	case errors.Is(err, portfolio.ErrInvalidIndexer):
		writeError(w, http.StatusBadRequest, "indexer_type must be prefixado, cdi_percentual, or ipca_spread")
	case errors.Is(err, portfolio.ErrMaturityRequired):
		writeError(w, http.StatusBadRequest, "maturity_date is required for an at_maturity holding")
	case errors.Is(err, portfolio.ErrPastMaturity):
		writeError(w, http.StatusBadRequest, "maturity_date must not be in the past")
	default:
		h.logger.ErrorContext(r.Context(), "holdings request failed", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
	}
	return true
}
