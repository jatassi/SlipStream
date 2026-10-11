import json, os, shutil, sys, time
from lib import MEDIA, SEED, EVID, copy, poll, Log
import plex
from mb import MB, lib_options

log = Log("s7")
LIB = "com.plexapp.plugins.library"
RESULTS = os.path.join(EVID, "s7.json")

PLEX_CASES = [
    ("P1", "Heat", 1995, 1, "partial", "gone", "same"),
    ("P2", "Collateral", 2004, 0, "partial", "gone", "same"),
    ("P3", "Thief", 1981, 1, "partial", "present", "same"),
    ("P4", "Manhunter", 1986, 0, "partial", "present", "same"),
    ("P5", "The Insider", 1999, 1, "full", "gone", "same"),
    ("P6", "Ali", 2001, 0, "full", "gone", "same"),
    ("P7", "Miami Vice", 2006, 1, "full", "present", "same"),
    ("P8", "Public Enemies", 2009, 0, "full", "present", "same"),
    ("P9", "Blackhat", 2015, 1, "partial", "gone", "other"),
    ("P10", "The Last of the Mohicans", 1992, 0, "partial", "gone", "other"),
    ("P11", "Ronin", 1998, 1, "partial", "present", "other"),
    ("P12", "Sicario", 2015, 0, "partial", "present", "other"),
]
PLEX_PURGE = ("P13", "Prisoners", 2013)

MB_CASES = [
    ("M1", "Drive", 2011, 64690, "remove-first", False),
    ("M2", "Zodiac", 2007, None, "remove-first", False),
    ("M3", "Arrival", 2016, 329865, "add-first", False),
    ("M4", "Gravity", 2013, None, "add-first", False),
    ("M5", "Inception", 2010, 27205, "realtime", True),
    ("M6", "Memento", 2000, None, "realtime", True),
]


def h(p):
    return os.path.join(MEDIA, p)


def save(key, value):
    d = json.load(open(RESULTS)) if os.path.exists(RESULTS) else {}
    d[key] = value
    json.dump(d, open(RESULTS, "w"), indent=1)


def plex_sections():
    s = plex.sections()
    return s["Moves A"]["key"], s["Moves C"]["key"]


def plex_setup():
    for top in ("plex-a", "plex-b", "plex-c"):
        os.makedirs(h(top), exist_ok=True)
    for cid, title, year, *_ in PLEX_CASES + [PLEX_PURGE]:
        name = f"{title} ({year})"
        copy(os.path.join(SEED, "1080.mkv"), h(f"plex-a/{name}/{name}.mkv"))
    log("plex prefs", plex.call("PUT", "/:/prefs", FSEventLibraryUpdatesEnabled=0, FSEventLibraryPartialScanEnabled=0,
                                ScheduledLibraryUpdatesEnabled=0, ButlerTaskRefreshLocalMedia=0, ButlerTaskRefreshPeriodicMetadata=0)[:2])
    q = "name=Moves%20A&type=movie&agent=tv.plex.agents.movie&scanner=Plex%20Movie&language=en-US&location=/media/plex-a&location=/media/plex-b"
    create = [("Moves A", lambda: plex.call("POST", "/library/sections?" + q)[0]),
              ("Moves C", lambda: plex.call("POST", "/library/sections", name="Moves C", type="movie", agent="tv.plex.agents.movie",
                                            scanner="Plex Movie", language="en-US", location="/media/plex-c")[0])]
    for name, make in create:
        code, _ = poll(lambda: make() in (200, 201), 300, 5)
        log("create", name, code)
    for n, d in plex.sections().items():
        log("plex section", n, d["key"], [l["path"] for l in d["Location"]])
    a, _ = plex_sections()
    want = len(PLEX_CASES) + 1
    _, took = poll(lambda: len([m for m in plex.leaves(a, 1) if m.get("guid", "").startswith("plex://")]) >= want and not plex.scanning(), 300, 2)
    log("plex initial scan + match took", took, "s")
    for m in plex.leaves(a, 1):
        log("  ", m["ratingKey"], m["title"], m.get("year"), m.get("guid"))


