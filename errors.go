package marketaux

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Sentinel errors. Use errors.Is to classify failures returned by the client.
var (
	ErrInvalidConfig = errors.New("marketaux: invalid configuration")
	ErrValidation    = errors.New("marketaux: validation error")
	ErrTransport     = errors.New("marketaux: transport error")
	ErrTimeout       = errors.New("marketaux: request timeout")
	ErrDecode        = errors.New("marketaux: response decode error")
	// ErrResponseTooLarge reports a response exceeding WithMaxResponseBytes.
	ErrResponseTooLarge = errors.New("marketaux: response body too large")
	ErrBadRequest       = errors.New("marketaux: bad request")
	ErrUnauthorized     = errors.New("marketaux: invalid api token")
	ErrUsageLimit       = errors.New("marketaux: usage limit reached")
	ErrForbidden        = errors.New("marketaux: endpoint access restricted")
	ErrNotFound         = errors.New("marketaux: resource not found")
	ErrEndpointNotFound = errors.New("marketaux: invalid api endpoint")
	ErrRateLimit        = errors.New("marketaux: rate limit reached")
	ErrServer           = errors.New("marketaux: server error")
	ErrMaintenance      = errors.New("marketaux: maintenance mode")
)

// ConfigError reports an invalid client configuration passed to NewClient or
// to one of the Option helpers.
type ConfigError struct {
	Field  string
	Reason string
}

// Error returns a human-readable description of the configuration error.
func (e *ConfigError) Error() string {
	return fmt.Sprintf("marketaux: invalid configuration: %s: %s", e.Field, e.Reason)
}

// Is reports whether the target is ErrInvalidConfig.
func (e *ConfigError) Is(target error) bool { return target == ErrInvalidConfig }

// ValidationError reports invalid request parameters detected on the client
// side, before any HTTP request is made.
type ValidationError struct {
	// Fields maps a parameter name to the reason it is invalid.
	Fields map[string]string
}

// Error returns a human-readable description of all validation failures in a
// stable key order.
func (e *ValidationError) Error() string {
	keys := make([]string, 0, len(e.Fields))
	for k := range e.Fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString("marketaux: validation error: ")
	for i, k := range keys {
		if i > 0 {
			b.WriteString("; ")
		}
		fmt.Fprintf(&b, "%s: %s", k, e.Fields[k])
	}
	return b.String()
}

// Is reports whether the target is ErrValidation.
func (e *ValidationError) Is(target error) bool { return target == ErrValidation }

// APIError is an error response returned by the Marketaux API. It is produced
// for every HTTP response outside the 2xx range, including unfollowed redirects.
type APIError struct {
	StatusCode      int
	Code            string
	Message         string
	URL             string // path + redacted query, never contains the token
	RequestID       string // cf-ray or x-request-id, when present
	RateLimitLimit  string // X-RateLimit-Limit header (429)
	UsageLimitLimit string // X-UsageLimit-Limit header (402)
	RetryAfter      time.Duration
	Body            []byte // truncated and redacted
}

// Error returns a human-readable description of the API error.
func (e *APIError) Error() string {
	code := e.Code
	if code == "" {
		code = "unknown_error"
	}
	msg := e.Message
	if msg == "" {
		msg = strings.TrimSpace(string(e.Body))
	}
	return fmt.Sprintf("marketaux: API error [%s] (HTTP %d): %s", code, e.StatusCode, msg)
}

// Is maps the error onto the sentinel error for its status code / error code
// pair so callers can use errors.Is.
func (e *APIError) Is(target error) bool {
	switch e.StatusCode {
	case 400:
		return target == ErrBadRequest
	case 401:
		return target == ErrUnauthorized
	case 402:
		return target == ErrUsageLimit
	case 403:
		return target == ErrForbidden
	case 404:
		if e.Code == "invalid_api_endpoint" {
			return target == ErrEndpointNotFound
		}
		return target == ErrNotFound
	case 429:
		return target == ErrRateLimit
	case 503:
		return target == ErrMaintenance
	}
	if e.StatusCode >= 500 {
		return target == ErrServer
	}
	return false
}

// TransportError wraps a network-level failure (DNS, TLS, connection reset,
// timeout) or an oversized response body.
type TransportError struct {
	Err     error
	URL     string // redacted
	Timeout bool
}

// Error returns a human-readable description of the transport error.
func (e *TransportError) Error() string {
	return fmt.Sprintf("marketaux: transport error: %s: %v", e.URL, e.Err)
}

// Unwrap returns the underlying error.
func (e *TransportError) Unwrap() error { return e.Err }

// Is reports whether the target is ErrTransport, or ErrTimeout when the
// failure was a timeout.
func (e *TransportError) Is(target error) bool {
	if target == ErrTransport {
		return true
	}
	if target == ErrTimeout && e.Timeout {
		return true
	}
	return false
}

// DecodeError reports a response body that could not be decoded as the
// expected JSON type.
type DecodeError struct {
	Err  error
	URL  string // redacted
	Body []byte // up to 500 bytes
}

// Error returns a human-readable description of the decode error.
func (e *DecodeError) Error() string {
	return fmt.Sprintf("marketaux: response decode error: %s: %v", e.URL, e.Err)
}

// Unwrap returns the underlying error.
func (e *DecodeError) Unwrap() error { return e.Err }

// Is reports whether the target is ErrDecode.
func (e *DecodeError) Is(target error) bool { return target == ErrDecode }
