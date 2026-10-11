import time
from scen import *
log = Log("s1")
import sys
FILES = [("movies/Ronin (1998)/Ronin (1998) - 1080p.mkv", "1080.mkv"), ("tv/The Wire/Season 01/The Wire - S01E03 - 1080p.mkv", "1080.mkv")]
for rel, seed in (FILES if len(sys.argv) < 2 else [(sys.argv[1], "1080.mkv")]):
    copy(os.path.join(SEED, seed), host(rel))
    log("added", rel)
    srv = servers()
    t0 = time.time()
    sent = {}
    for s in srv:
        if s.kind == "plex":
            sent[s.kind] = s.scan_folder(rel)
        else:
            sent[s.kind] = s.report(rel, "Created")
        log(s.kind, "route", sent[s.kind])
    pending = {s.kind: s for s in srv}
    while pending and time.time() - t0 < 180:
        for k, s in list(pending.items()):
            hit = s.find_part(rel) if k == "plex" else s.find(rel)
            if hit:
                log(k, "visible after", round(time.time() - t0, 1), "s", hit)
                pending.pop(k)
        time.sleep(0.5)
    for k in pending:
        log(k, "NOT visible after", round(time.time() - t0, 1), "s")
