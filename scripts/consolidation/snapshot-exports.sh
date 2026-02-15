#!/usr/bin/env bash
# Snapshot all exported symbols from given files.
# Run before and after a refactor, diff the outputs to detect API surface changes.
# Usage: ./scripts/consolidation/snapshot-exports.sh <file1> [file2] ...
# Example: ./scripts/consolidation/snapshot-exports.sh web/src/components/slots/DryRunModal/*.tsx
# Pipe to a file: ./scripts/consolidation/snapshot-exports.sh web/src/components/slots/DryRunModal/*.tsx > before.txt
set -euo pipefail

if [ $# -eq 0 ]; then
  echo "Usage: $0 <file1> [file2] ..."
  echo "Outputs sorted export declarations from each file."
  exit 1
fi

for file in "$@"; do
  # Normalize to just filename for comparison across directories
  BASENAME=$(basename "$file")
  echo "=== $BASENAME ($file) ==="
  if [ ! -f "$file" ]; then
    echo "  FILE NOT FOUND"
  else
    # Extract exports: function names, const names, type names, interface names, default
    grep -E '^export ' "$file" | \
      sed 's/export default /DEFAULT: /' | \
      sed 's/export function /fn: /' | \
      sed 's/export const /const: /' | \
      sed 's/export type /type: /' | \
      sed 's/export interface /interface: /' | \
      sed 's/export enum /enum: /' | \
      sed 's/{.*//' | \
      sed 's/(.*//' | \
      sort | \
      sed 's/^/  /'
  fi
  echo ""
done
