# marketaux-go

[![CI](https://github.com/tigusigalpa/marketaux-go/actions/workflows/ci.yml/badge.svg)](https://github.com/tigusigalpa/marketaux-go/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/tigusigalpa/marketaux-go.svg)](https://pkg.go.dev/github.com/tigusigalpa/marketaux-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/tigusigalpa/marketaux-go)](https://goreportcard.com/report/github.com/tigusigalpa/marketaux-go)
[![codecov](https://codecov.io/gh/tigusigalpa/marketaux-go/branch/main/graph/badge.svg)](https://codecov.io/gh/tigusigalpa/marketaux-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-%3E%3D1.23-blue.svg)](go.mod)

**English** | [Русский](README.ru.md)

Idiomatic, zero-dependency Go client for the [Marketaux](https://www.marketaux.com)
financial news API. Covers all 10 documented endpoints with typed params,
typed errors and context-first design — standard library only.

## Features

- All 10 API endpoints: news feed, similar news, news by UUID, entity stats
  (time series + aggregation), trending entities, entity search, entity
  types, industries, news sources
- `context.Context` in every method; `Client` is safe for concurrent use
- Functional options: timeouts, custom `http.Client`, retries, rate limit,
  logging, response size cap
- Typed errors compatible with `errors.Is`/`errors.As`
- Automatic retries with exponential backoff, jitter and `Retry-After`
- Client-side token-bucket rate limiter
- Lazy pagination via `iter.Seq2` (Go 1.23+)
- Client-side validation of all documented parameters
- API token redaction in errors and debug logs
- **Zero external dependencies**

## Table of Contents

- [Requirements](#requirements)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Service Map](#service-map)
- [Method Usage](#method-usage)
- [Params & Validation](#params--validation)
- [Pagination](#pagination)
- [Error Handling](#error-handling)
- [Retries & Rate Limits](#retries--rate-limits)
- [Plans & Limits](#plans--limits)
- [Concurrency](#concurrency)
- [Testing](#testing)
- [Compatibility](#compatibility)
- [Security](#security)
- [Endpoint Documentation](#endpoint-documentation)
- [Contributing](#contributing)
- [License](#license)

## Requirements

- Go 1.23+
- A Marketaux API token — get one at <https://www.marketaux.com/register>

## Installation

```bash
go get github.com/tigusigalpa/marketaux-go
```

## Quick Start

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/tigusigalpa/marketaux-go"
)

func main() {
	client, err := marketaux.NewClient(os.Getenv("MARKETAUX_API_TOKEN"))
	if err != nil {
		log.Fatal(err)
	}

	page, err := client.News().All(context.Background(), marketaux.NewsParams{
		Symbols:        []string{"TSLA", "AMZN"},
		Languages:      []string{"en"},
		FilterEntities: marketaux.Bool(true),
		SentimentGTE:   marketaux.Float64(0),
		Limit:          50,
	})
	if err != nil {
		log.Fatal(err)
	}
	for _, a := range page.Data {
		fmt.Println(a.Title, "-", a.URL)
		for _, e := range a.Entities {
			fmt.Println("  ", e.Symbol, e.SentimentScore)
		}
	}
}
```

## Configuration

`NewClient(token, opts...)` accepts functional options:

| Option | Default | Description |
| --- | --- | --- |
| `WithBaseURL(raw)` | `https://api.marketaux.com/v1` | Custom API base (tests, proxies) |
| `WithHTTPClient(hc)` | `http.Client{Timeout: 30s}` | Custom HTTP client (not mutated) |
| `WithTimeout(d)` | `30s` | Per-request timeout |
| `WithUserAgent(ua)` | `marketaux-go/1.0.0` | `User-Agent` header |
| `WithRetry(p)` | `DefaultRetryPolicy` | Retry policy, `MaxAttempts=1` disables |
| `WithRateLimit(rps, burst)` | off | Client-side token bucket |
| `WithLogger(l)` | discard | `*slog.Logger` for debug output |
| `WithDebug(bool)` | `false` | Debug request logging (token redacted) |
| `WithMaxResponseBytes(n)` | `32 MiB` | Response body size cap |

Environment variables used by examples: `MARKETAUX_API_TOKEN`,
`MARKETAUX_BASE_URL` (optional, smoke test only).

```go
client, err := marketaux.NewClient(token,
	marketaux.WithTimeout(30*time.Second),
	marketaux.WithRetry(marketaux.DefaultRetryPolicy),
	marketaux.WithRateLimit(5, 2),
	marketaux.WithDebug(true),
)
```

## Service Map

| Service method | HTTP endpoint | Availability | Documentation |
| --- | --- | --- | --- |
| `News().All` | `GET /news/all` | all plans | [#finance-news](https://www.marketaux.com/documentation#finance-news) |
| `News().Similar` | `GET /news/similar/{uuid}` | all plans | [#similar-news](https://www.marketaux.com/documentation#similar-news) |
| `News().ByUUID` | `GET /news/uuid/{uuid}` | all plans | [#news-by-uuid](https://www.marketaux.com/documentation#news-by-uuid) |
| `Stats().Intraday` | `GET /entity/stats/intraday` | Standard+ | [#entity-stats](https://www.marketaux.com/documentation#entity-stats) |
| `Stats().Aggregation` | `GET /entity/stats/aggregation` | Standard+ | [#entity-aggregation](https://www.marketaux.com/documentation#entity-aggregation) |
| `Stats().Trending` | `GET /entity/trending/aggregation` | Standard+ | [#trending-aggregation](https://www.marketaux.com/documentation#trending-aggregation) |
| `Entities().Search` | `GET /entity/search` | all plans | [#entity-search](https://www.marketaux.com/documentation#entity-search) |
| `Entities().Types` | `GET /entity/type/list` | all plans | [#entity-types](https://www.marketaux.com/documentation#entity-types) |
| `Entities().Industries` | `GET /entity/industry/list` | all plans | [#entity-industries](https://www.marketaux.com/documentation#entity-industries) |
| `Sources().All` | `GET /news/sources` | all plans | [#sources](https://www.marketaux.com/documentation#sources) |

Errors: <https://www.marketaux.com/documentation#errors>

## Method Usage

### News feed — `News().All(ctx, NewsParams)`

Latest world financial news with entity analysis (match score, sentiment,
highlights). Docs: <https://www.marketaux.com/documentation#finance-news>

```go
page, err := client.News().All(ctx, marketaux.NewsParams{
	Symbols:          []string{"TSLA", "AMZN", "MSFT"},
	EntityTypes:      []marketaux.EntityType{marketaux.EntityTypeEquity},
	Countries:        []string{"us"},
	Languages:        []string{"en"},
	SentimentGTE:     marketaux.Float64(0.1),
	MinMatchScore:    marketaux.Float64(10),
	FilterEntities:   marketaux.Bool(true),
	MustHaveEntities: marketaux.Bool(true),
	Search:           `"ipo" -nyse`,
	Domains:          []string{"wsj.com", "bloomberg.com"},
	PublishedOn:      marketaux.Date(2026, time.November, 13),
	Sort:             marketaux.NewsSortEntitySentimentScore,
	SortOrder:        marketaux.SortOrderDesc,
	Limit:            50,
	Page:             1,
})
```

`NewsParams` fields map 1:1 to the documented query parameters: `symbols`,
`entity_types`, `industries`, `countries`, `sentiment_gte`, `sentiment_lte`,
`min_match_score`, `filter_entities`, `must_have_entities`, `group_similar`,
`search`, `domains`, `exclude_domains`, `source_ids`, `exclude_source_ids`,
`language`, `published_before`, `published_after`, `published_on`, `sort`,
`sort_order`, `limit`, `page`.

Search operators: `+` (AND), `|` (OR), `-` (negation), `"..."` (exact phrase),
`*` suffix (prefix), `()` (grouping), `\` (escape). Example: `"ipo" -nyse`.

Note: the API returns a `snippet` and a link — never the full article text
(see [Marketaux FAQ](https://www.marketaux.com/faq)). Also, the documented
sort value is `published_on` while the response field is `published_at`; the
SDK follows the documentation literally.

### Similar news — `News().Similar(ctx, uuid, SimilarParams)`

Articles similar to a known article UUID. Same filters as the news feed minus
`search`, `sort`, `sort_order` and `min_match_score` (not supported by the
API and impossible to express in `SimilarParams`).
Docs: <https://www.marketaux.com/documentation#similar-news>

```go
similar, err := client.News().Similar(ctx,
	"70cb577e-c2dd-4dde-b501-f713823a4939",
	marketaux.SimilarParams{Limit: 10},
)
```

### News by UUID — `News().ByUUID(ctx, uuid)`

A single article previously saved by UUID. Returns `ErrNotFound` for missing
articles. Docs: <https://www.marketaux.com/documentation#news-by-uuid>

```go
article, err := client.News().ByUUID(ctx,
	"9b9fb67e-9438-4263-815e-ef6a7d36af07")
```

### Entity stats time series — `Stats().Intraday(ctx, StatsParams)`

Entity statistics bucketed by `interval` (`minute`/`hour`/`day`/`week`/
`month`/`quarter`/`year`). Without `symbols` the server returns the best
entities by metric. Requires a Standard plan or higher.
Docs: <https://www.marketaux.com/documentation#entity-stats>

```go
series, err := client.Stats().Intraday(ctx, marketaux.StatsParams{
	Symbols:         []string{"TSLA", "AMZN", "MSFT"},
	Interval:        marketaux.IntervalDay,
	GroupBy:         marketaux.GroupBySymbol,
	Countries:       []string{"us"},
	PublishedAfter:  marketaux.Date(2026, time.October, 1),
	PublishedBefore: marketaux.Date(2026, time.October, 31),
	Sort:            marketaux.StatsSortTotalDocuments,
	DateOrder:       marketaux.DateOrderDesc,
})
```

Maximum look-back per interval: minute 7d, hour 31d, day 3y, week/month 5y,
quarter/year 10y. Longer periods are silently truncated by the server — see
`Interval.MaxDays()`.

### Entity stats aggregation — `Stats().Aggregation(ctx, AggregationParams)`

Same filters as the time series, aggregated over a single period. The
response `Meta` only carries `Returned` and `Limit`.
Docs: <https://www.marketaux.com/documentation#entity-aggregation>

```go
agg, err := client.Stats().Aggregation(ctx, marketaux.AggregationParams{
	GroupBy: marketaux.GroupByIndustry,
})
```

### Trending entities — `Stats().Trending(ctx, TrendingParams)`

Entities gaining traction over the period, ranked by `Score`.
Docs: <https://www.marketaux.com/documentation#trending-aggregation>

```go
trending, err := client.Stats().Trending(ctx, marketaux.TrendingParams{
	Countries: []string{"us"},
})
for _, e := range trending.Data {
	fmt.Println(e.Key, e.Score)
}
```

### Entity search — `Entities().Search(ctx, EntitySearchParams)`

Find supported entities by name/symbol. The page limit is fixed at 50.
Docs: <https://www.marketaux.com/documentation#entity-search>

```go
found, err := client.Entities().Search(ctx, marketaux.EntitySearchParams{
	Search: "tesla",
	Types:  []marketaux.EntityType{marketaux.EntityTypeEquity},
})
```

### Entity types — `Entities().Types(ctx)`

All supported entity types (`equity`, `index`, `etf`, `mutualfund`,
`currency`, `cryptocurrency`, plus any future values — unknown types decode
safely as `EntityType`). Docs:
<https://www.marketaux.com/documentation#entity-types>

### Industries — `Entities().Industries(ctx)`

All supported industries.
Docs: <https://www.marketaux.com/documentation#entity-industries>

### News sources — `Sources().All(ctx, SourcesParams)`

News sources usable in `domains`/`source_ids` filters. Page limit fixed at
50; `DistinctDomain` groups by unique domain (`SourceID` becomes `nil`).
Docs: <https://www.marketaux.com/documentation#sources>

```go
src, err := client.Sources().All(ctx, marketaux.SourcesParams{
	DistinctDomain: marketaux.Bool(true),
	Languages:      []string{"en"},
})
```

## Params & Validation

- Optional booleans/floats use pointers (`*bool`, `*float64`) so `nil` means
  "not sent" — the API applies your plan defaults. Helpers: `marketaux.Bool`,
  `marketaux.Float64`, `marketaux.Int`, `marketaux.Time`, `marketaux.Date`.
- `Limit`/`Page` of `0` are not sent; the server applies the plan default.
- Lists are CSV-encoded with trimming, deduplication and lowercasing where
  the API expects it (countries, languages, entity types). Domains are
  normalized (scheme/path stripped, host lowercased).
- Validation happens client-side and returns `*ValidationError` with a
  `Fields` map before any request is made:

```go
_, err := client.News().All(ctx, marketaux.NewsParams{
	PublishedOn:     marketaux.Date(2026, 11, 1),
	PublishedBefore: marketaux.Date(2026, 11, 2), // conflict
})
var verr *marketaux.ValidationError
if errors.As(err, &verr) {
	for field, reason := range verr.Fields {
		fmt.Println(field, reason)
	}
}
```

Rules enforced: sentiment in `[-1, 1]`, `published_on` mutually exclusive
with the date range, `published_after < published_before`, `sort_order` only
with entity score sorts, `relevance_score` requires `search`, valid enums,
and `page*limit <= 20000` (the API result-set bound).

Supported languages include `ar bg bn cs da de el en es et fa fi fr he hi hr
hu id it ja ko lt multi nl no pl pt ro ru sk sv ta th tr uk vi zh` (`multi`
= multilingual sources). The list may grow — the SDK does not hard-reject
languages.

## Pagination

`Paginate` returns an `iter.Seq2` iterator that fetches pages lazily:

```go
for article, err := range client.News().Paginate(ctx, marketaux.NewsParams{
	Symbols: []string{"TSLA"},
}) {
	if err != nil {
		return err
	}
	fmt.Println(article.Title)
}
```

Iteration stops at the last page (`returned < limit`), on empty data, at the
20000-result API bound, on error (yielded once as the second value) or when
you `break`. `Entities().Paginate` and `Sources().Paginate` work the same way
with the fixed limit of 50.

For eager collection use `News().AllPages(ctx, params, maxPages)`.

## Error Handling

Every failure is a typed error usable with `errors.Is`/`errors.As`:

```go
page, err := client.News().All(ctx, params)
switch {
case err == nil:
	// ok
case errors.Is(err, marketaux.ErrRateLimit):
	var apiErr *marketaux.APIError
	if errors.As(err, &apiErr) {
		fmt.Println("limit:", apiErr.RateLimitLimit, "retry after:", apiErr.RetryAfter)
	}
case errors.Is(err, marketaux.ErrUsageLimit):
	// daily plan quota exhausted
case errors.Is(err, marketaux.ErrForbidden):
	// endpoint not on your plan (e.g. stats on Free/Basic)
case errors.Is(err, marketaux.ErrValidation):
	// client-side parameter problem
default:
	return err
}
```

| Sentinel | HTTP/code |
| --- | --- |
| `ErrInvalidConfig` | bad `NewClient`/option values |
| `ErrValidation` | client-side param validation |
| `ErrTransport` | network failures |
| `ErrTimeout` | request timeouts |
| `ErrDecode` | invalid JSON responses |
| `ErrBadRequest` | 400 `malformed_parameters` |
| `ErrUnauthorized` | 401 `invalid_api_token` |
| `ErrUsageLimit` | 402 `usage_limit_reached` |
| `ErrForbidden` | 403 `endpoint_access_restricted` |
| `ErrNotFound` | 404 `resource_not_found` |
| `ErrEndpointNotFound` | 404 `invalid_api_endpoint` |
| `ErrRateLimit` | 429 `rate_limit_reached` |
| `ErrServer` | 5xx `server_error` |
| `ErrMaintenance` | 503 `maintenance_mode` |

`*APIError` exposes `StatusCode`, `Code`, `Message`, `URL` (redacted),
`RequestID`, `RateLimitLimit`, `UsageLimitLimit`, `RetryAfter` and `Body`
(truncated).

## Retries & Rate Limits

`DefaultRetryPolicy`: 3 attempts, 500 ms base delay, 8 s cap, full jitter.
Retried: `429`, `500`, `502`, `503`, `504` and network errors. Not retried:
`4xx` (except 429), timeouts and canceled contexts. `Retry-After` (seconds or
HTTP-date) takes precedence when longer than the computed backoff.

The server reports limits in `X-RateLimit-Limit` (requests per 60 s, on 429)
and `X-UsageLimit-Limit` (daily quota, on 402) — both land on `*APIError`.

For preemptive throttling use `WithRateLimit(rps, burst)` — an in-process
token bucket, safe for concurrent use.

## Plans & Limits

| Plan | Price/mo | Requests/day | Articles/req | Entities/stat req | Stats API |
| --- | --- | --- | --- | --- | --- |
| Free | $0 | 100 | 3 | 0 | no |
| Basic | $29 | 2 500 | 20 | 0 | no |
| Standard | $49 | 10 000 | 50 | 20 | yes |
| Pro | $99 | 25 000 | 100 | 100 | yes |
| Pro 50K | $199 | 50 000 | 100 | 100 | yes |

Source: <https://www.marketaux.com/pricing>. Stats endpoints return
`403 endpoint_access_restricted` on Free/Basic. Entity search and sources
have a fixed page limit of 50. `page × limit` must stay ≤ 20 000.

## Concurrency

`Client` (and all services obtained from it) is immutable after construction
and safe to share across goroutines. Verified under `go test -race`.

## Testing

```bash
go test ./...        # unit tests (httptest only, no network)
go test -race ./...  # race detector
make cover           # coverage report (≥90%)

# live API smoke test (opt-in):
MARKETAUX_API_TOKEN=... go test -tags=integration ./...
MARKETAUX_API_TOKEN=... go run ./examples/smoke
```

## Compatibility

Go 1.23–1.27, tested in CI on every supported minor version. `go.mod` has no
`require` directives — the module is standard library only. v1 keeps the
public API backward compatible.

## Security

- `api_token` is sent only as a query parameter — the API does not support
  `Authorization` headers.
- The token is masked (`api_token=***`) in `*APIError`/`TransportError`/
  `DecodeError` URLs and in debug logs.
- Never commit tokens; `.env` is gitignored.
- Report vulnerabilities privately per [SECURITY.md](SECURITY.md).

## Endpoint Documentation

| Endpoint | Docs |
| --- | --- |
| `/news/all` | https://www.marketaux.com/documentation#finance-news |
| `/news/similar/{uuid}` | https://www.marketaux.com/documentation#similar-news |
| `/news/uuid/{uuid}` | https://www.marketaux.com/documentation#news-by-uuid |
| `/entity/stats/intraday` | https://www.marketaux.com/documentation#entity-stats |
| `/entity/stats/aggregation` | https://www.marketaux.com/documentation#entity-aggregation |
| `/entity/trending/aggregation` | https://www.marketaux.com/documentation#trending-aggregation |
| `/entity/search` | https://www.marketaux.com/documentation#entity-search |
| `/entity/type/list` | https://www.marketaux.com/documentation#entity-types |
| `/entity/industry/list` | https://www.marketaux.com/documentation#entity-industries |
| `/news/sources` | https://www.marketaux.com/documentation#sources |

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Bug reports and feature requests are
welcome in GitHub issues — never include your API token.

## License

MIT © [Igor Sazonov](https://github.com/tigusigalpa) — see [LICENSE](LICENSE).
