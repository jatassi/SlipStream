import time, shutil
from scen import *
import plex
log = Log("s4")
H4 = "movies/Heat (1995)/Heat (1995) - 4K HDR.mkv"
RON = "movies/Ronin (1998)/Ronin (1998) - 1080p.mkv"
srv = servers()
p = srv[0]
log("plex before: Heat rk", p.find_part(H4), "Ronin", p.find_part(RON))
for s in srv[1:]:
    log(s.kind, "before: Heat4K item", s.find(H4), "Ronin", s.find(RON))
os.remove(host(H4)); os.remove(host(H4[:-4] + ".fr.srt"))
shutil.rmtree(host("movies/Ronin (1998)"))
log("deleted", H4, "+ .fr.srt and folder Ronin (1998)")
t0 = time.time()
for s in srv:
    for rel in (H4, RON):
        sent = s.scan_folder(rel) if s.kind == "plex" else s.report(rel, "Deleted")
        log(s.kind, "route", sent)
pending = {(s.kind, rel): s for s in srv for rel in (H4, RON)}
while pending and time.time() - t0 < 90:
    for (k, rel), s in list(pending.items()):
        gone = not (s.find_part(rel) if k == "plex" else s.find(rel))
        if gone:
            log(k, rel, "gone after", round(time.time() - t0, 1), "s"); pending.pop((k, rel))
    time.sleep(0.5)
for (k, rel) in pending:
    log(k, rel, "STILL PRESENT after 90s")
log("plex Heat now:\n" + plex.describe(plex.meta("1")))
for s in srv[1:]:
    it, _ = s.source("movies/Heat (1995)/Heat (1995) - 1080p.mkv")
    log(s.kind, "Heat 1080p item versions now:", [x.get("Name") for x in it.get("MediaSources", [])])
