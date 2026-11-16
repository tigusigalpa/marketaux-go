package marketaux

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestRateLimiterBurst(t *testing.T) {
	t.Parallel()
	l := newRateLimiter(1000, 2)
	start := time.Now()
	for i := 0; i < 2; i++ {
		if err := l.wait(context.Background()); err != nil {
			t.Fatalf("wait: %v", err)
		}
	}
	if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
		t.Fatalf("burst too slow: %v", elapsed)
	}
}

func TestRateLimiterPacing(t *testing.T) {
	t.Parallel()
	l := newRateLimiter(100, 1)
	start := time.Now()
	for i := 0; i < 5; i++ {
		if err := l.wait(context.Background()); err != nil {
			t.Fatalf("wait: %v", err)
		}
	}
	// First call consumes the burst token; the next 4 are paced at 100 rps.
	elapsed := time.Since(start)
	if elapsed < 30*time.Millisecond {
		t.Fatalf("too fast: %v", elapsed)
	}
	if elapsed > 2*time.Second {
		t.Fatalf("too slow: %v", elapsed)
	}
}

func TestRateLimiterContextCancel(t *testing.T) {
	t.Parallel()
	l := newRateLimiter(0.001, 1) // ~16 minutes between tokens
	if err := l.wait(context.Background()); err != nil {
		t.Fatalf("first wait should be free: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := l.wait(ctx); err == nil {
		t.Fatal("expected context error")
	}
}

func TestRateLimiterConcurrency(t *testing.T) {
	t.Parallel()
	l := newRateLimiter(500, 4)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = l.wait(context.Background())
		}()
	}
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("concurrent waits hung")
	}
}
