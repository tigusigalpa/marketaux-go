# Marketaux Golang Client/SDK/Library

![Marketaux Golang Client/SDK/Library](https://i.postimg.cc/3xb1fbV7/marketaux-go-hero-code.jpg)

[![CI](https://github.com/tigusigalpa/marketaux-go/actions/workflows/ci.yml/badge.svg)](https://github.com/tigusigalpa/marketaux-go/actions/workflows/ci.yml)
[![Tests](https://github.com/tigusigalpa/marketaux-go/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/tigusigalpa/marketaux-go/actions/workflows/test.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/tigusigalpa/marketaux-go.svg)](https://pkg.go.dev/github.com/tigusigalpa/marketaux-go)
[![CodeQL](https://github.com/tigusigalpa/marketaux-go/actions/workflows/codeql.yml/badge.svg?branch=main)](https://github.com/tigusigalpa/marketaux-go/actions/workflows/codeql.yml)
[![Codecov](https://codecov.io/gh/tigusigalpa/marketaux-go/graph/badge.svg)](https://codecov.io/gh/tigusigalpa/marketaux-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-%3E%3D1.23-blue.svg)](go.mod)
[![GitHub Release](https://img.shields.io/github/v/release/tigusigalpa/marketaux-go?style=flat-square)](https://github.com/tigusigalpa/marketaux-go/releases)

Bring [Marketaux](https://www.marketaux.com) financial news into your Go
application. Find coverage of the companies you follow, explore sentiment
over time, or add a news feed to a dashboard. `marketaux-go` turns API
responses into Go structs and takes care of pagination, retries and request
limits along the way.

You can start with a token and one request, then add the configuration your
application needs. The package covers all 10 documented API endpoints, uses
only the standard library, and requires Go 1.23 or newer. This is a community
SDK maintained independently of Marketaux.

Marketaux provides article metadata, entity analysis, short snippets and links
to publishers. Full article text is not included in the API; see the
[Marketaux FAQ](https://www.marketaux.com/faq).

## Features

- All 10 API endpoints: news feed, similar news, news by UUID, entity stats
  (time series + aggregation), trending entities, entity search, entity
  types, industries, news sources
- Every API request accepts `context.Context`; one client can be shared by goroutines
- Functional options: timeouts, custom `http.Client`, retries, rate limit,
  logging, response size cap
- Typed errors compatible with `errors.Is`/`errors.As`
- Automatic retries with exponential backoff, jitter and `Retry-After`
- Client-side token-bucket rate limiter
- Lazy pagination via `iter.Seq2` (Go 1.23+)
- Typed parameters with client-side checks for invalid values and combinations
- API token redaction in errors and debug logs
- **Zero external dependencies**

## Table of Contents

- [Requirements](#requirements)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Choosing a Service](#choosing-a-service)
- [Recipes](#recipes)
- [Configuration](#configuration)
- [Service Map](#service-map)
- [Method Usage](#method-usage)
- [Reading Responses](#reading-responses)
- [Params & Validation](#params--validation)
- [Pagination](#pagination)
- [Error Handling](#error-handling)
- [Retries & Rate Limits](#retries--rate-limits)
- [Plans & Limits](#plans--limits)
- [Concurrency](#concurrency)
- [Using the SDK in a Service](#using-the-sdk-in-a-service)
- [Testing](#testing)
- [Compatibility](#compatibility)
- [Security](#security)
- [Troubleshooting](#troubleshooting)
- [Examples](#examples)
- [Endpoint Documentation](#endpoint-documentation)
- [Contributing](#contributing)
- [License](#license)

## Requirements

- Go 1.23+; use a currently supported, patched Go release for production.
- A [Marketaux API token](https://www.marketaux.com/register). The news
  quick start uses a limit of 3 to fit the Free plan.

## Installation

Add the package to an existing Go module:

```bash
go get github.com/tigusigalpa/marketaux-go
```

For a new project, create a module first:

```bash
mkdir marketaux-example
cd marketaux-example
go mod init example.com/marketaux-example
go get github.com/tigusigalpa/marketaux-go
```

The import path ends in `marketaux-go`, while the Go package is named
`marketaux`. That is why the examples call `marketaux.NewClient` after
importing `github.com/tigusigalpa/marketaux-go`.

## Quick Start

Set your token in the shell where you run the program:

```bash
export MARKETAUX_API_TOKEN='your-token'
```

On PowerShell:

```powershell
$env:MARKETAUX_API_TOKEN = 'your-token'
```

Save the following as `main.go` in your own Go module, then run `go run .`:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/tigusigalpa/marketaux-go"
)

func main() {
	client, err := marketaux.NewClient(os.Getenv("MARKETAUX_API_TOKEN"))
	if err != nil {
		log.Fatal(err)
	}

	defer client.CloseIdleConnections()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	page, err := client.News().All(ctx, marketaux.NewsParams{
		Symbols:        []string{"TSLA", "AMZN"},
		Languages:      []string{"en"},
		FilterEntities: marketaux.Bool(true),
		SentimentGTE:   marketaux.Float64(0),
		Limit:          3,
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

Create one client and reuse it across requests and goroutines. A successful
request can return no articles: an empty `page.Data` is a normal result.
The library does not read environment variables or load `.env` files; your
application passes the token to `NewClient`.

The quick start asks for English-language news mentioning TSLA or AMZN and
keeps entity mentions with neutral or positive sentiment. `FilterEntities`
also trims the returned entity lists to the matching mentions. To see a
broader feed, remove `SentimentGTE`; to inspect every entity attached to a
matching article, remove `FilterEntities`.

Your output will change as new articles arrive. If the request succeeds with
no results, try just `Symbols` and `Languages` first, then add the other
filters one at a time.

## Choosing a Service

Start with the question your application needs to answer:

| Your goal | Start here | What you get |
| --- | --- | --- |
| Show news about a watchlist | `News().All` | A page of articles with entity mentions |
| Process matching articles across pages | `News().Paginate` | Articles as each page arrives |
| Reopen an article saved earlier | `News().ByUUID` | One article identified by its UUID |
| Explore related coverage | `News().Similar` | A page of similar articles |
| Compare activity or sentiment over time | `Stats().Intraday` | Dated buckets of entity statistics |
| Summarize a period or find trending entities | `Stats().Aggregation` / `Trending` | Aggregate counts, sentiment and trending scores |
| Find the right symbol for a company | `Entities().Search` | Names, symbols, exchanges and countries |
| Discover available filters | `Entities().Types`, `Industries`, `Sources().All` | Metadata you can use in later requests |

An entity is a company or another tracked financial instrument. Symbols are
the identifiers you pass to news and stats filters. If a company name gives
unexpected results, use entity search to check the symbol, country and
exchange before building the query. The [service map](#service-map) lists the
HTTP endpoint and documentation for every method.

## Recipes

These functions build on the quick start. They accept an existing client so
you can use the same connection pool and rate limiter throughout your app.
The snippets use the `context`, `fmt` and `time` packages where needed.

### Collect a small watchlist feed

For a dashboard or a periodic digest, it is often useful to collect a bounded
number of pages. This function fetches at most three pages of three articles
each, with the matching entities kept in the response:

```go
func watchlistNews(ctx context.Context, client *marketaux.Client, symbols []string) ([]marketaux.Article, error) {
	if len(symbols) == 0 {
		return nil, fmt.Errorf("provide at least one watchlist symbol")
	}
	return client.News().AllPages(ctx, marketaux.NewsParams{
		Symbols:        symbols,
		Languages:      []string{"en"},
		FilterEntities: marketaux.Bool(true),
		Limit:          3,
	}, 3)
}
```

For example, pass `[]string{"TSLA", "AMZN"}` as `symbols`. The result may
contain fewer than nine articles. `AllPages` returns an error if any page
fails; use the iterator below when you want to handle each article as it arrives.

### Process yesterday's articles without collecting them all

A daily job can select a calendar date and pass each article to your own
handler. Calculate the date in UTC, since the SDK sends dates in UTC too:

```go
func processYesterday(ctx context.Context, client *marketaux.Client, handle func(marketaux.Article) error) error {
	if handle == nil {
		return fmt.Errorf("provide an article handler")
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	yesterday := time.Now().UTC().AddDate(0, 0, -1)
	for article, err := range client.News().Paginate(ctx, marketaux.NewsParams{
		Languages:   []string{"en"},
		PublishedOn: marketaux.Date(yesterday.Year(), yesterday.Month(), yesterday.Day()),
		Limit:       3,
	}) {
		if err != nil {
			return err
		}
		if err := handle(article); err != nil {
			return err
		}
	}
	return nil
}
```

The 30-second context covers requests, retries and page transitions. Your
handler runs synchronously; if it does slow work, pass a context to that work
too. A failure leaves previously handled articles in your application, so
store their UUIDs if rerunning the job should skip duplicates. An iterator
can still use many requests: narrow the filters to fit your daily quota.

### Read a week of daily sentiment

Stats return buckets rather than articles. This example requests the previous
seven UTC calendar days and checks for missing sentiment before printing it.
It requires access to the stats endpoints:

```go
func printDailySentiment(ctx context.Context, client *marketaux.Client, symbols []string) error {
	if len(symbols) == 0 {
		return fmt.Errorf("provide at least one symbol")
	}
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	series, err := client.Stats().Intraday(ctx, marketaux.StatsParams{
		Symbols:         symbols,
		Interval:        marketaux.IntervalDay,
		GroupBy:         marketaux.GroupBySymbol,
		PublishedAfter:  marketaux.Time(today.AddDate(0, 0, -7)),
		PublishedBefore: marketaux.Time(today),
		DateOrder:       marketaux.DateOrderAsc,
	})
	if err != nil {
		return err
	}
	for _, point := range series.Data {
		for _, stat := range point.Data {
			if stat.SentimentAvg == nil {
				fmt.Printf("%s %s: no sentiment available\n", point.Date.Format("2006-01-02"), stat.Key)
				continue
			}
			fmt.Printf("%s %s: %d documents, sentiment %.3f\n",
				point.Date.Format("2006-01-02"), stat.Key, stat.TotalDocuments, *stat.SentimentAvg)
		}
	}
	return nil
}
```

Missing sentiment is different from a score of zero: zero is a reported
neutral value. Keep that distinction when you build a chart or calculate
your own summaries.

## Configuration

Start with the defaults unless you need different behavior. Each option
changes one part of the client and is applied when `NewClient` runs:

| Option | Default | Description |
| --- | --- | --- |
| `WithBaseURL(raw)` | `https://api.marketaux.com/v1` | Custom API base (tests, proxies) |
| `WithHTTPClient(hc)` | `http.Client{Timeout: 30s}` | Shallow copy; transport and cookie jar stay shared |
| `WithTimeout(d)` | `30s` | Timeout for each HTTP attempt, including reading its body |
| `WithUserAgent(ua)` | `marketaux-go/1.1.0` | `User-Agent` header |
| `WithRetry(p)` | `DefaultRetryPolicy` | Retry policy, `MaxAttempts=1` disables |
| `WithRateLimit(rps, burst)` | off | Client-side token bucket |
| `WithLogger(l)` | discard | `*slog.Logger` for debug output |
| `WithDebug(bool)` | `false` | Debug request logging (token redacted) |
| `WithMaxResponseBytes(n)` | `32 MiB` | Response body size cap |

Environment variables used by examples: `MARKETAUX_API_TOKEN`,
`MARKETAUX_BASE_URL` (optional, smoke example and integration test).

Options are applied from left to right. A custom HTTP client must have a
positive `Timeout`, or be followed by `WithTimeout`. For example:

```go
client, err := marketaux.NewClient(token,
	marketaux.WithHTTPClient(&http.Client{Transport: http.DefaultTransport}),
	marketaux.WithTimeout(10*time.Second),
	marketaux.WithRateLimit(5, 2),
)
if err != nil {
	return err
}
```

`WithTimeout` applies to one attempt. Use `context.WithTimeout` to bound the
whole operation, including time spent waiting for rate limits and retries.
For an iterator, that context covers the entire walk through the pages.

To see request logs, provide a logger whose handler accepts debug messages:

```go
logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
	Level: slog.LevelDebug,
}))
client, err := marketaux.NewClient(token,
	marketaux.WithLogger(logger),
	marketaux.WithDebug(true),
)
if err != nil {
	return err
}
```

Logging is discarded by default. `WithDebug(true)` alone does not create an
output destination. The SDK logs the method, redacted URL, status, attempt
number and duration; it does not log response bodies.

`WithBaseURL` accepts an HTTP(S) URL, keeps its path prefix and removes its
query and fragment. Use HTTPS outside local tests. URLs with embedded user
credentials are rejected. Redirects may stay on the same origin; a change of
scheme, host or port is rejected before the next request is sent, even with
a custom `CheckRedirect` callback.

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

The snippets below assume an existing `client` and `ctx`, inside a function
that returns an error. The [examples directory](examples/README.md) contains
complete programs.

### News feed — `News().All(ctx, NewsParams)`

Use this for the main article feed. Each article can include entity matches,
sentiment and highlighted mentions. The example below combines several
filters to show the available controls; for your first request, choose only
the ones you need. Docs: <https://www.marketaux.com/documentation#finance-news>

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
if err != nil {
	return err
}
for _, article := range page.Data {
	fmt.Println(article.Title, article.PublishedAt, article.URL)
}
```

`NewsParams` fields map 1:1 to the documented query parameters: `symbols`,
`entity_types`, `industries`, `countries`, `sentiment_gte`, `sentiment_lte`,
`min_match_score`, `filter_entities`, `must_have_entities`, `group_similar`,
`search`, `domains`, `exclude_domains`, `source_ids`, `exclude_source_ids`,
`language`, `published_before`, `published_after`, `published_on`, `sort`,
`sort_order`, `limit`, `page`.

Search operators: `+` (AND), `|` (OR), `-` (negation), `"..."` (exact phrase),
`*` suffix (prefix), `()` (grouping), `\` (escape). Example: `"ipo" -nyse`.

The documented date-sort value is `published_on`, while the timestamp in an
article is named `published_at`. The SDK keeps those API names, so use
`NewsSortPublishedOn` when constructing a request and `Article.PublishedAt`
when reading a response.

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
if err != nil {
	return err
}
fmt.Println(len(similar.Data), "similar articles")
```

### News by UUID — `News().ByUUID(ctx, uuid)`

Save an article's UUID when you need to look it up again later. This method
returns the article directly, without a `Data` envelope, and returns
`ErrNotFound` for a missing article.
Docs: <https://www.marketaux.com/documentation#news-by-uuid>

```go
article, err := client.News().ByUUID(ctx,
	"9b9fb67e-9438-4263-815e-ef6a7d36af07")
if err != nil {
	return err
}
fmt.Println(article.Title, article.Snippet)
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
	PublishedBefore: marketaux.Date(2026, time.November, 1),
	Sort:            marketaux.StatsSortTotalDocuments,
	DateOrder:       marketaux.DateOrderDesc,
})
if err != nil {
	return err
}
for _, point := range series.Data {
	fmt.Println(point.Date, point.Data)
}
```

Maximum look-back per interval: minute 7d, hour 31d, day 3y, week/month 5y,
quarter/year 10y. Longer periods are silently truncated by the server — see
`Interval.MaxDays()`.

### Entity stats aggregation — `Stats().Aggregation(ctx, AggregationParams)`

Use aggregation when you need a summary for a period rather than a chart
with one point per interval. It shares the time-series filters but omits
`Interval` and `DateOrder`. The response `Meta` carries `Returned` and `Limit`.
Docs: <https://www.marketaux.com/documentation#entity-aggregation>

```go
agg, err := client.Stats().Aggregation(ctx, marketaux.AggregationParams{
	GroupBy: marketaux.GroupByIndustry,
})
if err != nil {
	return err
}
for _, stat := range agg.Data {
	fmt.Println(stat.Key, stat.TotalDocuments)
}
```

### Trending entities — `Stats().Trending(ctx, TrendingParams)`

Find entities gaining attention during the requested period. Each result
includes a trending `Score` alongside document counts and average sentiment.
Docs: <https://www.marketaux.com/documentation#trending-aggregation>

```go
trending, err := client.Stats().Trending(ctx, marketaux.TrendingParams{
	Countries: []string{"us"},
})
if err != nil {
	return err
}
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
if err != nil {
	return err
}
for _, entity := range found.Data {
	fmt.Println(entity.Symbol, entity.Name, entity.Country)
}
```

### Entity types — `Entities().Types(ctx)`

All supported entity types (`equity`, `index`, `etf`, `mutualfund`,
`currency`, `cryptocurrency`, plus any future values — unknown types decode
safely as `EntityType`). Docs:
<https://www.marketaux.com/documentation#entity-types>

```go
types, err := client.Entities().Types(ctx)
if err != nil {
	return err
}
fmt.Println(types)
```

### Industries — `Entities().Industries(ctx)`

All supported industries.
Docs: <https://www.marketaux.com/documentation#entity-industries>

```go
industries, err := client.Entities().Industries(ctx)
if err != nil {
	return err
}
fmt.Println(industries)
```

Type and industry lists are good candidates for application-level caching;
the SDK makes a request on every call.

### News sources — `Sources().All(ctx, SourcesParams)`

Use this catalog to discover values for `Domains` and `SourceIDs` filters.
The page size is fixed at 50. `DistinctDomain` groups by domain, in which
case `SourceID` can be `nil`.
Docs: <https://www.marketaux.com/documentation#sources>

```go
src, err := client.Sources().All(ctx, marketaux.SourcesParams{
	DistinctDomain: marketaux.Bool(true),
	Languages:      []string{"en"},
})
if err != nil {
	return err
}
for _, source := range src.Data {
	fmt.Println(source.Domain, source.Language)
}
```

## Reading Responses

List methods return a response envelope. Read its `Data` field for the
results and `Meta` for page information. `ByUUID`, `Types` and `Industries`
return their values directly:

| Method | Result | Useful fields |
| --- | --- | --- |
| News feed / similar | `*NewsResponse` | `Meta`, `Data []Article` |
| Article by UUID | `*Article` | `Title`, `Snippet`, `URL`, `PublishedAt`, `Entities`, `Similar` |
| Intraday stats | `*TimeSeriesResponse` | `Data []TimeSeriesPoint`, each with `Date` and `Data []EntityStat` |
| Stats aggregation | `*StatsResponse` | `Data []EntityStat`, `Meta.Returned`, `Meta.Limit` |
| Trending | `*TrendingResponse` | `Data []TrendingEntity`, including `Score` and embedded `EntityStat` |
| Entity search | `*EntitiesResponse` | `Data []EntityInfo`, `Meta` |
| Sources | `*SourcesResponse` | `Data []Source`, `Meta` |
| Types / industries | `[]EntityType` / `[]string` | Metadata values directly, without an envelope |

`Meta.Found` describes the matching result set; it can be larger than the
20,000-result pagination window. `Meta.Returned` describes this page and
`Meta.Limit` is its effective size. Intraday stats have no `Meta` object.

Dates decode into `time.Time`, including fractional seconds. Some values are
optional: check `Article.RelevanceScore`, `Entity.Exchange`,
`Entity.ExchangeLong` and `EntityStat.SentimentAvg` for `nil` before using
them. `Source.SourceID` can also be `nil` with `DistinctDomain` enabled.
Nullable text fields such as `Snippet` decode as an empty string. Empty and
null collections are safe to range over, but a null collection may stay nil.

Sentiment belongs to an entity mention; one article can have different scores
for different companies. `Entity.IsPositive()` / `IsNegative()` are shortcuts
for checking the sign. `Article.IsAbout("TSLA")` compares symbols without
regard to case. `Highlight.Text()` removes `<em>` markers, unescapes entities
and normalizes whitespace; escape its result when inserting it into HTML.

Unknown entity types in responses are preserved. `Entity.TypeKnown()` or
`EntityType.Valid()` tells you whether the SDK recognizes a value. Unknown
JSON fields are ignored for forward compatibility.

### Keep article identity separate from display fields

Store `Article.UUID` when you need to recognize an article again, and keep
`Article.URL` as the link you show readers. A title is useful for display but
is a poor duplicate key: different articles can share similar titles.
`Article.Similar` contains related coverage when grouping is enabled; it is
separate from the top-level `Data` list that pagination walks.

The SDK returns the data supplied by the API. It does not persist articles,
deduplicate repeated polls, or refresh previously returned structs. Your
application decides what to store and how long to cache it.

## Params & Validation

Parameter structs keep request construction close to the API while letting
you use Go values. Leave a field at its zero value when you want the server
default; use the pointer helpers when an explicit false or numeric zero matters.

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

Non-finite floats (`NaN` and infinities), negative pagination values and a
negative `MinMatchScore` are rejected too. Plan-specific limits and access
permissions are checked by Marketaux, since the SDK does not know your plan.

### Parameter reference

| Go fields | Query parameters | Available in |
| --- | --- | --- |
| `Symbols`, `EntityTypes`, `Industries`, `Countries` | `symbols`, `entity_types`, `industries`, `countries` | News, similar, all stats |
| `SentimentGTE`, `SentimentLTE` | `sentiment_gte`, `sentiment_lte` | News, similar, all stats |
| `Domains`, `ExcludeDomains` | `domains`, `exclude_domains` | News, similar, all stats |
| `SourceIDs`, `ExcludeSourceIDs` | `source_ids`, `exclude_source_ids` | News, similar, all stats |
| `Languages` | `language` (singular) | News, similar, all stats, sources |
| `PublishedBefore`, `PublishedAfter`, `PublishedOn` | `published_before`, `published_after`, `published_on` | News, similar, all stats |
| `FilterEntities`, `MustHaveEntities`, `GroupSimilar` | `filter_entities`, `must_have_entities`, `group_similar` | News, similar |
| `MinMatchScore` | `min_match_score` | News |
| `Search` | `search` | News, all stats, entity search |
| `Sort`, `SortOrder` | `sort`, `sort_order` | News, all stats; enum types differ |
| `Limit` | `limit` | News, similar, all stats |
| `Page` | `page` | News, similar, entity search, sources |
| `GroupBy`, `MinDocCount` | `group_by`, `min_doc_count` | All stats |
| `SentimentAvgGTE`, `SentimentAvgLTE` | `sentiment_avg_gte`, `sentiment_avg_lte` | All stats |
| `Interval`, `DateOrder` | `interval`, `date_order` | Intraday stats |
| `Search`, `Symbols`, `Exchanges`, `Types`, `Industries`, `Countries` | `search`, `symbols`, `exchanges`, `types`, `industries`, `countries` | Entity search |
| `DistinctDomain` | `distinct_domain` | Sources |

The exact field types are in the [Go reference](https://pkg.go.dev/github.com/tigusigalpa/marketaux-go).
Similar news deliberately has its own params type: it does not accept the
main feed's `Search`, `MinMatchScore`, `Sort` or `SortOrder` fields.

`Countries` selects the country of an entity's exchange, while `Languages`
selects the language of the news source. `FilterEntities` trims the entity
list inside each article to your filters. `MustHaveEntities` requires an
article to contain at least one recognized entity. `GroupSimilar` controls
whether related articles are grouped into the article's `Similar` field.

Omitted is different from false or zero:

```go
params := marketaux.NewsParams{
	FilterEntities: marketaux.Bool(false), // explicitly send false
	SentimentGTE:   marketaux.Float64(0),  // explicitly include neutral sentiment
	PublishedOn:    marketaux.Date(2026, time.November, 1),
}
```

Dates are `*time.Time`. `Time(t)` preserves an instant; `Date(y, m, d)` creates
midnight UTC. Before/after values are converted to UTC and sent with second
precision; `PublishedOn` sends the UTC calendar date. The API also accepts
shorter date strings, but this SDK uses one canonical format. An upper bound
at midnight does not include the rest of that day.

Supported languages include `ar bg bn cs da de el en es et fa fi fr he hi hr
hu id it ja ko lt multi nl no pl pt ro ru sk sv ta th tr uk vi zh` (`multi`
= multilingual sources). The list may grow — the SDK does not hard-reject
languages.

## Pagination

Use `All` when one page is enough, `Paginate` when you want to process results
as they arrive, and `AllPages` when you need a slice with a page budget.
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

The iterator starts at `params.Page`, or page 1 when it is zero. No request
is made until iteration starts, and `break` does not prefetch another page.
If your plan caps the requested size, the SDK follows `Meta.Limit` and uses
that effective size for subsequent requests. It stops before the next page
would exceed `page × limit <= 20000`; with a size of 3, the last allowed page
is 6666. To explore more results, narrow the date range or other filters.

For eager collection:

```go
articles, err := client.News().AllPages(ctx, marketaux.NewsParams{
	Symbols: []string{"TSLA"},
	Limit:   3,
}, 5) // at most 5 pages
if err != nil {
	return err
}
fmt.Println(len(articles))
```

`maxPages <= 0` defaults to 400 pages, which may be fewer than the entire
matching result set. `AllPages` holds the results in memory and discards
partial results on error. With `Paginate`, you keep anything already handled.
Each fetched page, and potentially each retry, uses another API request.
Similar news supports manual pagination through `SimilarParams.Page`.

### Polling a changing feed

A news feed can change between requests. New arrivals can move articles
between page numbers while your application is walking the results. The
iterator follows page numbers; it does not create a snapshot of the feed.

For a periodic job, keep a record of handled UUIDs and use an explicit date
window. If you deliberately overlap successive windows, skip UUIDs you have
already stored. This makes repeated polling easier to recover after a
failure. Choose the overlap and retention period for your application; the
SDK does not maintain a polling cursor.

For historical work, divide large queries into smaller date windows and
monitor how many results each window contains. Reaching the pagination
boundary is a reason to narrow the query before assuming you have collected
the full matching set.

## Error Handling

Use `errors.Is` to choose what to do and `errors.As` to inspect details.
Context cancellation and deadlines remain standard Go context errors:

```go
page, err := client.News().All(ctx, params)
switch {
case err == nil:
	fmt.Println(len(page.Data), "articles")
case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
	return err
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
| `ErrResponseTooLarge` | body exceeds `WithMaxResponseBytes`; also matches `ErrTransport` |
| `ErrBadRequest` | 400 `malformed_parameters` |
| `ErrUnauthorized` | 401 `invalid_api_token` |
| `ErrUsageLimit` | 402 `usage_limit_reached` |
| `ErrForbidden` | 403 `endpoint_access_restricted` |
| `ErrNotFound` | 404 `resource_not_found` |
| `ErrEndpointNotFound` | 404 `invalid_api_endpoint` |
| `ErrRateLimit` | 429 `rate_limit_reached` |
| `ErrServer` | 5xx other than 503 |
| `ErrMaintenance` | 503 `maintenance_mode` |

`*APIError` exposes `StatusCode`, `Code`, `Message`, `URL` (redacted),
`RequestID`, `RateLimitLimit`, `UsageLimitLimit`, `RetryAfter` and `Body`
(redacted, up to 2,000 bytes). `*DecodeError` includes a redacted body preview
of up to 500 bytes. A response with valid JSON but an unexpected field type
can also produce `ErrDecode`. Unfollowed redirects are `*APIError` values
with their original 3xx status.

A client timeout matches both `ErrTimeout` and `ErrTransport`. An expired
caller context matches `context.DeadlineExceeded` instead. Check the context
errors separately when deciding whether to retry at the application level.

Treat a validation error as feedback about the request you constructed.
Retrying the same parameters will not fix it. An API error describes a
response from Marketaux, so the next step depends on the status: check the
token for 401, endpoint access for 403, and account quota for 402. For a
temporary failure, remember that the SDK may already have used its configured
retry attempts before returning control to you.

When reporting a problem, the error class, HTTP status and request ID are
usually more useful than a complete request URL. Keep response previews in
private diagnostics, since they may contain details of your query even after
the token has been masked.

## Retries & Rate Limits

`DefaultRetryPolicy`: 3 attempts total, 500 ms base delay, 8 s backoff cap.
Jitter chooses a delay between 50% and 100% of that backoff (equal jitter).
Retried: `429`, `500`, `502`, `503`, `504` and network errors. Not retried:
`4xx` (except 429), timeouts, oversized responses, decode errors and canceled
contexts. `Retry-After` (seconds or HTTP-date) takes precedence when longer
than the computed backoff.

Copy the default policy before changing it, or use a policy with one attempt
to disable retries:

```go
policy := marketaux.DefaultRetryPolicy
policy.MaxAttempts = 4
policy.BaseDelay = 250 * time.Millisecond
client, err := marketaux.NewClient(token, marketaux.WithRetry(policy))
if err != nil {
	return err
}
// To disable: marketaux.WithRetry(marketaux.RetryPolicy{MaxAttempts: 1})
```

`MaxDelay` caps the SDK backoff; it does not shorten a longer `Retry-After`.
A caller context deadline can interrupt either wait. Avoid stacking another
automatic retry loop around the SDK without considering total attempts.

The server reports limits in `X-RateLimit-Limit` (requests per 60 s, on 429)
and `X-UsageLimit-Limit` (daily quota, on 402) — both land on `*APIError`.

For preemptive throttling use `WithRateLimit(rps, burst)` — an in-process
token bucket, safe for concurrent use. It applies to every attempt, including
retries. It is local to this client instance; it does not coordinate separate
processes or enforce a shared account's daily quota. Cancellation interrupts
waiting without reserving future capacity. Retried requests can consume your
daily quota, so choose a policy that fits your workload.

### Budget requests before increasing concurrency

Page size, request rate and daily quota solve different problems. A larger
page can reduce the number of requests needed to walk a result set, if your
plan allows it. A rate limiter spaces attempts over time. Neither one adds
more requests to your account's daily allowance.

For example, processing five pages takes five HTTP attempts when every page
succeeds immediately. If one page needs two extra attempts, the operation
can make seven attempts. Plan for that headroom when you run several jobs
with the same account, and avoid retrying a daily-quota error in a tight loop.

## Plans & Limits

Monthly billing prices and limits checked on 17 November 2026. Pricing can
change; confirm the current terms on the linked Marketaux pricing page.

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

Custom transports, log handlers and cookie jars must also support concurrent
use. Do not modify the shared transport, `DefaultRetryPolicy`, or parameter
slices/pointer values while requests use them. Responses belong to the caller.
`CloseIdleConnections` releases idle connections in the underlying transport;
it does not cancel active requests or permanently close the client. With a
shared transport, this can also close its other idle connections.

## Using the SDK in a Service

Create the client when your application starts and keep it with the other
shared dependencies. Pass the incoming request's context to SDK methods.
For a background job, derive a context from the worker's lifecycle and add
a deadline for each run. That gives shutdown and cancellation a clear path
through request waits and retries.

Choose timeouts around the amount of work you expect. A single-page web
request usually needs a different budget from an overnight import. If both
use the same client, their contexts can give them different overall budgets
without changing the shared client configuration.

Keep persistence and caching in your application. Cache type and industry
lists when that fits your freshness requirements, store article UUIDs for
repeatable processing, and decide whether news pages should be cached for
your readers. The SDK issues a request each time a service method is called;
it does not schedule refreshes or share a cache between callers.

At shutdown, cancel your workers and wait for active work to finish before
calling `CloseIdleConnections`. Call it for application cleanup rather than
after every request, so the connection pool can do its job.

For a local test server, pass its URL through `WithBaseURL` and use
`WithRetry(RetryPolicy{MaxAttempts: 1})` when you want one attempt per test.
The repository's [client tests](client_test.go) and
[request-pipeline tests](pipeline_test.go) show complete examples using the
standard library's `httptest` package.

## Testing

```bash
go test ./...        # local HTTP test servers; no Marketaux calls
go test -race ./...  # race detector
make cover           # coverage report (≥90%)

# live API smoke test (opt-in):
MARKETAUX_API_TOKEN=... go test -tags=integration ./...
MARKETAUX_API_TOKEN=... go run ./examples/smoke
```

On PowerShell, set `$env:MARKETAUX_API_TOKEN` first and run the commands
without the Unix-style assignment prefix. For coverage:

```powershell
go test -race '-coverprofile=coverage.out' '-covermode=atomic' .
go tool cover '-func=coverage.out'
```

The coverage target measures the library package; example programs are built
and checked separately. Live tests consume API requests and are skipped when
no token is set. See [CONTRIBUTING.md](CONTRIBUTING.md) for the full workflow.

## Compatibility

The CI compatibility matrix covers Go 1.23–1.27. `go.mod` has no
`require` directives — the module is standard library only. v1 keeps the
public API backward compatible.

## Security

- `api_token` is sent only as a query parameter — the API does not support
  `Authorization` headers.
- The token is masked (`api_token=***`) in `*APIError`/`TransportError`/
  `DecodeError` URLs and in debug logs.
- Known token representations echoed in error messages, diagnostic body
  previews and API error headers are redacted too. Original underlying error
  identity is preserved for `errors.Is`/`errors.As`; take care when logging
  custom transport errors obtained by unwrapping, or arbitrary response data.
- Custom transports, HTTP tracing and proxy access logs see the real request
  URL. Configure their logging to avoid recording the query token.
- Use a patched Go toolchain and run `govulncheck ./...` regularly. Standard
  library security updates still matter for a dependency-free SDK.
- Never commit tokens; `.env` is gitignored.
- Report vulnerabilities privately per [SECURITY.md](SECURITY.md).

## Troubleshooting

When a request surprises you, start with the error class and the simplest
query that should work. Add filters back gradually, check the response's
`Meta`, and enable redacted debug logging if you need to see the attempts.

| What you see | What to check |
| --- | --- |
| `ErrInvalidConfig` | Token is set; timeout is positive; `WithHTTPClient` precedes `WithTimeout` if the custom client has no timeout |
| `ErrUnauthorized` | Token belongs to the intended account and has no surrounding whitespace |
| `ErrForbidden` on stats | The account has access to that endpoint |
| `ErrUsageLimit` | Daily quota; repeated retries will not restore it |
| `ErrRateLimit` | Per-minute limit; lower RPS/burst and inspect `RetryAfter` |
| Fewer articles than requested | Your plan can cap `Limit`; inspect `Meta.Limit` |
| Empty results | Start with fewer filters, check UTC dates and the entity's exact symbol |
| No debug logs | Supply `WithLogger`, enable debug in its handler, then set `WithDebug(true)` |
| `ErrDecode` | Inspect the redacted preview; a proxy may have returned HTML or unexpected JSON |
| `ErrResponseTooLarge` | Reduce result size or deliberately raise `WithMaxResponseBytes` |

### When the articles look right but the entities do not

An article can match your watchlist and still mention several other
companies. Without `FilterEntities`, those other mentions remain in its
`Entities` list. Check which symbol each score belongs to before displaying
it as sentiment for your watchlist. Entity search can also help distinguish
instruments with similar names on different exchanges.

### When a job stops halfway through

Decide whether to keep partial work before choosing between an iterator and
`AllPages`. A handler called from `Paginate` may already have written several
articles when a later request fails. `AllPages` returns no collected slice
on error. You can start another iterator at a chosen `Page`, but page
contents may have moved in a live feed. A date window and stored UUIDs give
your application more control over reprocessing.

## Examples

Runnable programs are in [examples/README.md](examples/README.md):
[news](examples/news/main.go), [entities](examples/entities/main.go),
[stats](examples/stats/main.go), [pagination](examples/pagination/main.go),
[errors](examples/errors/main.go), and [smoke](examples/smoke/main.go).
For example, after setting your token, run `go run ./examples/news` from this
repository. Stats examples need access to the stats endpoints.

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

Found a surprising response or a missing example? Open an issue with the
method you called, the error or status you received, your Go version and a
small reproducer. Remove tokens and private query data first. For code
changes and local checks, see [CONTRIBUTING.md](CONTRIBUTING.md); report
security issues through [SECURITY.md](SECURITY.md).

## License

MIT © [Igor Sazonov](https://github.com/tigusigalpa) — see [LICENSE](LICENSE).
