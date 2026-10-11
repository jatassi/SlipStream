#!/bin/sh
# PROTOTYPE, throwaway. Reuses the direction prototype's artwork copy (never committed) and serves prototypes/ on $PORT (default 4174).
set -e
cd "$(dirname "$0")/.."
SRC="${V1_ART:-$HOME/Developer/SlipStream-v1/data/artwork}"
if [ ! -d ui-direction/art ]; then
  mkdir -p ui-direction/art/movie ui-direction/art/series
  for id in 693134 872585 157336 361743 545611 414906 284054 603 155 1124; do cp "$SRC/movie/${id}_"* ui-direction/art/movie/ 2>/dev/null || true; done
  for id in 95396 76479 94997 100088 60059 1399 63351; do cp "$SRC/series/${id}_"* ui-direction/art/series/ 2>/dev/null || true; done
fi
PORT="${PORT:-4174}"
echo "http://localhost:$PORT/torrent-client/"
exec python3 -m http.server "$PORT"
