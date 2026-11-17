# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed
- Apply client-side rate limiting to every HTTP attempt, including retries;
  canceled waits no longer leave reservations that delay later requests.
- Follow the server's effective page size, preserve pagination validation,
  and stop before requesting a page beyond the 20,000-result window.
- Redact credentials echoed in API errors, decode previews, response headers
  and transport diagnostics; handle malformed query strings in `RedactURL`.
- Block cross-origin redirects before credentials can reach another origin.
- Classify body-read timeouts and cancellations correctly, avoid retrying
  oversized responses, and close idle connections on the default transport.
- Prevent integer overflow in result-window validation, retry delays,
  `Retry-After` parsing and response-size limits; reject non-finite numbers.
- Use a golangci-lint v2-compatible action and run lint/security tools with a
  current Go toolchain independently of the compatibility test matrix.
- Honor `MARKETAUX_BASE_URL` in the smoke example.

### Added
- `ErrResponseTooLarge`, also matched by `ErrTransport`, for response limits.
- Regression tests for HTTP failures, redaction, pagination and rate limiting.

### Changed
- Expanded the English README with practical recipes, parameter reference,
  timeout budgets, quota behavior, polling guidance and troubleshooting.
- Consolidated documentation into the English README and removed the
  separate translation and language switcher.

## [1.0.0] - 2026-11-16

### Added
- Full coverage of all 10 Marketaux API endpoints: news feed, similar news,
  news by UUID, entity stats (intraday time series and aggregation), trending
  entities, entity search, entity types, industries and news sources.
- `Client` with functional options: `WithBaseURL`, `WithHTTPClient`,
  `WithTimeout`, `WithUserAgent`, `WithRetry`, `WithRateLimit`, `WithLogger`,
  `WithDebug`, `WithMaxResponseBytes`.
- Automatic retries with exponential backoff, jitter and `Retry-After`
  support via `RetryPolicy`/`DefaultRetryPolicy`.
- Client-side token-bucket rate limiter (`WithRateLimit`).
- Lazy pagination with `iter.Seq2` iterators on news, entity search and
  sources, plus `News().AllPages` convenience helper.
- Typed errors: `APIError`, `ValidationError`, `ConfigError`,
  `TransportError`, `DecodeError` and sentinel errors compatible with
  `errors.Is`/`errors.As` for every documented API error code.
- Token redaction: the `api_token` query parameter is masked in all error
  messages and debug logs.
- Strongly typed params with client-side validation (sentiment ranges, date
  windows, sort/order combinations, the 20000-result API bound).
- Pointer helpers `Bool`, `Int`, `Float64`, `Time`, `Date` for optional
  parameters.
- `Article`, `Entity`, `Highlight` convenience helpers (`HasEntities`,
  `IsAbout`, `IsPositive`, `IsNegative`, `TypeKnown`, `Text`).
- Examples for news, entities, stats, pagination and error handling in
  `examples/`, plus a live `examples/smoke` command.
- Integration test behind the `integration` build tag.
- GitHub Actions: CI matrix (Go 1.23–1.27), CodeQL, release workflow.
- Zero external dependencies: standard library only.