def plex_items(title):
    out = []
    for sid in plex_sections():
        for m in plex.mc(f"/library/sections/{sid}/all", type=1, includeGuids=1).get("Metadata", []):
            if m.get("title") == title:
                d = plex.meta(m["ratingKey"]) or {}
                out.append({
                    "section": sid, "ratingKey": d.get("ratingKey"), "guid": d.get("guid"),
                    "files": [p["file"] for md in d.get("Media", []) for p in md.get("Part", [])],
                    "deletedParts": [p["file"] for md in d.get("Media", []) for p in md.get("Part", []) if p.get("deletedAt")],
                    "deletedAt": d.get("deletedAt"), "viewCount": d.get("viewCount"), "lastViewedAt": d.get("lastViewedAt"),
                    "viewOffset": d.get("viewOffset"), "userRating": d.get("userRating"),
                    "collections": [c["tag"] for c in d.get("Collection", [])],
                })
    return out


def plex_history(title):
    c, r, _ = plex.call("GET", "/status/sessions/history/all", sort="viewedAt:desc")
    rows = (r or {}).get("MediaContainer", {}).get("Metadata", []) if isinstance(r, dict) else []
    return [{"ratingKey": x.get("ratingKey"), "viewedAt": x.get("viewedAt")} for x in rows if x.get("title") == title]


def plex_collection(title):
    for sid in plex_sections():
        for c in plex.mc(f"/library/sections/{sid}/collections").get("Metadata", []):
            if c.get("title") == title:
                kids = plex.mc(f"/library/collections/{c['ratingKey']}/children").get("Metadata", [])
                return {"key": c["ratingKey"], "section": sid, "members": [(k["ratingKey"], k["title"]) for k in kids]}
    return None


def plex_settle(path=None, sid=None, full=False):
    if full:
        r = plex.call("GET", f"/library/sections/{sid}/refresh")
    else:
        r = plex.call("GET", f"/library/sections/{sid}/refresh", path=path)
    log("   scan", "full" if full else path, "->", r[0])
    plex.wait_idle()
    time.sleep(4)
    plex.wait_idle()


def plex_arrive(path, sid, scan, title):
    plex_settle(path, sid, scan == "full")
    here = lambda: any(path in f for i in plex_items(title) for f in i["files"])
    ok, took = poll(here, 30, 2)
    log("   destination file on an item:", bool(ok), "after", took, "s")
    if not ok and scan == "partial":
        log("   partial scan did not pick it up; full scan of section", sid)
        plex_settle(None, sid, True)
        ok, took = poll(here, 60, 2)
        log("   destination file on an item after full scan:", bool(ok), took)


def plex_seed(cid, title):
    [it] = plex_items(title)
    rk = it["ratingKey"]
    mid = plex.mc("/identity")["machineIdentifier"]
    tl = dict(ratingKey=rk, key=f"/library/metadata/{rk}", duration=60000)
    log("   play to the end", plex.call("GET", "/:/timeline", state="playing", time=1000, **tl)[0],
        plex.call("GET", "/:/timeline", state="stopped", time=59000, **tl)[0],
        "partial rewatch", plex.call("GET", "/:/timeline", state="playing", time=15000, **tl)[0],
        plex.call("GET", "/:/timeline", state="stopped", time=20000, **tl)[0],
        "rate", plex.call("PUT", "/:/rate", identifier=LIB, key=rk, rating=8)[0],
        "collection", plex.call("POST", "/library/collections", type=1, title=f"Case {cid}", smart=0, sectionId=it["section"],
                                uri=f"server://{mid}/{LIB}/library/metadata/{rk}")[0])
    time.sleep(1)
    before = {"items": plex_items(title), "history": plex_history(title), "collection": plex_collection(f"Case {cid}")}
    log("   before", json.dumps(before))
    return before


