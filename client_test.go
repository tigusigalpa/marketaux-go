package marketaux_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	marketaux "github.com/tigusigalpa/marketaux-go"
)

const testToken = "test-token-123"

// fixture returns the contents of a testdata file.
func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return b
}

// fastRetry is a retry policy that never slows tests down.
var fastRetry = marketaux.RetryPolicy{
	MaxAttempts: 3,
	BaseDelay:   time.Millisecond,
	MaxDelay:    2 * time.Millisecond,
	Jitter:      false,
}

// newTestClient returns a client pointed at srv with a fast retry policy.
func newTestClient(t *testing.T, srv *httptest.Server, opts ...marketaux.Option) *marketaux.Client {
	t.Helper()
	opts = append([]marketaux.Option{
		marketaux.WithBaseURL(srv.URL),
		marketaux.WithRetry(fastRetry),
	}, opts...)
	c, err := marketaux.NewClient(testToken, opts...)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

// newServer returns a test server serving the given handler.
func newServer(t *testing.T, h http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv
}

func TestNewClientConfigErrors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		opts []marketaux.Option
		tok  string
	}{
		{"empty token", nil, ""},
		{"blank token", nil, "   "},
		{"bad base url", []marketaux.Option{marketaux.WithBaseURL("http://[::1")}, testToken},
		{"base url bad scheme", []marketaux.Option{marketaux.WithBaseURL("ftp://x")}, testToken},
		{"base url no host", []marketaux.Option{marketaux.WithBaseURL("https://")}, testToken},
		{"zero timeout", []marketaux.Option{marketaux.WithTimeout(0)}, testToken},
		{"negative timeout", []marketaux.Option{marketaux.WithTimeout(-time.Second)}, testToken},
		{"nil http client", []marketaux.Option{marketaux.WithHTTPClient(nil)}, testToken},
		{"empty user agent", []marketaux.Option{marketaux.WithUserAgent("  ")}, testToken},
		{"retry zero attempts", []marketaux.Option{marketaux.WithRetry(marketaux.RetryPolicy{MaxAttempts: 0})}, testToken},
		{"retry negative delay", []marketaux.Option{marketaux.WithRetry(marketaux.RetryPolicy{MaxAttempts: 1, BaseDelay: -1})}, testToken},
		{"retry max < base", []marketaux.Option{marketaux.WithRetry(marketaux.RetryPolicy{MaxAttempts: 1, BaseDelay: 2 * time.Second, MaxDelay: time.Second})}, testToken},
		{"rate limit rps", []marketaux.Option{marketaux.WithRateLimit(0, 1)}, testToken},
		{"rate limit burst", []marketaux.Option{marketaux.WithRateLimit(1, 0)}, testToken},
		{"nil logger", []marketaux.Option{marketaux.WithLogger(nil)}, testToken},
		{"zero max bytes", []marketaux.Option{marketaux.WithMaxResponseBytes(0)}, testToken},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := marketaux.NewClient(tc.tok, tc.opts...)
			if err == nil {
				t.Fatal("expected error")
			}
			var cfgErr *marketaux.ConfigError
			if !errors.As(err, &cfgErr) {
				t.Fatalf("expected *ConfigError, got %T", err)
			}
			if !errors.Is(err, marketaux.ErrInvalidConfig) {
				t.Fatalf("expected ErrInvalidConfig, got %v", err)
			}
		})
	}
}

func TestNewClientDefaults(t *testing.T) {
	t.Parallel()
	c, err := marketaux.NewClient(testToken)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if c.BaseURL() != "https://api.marketaux.com/v1" {
		t.Fatalf("BaseURL = %q", c.BaseURL())
	}
	if c.News() == nil || c.Stats() == nil || c.Entities() == nil || c.Sources() == nil {
		t.Fatal("services must not be nil")
	}
	c.CloseIdleConnections() // must not panic
}

