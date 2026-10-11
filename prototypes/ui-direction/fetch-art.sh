#!/bin/sh
# PROTOTYPE, throwaway. Fills art/ from TMDB's public pages when v1's artwork cache isn't on this machine. Never committed.

cd "$(dirname "$0")"
mkdir -p art/movie art/series
fetch() {
  page=$(curl -sSL --max-time 40 --retry 4 --retry-all-errors -A 'Mozilla/5.0' "https://www.themoviedb.org/$1/$2")
  urls=$(printf '%s' "$page" | grep -oE 'og:image" content="[^"]+"' | cut -d'"' -f3 | sed 's#media.themoviedb.org#image.tmdb.org#')
  poster=$(printf '%s\n' "$urls" | sed -n 1p)
  backdrop=$(printf '%s\n' "$urls" | sed -n 2p | sed 's#/w780/#/w1280/#')
  [ -n "$poster" ] && curl -sSf --max-time 60 --retry 4 --retry-all-errors -o "art/$3/$2_poster.jpg" "$poster"
  [ -n "$backdrop" ] && curl -sSf --max-time 60 --retry 4 --retry-all-errors -o "art/$3/$2_backdrop.jpg" "$backdrop"
  echo "$3 $2"
}
for id in 693134 872585 157336 361743 545611 414906 284054 603 155 1124 19995; do fetch movie "$id" movie; done
for id in 95396 76479 94997 100088 60059 1399 63351; do fetch tv "$id" series; done