def plex_case(cid, title, year, trash, scan, source, dest):
    name = f"{title} ({year})"
    a, c = plex_sections()
    log(f"== {cid} {name}: autoEmptyTrash={trash} scan={scan} source={source} dest={dest}")
    log("   pref", plex.call("PUT", "/:/prefs", autoEmptyTrash=trash)[0])
    before = plex_seed(cid, title)
    src, dst = h(f"plex-a/{name}"), h(f"plex-{'b' if dest == 'same' else 'c'}/{name}")
    dsid = a if dest == "same" else c
    cs, cd = "/media/plex-a/" + name, "/media/" + os.path.relpath(dst, MEDIA)
    if source == "gone":
        shutil.move(src, dst)
        log("   mv", src, "->", dst)
        plex_arrive(cd, dsid, scan, title)
        mid = {"items": plex_items(title)}
        plex_settle(cs, a, scan == "full") if (scan == "partial" or dest == "other") else None
    else:
        shutil.copytree(src, dst)
        log("   cp", src, "->", dst)
        plex_arrive(cd, dsid, scan, title)
        mid = {"items": plex_items(title)}
        shutil.rmtree(src)
        log("   rm", src)
        plex_settle(cs, a, scan == "full")
    log("   after destination scan", json.dumps(mid))
    after = {"items": plex_items(title), "history": plex_history(title), "collection": plex_collection(f"Case {cid}")}
    rk = before["items"][0]["ratingKey"]
    old = plex.meta(rk) or {}
    after["oldRatingKey"] = {"ratingKey": rk, "exists": bool(old), "deletedAt": old.get("deletedAt"), "viewCount": old.get("viewCount"),
                             "files": [p["file"] for md in old.get("Media", []) for p in md.get("Part", [])]}
    log("   after", json.dumps(after))
    save(cid, {"setup": dict(title=name, autoEmptyTrash=trash, scan=scan, source=source, dest=dest),
               "before": before, "afterDestinationScan": mid, "after": after})


def plex_purge(cid, title, year):
    name = f"{title} ({year})"
    a, _ = plex_sections()
    log(f"== {cid} {name}: purge, then re-add under the same GUID at a new path")
    log("   pref", plex.call("PUT", "/:/prefs", autoEmptyTrash=1)[0])
    before = plex_seed(cid, title)
    src = h(f"plex-a/{name}")
    keep = os.path.join(MEDIA, "..", "parked-" + name)
    shutil.move(src, keep)
    plex_settle(None, a, True)
    log("   emptyTrash", plex.call("PUT", f"/library/sections/{a}/emptyTrash")[0])
    plex.wait_idle()
    _, took = poll(lambda: not plex_items(title), 60, 2)
    log("   item gone after", took, "s")
    purged = {"items": plex_items(title), "history": plex_history(title), "collection": plex_collection(f"Case {cid}")}
    log("   purged", json.dumps(purged))
    shutil.move(keep, h(f"plex-b/{name}"))
    plex_arrive("/media/plex-b/" + name, a, "partial", title)
    poll(lambda: any((i["guid"] or "").startswith("plex://") for i in plex_items(title)), 60, 2)
    after = {"items": plex_items(title), "history": plex_history(title), "collection": plex_collection(f"Case {cid}")}
    log("   after", json.dumps(after))
    save(cid, {"setup": dict(title=name, purge=True), "before": before, "purged": purged, "after": after})


def mb_folder(title, year, tmdb):
    return f"{title} ({year})" + (f" [tmdbid-{tmdb}]" if tmdb else "")


def mb_lib(m):
    c, vf, _ = m.call("GET", "/Library/VirtualFolders")
    return next(v for v in vf if v["Name"] == "Moves")


