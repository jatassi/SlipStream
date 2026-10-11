import time, sys
from scen import *
log = Log("s2")
HEAT1 = "movies/Heat (1995)/Heat (1995) - 1080p.mkv"
HEAT4 = "movies/Heat (1995)/Heat (1995) - 4K HDR.mkv"
COL1 = "movies/Collateral (2004)/Collateral (2004) - 1080p.mkv"

def side(video, lang):
    return video[:-4] + f".{lang}.srt"

def wait_sub(s, video, lang, timeout=90):
    t0 = time.time()
    while time.time() - t0 < timeout:
        subs = s.subs(video) or []
        if any((x.get("lang") or "").startswith(lang[:2]) or (x.get("path") or "").endswith(f".{lang}.srt") for x in subs):
            return round(time.time() - t0, 1), subs
        time.sleep(1)
    return None, s.subs(video)

def run(case, s, video, lang, route):
    copy(os.path.join(SEED, {"de": "fr", "es": "fr"}.get(lang, lang) + ".srt"), host(side(video, lang)))
    log(s.kind, case, "added", side(video, lang))
    if route == "plexA":
        rk = s.find_part(video)["ratingKey"]; sent = s.refresh_item(rk)
    elif route == "plexB":
        sent = s.scan_folder(video)
    elif route == "video":
        sent = s.report(video, "Modified")
    elif route == "sidecar":
        sent = s.report(side(video, lang), "Created")
    log(s.kind, case, "route", sent)
    lat, subs = wait_sub(s, video, lang)
    log(s.kind, case, "subtitle on version:", "after %ss" % lat if lat is not None else "NOT within 90s", subs)
    return lat

if __name__ == "__main__":
    p, j, e = servers()
    which = sys.argv[1]
    if which == "plex":
        run("A(item refresh) Heat1080 en", p, HEAT1, "en", "plexA")
        run("A(item refresh) Collateral1080 en", p, COL1, "en", "plexA")
        run("B(folder scan) Heat4K fr", p, HEAT4, "fr", "plexB")
    else:
        s = j if which == "jellyfin" else e
        if len(sys.argv) > 2:
            run("S(sidecar path) Heat1080 de", s, HEAT1, "de", "sidecar")
            run("M(video path) Collateral1080 es", s, COL1, "es", "video")
            sys.exit()
        run("S(sidecar path) Heat1080 en", s, HEAT1, "en", "sidecar")
        run("M(video path) Collateral1080 en", s, COL1, "en", "video")
