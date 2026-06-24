package ollama_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/biel-ferreira/yield-forge/internal/insight"
	"github.com/biel-ferreira/yield-forge/internal/insight/ollama"
)

// chatReply renders Ollama's /api/chat response with the given assistant content.
func chatReply(content string) string {
	b, _ := json.Marshal(map[string]any{"message": map[string]string{"role": "assistant", "content": content}})
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

func TestAdapter_Success(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(chatReply(`{"insights":[{"title":"t","detail":"d","explanation":"e"}]}`)))
	})

	res, err := ollama.New(srv.URL, "test", 5*time.Second).Generate(context.Background(), testReq())
	require.NoError(t, err)
	require.Len(t, res.Insights, 1)
	require.Equal(t, "e", res.Insights[0].Explanation)
}

func TestAdapter_MalformedThenReask(t *testing.T) {
	calls := 0
	srv := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		calls++
		if calls == 1 {
			_, _ = w.Write([]byte(chatReply("desculpe, não consegui"))) // malformed
			return
		}
		_, _ = w.Write([]byte(chatReply(`{"insights":[{"title":"t","explanation":"e"}]}`)))
	})

	res, err := ollama.New(srv.URL, "test", 5*time.Second).Generate(context.Background(), testReq())
	require.NoError(t, err)
	require.Len(t, res.Insights, 1)
	require.Equal(t, 2, calls, "a malformed reply triggers exactly one re-ask")
}

func TestAdapter_MalformedTwiceDegrades(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(chatReply("não é json")))
	})

	_, err := ollama.New(srv.URL, "test", 5*time.Second).Generate(context.Background(), testReq())
	require.ErrorIs(t, err, insight.ErrInsightsUnavailable)
}

func TestAdapter_HTTPErrorDegrades(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	_, err := ollama.New(srv.URL, "test", 5*time.Second).Generate(context.Background(), testReq())
	require.ErrorIs(t, err, insight.ErrInsightsUnavailable)
}

func TestAdapter_InsufficientFacts(t *testing.T) {
	_, err := ollama.New("http://unused", "test", time.Second).
		Generate(context.Background(), insight.InsightRequest{Facts: nil})
	require.ErrorIs(t, err, insight.ErrInsufficientFacts)
}
