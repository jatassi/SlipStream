#!/usr/bin/env bash
# Lint summary — shows total counts and top violations by rule.
# Usage: ./scripts/lint/summary.sh
set -euo pipefail

cd "$(dirname "$0")/../../web"

OUTPUT=$(bunx eslint . 2>&1 || true)

TOTAL=$(echo "$OUTPUT" | tail -1)
echo "=== $TOTAL ==="
echo ""
echo "=== By Rule (top 25) ==="
echo "$OUTPUT" | grep -oE '[a-zA-Z@/_-]+$' | sort | uniq -c | sort -rn | head -25
