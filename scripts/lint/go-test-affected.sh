#!/usr/bin/env bash
# Test only Go packages affected by recently modified files.
# Usage: ./scripts/lint/go-test-affected.sh                  # uses git diff (unstaged + staged)
#        ./scripts/lint/go-test-affected.sh file1.go file2.go # explicit files
#        ./scripts/lint/go-test-affected.sh --staged          # only staged files
set -euo pipefail

cd "$(dirname "$0")/../.."

if [[ "${1:-}" == "--staged" ]]; then
  FILES=$(git diff --cached --name-only --diff-filter=ACMR -- '*.go' | grep -v '_test.go' || true)
elif [[ $# -gt 0 ]]; then
  FILES="$*"
else
  FILES=$(git diff --name-only --diff-filter=ACMR -- '*.go' | grep -v '_test.go' || true)
  STAGED=$(git diff --cached --name-only --diff-filter=ACMR -- '*.go' | grep -v '_test.go' || true)
  FILES=$(printf '%s\n%s' "$FILES" "$STAGED" | sort -u)
fi

if [[ -z "$FILES" ]]; then
  echo "No modified Go files found."
  exit 0
fi

PKGS=""
while IFS= read -r f; do
  [[ -z "$f" ]] && continue
  dir=$(dirname "$f")
  pkg="./$dir/..."
  PKGS=$(printf '%s\n%s' "$PKGS" "$pkg")
done <<< "$FILES"

PKGS=$(echo "$PKGS" | sort -u | grep -v '^$')
COUNT=$(echo "$PKGS" | wc -l | tr -d ' ')

echo "=== Testing $COUNT affected package(s) ==="
echo "$PKGS"
echo ""

go test $PKGS
