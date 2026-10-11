import time, json, re, threading, sys
import plex
from lib import Log, trim
log = Log("s6-plex")
RK = "15"   # Collateral
CID = "conformance-player"
H = {"X-Plex-Client-Identifier": CID, "X-Plex-Device-Name": "conformance-player", "X-Plex-Platform": "Chrome", "X-Plex-Product": "Plex Web", "X-Plex-Session-Identifier": "conf-sess-1"}
mode = sys.argv[1] if len(sys.argv) > 1 else "transcode"
notl = "notimeline" in sys.argv
m = plex.meta(RK)
idx = [i for i, md in enumerate(m["Media"]) if md["videoResolution"] == "1080"][0]
q = dict(path=f"/library/metadata/{RK}", mediaIndex=idx, partIndex=0, protocol="hls", fastSeek=1, session="conf-tc-1", hasMDE=1, location="lan")
if mode == "transcode":
    q.update(directPlay=0, directStream=0, videoResolution="640x360", maxVideoBitrate=400, videoQuality=50)
else:
    q.update(directPlay=0, directStream=1, directStreamAudio=1, videoResolution="3840x2160", maxVideoBitrate=200000, videoQuality=100)
c, dec, ms = plex.call("GET", "/video/:/transcode/universal/decision", headers=H, **q)
log(mode, "decision", c, ms, "ms", trim(dec, 1200))
c, body, ms = plex.call("GET", "/video/:/transcode/universal/start.m3u8", raw=True, headers=H, **q)
log(mode, "start.m3u8", c, ms, "ms", body[:300])
sub = [l for l in body.decode().splitlines() if l and not l.startswith("#")]
if sub:
    c, pl, ms = plex.call("GET", "/video/:/transcode/universal/" + sub[0], raw=True, headers=H)
    log("variant", c, pl[:200])
    segs = [l for l in pl.decode().splitlines() if l and not l.startswith("#")]
    if segs:
        c, seg, ms = plex.call("GET", "/video/:/transcode/universal/session/conf-tc-1/base/" + segs[0] if not segs[0].startswith("/") else segs[0], raw=True, headers=H)
        log("segment0", c, len(seg), "bytes", ms, "ms")
for t in ([] if notl else range(0, 12000, 3000)):
    c, r, _ = plex.call("GET", "/:/timeline", headers=H, ratingKey=RK, key=f"/library/metadata/{RK}", state="playing", time=t, duration=m["duration"], playbackTime=t)
    time.sleep(1)
log(mode, "timeline", c)
c, ss, _ = plex.call("GET", "/status/sessions")
log(mode, "notimeline" if notl else "", "/status/sessions", c, "size", ss["MediaContainer"].get("size"))
for md in ss["MediaContainer"].get("Metadata", []):
    log("  session keys:", sorted(k for k in md if k[0].isupper()))
    log("  Session:", md.get("Session"), "Player:", {k: md.get("Player", {}).get(k) for k in ("machineIdentifier", "product", "platform", "state", "local")})
    log("  TranscodeSession:", md.get("TranscodeSession"))
    for x in md.get("Media", []):
        for p in x.get("Part", []):
            log("  Media", x.get("id"), "selected", x.get("selected"), "Part.decision", p.get("decision"), "streams", [(st.get("streamType"), st.get("decision"), st.get("location")) for st in p.get("Stream", [])])
c, ts, _ = plex.call("GET", "/transcode/sessions")
log(mode, "/transcode/sessions", c, trim(ts, 2500))
time.sleep(4)
plex.call("GET", "/:/timeline", headers=H, ratingKey=RK, key=f"/library/metadata/{RK}", state="stopped", time=12000, duration=m["duration"])
plex.call("GET", "/video/:/transcode/universal/stop", headers=H, session="conf-tc-1")
time.sleep(2)
c, ss, _ = plex.call("GET", "/status/sessions")
log(mode, "after stop /status/sessions size", ss.get("MediaContainer", {}).get("size") if isinstance(ss, dict) else ss)
