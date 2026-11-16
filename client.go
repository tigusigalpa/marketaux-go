package marketaux

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// defaultBaseURL is the public Marketaux API root.
const defaultBaseURL = "https://api.marketaux.com/v1"

// defaultMaxResponseBytes bounds the size of a decoded response body.
const defaultMaxResponseBytes int64 = 32 << 20 // 32 MiB

// apiErrorBodyLimit bounds how many bytes of an error response are kept in
// APIError.Body.
const apiErrorBodyLimit = 2000

// decodeBodyPreview bounds the preview kept in DecodeError.Body.
const decodeBodyPreview = 500

// Client is a thread-safe Marketaux API client. It covers all documented
// endpoints through the News, Stats, Entities and Sources services.
type Client struct {
	token            string
	baseURL          *url.URL
	httpClient       *http.Client
	userAgent        string
	retry            RetryPolicy
	limiter          *rateLimiter
	logger           *slog.Logger
	debug            bool
	maxResponseBytes int64

	news     *NewsService
	stats    *StatsService
	entities *EntitiesService
	sources  *SourcesService
}

// NewClient creates a Marketaux client authenticated with token. The token is
// sent as the api_token query parameter on every request, never in a header.
// Options are applied in order; the last one wins.
func NewClient(token string, opts ...Option) (*Client, error) {
	if strings.TrimSpace(token) == "" {
		return nil, &ConfigError{Field: "token", Reason: "must not be empty"}
	}
	base, err := url.Parse(defaultBaseURL)
	if err != nil {
		return nil, &ConfigError{Field: "baseURL", Reason: err.Error()}
	}
	c := &Client{
		token:     token,
		baseURL:   base,
		userAgent: "marketaux-go/" + Version,
		retry:     DefaultRetryPolicy,
		logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		maxResponseBytes: defaultMaxResponseBytes,
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(c); err != nil {
			return nil, err
		}
	}
	if err := c.retry.validate(); err != nil {
		return nil, err
	}
	if c.httpClient == nil {
		c.httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	if c.httpClient.Timeout <= 0 {
		return nil, &ConfigError{Field: "timeout", Reason: "must be > 0"}
	}
	if c.maxResponseBytes <= 0 {
		return nil, &ConfigError{Field: "maxResponseBytes", Reason: "must be > 0"}
	}

	c.news = &NewsService{c: c}
	c.stats = &StatsService{c: c}
	c.entities = &EntitiesService{c: c}
	c.sources = &SourcesService{c: c}
	return c, nil
}

// News returns the service for the /news/* endpoints.
func (c *Client) News() *NewsService { return c.news }

// Stats returns the service for the /entity/stats/* and /entity/trending/*
// endpoints. These endpoints require a Standard plan or higher.
func (c *Client) Stats() *StatsService { return c.stats }

// Entities returns the service for the /entity/search, /entity/type/list and
// /entity/industry/list endpoints.
func (c *Client) Entities() *EntitiesService { return c.entities }

// Sources returns the service for the /news/sources endpoint.
func (c *Client) Sources() *SourcesService { return c.sources }

// BaseURL returns the configured API base URL.
func (c *Client) BaseURL() string { return c.baseURL.String() }

// CloseIdleConnections closes idle keep-alive connections held by the
// underlying HTTP transport, when supported.
func (c *Client) CloseIdleConnections() {
	if tr, ok := c.httpClient.Transport.(interface{ CloseIdleConnections() }); ok {
		tr.CloseIdleConnections()
	}
}

// get performs an authenticated GET request and decodes the JSON body into
// out. It handles rate limiting, retries and error mapping.
func (c *Client) get(ctx context.Context, path string, query url.Values, out any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if query == nil {
		query = url.Values{}
	}
	query.Set("api_token", c.token)

	u := c.baseURL.ResolveReference(&url.URL{Path: strings.TrimSuffix(c.baseURL.Path, "/") + path})
	u.RawQuery = query.Encode()
	full := u.String()
	redactedURL := RedactURL(full)

	if c.limiter != nil {
		if err := c.limiter.wait(ctx); err != nil {
			return err
		}
	}

	var lastErr error
	for attempt := 1; attempt <= c.retry.MaxAttempts; attempt++ {
		if attempt > 1 {
			c.logDebug(ctx, "retrying request",
				slog.Int("attempt", attempt),
				slog.String("url", redactedURL))
		}
		start := time.Now()
		status, retryAfter, err := c.doOnce(ctx, full, redactedURL, out)
		c.logDebug(ctx, "request completed",
			slog.String("method", http.MethodGet),
			slog.String("url", redactedURL),
			slog.Int("status", status),
			slog.Int("attempt", attempt),
			slog.Duration("duration", time.Since(start)))
		if err == nil {
			return nil
		}
		lastErr = err
		if attempt == c.retry.MaxAttempts || !isRetryable(err) {
			return err
		}
		if err := sleep(ctx, c.retry.delay(attempt, retryAfter)); err != nil {
			return err
		}
	}
	return lastErr
}

// doOnce executes a single HTTP attempt. It returns the status code, the
// parsed Retry-After delay (when present) and nil on success.
func (c *Client) doOnce(ctx context.Context, full, redactedURL string, out any) (int, time.Duration, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, full, nil)
	if err != nil {
		return 0, 0, &TransportError{Err: err, URL: redactedURL}
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return 0, 0, ctx.Err()
		}
		return 0, 0, &TransportError{
			Err:     unwrapNetErr(err),
			URL:     redactedURL,
			Timeout: isTimeoutErr(err),
		}
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := readBody(resp.Body, c.maxResponseBytes)
	if err != nil {
		drain(resp.Body)
		return resp.StatusCode, 0, &TransportError{Err: err, URL: redactedURL}
	}
	drain(resp.Body)

	if resp.StatusCode >= 400 {
		apiErr := newAPIError(resp, body, redactedURL)
		return resp.StatusCode, apiErr.RetryAfter, apiErr
	}

	if err := json.Unmarshal(body, out); err != nil {
		preview := body
		if len(preview) > decodeBodyPreview {
			preview = preview[:decodeBodyPreview]
		}
		return resp.StatusCode, 0, &DecodeError{
			Err:  err,
			URL:  redactedURL,
			Body: append([]byte(nil), preview...),
		}
	}
	return resp.StatusCode, 0, nil
}

