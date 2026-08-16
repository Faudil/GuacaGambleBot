#!/usr/bin/env bash
# Shared checks run by the git hooks, mirroring the CI workflow.
# Usage: run-checks.sh [commit|push]
set -uo pipefail

MODE="${1:-commit}"
ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

fail() {
  echo "❌ $1" >&2
  exit 1
}

echo "==> gofmt check"
unformatted=$(gofmt -l cmd internal tools)
if [ -n "$unformatted" ]; then
  echo "Files need gofmt:"
  echo "$unformatted"
  fail "gofmt: run 'gofmt -w <files>'"
fi

echo "==> go vet ./..."
go vet ./... || fail "go vet"

if [ "$MODE" = "push" ]; then
  echo "==> go test ./... -count=1 -race (full suite, same as CI)"
  go test ./... -count=1 -race || fail "go test"
else
  echo "==> go test ./... (skip --no-verify to bypass)"
  go test ./... -count=1 || fail "go test"
fi

echo "✅ all checks passed"
