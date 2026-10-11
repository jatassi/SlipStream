#!/bin/sh
# PROTOTYPE, throwaway. Copies v1's cached artwork (never committed) and serves prototypes/ on $PORT (default 4175),
# because the page loads Lit console's stylesheet from ../ui-direction.
set -e
cd "$(dirname "$0")"
SRC="${V1_ART:-$HOME/Developer/SlipStream-v1/data/artwork}"
if [ ! -d art ]; then
  mkdir -p art/movie art/series
  for id in 693134 872585 157336 361743 545611 414906 284054 603 155 1124 438631 335984 329865 76341 27205 496243 120467 244786 313369 49047 530915 577922 370172 76600 569094 346698 466420 792307 840430 666277 940721 786892 929590 945961 402431 558449 974576 1064213 933260 575264 603692 475557 680 13 78 62 348 238; do cp "$SRC/movie/${id}_"* art/movie/ 2>/dev/null || true; done
  for id in 95396 76479 94997 100088 60059 1399 63351; do cp "$SRC/series/${id}_"* art/series/ 2>/dev/null || true; done
fi
PORT="${PORT:-4175}"
echo "http://localhost:$PORT/conversion-queue/?variant=A"
cd ..
exec python3 -m http.server "$PORT"
