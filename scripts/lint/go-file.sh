#!/usr/bin/env bash
# Lint a single Go file or package.
# Usage: ./scripts/lint/go-file.sh internal/api/handlers.go
#        ./scripts/lint/go-file.sh ./internal/api/...
set -euo pipefail

if [ -z "${1:-}" ]; then
  echo "Usage: $0 <file-or-package>"
  echo "  e.g.: $0 internal/api/handlers.go"
  echo "        $0 ./internal/api/..."
  exit 1
fi

cd "$(dirname "$0")/../.."

GOLANGCI_LINT=$(command -v golangci-lint 2>/dev/null || echo "$(go env GOPATH)/bin/golangci-lint")
TARGET="$1"

if [[ "$TARGET" == *.go ]]; then
  PKG=$(dirname "$TARGET")
  $GOLANGCI_LINT run "./$PKG/..." 2>&1 | grep "$TARGET" || echo "No issues found."
else
  $GOLANGCI_LINT run "$TARGET" 2>&1 || true
fi