def mb_setup(kind):
    m = MB(kind); m.login()
    for top in ("mb-a", "mb-b"):
        os.makedirs(h(top), exist_ok=True)
    for cid, title, year, tmdb, *_ in MB_CASES:
        f = mb_folder(title, year, tmdb)
        if not os.path.exists(h(f"mb-a/{f}")):
            copy(os.path.join(SEED, "1080.mkv"), h(f"mb-a/{f}/{title} ({year}).mkv"))
    copy(os.path.join(SEED, "1080.mkv"), h("mb-b/Heat (1995)/Heat (1995).mkv"))
    c, cfg, _ = m.call("GET", "/System/Configuration")
    cfg["LibraryMonitorDelaySeconds" if kind == "emby" else "LibraryMonitorDelay"] = 5
    log(kind, "delay 5", m.call("POST", "/System/Configuration", cfg)[0])
    lo = lib_options(kind, "/media/mb-a", "movies")
    lo["PathInfos"] = [{"Path": "/media/mb-a"}, {"Path": "/media/mb-b"}]
    log(kind, "create Moves", m.call("POST", "/Library/VirtualFolders?name=Moves&collectionType=movies&refreshLibrary=false&paths=/media/mb-a&paths=/media/mb-b",
                                     {"LibraryOptions": lo})[:2])
    v = mb_lib(m)
    if "/media/mb-b" not in v.get("Locations", []):
        for body in ({"Name": "Moves", "PathInfo": {"Path": "/media/mb-b"}}, {"Name": "Moves", "Path": "/media/mb-b"}):
            log(kind, "add path", json.dumps(body), m.call("POST", "/Library/VirtualFolders/Paths", body, refreshLibrary="false")[:2])
            if "/media/mb-b" in mb_lib(m).get("Locations", []):
                break
        v = mb_lib(m)
    log(kind, "Moves", v.get("Locations"), "realtime", v.get("LibraryOptions", {}).get("EnableRealtimeMonitor"))
    assert sorted(v.get("Locations", [])) == ["/media/mb-a", "/media/mb-b"], v.get("Locations")
    c, tasks, _ = m.call("GET", "/ScheduledTasks")
    for t in tasks:
        if t.get("Key") == "RefreshLibrary" or t.get("Name", "").lower() == "scan media library":
            log(kind, "clear scan triggers", m.call("POST", f"/ScheduledTasks/{t['Id']}/Triggers", [])[0])
    log(kind, "refresh", m.call("POST", "/Library/Refresh")[0])
    _, took = poll(lambda: len(mb_movies(m)) >= len(MB_CASES), 180, 2)
    log(kind, "initial scan took", took)
    for it in mb_movies(m):
        log("  ", it["Id"], it["Name"], it.get("Path"), it.get("ProviderIds"))


def mb_movies(m):
    return m.items(IncludeItemTypes="Movie", Fields="Path,ProviderIds,DateCreated")


def mb_state(m, title, year, coll):
    items = [{"Id": i["Id"], "Path": i.get("Path"), "ProviderIds": i.get("ProviderIds"),
              "UserData": {k: i.get("UserData", {}).get(k) for k in ("Played", "PlayCount", "LastPlayedDate", "PlaybackPositionTicks", "IsFavorite")}}
             for i in mb_movies(m) if os.path.basename((i.get("Path") or "")).startswith(f"{title} ({year})")]
    members = [(i["Id"], i.get("Path")) for i in m.items(ParentId=coll, Recursive="false")] if coll else None
    return {"items": items, "collection": members}


def mb_userdata(m, iid):
    for new, old in ((f"/UserPlayedItems/{iid}", f"/Users/{m.uid}/PlayedItems/{iid}"), (f"/UserFavoriteItems/{iid}", f"/Users/{m.uid}/FavoriteItems/{iid}")):
        c = m.call("POST", new, userId=m.uid, datePlayed="2026-10-01T12:00:00Z")[0]
        if c not in (200, 204):
            c = m.call("POST", old, DatePlayed="20261001120000")[0]
        log("   ", m.kind, "POST", new, "or", old, c)
    body = {"Played": True, "PlayCount": 2, "PlaybackPositionTicks": 200000000, "IsFavorite": True, "LastPlayedDate": "2026-10-01T12:00:00.0000000Z"}
    for path, q in ((f"/UserItems/{iid}/UserData", {"userId": m.uid}), (f"/Users/{m.uid}/Items/{iid}/UserData", {})):
        c = m.call("POST", path, body, **q)[0]
        log("   ", m.kind, "POST", path, c)
        if c in (200, 204):
            return


def mb_realtime(m, on):
    v = mb_lib(m)
    lo = v["LibraryOptions"]; lo["EnableRealtimeMonitor"] = on
    log("   ", m.kind, "realtime", on, m.call("POST", "/Library/VirtualFolders/LibraryOptions", {"Id": v["ItemId"], "LibraryOptions": lo})[0],
        "read back", mb_lib(m)["LibraryOptions"].get("EnableRealtimeMonitor"))


