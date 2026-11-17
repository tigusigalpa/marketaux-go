package marketaux

import (
	"context"
	"math"
	"sync"
	"time"
)

// rateLimiter is a token-bucket limiter. It spawns no goroutines and is safe
// for concurrent use.
type rateLimiter struct {
	mu     sync.Mutex
	tokens float64
	burst  float64
	rps    float64
	last   time.Time
}

// newRateLimiter creates a token bucket allowing burst tokens at once,
// refilling at rps tokens per second.
func newRateLimiter(rps float64, burst int) *rateLimiter {
	return &rateLimiter{
		tokens: float64(burst),
		burst:  float64(burst),
		rps:    rps,
		last:   time.Now(),
	}
}

// reserve consumes an available token or reports when to try again.
// Waiting callers do not borrow tokens, so cancellation leaves no debt.
func (l *rateLimiter) reserve() time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(l.last).Seconds()
	l.last = now
	l.tokens += elapsed * l.rps
	if l.tokens > l.burst {
		l.tokens = l.burst
	}
	if l.tokens >= 1 {
		l.tokens--
		return 0
	}
	deficit := 1 - l.tokens
	wait := math.Ceil(deficit / l.rps * float64(time.Second))
	if wait >= float64(math.MaxInt64) {
		return time.Duration(math.MaxInt64)
	}
	return time.Duration(wait)
}

// wait blocks until a token is available or ctx is done.
func (l *rateLimiter) wait(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		d := l.reserve()
		if d <= 0 {
			return nil
		}
		if err := sleep(ctx, d); err != nil {
			return err
		}
	}
}
