package http

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/biel-ferreira/yield-forge/internal/auth"
	"github.com/biel-ferreira/yield-forge/internal/marketdata"
	"github.com/biel-ferreira/yield-forge/internal/platform/buildinfo"
	"github.com/biel-ferreira/yield-forge/internal/portfolio"
)

func holdingsRouter(svc PortfolioService) http.Handler {
	user := auth.User{ID: "u1", Email: "me@example.com"}
	return NewRouter(Deps{
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		Build:      buildinfo.Info{},
		Ready:      fakePinger{},
		Auth:       fakeAuth{authUser: user},
		Portfolio:  svc,
		CookieName: "yf_session",
		SessionTTL: time.Hour,
	})
}

// fakePortfolioService records the userID/id it was called with so we can assert identity is
// taken from the context (not the body/path), and returns configured results/errors.
type fakePortfolioService struct {
	gotUserID string
	gotID     string
	calls     int
	fiiResult portfolio.FIIHolding
	fiResult  portfolio.FixedIncomeHolding
	fiiList   []portfolio.FIIHolding
	err       error
}

func (f *fakePortfolioService) CreateFIIHolding(_ context.Context, userID string, _ portfolio.FIIInput) (portfolio.FIIHolding, error) {
	f.gotUserID, f.calls = userID, f.calls+1
	return f.fiiResult, f.err
}
func (f *fakePortfolioService) ListFIIHoldings(_ context.Context, userID string) ([]portfolio.FIIHolding, error) {
	f.gotUserID = userID
	return f.fiiList, f.err
}
func (f *fakePortfolioService) UpdateFIIHolding(_ context.Context, userID, id string, _ portfolio.FIIInput) (portfolio.FIIHolding, error) {
	f.gotUserID, f.gotID = userID, id
	return f.fiiResult, f.err
}
func (f *fakePortfolioService) DeleteFIIHolding(_ context.Context, userID, id string) error {
	f.gotUserID, f.gotID = userID, id
	return f.err
}
func (f *fakePortfolioService) CreateFixedIncomeHolding(_ context.Context, userID string, _ portfolio.FixedIncomeInput) (portfolio.FixedIncomeHolding, error) {
	f.gotUserID, f.calls = userID, f.calls+1
	return f.fiResult, f.err
}
func (f *fakePortfolioService) ListFixedIncomeHoldings(_ context.Context, userID string) ([]portfolio.FixedIncomeHolding, error) {
	f.gotUserID = userID
	return nil, f.err
}
func (f *fakePortfolioService) UpdateFixedIncomeHolding(_ context.Context, userID, id string, _ portfolio.FixedIncomeInput) (portfolio.FixedIncomeHolding, error) {
	f.gotUserID, f.gotID = userID, id
	return f.fiResult, f.err
}
func (f *fakePortfolioService) DeleteFixedIncomeHolding(_ context.Context, userID, id string) error {
	f.gotUserID, f.gotID = userID, id
	return f.err
}

func newHoldingsHandler(svc PortfolioService) holdingsHandler {
	return holdingsHandler{service: svc, logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
}

func sampleFII(userID string) portfolio.FIIHolding {
	q, _ := portfolio.ParseQuantity(100)
	now := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)
	return portfolio.FIIHolding{
		ID: "fii-1", UserID: userID, Ticker: marketdata.MustParseTicker("HGLG11"),
		Quantity: q, AveragePriceCentavos: 15_750, CreatedAt: now, UpdatedAt: now,
	}
}

func TestCreateFIIHolding_ContextIdentityAndMoney(t *testing.T) {
	svc := &fakePortfolioService{fiiResult: sampleFII("u1")}
	h := newHoldingsHandler(svc)

	body := `{"ticker":"HGLG11","quantity":100,"average_price_centavos":15750}`
	rec := httptest.NewRecorder()
	h.createFIIHolding(rec, authed(http.MethodPost, "/holdings/fii", body, "u1"))

	require.Equal(t, http.StatusCreated, rec.Code)
	require.Equal(t, "u1", svc.gotUserID, "service called with the context user_id")

	var resp fiiHoldingResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, int64(15_750), resp.AveragePriceCentavos, "money crosses the wire as integer centavos")
	require.Equal(t, "HGLG11", resp.Ticker)
}

func TestCreateFIIHolding_BodyUserIDRejected(t *testing.T) {
	svc := &fakePortfolioService{fiiResult: sampleFII("u1")}
	h := newHoldingsHandler(svc)

	body := `{"user_id":"hacker","ticker":"HGLG11","quantity":100,"average_price_centavos":15750}`
	rec := httptest.NewRecorder()
	h.createFIIHolding(rec, authed(http.MethodPost, "/holdings/fii", body, "u1"))

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Zero(t, svc.calls, "the service is never reached when the body is rejected")
}

