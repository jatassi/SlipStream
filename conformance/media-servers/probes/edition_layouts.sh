#!/bin/bash
set -e
cd "$(dirname "$0")"
D=..; F="$D/compose.edition.yml"
[ -f $D/creds.txt ] || printf "user=conformance\npassword=%s\n" "$(openssl rand -hex 12)" > $D/creds.txt
[ -f $D/seed/1080.mkv ] || $D/make_seed.sh
mkdir -p $D/evidence
for L in "$@"; do
  docker compose -f $F down -t 5 >/dev/null 2>&1
  rm -rf $D/config $D/transcode; mkdir -p $D/config/plex $D/config/jellyfin $D/config/emby $D/transcode $D/media/tv
  python3 edition_layouts.py build "$L" >/dev/null
  docker compose -f $F up -d >/dev/null 2>&1
  for u in localhost:8096/System/Info/Public localhost:8097/System/Info/Public localhost:32400/identity; do
    for i in $(seq 1 120); do curl -sf "$u" >/dev/null && break; sleep 1; done
  done
  sleep 5
  { python3 mb.py jellyfin; python3 mb.py emby; python3 setup_mb.py jellyfin; python3 setup_mb.py emby; python3 plex.py setup; } > $D/evidence/$L.setup.log 2>&1
  python3 - <<'PY'
import time, plex
for i in range(90):
    try:
        if plex.leaves(plex.sections()["Movies"]["key"], 1) and not plex.scanning(): break
    except KeyError:
        pass
    time.sleep(2)
PY
  sleep 15
  { python3 edition_layouts.py dump "$L"; python3 plex_refresh_dump.py; } | tee $D/evidence/$L.txt
done