func TestWithBaseURLTrailingSlash(t *testing.T) {
	t.Parallel()
	c, err := marketaux.NewClient(testToken, marketaux.WithBaseURL("https://example.com/v1/"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if c.BaseURL() != "https://example.com/v1" {
		t.Fatalf("BaseURL = %q", c.BaseURL())
	}
}

func TestWithHTTPClientDoesNotMutate(t *testing.T) {
	t.Parallel()
	orig := &http.Client{Timeout: 42 * time.Second}
	_, err := marketaux.NewClient(testToken,
		marketaux.WithHTTPClient(orig),
		marketaux.WithTimeout(time.Second),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if orig.Timeout != 42*time.Second {
		t.Fatalf("original http.Client mutated: %v", orig.Timeout)
	}
}

func TestRequestHeadersAndToken(t *testing.T) {
	t.Parallel()
	var gotToken, gotAccept, gotUA string
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotToken = r.URL.Query().Get("api_token")
		gotAccept = r.Header.Get("Accept")
		gotUA = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/json")
		w.Write(fixture(t, "news_all.json"))
	})
	c := newTestClient(t, srv)
	if _, err := c.News().All(context.Background(), marketaux.NewsParams{}); err != nil {
		t.Fatalf("All: %v", err)
	}
	if gotToken != testToken {
		t.Fatalf("api_token = %q", gotToken)
	}
	if gotAccept != "application/json" {
		t.Fatalf("Accept = %q", gotAccept)
	}
	if gotUA != "marketaux-go/"+marketaux.Version {
		t.Fatalf("User-Agent = %q", gotUA)
	}
	if strings.Contains(gotUA, "Bearer") {
		t.Fatal("bearer token must not be used")
	}
}

func TestEndpointPaths(t *testing.T) {
	t.Parallel()
	uuid := "9b9fb67e-9438-4263-815e-ef6a7d36af07"
	var mu sync.Mutex
	got := map[string]int{}
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		got[r.URL.Path]++
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/news/all":
			w.Write(fixture(t, "news_all.json"))
		case "/news/similar/" + uuid:
			w.Write(fixture(t, "news_similar.json"))
		case "/news/uuid/" + uuid:
			w.Write(fixture(t, "news_uuid.json"))
		case "/entity/stats/intraday":
			w.Write(fixture(t, "stats_intraday.json"))
		case "/entity/stats/aggregation":
			w.Write(fixture(t, "stats_aggregation.json"))
		case "/entity/trending/aggregation":
			w.Write(fixture(t, "trending.json"))
		case "/entity/search":
			w.Write(fixture(t, "entity_search.json"))
		case "/entity/type/list":
			w.Write(fixture(t, "entity_types.json"))
		case "/entity/industry/list":
			w.Write(fixture(t, "entity_industries.json"))
		case "/news/sources":
			w.Write(fixture(t, "sources.json"))
		default:
			http.Error(w, `{"error":{"code":"invalid_api_endpoint","message":"The requested API endpoint is invalid."}}`, http.StatusNotFound)
		}
	})
	c := newTestClient(t, srv)
	ctx := context.Background()

	if _, err := c.News().All(ctx, marketaux.NewsParams{}); err != nil {
		t.Errorf("News.All: %v", err)
	}
	if _, err := c.News().Similar(ctx, uuid, marketaux.SimilarParams{}); err != nil {
		t.Errorf("News.Similar: %v", err)
	}
	if _, err := c.News().ByUUID(ctx, uuid); err != nil {
		t.Errorf("News.ByUUID: %v", err)
	}
	if _, err := c.Stats().Intraday(ctx, marketaux.StatsParams{}); err != nil {
		t.Errorf("Stats.Intraday: %v", err)
	}
	if _, err := c.Stats().Aggregation(ctx, marketaux.AggregationParams{}); err != nil {
		t.Errorf("Stats.Aggregation: %v", err)
	}
	if _, err := c.Stats().Trending(ctx, marketaux.TrendingParams{}); err != nil {
		t.Errorf("Stats.Trending: %v", err)
	}
	if _, err := c.Entities().Search(ctx, marketaux.EntitySearchParams{}); err != nil {
		t.Errorf("Entities.Search: %v", err)
	}
	if _, err := c.Entities().Types(ctx); err != nil {
		t.Errorf("Entities.Types: %v", err)
	}
	if _, err := c.Entities().Industries(ctx); err != nil {
		t.Errorf("Entities.Industries: %v", err)
	}
	if _, err := c.Sources().All(ctx, marketaux.SourcesParams{}); err != nil {
		t.Errorf("Sources.All: %v", err)
	}

	for _, p := range []string{
		"/news/all", "/news/similar/" + uuid, "/news/uuid/" + uuid,
		"/entity/stats/intraday", "/entity/stats/aggregation",
		"/entity/trending/aggregation", "/entity/search",
		"/entity/type/list", "/entity/industry/list", "/news/sources",
	} {
		if got[p] == 0 {
			t.Errorf("path %s was never requested", p)
		}
	}
}