func TestCreateFIIHolding_ValidationError(t *testing.T) {
	svc := &fakePortfolioService{err: portfolio.ErrInvalidQuantity}
	h := newHoldingsHandler(svc)
	rec := httptest.NewRecorder()
	h.createFIIHolding(rec, authed(http.MethodPost, "/holdings/fii", `{"ticker":"HGLG11","quantity":0,"average_price_centavos":1}`, "u1"))
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdateFIIHolding_OwnershipNotFound(t *testing.T) {
	svc := &fakePortfolioService{err: portfolio.ErrHoldingNotFound}
	h := newHoldingsHandler(svc)
	req := authed(http.MethodPut, "/holdings/fii/abc", `{"ticker":"HGLG11","quantity":1,"average_price_centavos":1}`, "u1")
	req.SetPathValue("id", "abc")
	rec := httptest.NewRecorder()
	h.updateFIIHolding(rec, req)
	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Equal(t, "abc", svc.gotID)
}

func TestDeleteFIIHolding_NoContent(t *testing.T) {
	svc := &fakePortfolioService{}
	h := newHoldingsHandler(svc)
	req := authed(http.MethodDelete, "/holdings/fii/xyz", "", "u1")
	req.SetPathValue("id", "xyz")
	rec := httptest.NewRecorder()
	h.deleteFIIHolding(rec, req)
	require.Equal(t, http.StatusNoContent, rec.Code)
	require.Equal(t, "xyz", svc.gotID)
}

func TestCreateFixedIncome_MaturityDateParsing(t *testing.T) {
	t.Run("valid date", func(t *testing.T) {
		svc := &fakePortfolioService{fiResult: portfolio.FixedIncomeHolding{ID: "fi-1"}}
		h := newHoldingsHandler(svc)
		body := `{"name":"CDB","institution":"Banco","invested_amount_centavos":100000,"annual_rate_bps":1200,"maturity_date":"2030-01-15","liquidity_type":"at_maturity"}`
		rec := httptest.NewRecorder()
		h.createFixedIncomeHolding(rec, authed(http.MethodPost, "/holdings/fixed-income", body, "u1"))
		require.Equal(t, http.StatusCreated, rec.Code)
	})

	t.Run("malformed date rejected before the service", func(t *testing.T) {
		svc := &fakePortfolioService{}
		h := newHoldingsHandler(svc)
		body := `{"name":"CDB","institution":"Banco","invested_amount_centavos":100000,"annual_rate_bps":1200,"maturity_date":"15/01/2030","liquidity_type":"at_maturity"}`
		rec := httptest.NewRecorder()
		h.createFixedIncomeHolding(rec, authed(http.MethodPost, "/holdings/fixed-income", body, "u1"))
		require.Equal(t, http.StatusBadRequest, rec.Code)
		require.Zero(t, svc.calls)
	})
}

func TestListFIIHoldings_OK(t *testing.T) {
	svc := &fakePortfolioService{fiiList: []portfolio.FIIHolding{sampleFII("u1")}}
	h := newHoldingsHandler(svc)
	rec := httptest.NewRecorder()
	h.listFIIHoldings(rec, authed(http.MethodGet, "/holdings/fii", "", "u1"))
	require.Equal(t, http.StatusOK, rec.Code)
	var resp []fiiHoldingResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Len(t, resp, 1)
}

func TestHoldings_Unauthenticated(t *testing.T) {
	h := newHoldingsHandler(&fakePortfolioService{})
	rec := httptest.NewRecorder()
	h.listFIIHoldings(rec, httptest.NewRequest(http.MethodGet, "/holdings/fii", nil))
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

// TestHTTP_HoldingsSpanRouteNamed runs auth → holdings end to end and verifies the span is
// route-named and carries no money/ticker values (SPEC-102 FR-1028).
func TestHTTP_HoldingsSpanRouteNamed(t *testing.T) {
	exp := spanRecorder(t)
	router := holdingsRouter(&fakePortfolioService{fiiResult: sampleFII("u1")})

	body := `{"ticker":"HGLG11","quantity":100,"average_price_centavos":15750}`
	rr := doReq(router, http.MethodPost, "/holdings/fii", body, &http.Cookie{Name: "yf_session", Value: "tok"})
	require.Equal(t, http.StatusCreated, rr.Code)

	spans := exp.GetSpans()
	require.Len(t, spans, 1)
	require.Equal(t, "POST /holdings/fii", spans[0].Name)
	for _, kv := range spans[0].Attributes {
		v := kv.Value.Emit()
		require.NotContains(t, v, "15750", "money must not leak onto the span")
		require.NotContains(t, v, "HGLG11", "holding values must not leak onto the span")
	}
}

// TestHTTP_HoldingsIDRouteIsLowCardinality verifies the {id} routes are named by the route
// pattern, not the raw UUID, so the span stays low-cardinality (SPEC-004 BR-406 / FR-1028).
func TestHTTP_HoldingsIDRouteIsLowCardinality(t *testing.T) {
	exp := spanRecorder(t)
	router := holdingsRouter(&fakePortfolioService{fiiResult: sampleFII("u1")})

	const id = "9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d"
	body := `{"ticker":"HGLG11","quantity":1,"average_price_centavos":1}`
	rr := doReq(router, http.MethodPut, "/holdings/fii/"+id, body, &http.Cookie{Name: "yf_session", Value: "tok"})
	require.Equal(t, http.StatusOK, rr.Code)

	spans := exp.GetSpans()
	require.Len(t, spans, 1)
	require.Equal(t, "PUT /holdings/fii/{id}", spans[0].Name, "named by the route pattern, not the raw id")
	require.NotContains(t, spans[0].Name, id)
}
