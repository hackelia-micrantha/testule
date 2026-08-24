#!/usr/bin/env bash
set -euo pipefail

section() {
  printf '\n==> %s\n' "$1"
}

# Dubnium JIT runners intentionally mount /tmp noexec. Go test/fuzz builds
# executable test binaries in its temporary build directory, so keep that
# build area inside the checked-out workspace rather than weakening the host.
GOTMPDIR="$PWD/.tmp/go"
export GOTMPDIR
mkdir -p "$GOTMPDIR"
trap 'rm -rf "$PWD/.tmp"' EXIT

section "format"
files="$(gofmt -l .)"
if [[ -n "$files" ]]; then
  printf '%s\n' "$files"
  # shellcheck disable=SC2086
  gofmt -d $files
  exit 1
fi

section "module metadata"
go mod tidy
git diff --exit-code -- go.mod go.sum

section "vet"
go vet ./...

section "staticcheck"
staticcheck ./...

section "race tests"
go test -race ./...

section "fuzz smoke: plan"
go test ./internal/plan -run=^$ -fuzz=FuzzDecodeNeverPanics -fuzztime=5s

section "fuzz smoke: evidence"
go test ./internal/evidence -run=^$ -fuzz=FuzzDecodeNeverPanics -fuzztime=5s

section "build"
go build ./cmd/testule
