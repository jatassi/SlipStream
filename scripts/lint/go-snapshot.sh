#!/usr/bin/env bash
# Save or compare lint count snapshots for tracking progress across phases.
# Usage: ./scripts/lint/go-snapshot.sh save phase1    # save current counts
#        ./scripts/lint/go-snapshot.sh compare phase1  # compare current vs saved
#        ./scripts/lint/go-snapshot.sh list             # list saved snapshots
set -euo pipefail

cd "$(dirname "$0")/../.."

SNAPSHOT_DIR=".lint-snapshots"
GOLANGCI_LINT=$(command -v golangci-lint 2>/dev/null || echo "$(go env GOPATH)/bin/golangci-lint")

get_counts() {
  OUTPUT=$($GOLANGCI_LINT run ./... --max-issues-per-linter 0 --max-same-issues 0 2>&1 || true)
  echo "$OUTPUT" | grep -E '^\S+\.go:[0-9]+' | grep -oE '\([a-zA-Z-]+\)$' | tr -d '()' | sort | uniq -c | sort -rn
}

case "${1:-}" in
  save)
    NAME="${2:?Usage: $0 save <name>}"
    mkdir -p "$SNAPSHOT_DIR"
    get_counts > "$SNAPSHOT_DIR/$NAME.txt"
    TOTAL=$(awk '{s+=$1} END{print s}' "$SNAPSHOT_DIR/$NAME.txt")
    echo "Saved snapshot '$NAME' ($TOTAL total issues)"
    ;;
  compare)
    NAME="${2:?Usage: $0 compare <name>}"
    FILE="$SNAPSHOT_DIR/$NAME.txt"
    if [[ ! -f "$FILE" ]]; then
      echo "No snapshot '$NAME' found. Run: $0 save $NAME"
      exit 1
    fi
    SAVED_TOTAL=$(awk '{s+=$1} END{print s}' "$FILE")
    echo "=== Snapshot '$NAME' ($SAVED_TOTAL total) ==="
    cat "$FILE"
    echo ""
    echo "=== Current ==="
    CURRENT=$(get_counts)
    CURRENT_TOTAL=$(echo "$CURRENT" | awk '{s+=$1} END{print s}')
    echo "$CURRENT"
    echo ""
    DIFF=$((CURRENT_TOTAL - SAVED_TOTAL))
    if [[ $DIFF -lt 0 ]]; then
      echo "=== Delta: $DIFF issues (improved) ==="
    elif [[ $DIFF -gt 0 ]]; then
      echo "=== Delta: +$DIFF issues (regressed) ==="
    else
      echo "=== Delta: 0 (no change) ==="
    fi
    ;;
  list)
    if [[ ! -d "$SNAPSHOT_DIR" ]]; then
      echo "No snapshots yet. Run: $0 save <name>"
      exit 0
    fi
    for f in "$SNAPSHOT_DIR"/*.txt; do
      [[ -f "$f" ]] || continue
      NAME=$(basename "$f" .txt)
      TOTAL=$(awk '{s+=$1} END{print s}' "$f")
      DATE=$(stat -f '%Sm' -t '%Y-%m-%d %H:%M' "$f" 2>/dev/null || stat -c '%y' "$f" 2>/dev/null | cut -d. -f1)
      printf "  %-20s %5d issues  (%s)\n" "$NAME" "$TOTAL" "$DATE"
    done
    ;;
  *)
    echo "Usage: $0 {save|compare|list} [name]"
    echo "  save <name>     Save current lint counts as a snapshot"
    echo "  compare <name>  Compare current counts against a snapshot"
    echo "  list            List saved snapshots"
    exit 1
    ;;
esac
