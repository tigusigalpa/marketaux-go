package marketaux_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	marketaux "github.com/tigusigalpa/marketaux-go"
)

func TestAPIErrorIs(t *testing.T) {
	t.Parallel()
	cases := []struct {
		status int
		code   string
		want   error
	}{
		{400, "malformed_parameters", marketaux.ErrBadRequest},
		{401, "invalid_api_token", marketaux.ErrUnauthorized},
		{402, "usage_limit_reached", marketaux.ErrUsageLimit},
		{403, "endpoint_access_restricted", marketaux.ErrForbidden},
		{404, "resource_not_found", marketaux.ErrNotFound},
		{404, "invalid_api_endpoint", marketaux.ErrEndpointNotFound},
		{429, "rate_limit_reached", marketaux.ErrRateLimit},
		{500, "server_error", marketaux.ErrServer},
		{502, "server_error", marketaux.ErrServer},
		{503, "maintenance_mode", marketaux.ErrMaintenance},
		{418, "teapot", nil},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", tc.status, tc.code), func(t *testing.T) {
			t.Parallel()
			err := error(&marketaux.APIError{StatusCode: tc.status, Code: tc.code})
			if tc.want == nil {
				for _, s := range []error{
					marketaux.ErrBadRequest, marketaux.ErrUnauthorized,
					marketaux.ErrUsageLimit, marketaux.ErrForbidden,
					marketaux.ErrNotFound, marketaux.ErrEndpointNotFound,
					marketaux.ErrRateLimit, marketaux.ErrServer,
					marketaux.ErrMaintenance,
				} {
					if errors.Is(err, s) {
						t.Errorf("unexpected match for %v", s)
					}
				}
				return
			}
			if !errors.Is(err, tc.want) {
				t.Fatalf("errors.Is(_, %v) failed", tc.want)
			}
		})
	}
}

func TestAPIErrorMessage(t *testing.T) {
	t.Parallel()
	e := &marketaux.APIError{
		StatusCode: 401,
		Code:       "invalid_api_token",
		Message:    "An invalid API token was supplied.",
	}
	want := "marketaux: API error [invalid_api_token] (HTTP 401): An invalid API token was supplied."
	if e.Error() != want {
		t.Fatalf("Error() = %q", e.Error())
	}
	e2 := &marketaux.APIError{StatusCode: 500, Body: []byte("oops")}
	if !strings.Contains(e2.Error(), "unknown_error") {
		t.Fatalf("Error() = %q", e2.Error())
	}
}

func TestValidationErrorStableOrder(t *testing.T) {
	t.Parallel()
	e := &marketaux.ValidationError{Fields: map[string]string{
		"limit": "too big",
		"page":  "negative",
		"sort":  "bad",
	}}
	msg := e.Error()
	il := strings.Index(msg, "limit")
	ip := strings.Index(msg, "page")
	is := strings.Index(msg, "sort")
	if il >= ip || ip >= is {
		t.Fatalf("unstable order: %q", msg)
	}
	if !errors.Is(e, marketaux.ErrValidation) {
		t.Fatal("errors.Is failed")
	}
}

func TestTransportAndDecodeErrors(t *testing.T) {
	t.Parallel()
	inner := errors.New("conn reset")
	tr := &marketaux.TransportError{Err: inner, URL: "http://x", Timeout: true}
	if !errors.Is(tr, marketaux.ErrTransport) {
		t.Fatal("expected ErrTransport")
	}
	if !errors.Is(tr, marketaux.ErrTimeout) {
		t.Fatal("expected ErrTimeout")
	}
	if !errors.Is(tr, inner) {
		t.Fatal("Unwrap failed")
	}
	tr.Timeout = false
	if errors.Is(tr, marketaux.ErrTimeout) {
		t.Fatal("unexpected ErrTimeout")
	}

	dec := &marketaux.DecodeError{Err: inner, URL: "http://x"}
	if !errors.Is(dec, marketaux.ErrDecode) {
		t.Fatal("expected ErrDecode")
	}
	if !errors.Is(dec, inner) {
		t.Fatal("Unwrap failed")
	}
	if !strings.Contains(dec.Error(), "response decode error") {
		t.Fatalf("Error() = %q", dec.Error())
	}
	if !strings.Contains(tr.Error(), "transport error") {
		t.Fatalf("Error() = %q", tr.Error())
	}
}

func TestConfigError(t *testing.T) {
	t.Parallel()
	e := &marketaux.ConfigError{Field: "token", Reason: "must not be empty"}
	if !errors.Is(e, marketaux.ErrInvalidConfig) {
		t.Fatal("errors.Is failed")
	}
	if !strings.Contains(e.Error(), "token") {
		t.Fatalf("Error() = %q", e.Error())
	}
}

func TestRetryAfterHTTPDate(t *testing.T) {
	t.Parallel()
	// A server returning Retry-After as an HTTP-date should populate
	// APIError.RetryAfter with the remaining duration.
	future := time.Now().Add(30 * time.Second).UTC().Format(http.TimeFormat)
	srv := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", future)
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write(fixture(t, "error_429.json"))
	})
	c := newTestClient(t, srv, marketaux.WithRetry(marketaux.RetryPolicy{MaxAttempts: 1}))
	_, err := c.News().All(context.Background(), marketaux.NewsParams{})
	var apiErr *marketaux.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %v", err)
	}
	if apiErr.RetryAfter < 20*time.Second || apiErr.RetryAfter > 31*time.Second {
		t.Fatalf("RetryAfter = %v", apiErr.RetryAfter)
	}
}
