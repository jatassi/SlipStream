#!/bin/sh
# PROTOTYPE, throwaway. Copies v1's cached artwork, or fetches it from TMDB when absent (never committed), and serves on $PORT (default 4173).
set -e
cd "$(dirname "$0")"
SRC="${V1_ART:-$HOME/Developer/SlipStream-v1/data/artwork}"
if [ ! -e art/movie/19995_poster.jpg ]; then
  mkdir -p art/movie art/series
  for id in 693134 872585 157336 361743 545611 414906 284054 603 155 1124 19995; do cp "$SRC/movie/${id}_"* art/movie/ 2>/dev/null || true; done
  for id in 95396 76479 94997 100088 60059 1399 63351; do cp "$SRC/series/${id}_"* art/series/ 2>/dev/null || true; done
  [ -e art/movie/19995_poster.jpg ] || ./fetch-art.sh
fi
PORT="${PORT:-4173}"
echo "http://localhost:$PORT/?variant=A"
echo "http://localhost:$PORT/scale.html (Work page at scale)"
exec python3 -m http.server "$PORT"
