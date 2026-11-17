# Contributing

Thank you for considering a contribution to marketaux-go.

## Requirements

- Go 1.23 or newer
- `golangci-lint` v2 (`go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0`)
- `govulncheck` (`go install golang.org/x/vuln/cmd/govulncheck@latest`)

## Workflow

1. Fork and clone the repository.
2. Create a branch from `main`.
3. Make your changes — keep the public API backward compatible within v1.
4. Run the full check suite:

   ```bash
   make check   # tidy + fmt + vet + race tests + lint
   ```

5. Commit using [Conventional Commits](https://www.conventionalcommits.org/):
   `feat:`, `fix:`, `test:`, `docs:`, `ci:`, `chore:`.
6. Open a pull request and fill in the template.

## Testing

- Unit tests: `go test ./...`
- With race detector: `go test -race ./...`
- Coverage: `make cover` (goal: ≥ 90%)
- A single test: `go test -run TestNewsAllDecode ./...`

All tests must use `net/http/httptest` — tests never hit the real API.
Fixtures live in `testdata/`.

Custom `http.RoundTripper` implementations are also useful for simulating
transport and response-body failures without network access. Run `govulncheck
./...` with a current, patched Go toolchain: having no third-party dependencies
does not remove vulnerabilities in the standard library.

On PowerShell, quote arguments containing file names, for example:

```powershell
go test -race '-coverprofile=coverage.out' '-covermode=atomic' .
go tool cover '-func=coverage.out'
```

### Integration tests

Live tests are behind the `integration` build tag and require a real token:

```bash
MARKETAUX_API_TOKEN=... go test -tags=integration ./...
```

## Adding a new endpoint

1. Add the params type and its `validate()`/`values()` in `params.go`.
2. Add the response types in `types.go`.
3. Add a method on the appropriate service (`news.go`, `stats.go`,
   `entities.go`, `sources.go`).
4. Add a fixture in `testdata/` and tests covering decoding, query encoding
   and validation.
5. Update the service map in `README.md`/`README.ru.md` and `CHANGELOG.md`.

## Rules

- No external dependencies — `go.mod` must stay free of `require`.
- No `panic`, `fmt.Println` or `log.Print` in library code.
- Every exported identifier needs a godoc comment starting with its name.
- The API token must never appear in errors, logs or committed files.
