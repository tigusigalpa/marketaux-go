# Examples

Every example reads the API token from the `MARKETAUX_API_TOKEN` environment
variable. Register at https://www.marketaux.com/register to get one.

```bash
export MARKETAUX_API_TOKEN=your_token

go run ./examples/news        # news feed filtered by symbols
go run ./examples/entities    # entity search + type/industry lists
go run ./examples/stats       # intraday, aggregation, trending (Standard+ plan)
go run ./examples/pagination  # lazy iteration over pages with iter.Seq2
go run ./examples/errors      # typed error handling (errors.Is / errors.As)
go run ./examples/smoke       # live smoke test with debug logging
```

The examples compile as part of `go build ./...` but are never executed by
`go test` — they require a real token and network access.
