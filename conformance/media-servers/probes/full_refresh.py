import time, plex
from mb import MB
for k in ("jellyfin", "emby"):
    m = MB(k); m.login(); print(k, "POST /Library/Refresh", m.call("POST", "/Library/Refresh")[:2])
for name, d in plex.sections().items():
    print("plex", name, plex.call("GET", f"/library/sections/{d['key']}/refresh")[:2])
time.sleep(45)
