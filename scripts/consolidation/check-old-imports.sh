#!/usr/bin/env bash
# Verify no file still imports from old/deleted paths.
# Usage: ./scripts/consolidation/check-old-imports.sh <pattern1> [pattern2] ...
# Example (after Wave 1): ./scripts/consolidation/check-old-imports.sh \
#   "DryRunModal/summary-card" "DryRunModal/file-item" \
#   "MigrationPreviewModal/summary-card" "MigrationPreviewModal/file-item"
set -euo pipefail

if [ $# -eq 0 ]; then
  echo "Usage: $0 <import-pattern1> [import-pattern2] ..."
  echo "Verifies no .ts/.tsx file imports from paths matching the given patterns."
  exit 1
fi

WEB_DIR="$(cd "$(dirname "$0")/../../web/src" && pwd)"
ERRORS=0

for PATTERN in "$@"; do
  HITS=$(grep -rn --include='*.ts' --include='*.tsx' \
    -E "(import|export).*from.*['\"].*${PATTERN}['\"]" \
    "$WEB_DIR" 2>/dev/null || true)

  if [ -n "$HITS" ]; then
    echo "FAIL: Found imports still referencing '$PATTERN':"
    echo "$HITS" | sed 's/^/  /'
    ERRORS=$((ERRORS + 1))
  else
    echo "PASS: No imports reference '$PATTERN'"
  fi
done

echo ""
if [ $ERRORS -gt 0 ]; then
  echo "FAILED: $ERRORS stale import patterns found"
  exit 1
else
  echo "ALL CLEAN: No stale imports"
fi
