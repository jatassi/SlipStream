#!/usr/bin/env bash
# Post-wave verification: tsc + build + lint count
# Usage: ./scripts/consolidation/verify.sh
set -euo pipefail

WEB_DIR="$(cd "$(dirname "$0")/../../web" && pwd)"
cd "$WEB_DIR"

PASS=0
FAIL=0

echo "=== TypeScript type check ==="
if bunx tsc --noEmit 2>&1; then
  echo "PASS: tsc --noEmit"
  PASS=$((PASS + 1))
else
  echo "FAIL: tsc --noEmit"
  FAIL=$((FAIL + 1))
fi

echo ""
echo "=== Production build ==="
if bun run build 2>&1; then
  echo "PASS: bun run build"
  PASS=$((PASS + 1))
else
  echo "FAIL: bun run build"
  FAIL=$((FAIL + 1))
fi

echo ""
echo "=== ESLint error count ==="
LINT_COUNT=$(bunx eslint src/ --quiet 2>&1 | grep -c "error" || true)
echo "Lint errors: $LINT_COUNT"
PASS=$((PASS + 1))

echo ""
echo "================================"
echo "Results: $PASS passed, $FAIL failed"
if [ $FAIL -gt 0 ]; then
  echo "VERIFICATION FAILED"
  exit 1
else
  echo "VERIFICATION PASSED"
fi
