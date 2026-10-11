#!/bin/bash
set -e
cd "$(dirname "$0")"
D=..; F="$D/${COMPOSE:-compose.yml}"
[ -f $D/creds.txt ] || printf "user=conformance\npassword=%s\n" "$(openssl rand -hex 12)" > $D/creds.txt
[ -f $D/seed/1080.mkv ] || $D/make_seed.sh
docker compose -f $F down -t 5 >/dev/null 2>&1 || true
rm -rf $D/config $D/transcode $D/media $D/evidence/s7.log $D/evidence/s7.json $D/parked-*
mkdir -p $D/config/plex $D/config/jellyfin $D/config/emby $D/transcode $D/media/movies $D/media/tv $D/evidence
docker compose -f $F up -d
for u in localhost:8096/System/Info/Public localhost:8097/System/Info/Public localhost:32400/identity; do
  for i in $(seq 1 180); do curl -sf "$u" >/dev/null && break; sleep 1; done
  curl -s "$u" -H "Accept: application/json" | tee -a $D/evidence/s7.versions; echo >> $D/evidence/s7.versions
done
sleep 5
python3 mb.py jellyfin; python3 mb.py emby
python3 s7_move.py setup
[ -n "$SKIP_PLEX" ] || python3 s7_move.py plex
python3 s7_move.py mb
echo "evidence in $D/evidence/s7.log and s7.json"