def mb_present(m, path):
    return any(i.get("Path") == path for i in mb_movies(m))


def mb_case(cid, title, year, tmdb, order, realtime):
    f = mb_folder(title, year, tmdb)
    file = f"{title} ({year}).mkv"
    old, new = f"/media/mb-a/{f}/{file}", f"/media/mb-b/{f}/{file}"
    log(f"== {cid} {f}: order={order} realtime={realtime} providerIds={'present' if tmdb else 'absent'}")
    ms = [MB("jellyfin"), MB("emby")]
    for m in ms:
        m.login()
        [it] = [i for i in mb_movies(m) if i.get("Path") == old]
        mb_userdata(m, it["Id"])
        c, r, _ = m.call("POST", "/Collections", Name=f"Case {cid}", Ids=it["Id"])
        m.coll = (r or {}).get("Id") if isinstance(r, dict) else None
        log("   ", m.kind, "collection", c, m.coll)
        if realtime:
            mb_realtime(m, True)
    time.sleep(2)
    before = {m.kind: mb_state(m, title, year, m.coll) for m in ms}
    log("   before", json.dumps(before))
    t0 = time.time()
    timing = {}
    if order == "remove-first":
        shutil.move(h(f"mb-a/{f}"), h(f"mb-b/{f}"))
        for m in ms:
            log("   ", m.kind, "Deleted", m.media_updated(old, "Deleted")[0])
        for m in ms:
            _, timing[m.kind + " old gone"] = poll(lambda: not mb_present(m, old), 60, 0.5)
        for m in ms:
            log("   ", m.kind, "Created", m.media_updated(new, "Created")[0])
        for m in ms:
            _, timing[m.kind + " new visible"] = poll(lambda: mb_present(m, new), 60, 0.5)
    elif order == "add-first":
        shutil.copytree(h(f"mb-a/{f}"), h(f"mb-b/{f}"))
        for m in ms:
            log("   ", m.kind, "Created", m.media_updated(new, "Created")[0])
        for m in ms:
            _, timing[m.kind + " new visible"] = poll(lambda: mb_present(m, new), 60, 0.5)
        mid = {m.kind: mb_state(m, title, year, m.coll) for m in ms}
        log("   both paths present", json.dumps(mid))
        shutil.rmtree(h(f"mb-a/{f}"))
        for m in ms:
            log("   ", m.kind, "Deleted", m.media_updated(old, "Deleted")[0])
        for m in ms:
            _, timing[m.kind + " old gone"] = poll(lambda: not mb_present(m, old), 60, 0.5)
    else:
        shutil.move(h(f"mb-a/{f}"), h(f"mb-b/{f}"))
        log("   mv only, no route")
        for m in ms:
            _, timing[m.kind + " new visible"] = poll(lambda: mb_present(m, new) and not mb_present(m, old), 150, 1)
    if not realtime:
        for m in ms:
            if mb_present(m, new):
                continue
            log("   ", m.kind, "new path missing; Created for the folder", m.media_updated(os.path.dirname(new), "Created")[0])
            ok, timing[m.kind + " new visible after folder report"] = poll(lambda: mb_present(m, new), 30, 0.5)
            if not ok:
                log("   ", m.kind, "still missing; library scan", m.call("POST", "/Library/Refresh")[0])
                _, timing[m.kind + " new visible after library scan"] = poll(lambda: mb_present(m, new), 120, 1)
    time.sleep(3)
    after = {m.kind: mb_state(m, title, year, m.coll) for m in ms}
    log("   timing", timing)
    log("   after", json.dumps(after))
    for m in ms:
        if realtime:
            mb_realtime(m, False)
    save(cid, {"setup": dict(folder=f, order=order, realtime=realtime, providerIds=bool(tmdb)), "before": before, "after": after, "timing": timing})


if __name__ == "__main__":
    cmd = sys.argv[1]
    if cmd == "setup":
        plex_setup(); mb_setup("jellyfin"); mb_setup("emby")
    elif cmd == "plex":
        for case in PLEX_CASES:
            plex_case(*case)
        plex_purge(*PLEX_PURGE)
    elif cmd == "mb":
        for case in MB_CASES:
            mb_case(*case)
