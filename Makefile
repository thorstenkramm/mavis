BINARY := mavis
COVERAGE_MIN := 85

.PHONY: build test lint coverage docs docs-serve clean

build:
	CGO_ENABLED=0 go build -o $(BINARY) ./cmd/mavis

test:
	go test -race ./...

lint:
	@test -z "$$(gofmt -l .)" || { echo "gofmt: files need formatting:"; gofmt -l .; exit 1; }
	go vet ./...
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0 run ./...
	go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./...

coverage:
	go test -coverprofile=coverage.out $$(go list ./... | grep -v /internal/testutil)
	go tool cover -func=coverage.out
	@total=$$(go tool cover -func=coverage.out | awk '/^total:/ {sub("%", "", $$3); print $$3}'); \
	echo "total coverage: $$total% (minimum $(COVERAGE_MIN)%)"; \
	awk -v t="$$total" -v min="$(COVERAGE_MIN)" 'BEGIN { exit (t + 0 >= min + 0) ? 0 : 1 }' \
		|| { echo "FAIL: total coverage $$total% is below $(COVERAGE_MIN)%"; exit 1; }

docs:
	@git_info=$$(git rev-parse --verify HEAD >/dev/null 2>&1 && echo true || echo false); \
	HUGO_ENABLEGITINFO=$$git_info hugo --source docs --minify --gc --cleanDestinationDir --panicOnWarning

docs-serve:
	@git_info=$$(git rev-parse --verify HEAD >/dev/null 2>&1 && echo true || echo false); \
	HUGO_ENABLEGITINFO=$$git_info hugo server --source docs

clean:
	rm -f $(BINARY) coverage.out
	rm -rf docs/public docs/resources docs/.hugo_build.lock docs/assets/generated
