package marketaux_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	marketaux "github.com/tigusigalpa/marketaux-go"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type readerFunc func([]byte) (int, error)

func (f readerFunc) Read(p []byte) (int, error) { return f(p) }

func TestErrorsRedactEchoedCredentials(t *testing.T) {
	t.Parallel()
	const token = `sensitive<key>+&/"value`
	for _, status := range []int{http.StatusBadRequest, http.StatusOK} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			payload, err := json.Marshal(map[string]any{
				"error": map[string]string{"code": "bad_request", "message": token},
				"data":  "wrong type: " + token,
				"query": url.QueryEscape(token),
			})
			if err != nil {
				t.Fatal(err)
			}
			srv := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("X-Request-ID", token)
				w.WriteHeader(status)
				w.Write(payload)
			})
			c, err := marketaux.NewClient(token, marketaux.WithBaseURL(srv.URL))
			if err != nil {
				t.Fatal(err)
			}
			_, err = c.News().All(context.Background(), marketaux.NewsParams{})
			var body []byte
			var apiErr *marketaux.APIError
			var decodeErr *marketaux.DecodeError
			switch {
			case errors.As(err, &apiErr):
				body = apiErr.Body
				if strings.Contains(apiErr.RequestID, token) {
					t.Fatal("credential leaked in request ID")
				}
			case errors.As(err, &decodeErr):
				body = decodeErr.Body
			default:
				t.Fatalf("expected API or decode error, got %v", err)
			}
			quoted, _ := json.Marshal(token)
			for _, secret := range []string{token, url.QueryEscape(token), string(quoted[1 : len(quoted)-1])} {
				if strings.Contains(err.Error(), secret) || strings.Contains(string(body), secret) {
					t.Fatal("credential leaked in error message or body")
				}
			}
		})
	}
}

func TestTransportRedactionPreservesErrorIdentity(t *testing.T) {
	t.Parallel()
	var original error
	hc := &http.Client{Timeout: time.Second, Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		original = fmt.Errorf("proxy failed for %s", r.URL.String())
		return nil, original
	})}
	c, err := marketaux.NewClient(testToken, marketaux.WithHTTPClient(hc),
		marketaux.WithRetry(marketaux.RetryPolicy{MaxAttempts: 1}))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.Entities().Types(context.Background())
	if !errors.Is(err, original) || !errors.Is(err, marketaux.ErrTransport) {
		t.Fatalf("error identity lost: %v", err)
	}
	var transportErr *marketaux.TransportError
	if !errors.As(err, &transportErr) || strings.Contains(transportErr.Err.Error(), testToken) || strings.Contains(err.Error(), testToken) {
		t.Fatal("credential leaked from transport error")
	}
}

func TestRetriesRespectRateLimit(t *testing.T) {
	t.Parallel()
	h, calls := flaky(t, 500, 2, "news_all.json")
	srv := newServer(t, h)
	c := newTestClient(t, srv, marketaux.WithRateLimit(20, 1))
	start := time.Now()
	if _, err := c.News().All(context.Background(), marketaux.NewsParams{}); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 3 || time.Since(start) < 90*time.Millisecond {
		t.Fatal("retry attempts bypassed rate limiter")
	}
}

func TestOversizedResponseIsNotRetried(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	srv := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Write([]byte(strings.Repeat("x", 100)))
	})
	c := newTestClient(t, srv, marketaux.WithMaxResponseBytes(10))
	_, err := c.Entities().Types(context.Background())
	if !errors.Is(err, marketaux.ErrResponseTooLarge) || !errors.Is(err, marketaux.ErrTransport) || calls.Load() != 1 {
		t.Fatalf("got %v, attempts %d", err, calls.Load())
	}
}

func TestBodyReadFailureClassification(t *testing.T) {
	t.Parallel()
	for _, canceled := range []bool{false, true} {
		t.Run(fmt.Sprint(canceled), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var calls atomic.Int32
			hc := &http.Client{Timeout: time.Second, Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(readerFunc(func([]byte) (int, error) {
					if canceled {
						cancel()
						return 0, context.Canceled
					}
					return 0, context.DeadlineExceeded
				}))}, nil
			})}
			c, err := marketaux.NewClient(testToken, marketaux.WithHTTPClient(hc))
			if err != nil {
				t.Fatal(err)
			}
			_, err = c.Entities().Types(ctx)
			if canceled && !errors.Is(err, context.Canceled) {
				t.Fatalf("expected cancellation, got %v", err)
			}
			if !canceled && !errors.Is(err, marketaux.ErrTimeout) {
				t.Fatalf("expected timeout, got %v", err)
			}
			if calls.Load() != 1 {
				t.Fatalf("body failure retried %d times", calls.Load())
			}
		})
	}
}

