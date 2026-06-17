package httpserver

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/biel-ferreira/yield-forge/internal/platform/config"
)

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func testConfig(port int) config.Config {
	return config.Config{
		Port:            port,
		ReadTimeout:     5 * time.Second,
		WriteTimeout:    5 * time.Second,
		IdleTimeout:     5 * time.Second,
		ShutdownTimeout: 5 * time.Second,
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// waitUp blocks until the server accepts TCP connections on the port, or fails.
func waitUp(t *testing.T, port int) {
	t.Helper()
	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			conn.Close()
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("server did not come up on port %d", port)
}

func TestRun_ServesAndStopsCleanly(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}
	port := freePort(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "pong")
	})

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- Run(ctx, testConfig(port), mux, discardLogger()) }()

	waitUp(t, port)

	resp, err := http.Get("http://127.0.0.1:" + strconv.Itoa(port) + "/ping")
	if err != nil {
		t.Fatalf("GET /ping: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || string(body) != "pong" {
		t.Errorf("GET /ping = %d %q, want 200 pong", resp.StatusCode, body)
	}

	cancel() // trigger graceful shutdown
	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("Run returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after shutdown")
	}
}

func TestRun_GracefulShutdownDrainsInFlight(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}
	port := freePort(t)

	started := make(chan struct{})
	mux := http.NewServeMux()
	mux.HandleFunc("/slow", func(w http.ResponseWriter, _ *http.Request) {
		close(started)
		time.Sleep(300 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "done")
	})

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- Run(ctx, testConfig(port), mux, discardLogger()) }()
	waitUp(t, port)

	type result struct {
		status int
		body   string
		err    error
	}
	reqDone := make(chan result, 1)
	go func() {
		resp, err := http.Get("http://127.0.0.1:" + strconv.Itoa(port) + "/slow")
		if err != nil {
			reqDone <- result{err: err}
			return
		}
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		reqDone <- result{status: resp.StatusCode, body: string(b)}
	}()

	<-started // the request is now in-flight inside the handler
	cancel()  // begin graceful shutdown while the request is still running

	select {
	case r := <-reqDone:
		if r.err != nil {
			t.Fatalf("in-flight request failed during shutdown: %v", r.err)
		}
		if r.status != http.StatusOK || r.body != "done" {
			t.Errorf("in-flight request = %d %q, want 200 done", r.status, r.body)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("in-flight request did not complete during graceful shutdown")
	}

	if err := <-errCh; err != nil {
		t.Errorf("Run returned error: %v", err)
	}
}

func TestRun_ReturnsErrorWhenPortInUse(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}
	port := freePort(t)
	// Occupy the exact address the server will bind (all interfaces), so the
	// conflict is unambiguous on every platform.
	l, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	// A timeout context guarantees this test can never hang, even if the bind
	// unexpectedly succeeds on some platform.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := Run(ctx, testConfig(port), http.NewServeMux(), discardLogger()); err == nil {
		t.Error("expected an error when the port is already in use, got nil")
	}
}
