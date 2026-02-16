#!/usr/bin/env bash
# Full linter breakdown — uncapped per-linter and per-file counts.
# Usage: ./scripts/lint/go-breakdown.sh
#        ./scripts/lint/go-breakdown.sh --linter gocritic   # filter to one linter
set -euo pipefail

cd "$(dirname "$0")/../.."

GOLANGCI_LINT=$(command -v golangci-lint 2>/dev/null || echo "$(go env GOPATH)/bin/golangci-lint")
LINTER_FILTER="${2:-}"

if [[ "${1:-}" == "--linter" && -n "$LINTER_FILTER" ]]; then
  OUTPUT=$($GOLANGCI_LINT run ./... --max-issues-per-linter 0 --max-same-issues 0 2>&1 | grep "($LINTER_FILTER)" || true)
  TOTAL=$(echo "$OUTPUT" | grep -cE '^\S+\.go:[0-9]+' || true)
  echo "=== $LINTER_FILTER: $TOTAL issues ==="
  echo ""
  echo "=== By File ==="
  echo "$OUTPUT" | grep -oE '^\S+\.go' | sort | uniq -c | sort -rn
  echo ""
  echo "=== Issues ==="
  echo "$OUTPUT"
else
  OUTPUT=$($GOLANGCI_LINT run ./... --max-issues-per-linter 0 --max-same-issues 0 2>&1 || true)
  TOTAL=$(echo "$OUTPUT" | grep -cE '^\S+\.go:[0-9]+' || true)
  echo "=== Total: $TOTAL issues ==="
  echo ""
  echo "=== By Linter ==="
  echo "$OUTPUT" | grep -E '^\S+\.go:[0-9]+' | grep -oE '\([a-zA-Z-]+\)$' | tr -d '()' | sort | uniq -c | sort -rn
  echo ""
  echo "=== By File (top 20) ==="
  echo "$OUTPUT" | grep -oE '^\S+\.go' | sort | uniq -c | sort -rn | head -20
fi
