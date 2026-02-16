#!/usr/bin/env bash
# Go lint summary — shows total issues and top linters/files.
# Usage: ./scripts/lint/go-summary.sh
set -euo pipefail

cd "$(dirname "$0")/../.."

GOLANGCI_LINT=$(command -v golangci-lint 2>/dev/null || echo "$(go env GOPATH)/bin/golangci-lint")
OUTPUT=$($GOLANGCI_LINT run ./... --max-issues-per-linter 0 --max-same-issues 0 2>&1 || true)

TOTAL=$(echo "$OUTPUT" | grep -cE '^\S+\.go:[0-9]+' || true)
echo "=== Total: $TOTAL issues ==="

echo ""
echo "=== By Linter (top 15) ==="
echo "$OUTPUT" | grep -E '^\S+\.go:[0-9]+' | grep -oE '\([a-zA-Z-]+\)$' | tr -d '()' | sort | uniq -c | sort -rn | head -15

echo ""
echo "=== By File (top 15) ==="
echo "$OUTPUT" | grep -oE '^\S+\.go' | sort | uniq -c | sort -rn | head -15
