import time, plex
from edition_layouts import dump_plex
print("-- before item refresh"); dump_plex()
sid = plex.sections()["Movies"]["key"]
for m in plex.leaves(sid, 1):
    print("refresh", m["ratingKey"], plex.call("PUT", f"/library/metadata/{m['ratingKey']}/refresh")[:2])
time.sleep(20); plex.wait_idle()
print("-- after item refresh"); dump_plex()
