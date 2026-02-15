#!/usr/bin/env bash
# Find all files that import from a path containing the given module name.
# Usage: ./scripts/consolidation/find-importers.sh <module-name>
# Example: ./scripts/consolidation/find-importers.sh summary-card
# Example: ./scripts/consolidation/find-importers.sh DryRunModal/summary-card
set -euo pipefail

if [ $# -eq 0 ]; then
  echo "Usage: $0 <module-name>"
  echo "Finds all .ts/.tsx files whose import/export statements reference the given module."
  exit 1
fi

MODULE="$1"
WEB_DIR="$(cd "$(dirname "$0")/../../web/src" && pwd)"

echo "Searching for imports of '$MODULE' in $WEB_DIR"
echo "---"

RESULTS=$(grep -rn --include='*.ts' --include='*.tsx' \
  -E "(import|export).*from.*['\"].*${MODULE}['\"]" \
  "$WEB_DIR" 2>/dev/null || true)

if [ -z "$RESULTS" ]; then
  echo "No imports found for: $MODULE"
else
  echo "$RESULTS"
  echo "---"
  echo "Total: $(echo "$RESULTS" | wc -l | tr -d ' ') import sites"
fi
