#!/usr/bin/env bash
# Capture or compare exported Go function/method signatures for regression detection.
# Usage: ./scripts/lint/go-check-signatures.sh save <name>       # save current signatures
#        ./scripts/lint/go-check-signatures.sh compare <name>    # compare current vs saved
#        ./scripts/lint/go-check-signatures.sh list              # list saved snapshots
#        ./scripts/lint/go-check-signatures.sh save <name> <pkg> # save single package
set -euo pipefail

cd "$(dirname "$0")/../.."

SNAPSHOT_DIR=".lint-snapshots/signatures"

extract_signatures() {
  local target="${1:-./internal/...}"
  # Extract all exported func/method declarations from non-test, non-generated Go files
  grep -rn '^func ' --include='*.go' \
    $(go list -f '{{.Dir}}' "$target" 2>/dev/null | tr '\n' ' ') 2>/dev/null \
    | grep -v '_test.go:' \
    | grep -v 'internal/database/sqlc/' \
    | grep -v 'internal/database/migrations/' \
    | sed "s|$(pwd)/||" \
    | grep -E '^[^:]+:[0-9]+:func [^a-z]|^[^:]+:[0-9]+:func \([^)]+\) [A-Z]' \
    | sed 's/:[0-9]*:func /: func /' \
    | sort
}

case "${1:-}" in
  save)
    NAME="${2:?Usage: $0 save <name> [package]}"
    PKG="${3:-./internal/...}"
    mkdir -p "$SNAPSHOT_DIR"
    extract_signatures "$PKG" > "$SNAPSHOT_DIR/$NAME.txt"
    COUNT=$(wc -l < "$SNAPSHOT_DIR/$NAME.txt" | tr -d ' ')
    echo "Saved $COUNT signatures to '$NAME'"
    ;;
  compare)
    NAME="${2:?Usage: $0 compare <name> [package]}"
    PKG="${3:-./internal/...}"
    FILE="$SNAPSHOT_DIR/$NAME.txt"
    if [[ ! -f "$FILE" ]]; then
      echo "No snapshot '$NAME' found. Run: $0 save $NAME"
      exit 1
    fi
    CURRENT=$(mktemp)
    extract_signatures "$PKG" > "$CURRENT"
    SAVED_COUNT=$(wc -l < "$FILE" | tr -d ' ')
    CURRENT_COUNT=$(wc -l < "$CURRENT" | tr -d ' ')
    echo "=== Snapshot '$NAME': $SAVED_COUNT signatures ==="
    echo "=== Current: $CURRENT_COUNT signatures ==="
    echo ""
    DIFF_OUT=$(diff --unified=0 "$FILE" "$CURRENT" || true)
    if [[ -z "$DIFF_OUT" ]]; then
      echo "No signature changes detected."
    else
      REMOVED=$(echo "$DIFF_OUT" | grep '^-[^-]' | wc -l | tr -d ' ')
      ADDED=$(echo "$DIFF_OUT" | grep '^+[^+]' | wc -l | tr -d ' ')
      echo "Changes: -$REMOVED removed, +$ADDED added"
      echo ""
      echo "$DIFF_OUT" | grep '^[+-]' | grep -v '^[+-][+-][+-]'
    fi
    rm -f "$CURRENT"
    ;;
  list)
    if [[ ! -d "$SNAPSHOT_DIR" ]]; then
      echo "No signature snapshots yet. Run: $0 save <name>"
      exit 0
    fi
    for f in "$SNAPSHOT_DIR"/*.txt; do
      [[ -f "$f" ]] || continue
      NAME=$(basename "$f" .txt)
      COUNT=$(wc -l < "$f" | tr -d ' ')
      DATE=$(stat -f '%Sm' -t '%Y-%m-%d %H:%M' "$f" 2>/dev/null || stat -c '%y' "$f" 2>/dev/null | cut -d. -f1)
      printf "  %-20s %5d signatures  (%s)\n" "$NAME" "$COUNT" "$DATE"
    done
    ;;
  *)
    echo "Usage: $0 {save|compare|list} [name] [package]"
    echo "  save <name> [pkg]     Save exported function signatures"
    echo "  compare <name> [pkg]  Compare current signatures against snapshot"
    echo "  list                  List saved snapshots"
    exit 1
    ;;
esac
