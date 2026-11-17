package marketaux

import (
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Option configures a Client. Options are applied in order during NewClient;
// a later option overrides an earlier one.
type Option func(*Client) error

// WithBaseURL overrides the API base URL. It is mostly useful for tests and
// proxies. Trailing slashes are tolerated.
func WithBaseURL(raw string) Option {
	return func(c *Client) error {
		u, err := url.Parse(raw)
		if err != nil {
			return &ConfigError{Field: "baseURL", Reason: "must be a valid URL"}
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			return &ConfigError{Field: "baseURL", Reason: "scheme must be http or https"}
		}
		if u.Host == "" {
			return &ConfigError{Field: "baseURL", Reason: "host must not be empty"}
		}
		if u.User != nil {
			return &ConfigError{Field: "baseURL", Reason: "must not contain user credentials"}
		}
		u.Path = strings.TrimSuffix(u.Path, "/")
		u.RawQuery = ""
		u.Fragment = ""
		c.baseURL = u
		return nil
	}
}

// WithHTTPClient sets the HTTP client used for requests. The provided client
// is shallow-copied, so its Transport and Jar remain shared. Configure a
// positive Timeout, or use WithTimeout after this option. Redirects must stay
// on the original origin, including when a custom CheckRedirect is provided.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) error {
		if hc == nil {
			return &ConfigError{Field: "httpClient", Reason: "must not be nil"}
		}
		clone := *hc
		c.httpClient = &clone
		return nil
	}
}

// WithTimeout sets the per-request timeout on a copy of the current HTTP
// client. The default is 30 seconds.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) error {
		if d <= 0 {
			return &ConfigError{Field: "timeout", Reason: "must be > 0"}
		}
		if c.httpClient == nil {
			c.httpClient = &http.Client{}
		}
		clone := *c.httpClient
		clone.Timeout = d
		c.httpClient = &clone
		return nil
	}
}

// WithUserAgent overrides the User-Agent header. The default is
// "marketaux-go/<version>".
func WithUserAgent(ua string) Option {
	return func(c *Client) error {
		if strings.TrimSpace(ua) == "" {
			return &ConfigError{Field: "userAgent", Reason: "must not be empty"}
		}
		for _, ch := range ua {
			if ch < 32 || ch > 126 {
				return &ConfigError{Field: "userAgent", Reason: "must contain only printable ASCII characters"}
			}
		}
		c.userAgent = ua
		return nil
	}
}

// WithRetry overrides the retry policy. See RetryPolicy and
// DefaultRetryPolicy.
func WithRetry(p RetryPolicy) Option {
	return func(c *Client) error {
		if err := p.validate(); err != nil {
			return err
		}
		c.retry = p
		return nil
	}
}

// WithRateLimit enables a client-side token-bucket rate limiter allowing
// burst requests at once and rps requests per second on average.
func WithRateLimit(rps float64, burst int) Option {
	return func(c *Client) error {
		if rps <= 0 || math.IsNaN(rps) || math.IsInf(rps, 0) {
			return &ConfigError{Field: "rateLimit.rps", Reason: "must be finite and > 0"}
		}
		if burst < 1 {
			return &ConfigError{Field: "rateLimit.burst", Reason: "must be >= 1"}
		}
		c.limiter = newRateLimiter(rps, burst)
		return nil
	}
}

// WithLogger sets the slog logger used for debug output. By default logging
// goes to io.Discard. Passing nil is a configuration error.
func WithLogger(l *slog.Logger) Option {
	return func(c *Client) error {
		if l == nil {
			return &ConfigError{Field: "logger", Reason: "must not be nil"}
		}
		c.logger = l
		return nil
	}
}

// WithDebug enables debug-level logging of request method, redacted URL,
// status, duration and attempt number. The API token is never logged.
func WithDebug(enabled bool) Option {
	return func(c *Client) error {
		c.debug = enabled
		return nil
	}
}

// WithMaxResponseBytes bounds the size of a response body the client will
// read. The default is 32 MiB.
func WithMaxResponseBytes(n int64) Option {
	return func(c *Client) error {
		if n <= 0 {
			return &ConfigError{Field: "maxResponseBytes", Reason: "must be > 0"}
		}
		c.maxResponseBytes = n
		return nil
	}
}
