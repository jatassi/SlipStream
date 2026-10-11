#!/bin/bash
set -e
cd "$(dirname "$0")"
D=..; F="$D/compose.edition.yml"
P="$D/config/plex/Library/Application Support/Plex Media Server/Preferences.xml"
[ -f $D/seed/1080.mkv ] || $D/make_seed.sh
mkdir -p $D/evidence
docker compose -f $F down -t 5 >/dev/null 2>&1 || true
rm -rf $D/config/plex $D/transcode; mkdir -p $D/config/plex $D/transcode $D/media/tv
python3 edition_layouts.py build L0 >/dev/null
docker compose -f $F up -d plex
for i in $(seq 1 120); do curl -sf localhost:32400/identity >/dev/null && break; sleep 1; done
curl -s -o /dev/null -w "claim HTTP %{http_code}\n" -X POST -H "X-Plex-Client-Identifier: conformance-probe" "localhost:32400/myplex/claim?token=$PLEX_CLAIM"
for i in $(seq 1 30); do grep -q PlexOnlineToken "$P" 2>/dev/null && break; sleep 1; done
export PLEX_TOKEN=$(grep -o 'PlexOnlineToken="[^"]*"' "$P" | cut -d'"' -f2)
[ -n "$PLEX_TOKEN" ] || { echo "claim failed"; exit 1; }
curl -s -H "X-Plex-Token: $PLEX_TOKEN" -H "Accept: application/json" localhost:32400/ | python3 -c 'import json,sys; m=json.load(sys.stdin)["MediaContainer"]; print("version", m.get("version"), "myPlexSubscription", m.get("myPlexSubscription"))'
for L in "$@"; do
  python3 - <<'PY'
import plex
for n, d in plex.sections().items():
    print("delete", n, plex.call("DELETE", f"/library/sections/{d['key']}")[:2])
PY
  python3 edition_layouts.py build "$L" >/dev/null
  python3 plex.py setup > $D/evidence/pp-$L.setup.log 2>&1
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
  { echo "### Plex Pass layout $L"; python3 plex_refresh_dump.py; } | tee $D/evidence/pp-$L.txt
done
curl -s -o /dev/null -w "unclaim HTTP %{http_code}\n" -X DELETE -H "X-Plex-Token: $PLEX_TOKEN" localhost:32400/myplex/account
