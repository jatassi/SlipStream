#!/usr/bin/env bash
# Lint a single file and show violations.
# Usage: ./scripts/lint/file.sh src/routes/movies/\$id.tsx
set -euo pipefail

if [ -z "${1:-}" ]; then
  echo "Usage: $0 <file-path>"
  echo "  Path relative to web/, e.g.: src/routes/movies/\\\$id.tsx"
  exit 1
fi

cd "$(dirname "$0")/../../web"
bunx eslint "$1" 2>&1 || true
