#!/bin/sh
# PROTOTYPE, throwaway. Fills ui-direction/art (gitignored) when empty, then serves prototypes/ on $PORT (default 4175).
set -e
cd "$(dirname "$0")/.."
[ -f ui-direction/art/movie/414906_poster.jpg ] || sh ui-direction/fetch-art.sh
PORT="${PORT:-4175}"
echo "http://localhost:$PORT/manual-search/"
exec python3 -m http.server "$PORT"
