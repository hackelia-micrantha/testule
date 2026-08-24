#!/usr/bin/env bash
set -euo pipefail

section() {
  printf '\n==> %s\n' "$1"
}

# Dubnium JIT runners intentionally mount /tmp noexec. Go test/fuzz builds
# executable binaries and tests also create nested temporary workspaces, so
# keep both generic and Go-specific temporary state inside the executable
# checkout workspace rather than weakening the host mount policy.
TMPDIR="$PWD/.tmp/runtime"
GOTMPDIR="$PWD/.tmp/go"
export TMPDIR GOTMPDIR
mkdir -p "$TMPDIR" "$GOTMPDIR"
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
