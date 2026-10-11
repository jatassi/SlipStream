#!/bin/sh
# PROTOTYPE, throwaway. Copies v1's cached artwork for Dune: Part Two and Severance (never committed),
# then serves prototypes/ so this folder can reuse Lit console's stylesheet. Port $PORT (default 4175).
set -e
cd "$(dirname "$0")"
SRC="${V1_ART:-$HOME/Developer/SlipStream-v1/data/artwork}"
if [ ! -d art ]; then
  mkdir -p art/movie art/series
  cp "$SRC/movie/693134_"* art/movie/ 2>/dev/null || true
  cp "$SRC/series/95396_"* art/series/ 2>/dev/null || true
fi
PORT="${PORT:-4175}"
echo "http://localhost:$PORT/subtitles/?variant=A"
cd ..
exec python3 -m http.server "$PORT"
