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
func (f *fakePortfolioService) ReconcileFixedIncomeHolding(_ context.Context, userID, id string, _, _ int64) (portfolio.FixedIncomeHolding, error) {
	f.gotUserID, f.gotID = userID, id
	return f.fiResult, f.err
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

// sampleFixedIncome mirrors sampleFII's role for the fixed-income CRUD/span tests below.
func sampleFixedIncome(userID string) portfolio.FixedIncomeHolding {
	now := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)
	return portfolio.FixedIncomeHolding{
		ID: "fi-1", UserID: userID, Name: "CDB Banco X", Institution: "Banco X",
		InvestedAmountCentavos: 1_000_000, AnnualRateBps: 12_000,
		IndexerType: portfolio.IndexerCDIPercentual, EffectiveAnnualRateBps: 1_260,
		LiquidityType: portfolio.LiquidityDaily, CreatedAt: now, UpdatedAt: now,
	}
}

// TestCreateFixedIncome_IndexerRoundTrip proves indexer_type crosses the wire on write and both
// it and the resolved (never-accepted-on-write, SPEC-109 FR-1092) effective_annual_rate_bps come
// back on read — the fixed-income analogue of TestCreateFIIHolding_ContextIdentityAndMoney's
// money-round-trip assertion.
func TestCreateFixedIncome_IndexerRoundTrip(t *testing.T) {
	svc := &fakePortfolioService{fiResult: sampleFixedIncome("u1")}
	h := newHoldingsHandler(svc)

	body := `{"name":"CDB Banco X","institution":"Banco X","invested_amount_centavos":1000000,` +
		`"annual_rate_bps":12000,"indexer_type":"cdi_percentual","liquidity_type":"daily"}`
	rec := httptest.NewRecorder()
	h.createFixedIncomeHolding(rec, authed(http.MethodPost, "/holdings/fixed-income", body, "u1"))

	require.Equal(t, http.StatusCreated, rec.Code)
	var resp fixedIncomeResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, "cdi_percentual", resp.IndexerType)
	require.Equal(t, 1_260, resp.EffectiveAnnualRateBps, "the resolved rate is server-computed, never client-supplied")
	require.Equal(t, int64(1_000_000), resp.InvestedAmountCentavos, "money crosses the wire as integer centavos")
}

// TestCreateFixedIncome_InvalidIndexer_ValidationError proves garbage indexer_type surfaces as a
// 400 (the service's ErrInvalidIndexer sentinel is mapped in writeHoldingError), the fixed-income
// analogue of TestCreateFIIHolding_ValidationError.
func TestCreateFixedIncome_InvalidIndexer_ValidationError(t *testing.T) {
	svc := &fakePortfolioService{err: portfolio.ErrInvalidIndexer}
	h := newHoldingsHandler(svc)
	body := `{"name":"CDB","institution":"Banco","invested_amount_centavos":100000,` +
		`"annual_rate_bps":1200,"indexer_type":"garbage","liquidity_type":"daily"}`
	rec := httptest.NewRecorder()
	h.createFixedIncomeHolding(rec, authed(http.MethodPost, "/holdings/fixed-income", body, "u1"))
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

// TestReconcileFixedIncomeHolding_OK proves SPEC-110 FR-1103: the endpoint calls the service
// with the caller's context identity + path id (never trusting a client-supplied id), and the
// new fields (total_contributed_centavos, estimated_interest_centavos, reconciliation_due,
// last_reconciled_at) cross the wire.
func TestReconcileFixedIncomeHolding_OK(t *testing.T) {
	reconciled := sampleFixedIncome("u1")
	reconciled.InvestedAmountCentavos = 1_000_963
	reconciled.TotalContributedCentavos = 1_000_000
	reconciled.LastReconciledAt = reconciled.CreatedAt.Add(31 * 24 * time.Hour)
	svc := &fakePortfolioService{fiResult: reconciled}
	h := newHoldingsHandler(svc)

	body := `{"confirmed_interest_centavos":963,"contribution_centavos":0}`
	rec := httptest.NewRecorder()
	req := authed(http.MethodPost, "/holdings/fixed-income/fi-1/reconcile", body, "u1")
	req.SetPathValue("id", "fi-1")
	h.reconcileFixedIncomeHolding(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "u1", svc.gotUserID, "identity comes from context, never the body")
	require.Equal(t, "fi-1", svc.gotID, "id comes from the path")

	var resp fixedIncomeResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, int64(1_000_963), resp.InvestedAmountCentavos)
	require.Equal(t, int64(1_000_000), resp.TotalContributedCentavos, "money crosses the wire as integer centavos")
}

