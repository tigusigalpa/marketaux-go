package marketaux

import (
	"context"
	"testing"
	"time"
)

func TestRetryDelayBackoff(t *testing.T) {
	t.Parallel()
	p := RetryPolicy{MaxAttempts: 5, BaseDelay: 100 * time.Millisecond, MaxDelay: 1 * time.Second}
	cases := []struct {
		attempt  int
		min, max time.Duration
	}{
		{1, 50 * time.Millisecond, 100 * time.Millisecond},
		{2, 100 * time.Millisecond, 200 * time.Millisecond},
		{3, 200 * time.Millisecond, 400 * time.Millisecond},
		{4, 400 * time.Millisecond, 800 * time.Millisecond},
		{5, 500 * time.Millisecond, 1 * time.Second},
	}
	for _, tc := range cases {
		for i := 0; i < 20; i++ {
			d := p.delay(tc.attempt, 0)
			if d < tc.min || d > tc.max {
				t.Fatalf("attempt %d: delay %v outside [%v, %v]", tc.attempt, d, tc.min, tc.max)
			}
		}
	}
}

func TestRetryDelayNoJitter(t *testing.T) {
	t.Parallel()
	p := RetryPolicy{MaxAttempts: 3, BaseDelay: 100 * time.Millisecond, MaxDelay: time.Second}
	if d := p.delay(1, 0); d != 100*time.Millisecond {
		t.Fatalf("delay = %v", d)
	}
	if d := p.delay(2, 0); d != 200*time.Millisecond {
		t.Fatalf("delay = %v", d)
	}
	if d := p.delay(10, 0); d != time.Second {
		t.Fatalf("delay = %v, expected cap", d)
	}
}

func TestRetryDelayRetryAfter(t *testing.T) {
	t.Parallel()
	p := RetryPolicy{MaxAttempts: 3, BaseDelay: time.Millisecond, MaxDelay: 2 * time.Millisecond}
	if d := p.delay(1, 2*time.Second); d != 2*time.Second {
		t.Fatalf("Retry-After not honored: %v", d)
	}
	if d := p.delay(1, time.Microsecond); d != time.Millisecond {
		t.Fatalf("delay = %v", d)
	}
}

func TestParseRetryAfter(t *testing.T) {
	t.Parallel()
	now := time.Now()
	if d := parseRetryAfter("3", now); d != 3*time.Second {
		t.Fatalf("seconds: %v", d)
	}
	if d := parseRetryAfter("", now); d != 0 {
		t.Fatalf("empty: %v", d)
	}
	if d := parseRetryAfter("-5", now); d != 0 {
		t.Fatalf("negative: %v", d)
	}
	if d := parseRetryAfter("garbage", now); d != 0 {
		t.Fatalf("garbage: %v", d)
	}
	future := now.Add(10 * time.Second).UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")
	if d := parseRetryAfter(future, now); d <= 0 || d > 11*time.Second {
		t.Fatalf("http-date: %v", d)
	}
	past := now.Add(-10 * time.Second).UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")
	if d := parseRetryAfter(past, now); d != 0 {
		t.Fatalf("past date: %v", d)
	}
}

func TestRetryableStatus(t *testing.T) {
	t.Parallel()
	for _, code := range []int{429, 500, 502, 503, 504} {
		if !retryableStatus(code) {
			t.Errorf("%d should be retryable", code)
		}
	}
	for _, code := range []int{200, 400, 401, 402, 403, 404, 418} {
		if retryableStatus(code) {
			t.Errorf("%d should not be retryable", code)
		}
	}
}

func TestSleepContext(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := sleep(ctx, time.Hour); err == nil {
		t.Fatal("expected error")
	}
	if err := sleep(context.Background(), 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
