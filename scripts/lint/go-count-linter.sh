#!/usr/bin/env bash
# Count violations for a specific Go linter, grouped by file.
# Usage: ./scripts/lint/go-count-linter.sh nakedret
set -euo pipefail

if [ -z "${1:-}" ]; then
  echo "Usage: $0 <linter-name>"
  echo "  e.g.: $0 nakedret"
  echo "        $0 funlen"
  exit 1
fi

LINTER="$1"
cd "$(dirname "$0")/../.."

GOLANGCI_LINT=$(command -v golangci-lint 2>/dev/null || echo "$(go env GOPATH)/bin/golangci-lint")
OUTPUT=$($GOLANGCI_LINT run ./... --max-issues-per-linter 0 --max-same-issues 0 2>&1 || true)

echo "=== Files with $LINTER violations ==="
echo "$OUTPUT" | grep "($LINTER)" | grep -oE '^\S+\.go' | sort | uniq -c | sort -rn

echo ""
TOTAL=$(echo "$OUTPUT" | grep -c "($LINTER)" || true)
echo "Total: $TOTAL"