func TestErrorMapping(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		status   int
		fixture  string
		want     error
		wantCode string
	}{
		{"400", 400, "error_400.json", marketaux.ErrBadRequest, "malformed_parameters"},
		{"401", 401, "error_401.json", marketaux.ErrUnauthorized, "invalid_api_token"},
		{"402", 402, "error_402.json", marketaux.ErrUsageLimit, "usage_limit_reached"},
		{"403", 403, "error_403.json", marketaux.ErrForbidden, "endpoint_access_restricted"},
		{"404 resource", 404, "error_404.json", marketaux.ErrNotFound, "resource_not_found"},
		{"429", 429, "error_429.json", marketaux.ErrRateLimit, "rate_limit_reached"},
		{"500", 500, "error_500.json", marketaux.ErrServer, "server_error"},
		{"503", 503, "error_503.json", marketaux.ErrMaintenance, "maintenance_mode"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			srv := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-RateLimit-Limit", "60")
				w.Header().Set("X-UsageLimit-Limit", "100")
				w.Header().Set("Retry-After", "0")
				w.WriteHeader(tc.status)
				w.Write(fixture(t, tc.fixture))
			})
			c := newTestClient(t, srv)
			_, err := c.News().All(context.Background(), marketaux.NewsParams{})
			if err == nil {
				t.Fatal("expected error")
			}
			if !errors.Is(err, tc.want) {
				t.Fatalf("errors.Is(_, %v) failed for %v", tc.want, err)
			}
			var apiErr *marketaux.APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("expected *APIError, got %T", err)
			}
			if apiErr.Code != tc.wantCode {
				t.Fatalf("code = %q, want %q", apiErr.Code, tc.wantCode)
			}
			if apiErr.StatusCode != tc.status {
				t.Fatalf("status = %d", apiErr.StatusCode)
			}
			if strings.Contains(apiErr.URL, testToken) || strings.Contains(err.Error(), testToken) {
				t.Fatal("token leaked into error")
			}
		})
	}
}

func TestEndpointNotFoundError(t *testing.T) {
	t.Parallel()
	srv := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":{"code":"invalid_api_endpoint","message":"The requested API endpoint is invalid."}}`))
	})
	c := newTestClient(t, srv)
	_, err := c.News().ByUUID(context.Background(), "9b9fb67e-9438-4263-815e-ef6a7d36af07")
	if !errors.Is(err, marketaux.ErrEndpointNotFound) {
		t.Fatalf("expected ErrEndpointNotFound, got %v", err)
	}
}