func TestCrossOriginRedirectRejected(t *testing.T) {
	t.Parallel()
	var leaked atomic.Bool
	target := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		leaked.Store(true)
		w.Write([]byte(`{"data":[]}`))
	})
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+"?api_token="+testToken, http.StatusFound)
	})
	c := newTestClient(t, srv, marketaux.WithHTTPClient(&http.Client{
		Timeout:       time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return nil },
	}), marketaux.WithRetry(marketaux.RetryPolicy{MaxAttempts: 1}))
	_, err := c.Entities().Types(context.Background())
	if err == nil || leaked.Load() || strings.Contains(err.Error(), testToken) {
		t.Fatal("cross-origin redirect was followed or exposed the token")
	}
}

func TestUnfollowedRedirectIsAPIError(t *testing.T) {
	t.Parallel()
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/other", http.StatusFound)
	})
	c := newTestClient(t, srv, marketaux.WithHTTPClient(&http.Client{
		Timeout:       time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}))
	_, err := c.Entities().Types(context.Background())
	var apiErr *marketaux.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusFound {
		t.Fatalf("expected HTTP error for unfollowed redirect, got %v", err)
	}
}

func TestInvalidNumericConfiguration(t *testing.T) {
	t.Parallel()
	for _, rate := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		_, err := marketaux.NewClient(testToken, marketaux.WithRateLimit(rate, 1))
		if !errors.Is(err, marketaux.ErrInvalidConfig) {
			t.Fatalf("invalid rate accepted: %v", rate)
		}
	}
	for _, opt := range []marketaux.Option{
		marketaux.WithBaseURL("https://user:password@example.com/v1"),
		marketaux.WithUserAgent("bad\r\nheader: value"),
	} {
		if _, err := marketaux.NewClient(testToken, opt); !errors.Is(err, marketaux.ErrInvalidConfig) {
			t.Fatalf("expected config error, got %v", err)
		}
	}
}

func TestPaginationUsesCappedServerLimit(t *testing.T) {
	t.Parallel()
	for _, eager := range []bool{false, true} {
		t.Run(fmt.Sprint(eager), func(t *testing.T) {
			var calls atomic.Int32
			srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Query().Get("page") == "1" {
					w.Write([]byte(`{"meta":{"returned":2,"limit":2},"data":[{"uuid":"a"},{"uuid":"b"}]}`))
					return
				}
				if r.URL.Query().Get("limit") != "2" {
					t.Error("next page did not use the effective server limit")
				}
				w.Write([]byte(`{"meta":{"returned":1,"limit":2},"data":[{"uuid":"c"}]}`))
			})
			c := newTestClient(t, srv)
			var count int
			var err error
			if eager {
				var articles []marketaux.Article
				articles, err = c.News().AllPages(context.Background(), marketaux.NewsParams{Limit: 100}, 3)
				count = len(articles)
			} else {
				count, err = collect(t, c, marketaux.NewsParams{Limit: 100})
			}
			if err != nil || count != 3 || calls.Load() != 2 {
				t.Fatalf("count %d, calls %d, error %v", count, calls.Load(), err)
			}
		})
	}
}

func TestPaginationStopsBeforeInvalidNextPage(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	srv := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Write([]byte(`{"meta":{"returned":3,"limit":3},"data":[{},{},{}]}`))
	})
	c := newTestClient(t, srv)
	n, err := collect(t, c, marketaux.NewsParams{Page: 6666, Limit: 3})
	if err != nil || n != 3 || calls.Load() != 1 {
		t.Fatalf("count %d, calls %d, error %v", n, calls.Load(), err)
	}
}

func TestPaginationRejectsNegativePageAndStopsOnBreak(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	srv := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Write([]byte(`{"meta":{"returned":1,"limit":1},"data":[{}]}`))
	})
	c := newTestClient(t, srv)
	if _, err := collect(t, c, marketaux.NewsParams{Page: -1}); !errors.Is(err, marketaux.ErrValidation) {
		t.Fatalf("negative page accepted: %v", err)
	}
	if calls.Load() != 0 {
		t.Fatal("invalid pagination sent a request")
	}
	for _, err := range c.News().Paginate(context.Background(), marketaux.NewsParams{}) {
		if err != nil {
			t.Fatal(err)
		}
		break
	}
	if calls.Load() != 1 {
		t.Fatal("break fetched an extra page")
	}
}

func TestMetadataPaginationRespectsResultBound(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	srv := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Write([]byte(`{"meta":{"returned":50,"limit":50},"data":[{}]}`))
	})
	c := newTestClient(t, srv)
	for _, err := range c.Entities().Paginate(context.Background(), marketaux.EntitySearchParams{Page: 400}) {
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, err := range c.Sources().Paginate(context.Background(), marketaux.SourcesParams{Page: 400}) {
		if err != nil {
			t.Fatal(err)
		}
	}
	if calls.Load() != 2 {
		t.Fatalf("requests beyond result bound: %d", calls.Load())
	}
}
