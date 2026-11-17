package marketaux

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"
	"time"
)

func TestRetryDurationOverflow(t *testing.T) {
	t.Parallel()
	p := RetryPolicy{MaxAttempts: 3, BaseDelay: time.Duration(math.MaxInt64/2 + 1), MaxDelay: time.Duration(math.MaxInt64)}
	if got := p.delay(2, 0); got != p.MaxDelay {
		t.Fatalf("overflowed delay: %v", got)
	}
	if got := parseRetryAfter("9223372036854775807", time.Now()); got != time.Duration(math.MaxInt64) {
		t.Fatalf("overflowed Retry-After: %v", got)
	}
	p = RetryPolicy{MaxAttempts: 3, BaseDelay: time.Second, MaxDelay: 2 * time.Second, Jitter: true}
	for i := 0; i < 100; i++ {
		if got := p.delay(2, 0); got < time.Second || got > 2*time.Second {
			t.Fatalf("jitter out of bounds: %v", got)
		}
	}
}

func TestReadBodyAtInt64Limit(t *testing.T) {
	t.Parallel()
	body, err := readBody(strings.NewReader("{}"), math.MaxInt64)
	if err != nil || string(body) != "{}" {
		t.Fatalf("body %q, error %v", body, err)
	}
	if _, err := readBody(strings.NewReader("{}"), 2); err != nil {
		t.Fatalf("exact body limit rejected: %v", err)
	}
}

func TestNumericParamsRejectNonFiniteValues(t *testing.T) {
	t.Parallel()
	for _, value := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		for _, err := range []error{
			(NewsParams{SentimentGTE: Float64(value)}).validate(),
			(SimilarParams{SentimentLTE: Float64(value)}).validate(),
			(StatsParams{SentimentAvgGTE: Float64(value)}).validate(),
			(AggregationParams{SentimentGTE: Float64(value)}).validate(),
			(TrendingParams{SentimentAvgLTE: Float64(value)}).validate(),
			(NewsParams{MinMatchScore: Float64(value)}).validate(),
		} {
			if !errors.Is(err, ErrValidation) {
				t.Fatalf("non-finite value accepted: %v", value)
			}
		}
	}
}

func TestResultSetValidationCannotOverflow(t *testing.T) {
	t.Parallel()
	for _, p := range []NewsParams{
		{Page: math.MaxInt, Limit: math.MaxInt},
		{Page: math.MaxInt, Limit: 2},
		{Limit: maxResultSet + 1},
		{Page: maxResultSet + 1},
	} {
		if err := p.validate(); !errors.Is(err, ErrValidation) {
			t.Fatalf("invalid result window accepted: %+v", p)
		}
	}
}

func TestCanceledRateWaitDoesNotDelayNextRequest(t *testing.T) {
	t.Parallel()
	l := newRateLimiter(20, 1)
	if err := l.wait(context.Background()); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), time.Microsecond)
		_ = l.wait(ctx)
		cancel()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	if err := l.wait(ctx); err != nil {
		t.Fatalf("canceled waits left future reservations: %v", err)
	}
}
