import time
from scen import *
import plex
log = Log("s3")
VID = "movies/Heat (1995)/Heat (1995) - 1080p.mkv"; SUB = VID[:-4] + ".en.srt"
def vtt(s):
    it, src = s.source(VID)
    st = [x for x in src["MediaStreams"] if x["Type"] == "Subtitle" and (x.get("Path") or "").endswith(".en.srt")][0]
    url = f"/Videos/{it['Id']}/{src['Id']}/Subtitles/{st['Index']}/Stream.vtt"
    c, b, _ = s.m.call("GET", url, raw=True); b = b.decode(errors="replace")
    return url, c, "Replaced cue" if "Replaced" in b else ("Hello conformance" if "Hello" in b else b[:60])
js = servers()[1:]
for seed, label in [("en.srt", "back to v1"), ("en-v2.srt", "v2 again")]:
    copy(os.path.join(SEED, seed), host(SUB)); time.sleep(1)
    for s in js: log(s.kind, "VTT conversion after", label, "no route:", vtt(s))
for s in js: s.report(VID, "Modified")
time.sleep(8)
for s in js: log(s.kind, "VTT after route + 8s:", vtt(s))
