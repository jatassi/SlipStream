#!/bin/bash
set -e
cd "$(dirname "$0")"
[ -f ../creds.txt ] || printf "user=conformance\npassword=%s\n" "$(openssl rand -hex 12)" > ../creds.txt
[ -f ../seed/1080.mkv ] || ../make_seed.sh
python3 mb.py jellyfin >/dev/null; python3 mb.py emby >/dev/null
python3 setup_mb.py jellyfin; python3 setup_mb.py emby
python3 plex.py setup
python3 reset_media.py && python3 full_refresh.py
python3 dump_plex.py; python3 dump_mb.py jellyfin; python3 dump_emby_all.py
pgrep -f "ws.py plex" >/dev/null || (nohup python3 ws.py plex > ../evidence/ws-plex.stdout 2>&1 &)
pgrep -f "ws.py jellyfin" >/dev/null || (nohup python3 ws.py jellyfin > ../evidence/ws-jellyfin.stdout 2>&1 &)
pgrep -f "ws.py emby" >/dev/null || (nohup python3 ws.py emby > ../evidence/ws-emby.stdout 2>&1 &)
python3 set_delay.py 60 >/dev/null; python3 - <<'PY'
from mb import MB
m = MB("emby"); m.login(); c, cfg, _ = m.call("GET", "/System/Configuration"); cfg["LibraryMonitorDelaySeconds"] = 90; m.call("POST", "/System/Configuration", cfg)
PY
python3 s1_new_video.py
python3 set_delay.py 5
# Sequential: a second login with the same DeviceId revokes the first token.
python3 s2_sidecar.py plex; python3 s2_sidecar.py jellyfin; python3 s2_sidecar.py emby
python3 s3_replace.py; python3 s3b_vtt.py
python3 s4_remove.py
python3 s6_plex.py transcode notimeline; python3 s6_plex.py transcode; python3 s6_plex.py direct; python3 s6_plex_directplay.py
for k in jellyfin emby; do for a in "transcode" "transcode noreport" "profile" "direct"; do python3 s6_mb.py $k $a; done; done
echo "evidence in ../evidence"
