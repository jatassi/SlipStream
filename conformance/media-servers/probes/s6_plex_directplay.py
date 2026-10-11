import time, plex
from lib import Log
log = Log("s6-plex")
H = {"X-Plex-Client-Identifier": "conformance-dp", "X-Plex-Device-Name": "dp", "X-Plex-Platform": "Chrome", "X-Plex-Product": "Plex Web", "X-Plex-Session-Identifier": "conf-dp-1"}
m = plex.meta("15"); part = [p for md in m["Media"] if md["videoResolution"] == "1080" for p in md["Part"]][0]
c, b, ms = plex.call("GET", part["key"], raw=True, headers=H)
log("directplay part download", part["key"], c, len(b))
for t in (0, 3000, 6000):
    plex.call("GET", "/:/timeline", headers=H, ratingKey="15", key="/library/metadata/15", state="playing", time=t, duration=m["duration"]); time.sleep(1)
ss = plex.call("GET", "/status/sessions")[1]["MediaContainer"]
for md in ss.get("Metadata", []):
    log("directplay session: TranscodeSession", md.get("TranscodeSession"), "Session", md.get("Session"), "Media", [(x.get("id"), x.get("selected"), [p.get("decision") for p in x.get("Part", [])]) for x in md["Media"]])
log("directplay /transcode/sessions size", plex.call("GET", "/transcode/sessions")[1]["MediaContainer"].get("size"))
plex.call("GET", "/:/timeline", headers=H, ratingKey="15", key="/library/metadata/15", state="stopped", time=6000, duration=m["duration"])
