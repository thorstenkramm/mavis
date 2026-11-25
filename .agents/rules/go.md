# Go coding and verification rules

Use these rules for Go implementation, tests, CI, and release changes.

## Toolchain and formatting

- Use the Go version declared in `go.mod`; the baseline is Go 1.27.2.
- Format Go source with `gofmt` and keep source files LF-terminated on every OS.
- Do not convert files under `testdata/` or golden fixtures to a different line
  ending; tests may compare their bytes exactly.
- Keep implementation packages under `internal/` unless a public Go API is an
  explicit requirement.

## Tests and coverage

- Run `go test -race ./...` for the full test suite (`make test`).
- Add or update tests with the behavior they cover; do not defer a group's
  tests to a final cleanup.
- Maintain at least 85% total statement coverage, excluding test-only support
  packages such as `internal/testutil` from the coverage denominator.
- Use table-driven tests for configuration precedence, path traversal,
  wildcard matching, hidden files, symlink containment, and error mapping.
- Run platform-specific tests for path handling and listeners. Skips must be
  explicit and limited to genuine operating-system differences.
- Do not weaken adversarial, security, or cross-platform assertions merely to
  make an implementation easier.

## Static analysis and dependencies

- Run `make lint`: it checks `gofmt`, `go vet`, golangci-lint, and staticcheck.
- Pin tool versions in Makefile and CI rather than depending on an arbitrary
  developer installation or an unpinned latest release.
- Keep `.golangci.yml` aligned with CI. Enabled checks include errcheck,
  errorlint, govet, ineffassign, misspell, nilerr, staticcheck, unconvert, and
  unused.
- Check errors, including blank assignments and type assertions where
  configured. Do not discard errors to silence a linter.
- Keep production dependencies minimal and recorded in `go.mod`/`go.sum`.
  Pin development-tool invocations separately.

## Commands and CI

- Keep `make build`, `make test`, `make lint`, and `make coverage` working as
  the stable local quality entry points.
- CI must enforce the same formatting, tests, race checks, static analysis, and
  coverage expectations as local development.
- Build standalone binaries with `CGO_ENABLED=0` unless a requirement explicitly
  introduces a native dependency.
- Keep Mailfire-specific publishing, credentials, and package-manager
  automation out of Mavis unless separately requested.
- Use `./run-all-tests.sh` for the comprehensive local suite. In addition to
  Go checks and coverage, it enforces jscpd duplication checks, optional
  govulncheck, the Hugo documentation build, and Trivy with `--exit-code 1`.
- Keep `.jscpd.json` and `.trivyignore.yaml` aligned with local and CI scans.
  Do not add duplication or vulnerability exclusions without documenting why.
