package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/biel-ferreira/yield-forge/internal/insight"
	"github.com/biel-ferreira/yield-forge/internal/rebalancing"
)

type fakeRebalancingEngine struct {
	gotUserID        string
	gotContribution  int64
	gotIncludeShares bool
	result           rebalancing.Rebalancing
	err              error
}

func (f *fakeRebalancingEngine) Rebalance(_ context.Context, userID string, c rebalancing.Contribution, opts rebalancing.Options) (rebalancing.Rebalancing, error) {
	f.gotUserID = userID
	f.gotContribution = c.Centavos()
	f.gotIncludeShares = opts.IncludeAssetShares
	return f.result, f.err
}

func newRebalancingHandler(svc RebalancingEngine) rebalancingHandler {
	return rebalancingHandler{service: svc, logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
}

func sampleRebalancing() rebalancing.Rebalancing {
	return rebalancing.Rebalancing{
		Areas: []rebalancing.Area{
			{Class: "fii", SuggestedShareBps: 5000, SuggestedAmountCentavos: 25_000, Title: "FIIs", Explanation: "porque ..."},
			{Class: "fixed_income", SuggestedShareBps: 5000, SuggestedAmountCentavos: 25_000, Title: "Renda Fixa", Explanation: "porque ..."},
		},
		Candidates: []rebalancing.Candidate{{Ticker: "HGLG11", Sector: "logistics", Explanation: "sólido"}},
		Disclaimer: insight.Disclaimer,
		Available:  true,
	}
}

func postRebalancing(t *testing.T, body string) *http.Request {
	t.Helper()
	return authed(http.MethodPost, "/rebalancing", body, "u1")
}

func TestPostRebalancing_IdentityShapeAndNestedCandidates(t *testing.T) {
	svc := &fakeRebalancingEngine{result: sampleRebalancing()}
	h := newRebalancingHandler(svc)

	rec := httptest.NewRecorder()
	h.postRebalancing(rec, postRebalancing(t, `{"contribution_centavos":50000}`))

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "u1", svc.gotUserID, "identity from context")
	require.Equal(t, int64(50_000), svc.gotContribution)
	require.False(t, svc.gotIncludeShares)

	var resp rebalancingResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.True(t, resp.Available)
	require.Len(t, resp.Areas, 2)
	require.Equal(t, 5000, resp.Areas[0].SuggestedShareBps)
	require.NotEmpty(t, resp.Areas[0].Explanation, "every area carries an explanation")
	require.NotEmpty(t, resp.Disclaimer)

	// Candidates nest under the FII area; the fixed_income area has an empty (not null) list.
	require.Len(t, resp.Areas[0].Candidates, 1)
	require.Equal(t, "HGLG11", resp.Areas[0].Candidates[0].Ticker)
	require.NotNil(t, resp.Areas[1].Candidates)
	require.Empty(t, resp.Areas[1].Candidates)
}

func TestPostRebalancing_IncludeAssetSharesFlag(t *testing.T) {
	svc := &fakeRebalancingEngine{result: sampleRebalancing()}
	h := newRebalancingHandler(svc)
	rec := httptest.NewRecorder()
	h.postRebalancing(rec, postRebalancing(t, `{"contribution_centavos":50000,"include_asset_shares":true}`))
	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, svc.gotIncludeShares, "the flag reaches the engine options")
}

func TestPostRebalancing_RejectsNonPositiveAndFloat(t *testing.T) {
	for _, body := range []string{
		`{"contribution_centavos":0}`,
		`{"contribution_centavos":-100}`,
		`{"contribution_centavos":500.5}`, // float rejected at the edge (money is integer)
		`{}`,
	} {
		t.Run(body, func(t *testing.T) {
			svc := &fakeRebalancingEngine{result: sampleRebalancing()}
			h := newRebalancingHandler(svc)
			rec := httptest.NewRecorder()
			h.postRebalancing(rec, postRebalancing(t, body))
			require.Equal(t, http.StatusBadRequest, rec.Code)
			require.Empty(t, svc.gotUserID, "no engine call on a bad request")
		})
	}
}

func TestPostRebalancing_DegradedIsStill200(t *testing.T) {
	svc := &fakeRebalancingEngine{result: rebalancing.Rebalancing{Disclaimer: insight.Disclaimer, Available: false}}
	h := newRebalancingHandler(svc)
	rec := httptest.NewRecorder()
	h.postRebalancing(rec, postRebalancing(t, `{"contribution_centavos":50000}`))
	require.Equal(t, http.StatusOK, rec.Code, "an LLM outage is a 200 available:false")
	var resp rebalancingResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.False(t, resp.Available)
	require.NotNil(t, resp.Areas)
}

func TestPostRebalancing_ServiceError(t *testing.T) {
	svc := &fakeRebalancingEngine{err: errors.New("boom")}
	h := newRebalancingHandler(svc)
	rec := httptest.NewRecorder()
	h.postRebalancing(rec, postRebalancing(t, `{"contribution_centavos":50000}`))
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestPostRebalancing_Unauthenticated(t *testing.T) {
	h := newRebalancingHandler(&fakeRebalancingEngine{})
	rec := httptest.NewRecorder()
	h.postRebalancing(rec, httptest.NewRequest(http.MethodPost, "/rebalancing", strings.NewReader(`{"contribution_centavos":50000}`)))
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}
