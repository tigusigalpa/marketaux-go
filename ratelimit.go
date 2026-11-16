package marketaux

import (
	"context"
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

// reserve returns the duration the caller must wait before one token is
// available at time now.
func (l *rateLimiter) reserve(now time.Time) time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()

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
	// Borrow from the future: report the deficit and move the refill clock
	// forward so subsequent reservations queue behind this one.
	deficit := 1 - l.tokens
	l.tokens = 0
	wait := time.Duration(deficit / l.rps * float64(time.Second))
	l.last = now.Add(wait)
	return wait
}

// wait blocks until a token is available or ctx is done.
func (l *rateLimiter) wait(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	d := l.reserve(time.Now())
	if d <= 0 {
		return nil
	}
	return sleep(ctx, d)
}
