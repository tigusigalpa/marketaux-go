package marketaux

import (
	"context"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// RetryPolicy controls automatic retries of failed requests.
type RetryPolicy struct {
	// MaxAttempts is the total number of attempts, including the first one.
	// A value of 1 disables retries.
	MaxAttempts int
	// BaseDelay is the delay before the first retry. It doubles after every
	// subsequent attempt.
	BaseDelay time.Duration
	// MaxDelay caps the exponential backoff.
	MaxDelay time.Duration
	// Jitter adds full jitter (50-100% of the computed delay) when true.
	Jitter bool
}

// DefaultRetryPolicy is used when WithRetry is not provided.
var DefaultRetryPolicy = RetryPolicy{
	MaxAttempts: 3,
	BaseDelay:   500 * time.Millisecond,
	MaxDelay:    8 * time.Second,
	Jitter:      true,
}

// validate checks the policy for sane values.
func (p RetryPolicy) validate() error {
	if p.MaxAttempts < 1 {
		return &ConfigError{Field: "retry.MaxAttempts", Reason: "must be >= 1"}
	}
	if p.BaseDelay < 0 {
		return &ConfigError{Field: "retry.BaseDelay", Reason: "must be >= 0"}
	}
	if p.MaxDelay < p.BaseDelay {
		return &ConfigError{Field: "retry.MaxDelay", Reason: "must be >= BaseDelay"}
	}
	return nil
}

// delay returns the backoff delay before the given attempt (attempt is the
// 1-based number of the attempt that just failed), honoring Retry-After.
func (p RetryPolicy) delay(attempt int, retryAfter time.Duration) time.Duration {
	d := p.BaseDelay
	for i := 1; i < attempt; i++ {
		d *= 2
		if d >= p.MaxDelay {
			d = p.MaxDelay
			break
		}
	}
	if d > p.MaxDelay {
		d = p.MaxDelay
	}
	if p.Jitter && d > 0 {
		// Full jitter in the 0.5..1.0 range of the computed delay.
		// #nosec G404 -- jitter randomness is not security-sensitive.
		d = d/2 + time.Duration(rand.Int64N(int64(d/2)+1))
	}
	if retryAfter > d {
		d = retryAfter
	}
	return d
}

// retryableStatus reports whether the HTTP status code is worth retrying.
func retryableStatus(code int) bool {
	switch code {
	case http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	}
	return false
}

// parseRetryAfter parses a Retry-After header value expressed either as a
// number of seconds or as an HTTP-date.
func parseRetryAfter(h string, now time.Time) time.Duration {
	h = strings.TrimSpace(h)
	if h == "" {
		return 0
	}
	if secs, err := strconv.ParseInt(h, 10, 64); err == nil {
		if secs <= 0 {
			return 0
		}
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(h); err == nil {
		if d := t.Sub(now); d > 0 {
			return d
		}
	}
	return 0
}

// sleep waits for d or until ctx is done.
func sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
