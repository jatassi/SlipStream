import time, sys
from scen import *
import plex
log = Log("s3")
VID = "movies/Heat (1995)/Heat (1995) - 1080p.mkv"
SUB = VID[:-4] + ".en.srt"

def fetch(s):
    """Return (url, served text) for the en sidecar on VID's version."""
    if s.kind == "plex":
        st = [x for x in s.subs(VID) if x["lang"] == "eng"]
        if not st: return None, None
        url = st[0]["key"]
        c, body, _ = plex.call("GET", url, raw=True)
        return f"GET {url} -> {c}", body.decode(errors="replace")
    it, src = s.source(VID)
    st = [x for x in src.get("MediaStreams", []) if x["Type"] == "Subtitle" and (x.get("Path") or "").endswith(".en.srt")]
    if not st: return None, None
    url = f"/Videos/{it['Id']}/{src['Id']}/Subtitles/{st[0]['Index']}/Stream.srt"
    c, body, _ = s.m.call("GET", url, raw=True)
    return f"GET {url} -> {c}", body.decode(errors="replace")

def cue(text):
    return "Replaced cue" if text and "Replaced cue" in text else ("Hello conformance" if text and "Hello" in text else repr((text or "")[:80]))

srv = servers()
for s in srv:
    log(s.kind, "before:", *[cue(x) if i else x for i, x in enumerate(fetch(s))])
copy(os.path.join(SEED, "en-v2.srt"), host(SUB))
log("replaced", SUB, "size", os.path.getsize(host(SUB)))
time.sleep(1)
for s in srv:
    log(s.kind, "after replace, no route:", *[cue(x) if i else x for i, x in enumerate(fetch(s))])
t0 = time.time()
for s in srv:
    if s.kind == "plex":
        sent = s.refresh_item(s.find_part(VID)["ratingKey"])
    else:
        sent = s.report(VID, "Modified")
    log(s.kind, "route", sent)
pending = {s.kind: s for s in srv}
while pending and time.time() - t0 < 60:
    for k, s in list(pending.items()):
        u, txt = fetch(s)
        if cue(txt) == "Replaced cue":
            log(k, "serves new cue after", round(time.time() - t0, 1), "s via", u); pending.pop(k)
    time.sleep(1)
for k, s in pending.items():
    log(k, "still old after 60s:", *[cue(x) if i else x for i, x in enumerate(fetch(s))])