// TestReconcileFixedIncomeHolding_NegativeAmount_ValidationError proves a negative amount is
// rejected with a field-agnostic message (ErrNegativeAmount is shared with FII's
// average_price_centavos, so the mapped message must not assume that field).
func TestReconcileFixedIncomeHolding_NegativeAmount_ValidationError(t *testing.T) {
	svc := &fakePortfolioService{err: portfolio.ErrNegativeAmount}
	h := newHoldingsHandler(svc)
	body := `{"confirmed_interest_centavos":-1,"contribution_centavos":0}`
	rec := httptest.NewRecorder()
	req := authed(http.MethodPost, "/holdings/fixed-income/fi-1/reconcile", body, "u1")
	req.SetPathValue("id", "fi-1")
	h.reconcileFixedIncomeHolding(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.NotContains(t, rec.Body.String(), "average_price_centavos", "the message must not assume FII context")
}

// TestReconcileFixedIncomeHolding_NotFound proves an unowned/missing id maps to 404.
func TestReconcileFixedIncomeHolding_NotFound(t *testing.T) {
	svc := &fakePortfolioService{err: portfolio.ErrHoldingNotFound}
	h := newHoldingsHandler(svc)
	body := `{"confirmed_interest_centavos":100,"contribution_centavos":0}`
	rec := httptest.NewRecorder()
	req := authed(http.MethodPost, "/holdings/fixed-income/ghost/reconcile", body, "u1")
	req.SetPathValue("id", "ghost")
	h.reconcileFixedIncomeHolding(rec, req)
	require.Equal(t, http.StatusNotFound, rec.Code)
}

// TestHTTP_FixedIncomeSpanRouteNamed is TestHTTP_HoldingsSpanRouteNamed's fixed-income analogue:
// the span is route-named and leaks no money/indexer/holding values (SPEC-109 Phase 5, mirroring
// SPEC-102 FR-1028's existing no-PII-on-spans convention).
func TestHTTP_FixedIncomeSpanRouteNamed(t *testing.T) {
	exp := spanRecorder(t)
	router := holdingsRouter(&fakePortfolioService{fiResult: sampleFixedIncome("u1")})

	body := `{"name":"CDB Banco X","institution":"Banco X","invested_amount_centavos":1000000,` +
		`"annual_rate_bps":12000,"indexer_type":"cdi_percentual","liquidity_type":"daily"}`
	rr := doReq(router, http.MethodPost, "/holdings/fixed-income", body, &http.Cookie{Name: "yf_session", Value: "tok"})
	require.Equal(t, http.StatusCreated, rr.Code)

	spans := exp.GetSpans()
	require.Len(t, spans, 1)
	require.Equal(t, "POST /holdings/fixed-income", spans[0].Name)
	for _, kv := range spans[0].Attributes {
		v := kv.Value.Emit()
		require.NotContains(t, v, "1000000", "money must not leak onto the span")
		require.NotContains(t, v, "Banco X", "institution must not leak onto the span")
		require.NotContains(t, v, "cdi_percentual", "indexer type must not leak onto the span")
	}
}

// TestHTTP_ReconcileFixedIncomeSpanRouteNamed proves the new SPEC-110 endpoint's span is named
// by the route pattern (not the raw id, SPEC-004 BR-406/FR-1028) and leaks no money/holding
// values — Phase 5's deliverable for the new route (mirrors TestHTTP_FixedIncomeSpanRouteNamed).
func TestHTTP_ReconcileFixedIncomeSpanRouteNamed(t *testing.T) {
	exp := spanRecorder(t)
	router := holdingsRouter(&fakePortfolioService{fiResult: sampleFixedIncome("u1")})

	const id = "9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d"
	body := `{"confirmed_interest_centavos":963,"contribution_centavos":500000}`
	rr := doReq(router, http.MethodPost, "/holdings/fixed-income/"+id+"/reconcile", body, &http.Cookie{Name: "yf_session", Value: "tok"})
	require.Equal(t, http.StatusOK, rr.Code)

	spans := exp.GetSpans()
	require.Len(t, spans, 1)
	require.Equal(t, "POST /holdings/fixed-income/{id}/reconcile", spans[0].Name, "named by the route pattern, not the raw id")
	require.NotContains(t, spans[0].Name, id)
	for _, kv := range spans[0].Attributes {
		v := kv.Value.Emit()
		require.NotContains(t, v, "963", "money must not leak onto the span")
		require.NotContains(t, v, "500000", "money must not leak onto the span")
		require.NotContains(t, v, "CDB Banco X", "holding values must not leak onto the span")
	}
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
