#!/usr/bin/env bash

set -euo pipefail

go_files=()
while IFS= read -r file; do
  go_files+=("$file")
done < <(git ls-files --cached --others --exclude-standard -- '*.go')

if ((${#go_files[@]} == 0)); then
  echo "No Go files found; the Go module has not been scaffolded yet." >&2
  exit 1
fi

echo "Checking gofmt ..."
unformatted="$(gofmt -l "${go_files[@]}")"
if [[ -n "$unformatted" ]]; then
  echo "Unformatted Go files:"
  echo "$unformatted"
  exit 1
fi

echo "Running Go tests ..."
go test ./...

echo "Running go vet ..."
go vet ./...

echo "Running golangci-lint ..."
go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0 run ./...

echo "Running staticcheck ..."
go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./...

echo "Running Go race tests ..."
go test -race ./...

echo "Checking coverage ..."
make coverage

# An unmatched ignore marker can suppress duplication detection for the rest of
# a file without failing the scan, so marker balance is checked separately.
echo "Checking duplication marker balance ..."
unbalanced=0
for file in "${go_files[@]}"; do
  starts="$(grep -c 'jscpd:ignore-start' "$file" || true)"
  ends="$(grep -c 'jscpd:ignore-end' "$file" || true)"
  if [[ "$starts" -ne "$ends" ]]; then
    echo "$file has $starts jscpd:ignore-start and $ends jscpd:ignore-end markers"
    unbalanced=1
  fi
done
if [[ "$unbalanced" -ne 0 ]]; then
  echo "Unbalanced duplication markers hide later clones in the same file."
  exit 1
fi

echo "Running code duplication search ..."
npx --yes jscpd@5.1.2 --config .jscpd.json --no-tips .

if command -v govulncheck >/dev/null 2>&1; then
  echo "Running Go vulnerability scan ..."
  govulncheck ./...
fi

if command -v hugo >/dev/null 2>&1; then
  echo "Building documentation site ..."
  make docs
else
  echo "Skipping documentation site build (needs hugo) ..."
fi

echo "Running Trivy vulnerability scan ..."
trivy fs . --skip-dirs .agents --ignorefile .trivyignore.yaml --exit-code 1

echo "🥳🥳 All tests passed. 🎉🎉 "
