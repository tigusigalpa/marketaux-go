package marketaux_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	marketaux "github.com/tigusigalpa/marketaux-go"
)

// flaky returns a handler that fails with status `fail` the first `times`
// requests and serves the fixture afterwards.
func flaky(t *testing.T, fail, times int, fixtureName string) (http.HandlerFunc, *atomic.Int32) {
	t.Helper()
	var calls atomic.Int32
	return func(w http.ResponseWriter, _ *http.Request) {
		n := calls.Add(1)
		if n <= int32(times) {
			w.WriteHeader(fail)
			w.Write(fixture(t, "error_500.json"))
			return
		}
		w.Write(fixture(t, fixtureName))
	}, &calls
}

func TestRetryOn500ThenSuccess(t *testing.T) {
	t.Parallel()
	h, calls := flaky(t, 500, 1, "news_all.json")
	srv := newServer(t, h)
	c := newTestClient(t, srv)
	resp, err := c.News().All(context.Background(), marketaux.NewsParams{})
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	if len(resp.Data) == 0 {
		t.Fatal("expected data")
	}
	if calls.Load() != 2 {
		t.Fatalf("attempts = %d, want 2", calls.Load())
	}
}

func TestRetryOn429ThenSuccess(t *testing.T) {
	t.Parallel()
	h, calls := flaky(t, 429, 1, "news_all.json")
	srv := newServer(t, h)
	c := newTestClient(t, srv)
	if _, err := c.News().All(context.Background(), marketaux.NewsParams{}); err != nil {
		t.Fatalf("All: %v", err)
	}
	if calls.Load() != 2 {
		t.Fatalf("attempts = %d", calls.Load())
	}
}

func TestNoRetryOn4xx(t *testing.T) {
	t.Parallel()
	for _, status := range []int{400, 401, 402, 403, 404} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			t.Parallel()
			var calls atomic.Int32
			srv := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
				calls.Add(1)
				w.WriteHeader(status)
				w.Write(fixture(t, "error_400.json"))
			})
			c := newTestClient(t, srv)
			_, err := c.News().All(context.Background(), marketaux.NewsParams{})
			if err == nil {
				t.Fatal("expected error")
			}
			if calls.Load() != 1 {
				t.Fatalf("attempts = %d, want 1", calls.Load())
			}
		})
	}
}

func TestRetryExhausted(t *testing.T) {
	t.Parallel()
	h, calls := flaky(t, 500, 100, "news_all.json")
	srv := newServer(t, h)
	c := newTestClient(t, srv)
	_, err := c.News().All(context.Background(), marketaux.NewsParams{})
	if !errors.Is(err, marketaux.ErrServer) {
		t.Fatalf("expected ErrServer, got %v", err)
	}
	if calls.Load() != int32(fastRetry.MaxAttempts) {
		t.Fatalf("attempts = %d, want %d", calls.Load(), fastRetry.MaxAttempts)
	}
}

func TestRetryMaxAttempts1(t *testing.T) {
	t.Parallel()
	h, calls := flaky(t, 500, 100, "news_all.json")
	srv := newServer(t, h)
	c := newTestClient(t, srv, marketaux.WithRetry(marketaux.RetryPolicy{MaxAttempts: 1}))
	_, err := c.News().All(context.Background(), marketaux.NewsParams{})
	if err == nil {
		t.Fatal("expected error")
	}
	if calls.Load() != 1 {
		t.Fatalf("attempts = %d", calls.Load())
	}
}

func TestRetryAfterRespected(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	srv := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write(fixture(t, "error_429.json"))
			return
		}
		w.Write(fixture(t, "news_all.json"))
	})
	c := newTestClient(t, srv) // BaseDelay 1ms — Retry-After must dominate
	start := time.Now()
	if _, err := c.News().All(context.Background(), marketaux.NewsParams{}); err != nil {
		t.Fatalf("All: %v", err)
	}
	if elapsed := time.Since(start); elapsed < time.Second {
		t.Fatalf("Retry-After ignored, elapsed %v", elapsed)
	}
}

func TestRetryContextCancel(t *testing.T) {
	t.Parallel()
	srv := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(fixture(t, "error_500.json"))
	})
	c := newTestClient(t, srv, marketaux.WithRetry(marketaux.RetryPolicy{
		MaxAttempts: 5,
		BaseDelay:   5 * time.Second,
		MaxDelay:    10 * time.Second,
	}))
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	_, err := c.News().All(ctx, marketaux.NewsParams{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestRateLimitClientSide(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	srv := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Write(fixture(t, "entity_types.json"))
	})
	c := newTestClient(t, srv, marketaux.WithRateLimit(200, 1))
	start := time.Now()
	for i := 0; i < 3; i++ {
		if _, err := c.Entities().Types(context.Background()); err != nil {
			t.Fatalf("Types: %v", err)
		}
	}
	if elapsed := time.Since(start); elapsed < 8*time.Millisecond {
		t.Fatalf("rate limiter not applied: %v", elapsed)
	}
	if calls.Load() != 3 {
		t.Fatalf("requests = %d", calls.Load())
	}
}

func TestDebugLogRedactsToken(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	srv := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write(fixture(t, "error_401.json"))
	})
	c := newTestClient(t, srv,
		marketaux.WithLogger(logger),
		marketaux.WithDebug(true),
		marketaux.WithRetry(marketaux.RetryPolicy{MaxAttempts: 1}),
	)
	_, err := c.News().All(context.Background(), marketaux.NewsParams{})
	if err == nil {
		t.Fatal("expected error")
	}
	out := buf.String()
	if out == "" {
		t.Fatal("expected debug logs")
	}
	if strings.Contains(out, testToken) {
		t.Fatalf("token leaked into logs: %s", out)
	}
	if !strings.Contains(out, "api_token") {
		t.Fatalf("expected redacted api_token param in logs: %s", out)
	}
}