func TestAPIErrorRateLimitHeaders(t *testing.T) {
	t.Parallel()
	srv := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-RateLimit-Limit", "120")
		w.Header().Set("Retry-After", "1")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write(fixture(t, "error_429.json"))
	})
	c := newTestClient(t, srv, marketaux.WithRetry(marketaux.RetryPolicy{MaxAttempts: 1}))
	_, err := c.News().All(context.Background(), marketaux.NewsParams{})
	var apiErr *marketaux.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %v", err)
	}
	if apiErr.RateLimitLimit != "120" {
		t.Fatalf("RateLimitLimit = %q", apiErr.RateLimitLimit)
	}
	if apiErr.RetryAfter != time.Second {
		t.Fatalf("RetryAfter = %v", apiErr.RetryAfter)
	}
}

func TestUsageLimitHeader(t *testing.T) {
	t.Parallel()
	srv := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-UsageLimit-Limit", "100")
		w.WriteHeader(http.StatusPaymentRequired)
		w.Write(fixture(t, "error_402.json"))
	})
	c := newTestClient(t, srv)
	_, err := c.News().All(context.Background(), marketaux.NewsParams{})
	var apiErr *marketaux.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %v", err)
	}
	if apiErr.UsageLimitLimit != "100" {
		t.Fatalf("UsageLimitLimit = %q", apiErr.UsageLimitLimit)
	}
}

func TestDecodeError(t *testing.T) {
	t.Parallel()
	srv := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("this is not json"))
	})
	c := newTestClient(t, srv)
	_, err := c.News().All(context.Background(), marketaux.NewsParams{})
	if !errors.Is(err, marketaux.ErrDecode) {
		t.Fatalf("expected ErrDecode, got %v", err)
	}
	var decErr *marketaux.DecodeError
	if !errors.As(err, &decErr) {
		t.Fatalf("expected *DecodeError, got %T", err)
	}
	if len(decErr.Body) == 0 {
		t.Fatal("expected body preview")
	}
}

func TestTransportError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close() // immediately dead server

	c, err := marketaux.NewClient(testToken,
		marketaux.WithBaseURL(url),
		marketaux.WithRetry(marketaux.RetryPolicy{MaxAttempts: 1}),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_, err = c.News().All(context.Background(), marketaux.NewsParams{})
	if !errors.Is(err, marketaux.ErrTransport) {
		t.Fatalf("expected ErrTransport, got %v", err)
	}
}

func TestTimeoutError(t *testing.T) {
	t.Parallel()
	srv := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(300 * time.Millisecond)
		w.Write(fixture(t, "news_all.json"))
	})
	c := newTestClient(t, srv, marketaux.WithTimeout(50*time.Millisecond),
		marketaux.WithRetry(marketaux.RetryPolicy{MaxAttempts: 1}))
	_, err := c.News().All(context.Background(), marketaux.NewsParams{})
	if !errors.Is(err, marketaux.ErrTimeout) {
		t.Fatalf("expected ErrTimeout, got %v", err)
	}
}

func TestContextCanceled(t *testing.T) {
	t.Parallel()
	srv := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Write(fixture(t, "news_all.json"))
	})
	c := newTestClient(t, srv)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.News().All(ctx, marketaux.NewsParams{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestResponseTooLarge(t *testing.T) {
	t.Parallel()
	srv := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Write(fixture(t, "news_all.json"))
	})
	c := newTestClient(t, srv, marketaux.WithMaxResponseBytes(100))
	_, err := c.News().All(context.Background(), marketaux.NewsParams{})
	if !errors.Is(err, marketaux.ErrTransport) {
		t.Fatalf("expected ErrTransport, got %v", err)
	}
}

func TestConcurrentUse(t *testing.T) {
	t.Parallel()
	srv := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Write(fixture(t, "news_all.json"))
	})
	c := newTestClient(t, srv)
	var wg sync.WaitGroup
	errs := make(chan error, 50)
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := c.News().All(context.Background(), marketaux.NewsParams{
				Symbols: []string{"TSLA"},
			}); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("concurrent request failed: %v", err)
	}
}

