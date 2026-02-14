#!/usr/bin/env bash
# Count violations for a specific rule, grouped by file.
# Usage: ./scripts/lint/count-rule.sh no-floating-promises
set -euo pipefail

if [ -z "${1:-}" ]; then
  echo "Usage: $0 <rule-name>"
  echo "  e.g.: $0 no-floating-promises"
  exit 1
fi

RULE="$1"
cd "$(dirname "$0")/../../web"

OUTPUT=$(bunx eslint . 2>&1 || true)
echo "=== Files with $RULE violations ==="
echo "$OUTPUT" | awk -v rule="$RULE" '
  /^\/Users/ { file=$1 }
  $0 ~ rule { count[file]++ }
  END { for (f in count) print count[f], f }
' | sort -rn | sed "s|$(pwd)/||"

echo ""
TOTAL=$(echo "$OUTPUT" | grep -c "$RULE" || true)
echo "Total: $TOTAL"