// readBody reads r up to limit+1 bytes and fails when the body exceeds limit.
func readBody(r io.Reader, limit int64) ([]byte, error) {
	lr := &io.LimitedReader{R: r, N: limit + 1}
	body, err := io.ReadAll(lr)
	if err != nil {
		return nil, err
	}
	if lr.N == 0 {
		return nil, fmt.Errorf("response body exceeds %d bytes", limit)
	}
	return body, nil
}

// drain consumes what is left of the body so the connection can be reused.
func drain(r io.Reader) {
	_, _ = io.Copy(io.Discard, io.LimitReader(r, 1<<20))
}

// newAPIError builds an APIError from an HTTP error response.
func newAPIError(resp *http.Response, body []byte, redactedURL string) *APIError {
	var payload struct {
		Err struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	_ = json.Unmarshal(body, &payload)

	trimmed := body
	if len(trimmed) > apiErrorBodyLimit {
		trimmed = trimmed[:apiErrorBodyLimit]
	}

	return &APIError{
		StatusCode:      resp.StatusCode,
		Code:            payload.Err.Code,
		Message:         payload.Err.Message,
		URL:             redactedURL,
		RequestID:       firstHeader(resp.Header, "cf-ray", "x-request-id"),
		RateLimitLimit:  resp.Header.Get("X-RateLimit-Limit"),
		UsageLimitLimit: resp.Header.Get("X-UsageLimit-Limit"),
		RetryAfter:      parseRetryAfter(resp.Header.Get("Retry-After"), time.Now()),
		Body:            append([]byte(nil), trimmed...),
	}
}

// firstHeader returns the first non-empty header value among names.
func firstHeader(h http.Header, names ...string) string {
	for _, n := range names {
		if v := h.Get(n); v != "" {
			return v
		}
	}
	return ""
}

// isRetryable reports whether err justifies another attempt.
func isRetryable(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return retryableStatus(apiErr.StatusCode)
	}
	var trErr *TransportError
	if errors.As(err, &trErr) {
		if trErr.Timeout {
			// Local timeouts are not retried: the server may have processed
			// the request, and context deadlines would only burn attempts.
			return false
		}
		return true
	}
	return false
}

// isTimeoutErr reports whether err is a network timeout.
func isTimeoutErr(err error) bool {
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return true
	}
	return errors.Is(err, context.DeadlineExceeded)
}

// unwrapNetErr extracts the meaningful error from *url.Error wrappers.
func unwrapNetErr(err error) error {
	var ue *url.Error
	if errors.As(err, &ue) && ue.Err != nil {
		return ue.Err
	}
	return err
}

// logDebug emits a debug-level log entry when debug logging is enabled.
func (c *Client) logDebug(ctx context.Context, msg string, attrs ...slog.Attr) {
	if !c.debug {
		return
	}
	c.logger.LogAttrs(ctx, slog.LevelDebug, msg, attrs...)
}