func TestQueryEncoding(t *testing.T) {
	t.Parallel()
	var got url.Values
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Query()
		w.Write(fixture(t, "news_all.json"))
	})
	c := newTestClient(t, srv)
	now := time.Date(2024, 11, 10, 12, 30, 0, 0, time.UTC)
	day := time.Date(2024, 11, 9, 0, 0, 0, 0, time.UTC)
	_, err := c.News().All(context.Background(), marketaux.NewsParams{
		Symbols:          []string{"TSLA", " AMZN ", "", "TSLA"},
		EntityTypes:      []marketaux.EntityType{marketaux.EntityTypeEquity, marketaux.EntityTypeIndex},
		Industries:       []string{"Technology", "Industrials"},
		Countries:        []string{"US", "ca"},
		SentimentGTE:     marketaux.Float64(-0.5),
		SentimentLTE:     marketaux.Float64(0.8),
		MinMatchScore:    marketaux.Float64(10.5),
		FilterEntities:   marketaux.Bool(true),
		MustHaveEntities: marketaux.Bool(false),
		GroupSimilar:     marketaux.Bool(true),
		Search:           `"ipo" -nyse`,
		Domains:          []string{"https://Adweek.com/some/path?q=1"},
		ExcludeDomains:   []string{"foo.com"},
		SourceIDs:        []string{"adweek.com-1"},
		ExcludeSourceIDs: []string{"x.com-2"},
		Languages:        []string{"EN", "es"},
		PublishedAfter:   &day,
		PublishedBefore:  &now,
		Sort:             marketaux.NewsSortEntityMatchScore,
		SortOrder:        marketaux.SortOrderAsc,
		Limit:            25,
		Page:             2,
	})
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	want := map[string]string{
		"symbols":            "TSLA,AMZN",
		"entity_types":       "equity,index",
		"industries":         "Technology,Industrials",
		"countries":          "us,ca",
		"sentiment_gte":      "-0.5",
		"sentiment_lte":      "0.8",
		"min_match_score":    "10.5",
		"filter_entities":    "true",
		"must_have_entities": "false",
		"group_similar":      "true",
		"search":             `"ipo" -nyse`,
		"domains":            "adweek.com",
		"exclude_domains":    "foo.com",
		"source_ids":         "adweek.com-1",
		"exclude_source_ids": "x.com-2",
		"language":           "en,es",
		"published_after":    "2024-11-09T00:00:00",
		"published_before":   "2024-11-10T12:30:00",
		"sort":               "entity_match_score",
		"sort_order":         "asc",
		"limit":              "25",
		"page":               "2",
	}
	for k, v := range want {
		if got.Get(k) != v {
			t.Errorf("param %s = %q, want %q", k, got.Get(k), v)
		}
	}
}

func TestEmptyParamsNotSent(t *testing.T) {
	t.Parallel()
	var got url.Values
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Query()
		w.Write(fixture(t, "news_all.json"))
	})
	c := newTestClient(t, srv)
	if _, err := c.News().All(context.Background(), marketaux.NewsParams{}); err != nil {
		t.Fatalf("All: %v", err)
	}
	for k := range got {
		if k != "api_token" {
			t.Errorf("unexpected param %q sent", k)
		}
	}
}

func TestPublishedOnFormat(t *testing.T) {
	t.Parallel()
	var got url.Values
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Query()
		w.Write(fixture(t, "news_all.json"))
	})
	c := newTestClient(t, srv)
	day := time.Date(2024, 3, 5, 18, 45, 30, 0, time.UTC)
	_, err := c.News().All(context.Background(), marketaux.NewsParams{
		PublishedOn: &day,
	})
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	if got.Get("published_on") != "2024-03-05" {
		t.Fatalf("published_on = %q", got.Get("published_on"))
	}
	if got.Has("published_before") || got.Has("published_after") {
		t.Fatal("date range must not be sent")
	}
}
