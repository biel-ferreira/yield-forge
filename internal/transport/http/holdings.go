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
	MaturityDate           *string `json:"maturity_date"` // "YYYY-MM-DD" or null
	LiquidityType          string  `json:"liquidity_type"`
}

type fixedIncomeResponse struct {
	ID                     string    `json:"id"`
	Name                   string    `json:"name"`
	Institution            string    `json:"institution"`
	InvestedAmountCentavos int64     `json:"invested_amount_centavos"`
	AnnualRateBps          int       `json:"annual_rate_bps"`
	MaturityDate           *string   `json:"maturity_date"`
	LiquidityType          string    `json:"liquidity_type"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
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
		MaturityDate: maturity, LiquidityType: req.LiquidityType,
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
		InvestedAmountCentavos: h.InvestedAmountCentavos, AnnualRateBps: h.AnnualRateBps,
		MaturityDate: formatDate(h.MaturityDate), LiquidityType: string(h.LiquidityType),
		CreatedAt: h.CreatedAt, UpdatedAt: h.UpdatedAt,
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
		writeError(w, http.StatusBadRequest, "average_price_centavos must not be negative")
	case errors.Is(err, portfolio.ErrEmptyField):
		writeError(w, http.StatusBadRequest, "name and institution are required")
	case errors.Is(err, portfolio.ErrInvalidAmount):
		writeError(w, http.StatusBadRequest, "invested_amount_centavos must be positive")
	case errors.Is(err, portfolio.ErrInvalidRate):
		writeError(w, http.StatusBadRequest, "annual_rate_bps must not be negative")
	case errors.Is(err, portfolio.ErrInvalidLiquidityType):
		writeError(w, http.StatusBadRequest, "liquidity_type must be daily or at_maturity")
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
