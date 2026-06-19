package http

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"
)

type ctxKey int

const (
	requestIDKey ctxKey = iota
	logFieldsKey
)

// logFields carries request-scoped attributes that inner middleware (e.g. auth)
// fills in so the outer request logger can include them. It is a pointer in the
// context so a value set deep in the chain is visible when logRequests logs.
type logFields struct {
	userID string
}

func withLogFields(ctx context.Context) (context.Context, *logFields) {
	lf := &logFields{}
	return context.WithValue(ctx, logFieldsKey, lf), lf
}

func logFieldsFromContext(ctx context.Context) *logFields {
	lf, _ := ctx.Value(logFieldsKey).(*logFields)
	return lf
}

// requestID assigns a request id to each request — reusing an incoming
// X-Request-Id header when present, otherwise generating one — and exposes it on
// the response header and in the request context.
func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if id == "" {
			id = newID()
		}
		w.Header().Set("X-Request-Id", id)
		ctx := context.WithValue(r.Context(), requestIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequestIDFromContext returns the request id stored in ctx, or "" if absent.
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}

func newID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(b)
}

// logRequests logs one structured line per request: method, path, status,
// duration, and request id.
func logRequests(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

			// Seed the per-request log fields so inner middleware (auth) can add the
			// resolved user_id, then serve with that context.
			ctx, lf := withLogFields(r.Context())
			r = r.WithContext(ctx)

			next.ServeHTTP(rec, r)

			attrs := []slog.Attr{
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rec.status),
				slog.Float64("duration_ms", float64(time.Since(start).Microseconds())/1000.0),
				slog.String("request_id", RequestIDFromContext(r.Context())),
			}
			if lf.userID != "" {
				attrs = append(attrs, slog.String("user_id", lf.userID))
			}
			logger.LogAttrs(r.Context(), slog.LevelInfo, "http request", attrs...)
		})
	}
}

// statusRecorder wraps http.ResponseWriter to capture the status code written.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}
