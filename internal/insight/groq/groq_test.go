package groq_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/biel-ferreira/yield-forge/internal/insight"
	"github.com/biel-ferreira/yield-forge/internal/insight/groq"
)

// completion renders an OpenAI-compatible chat-completions response.
func completion(content string) string {
	b, _ := json.Marshal(map[string]any{
		"choices": []map[string]any{{"message": map[string]string{"role": "assistant", "content": content}}},
	})
	return string(b)
}

func testReq() insight.InsightRequest {
	return insight.InsightRequest{Facts: insight.Facts{"total_centavos": 100000}, Task: "overview", UserID: "u1"}
}

func newServer(t *testing.T, h http.HandlerFunc) *httptest.Server {
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv
}

func TestAdapter_Success_SendsKeyAndJSONFormat(t *testing.T) {
	var gotAuth, gotBody string
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		_, _ = w.Write([]byte(completion(`{"insights":[{"title":"t","explanation":"e"}]}`)))
	})

	res, err := groq.New(srv.URL, "gsk_secret", "test-model", 5*time.Second).Generate(context.Background(), testReq())
	require.NoError(t, err)
	require.Len(t, res.Insights, 1)
	require.Equal(t, "Bearer gsk_secret", gotAuth, "the API key is sent as a Bearer token")
	require.Contains(t, gotBody, `"json_object"`, "requests JSON response format")
}

func TestAdapter_MalformedThenReask(t *testing.T) {
	calls := 0
	srv := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		calls++
		if calls == 1 {
			_, _ = w.Write([]byte(completion("sorry, no json")))
			return
		}
		_, _ = w.Write([]byte(completion(`{"insights":[{"title":"t","explanation":"e"}]}`)))
	})

	res, err := groq.New(srv.URL, "k", "m", 5*time.Second).Generate(context.Background(), testReq())
	require.NoError(t, err)
	require.Len(t, res.Insights, 1)
	require.Equal(t, 2, calls)
}

func TestAdapter_RateLimitedDegradesWithoutRetry(t *testing.T) {
	calls := 0
	srv := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.WriteHeader(http.StatusTooManyRequests) // 429
	})

	_, err := groq.New(srv.URL, "k", "m", 5*time.Second).Generate(context.Background(), testReq())
	require.ErrorIs(t, err, insight.ErrInsightsUnavailable)
	require.Equal(t, 1, calls, "a rate-limit must not be retried (no charge-incurring retry, BR-504)")
}

func TestAdapter_UnauthorizedDegrades(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized) // 401
	})

	_, err := groq.New(srv.URL, "bad-key", "m", 5*time.Second).Generate(context.Background(), testReq())
	require.ErrorIs(t, err, insight.ErrInsightsUnavailable)
}

func TestAdapter_NeverLogsKeyInError(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	_, err := groq.New(srv.URL, "gsk_supersecret", "m", 5*time.Second).Generate(context.Background(), testReq())
	require.Error(t, err)
	require.False(t, strings.Contains(err.Error(), "gsk_supersecret"), "the API key must never appear in errors")
}
