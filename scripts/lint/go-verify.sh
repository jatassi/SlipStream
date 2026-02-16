#!/usr/bin/env bash
# Full Go verification suite — vet, build, test, lint count.
# Usage: ./scripts/lint/go-verify.sh
# Returns exit code 0 only if vet and build pass.
set -euo pipefail

cd "$(dirname "$0")/../.."

echo "=== Go Vet ==="
if go vet ./... 2>&1; then
  echo "PASS"
else
  echo "FAIL — vet errors above"
  exit 1
fi

echo ""
echo "=== Build ==="
if go build ./cmd/slipstream 2>&1; then
  echo "PASS"
else
  echo "FAIL — build errors above"
  exit 1
fi

echo ""
echo "=== Tests ==="
if go test ./... 2>&1 | tail -5; then
  echo "PASS"
else
  echo "FAIL — test errors above"
  exit 1
fi

echo ""
echo "=== Lint Count ==="
GOLANGCI_LINT=$(command -v golangci-lint 2>/dev/null || echo "$(go env GOPATH)/bin/golangci-lint")
TOTAL=$($GOLANGCI_LINT run ./... --max-issues-per-linter 0 --max-same-issues 0 2>&1 | grep -cE '^\S+\.go:[0-9]+' || true)
echo "Total lint issues: $TOTAL"
