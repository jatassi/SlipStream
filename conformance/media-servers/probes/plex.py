import json, sys, time, urllib.parse
from lib import req
BASE = "http://localhost:32400"
HDR = {"X-Plex-Client-Identifier": "conformance-probe", "X-Plex-Product": "conformance", "X-Plex-Version": "1.0",
       "X-Plex-Platform": "Chrome", "X-Plex-Device": "probe", "X-Plex-Device-Name": "probe"}

def call(method, _p, raw=False, headers=None, **q):
    url = BASE + _p
    if q:
        url += ("&" if "?" in url else "?") + urllib.parse.urlencode(q)
    h = dict(HDR); h.update(headers or {})
    return req(method, url, None, h, raw=raw)

def mc(path, **q):
    c, r, _ = call("GET", path, **q)
    return r.get("MediaContainer", {}) if isinstance(r, dict) else {}

def sections():
    return {d["title"]: d for d in mc("/library/sections").get("Directory", [])}

def scanning():
    return any(d.get("refreshing") for d in sections().values())

def wait_idle(timeout=300):
    t = time.time(); time.sleep(1)
    while time.time() - t < timeout and scanning():
        time.sleep(1)
    return round(time.time() - t, 1)

def leaves(sid, typ):
    return mc(f"/library/sections/{sid}/all", type=typ, includeGuids=1).get("Metadata", [])

def meta(rk):
    m = mc(f"/library/metadata/{rk}").get("Metadata", [])
    return m[0] if m else None

def describe(m):
    out = [f"{m['type']} {m.get('title')!r} ratingKey={m['ratingKey']} editionTitle={m.get('editionTitle')!r}"]
    for md in m.get("Media", []):
        for p in md.get("Part", []):
            subs = [(s.get("id"), s.get("languageCode"), s.get("key"), s.get("codec")) for s in p.get("Stream", []) if s.get("streamType") == 3]
            out.append(f"   Media id={md['id']} {md.get('videoResolution')} {md.get('width')}x{md.get('height')} title={md.get('title')!r} Part id={p['id']} file={p['file']} subs={subs}")
    return "\n".join(out)

if __name__ == "__main__" and sys.argv[1:] == ["setup"]:
    print("prefs", call("PUT", "/:/prefs", FSEventLibraryUpdatesEnabled=0, FSEventLibraryPartialScanEnabled=0, ScheduledLibraryUpdatesEnabled=0, ButlerTaskRefreshLocalMedia=0, ButlerTaskRefreshPeriodicMetadata=0)[:2])
    s = sections()
    for name, typ, agent, scanner, loc in [("Movies", "movie", "tv.plex.agents.movie", "Plex Movie", "/media/movies"),
                                           ("Shows", "show", "tv.plex.agents.series", "Plex TV Series", "/media/tv")]:
        if name not in s:
            print("create", name, call("POST", "/library/sections", name=name, type=typ, agent=agent, scanner=scanner, language="en-US", location=loc)[:2])
    for n, d in sections().items():
        print(n, d["key"], d["agent"], d["scanner"], [l["path"] for l in d["Location"]])
    print("initial scan wait", wait_idle())
