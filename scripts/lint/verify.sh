#!/usr/bin/env bash
# Full verification suite — types, build, lint count.
# Usage: ./scripts/lint/verify.sh
# Returns exit code 0 only if types and build pass.
set -euo pipefail

cd "$(dirname "$0")/../../web"

echo "=== Type Check ==="
if bunx tsc --noEmit 2>&1; then
  echo "PASS"
else
  echo "FAIL — type errors above"
  exit 1
fi

echo ""
echo "=== Build ==="
if bun run build 2>&1 | tail -3; then
  echo "PASS"
else
  echo "FAIL — build errors above"
  exit 1
fi

echo ""
echo "=== Lint Count ==="
LINT_OUTPUT=$(bunx eslint . 2>&1 || true)
echo "$LINT_OUTPUT" | tail -3
