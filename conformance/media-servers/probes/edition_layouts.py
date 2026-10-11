import json, os, shutil, sys, time
from lib import MEDIA, SEED, copy
import plex
from mb import MB

STD = ["Heat (1995)/Heat (1995) - 1080p.mkv", "Heat (1995)/Heat (1995) - 4K HDR.mkv"]


def dc(fmt):
    return [fmt.format(q="4K HDR"), fmt.format(q="1080p")]


LAYOUTS = {
    "L0": STD,
    "L1": STD + dc("Heat (1995)/Heat (1995) {{edition-Director's Cut}} - {q}.mkv"),
    "L2": STD + dc("Heat (1995)/Heat (1995) - {{edition-Director's Cut}} {q}.mkv"),
    "L3": STD + dc("Heat (1995)/Heat (1995) - {q} {{edition-Director's Cut}}.mkv"),
    "L4": STD + dc("Heat (1995)/Heat (1995) - Director's Cut {q}.mkv"),
    "L5": STD + dc("Heat (1995) {{edition-Director's Cut}}/Heat (1995) {{edition-Director's Cut}} - {q}.mkv"),
    "L6": STD + dc("Heat (1995)/Heat (1995) {{edition-Director's Cut}}/Heat (1995) {{edition-Director's Cut}} - {q}.mkv"),
    "Sspace": STD + dc("Heat (1995)/Heat (1995) Director's Cut {q}.mkv"),
    "Sdot": STD + dc("Heat (1995)/Heat (1995).Director's Cut {q}.mkv"),
    "Sunder": STD + dc("Heat (1995)/Heat (1995)_Director's Cut {q}.mkv"),
    "Sbracket": STD + dc("Heat (1995)/Heat (1995) [Director's Cut] - {q}.mkv"),
    "Sbrace": STD + dc("Heat (1995)/Heat (1995) {{Director's Cut}} - {q}.mkv"),
    "Sedit_dash": STD + dc("Heat (1995)/Heat (1995) - {{edition-Director's Cut}} - {q}.mkv"),
}


def build(name):
    movies = os.path.join(MEDIA, "movies")
    shutil.rmtree(movies, ignore_errors=True)
    os.makedirs(movies)
    for rel in LAYOUTS[name]:
        p = os.path.join(movies, rel)
        copy(os.path.join(SEED, "2160.mkv" if "4K" in rel else "1080.mkv"), p)
        with open(p[:-4] + ".en.srt", "w") as f:
            f.write("1\n00:00:01,000 --> 00:00:04,000\n%s\n\n" % os.path.basename(p)[:-4])
        print("wrote", rel)


def base(p):
    return os.path.basename((p or "").replace("\\", "/"))


def dump_jellyfin():
    m = MB("jellyfin"); m.login()
    items = m.items(IncludeItemTypes="Movie", Fields="Path,MediaSources,MediaStreams,ProductionYear")
    print(f"JELLYFIN movies={len(items)}")
    for it in items:
        print(f"  item name={it['Name']!r} year={it.get('ProductionYear')} path={base(it.get('Path'))!r}")
        for ms in it.get("MediaSources", []):
            subs = [base(s.get("Path")) for s in ms.get("MediaStreams", []) if s["Type"] == "Subtitle"]
            print(f"     source name={ms.get('Name')!r} file={base(ms.get('Path'))!r} subs={subs}")


def dump_emby():
    m = MB("emby"); m.login()
    c, r, _ = m.call("GET", "/Items", Recursive="true", IncludeItemTypes="Movie", Fields="Path,PresentationUniqueKey,ProductionYear")
    raw = r["Items"]
    users = m.items(IncludeItemTypes="Movie", Fields="Path")
    print(f"EMBY movie items={len(raw)} user-listing={len(users)} distinct PUK={len({i.get('PresentationUniqueKey') for i in raw})}")
    for it in raw:
        d = m.item(it["Id"])
        print(f"  item id={it['Id']} name={it['Name']!r} year={it.get('ProductionYear')} path={base(it.get('Path'))!r} PUK={it.get('PresentationUniqueKey')}")
        for ms in d.get("MediaSources", []):
            subs = [base(s.get("Path")) for s in ms.get("MediaStreams", []) if s["Type"] == "Subtitle"]
            print(f"     source name={ms.get('Name')!r} file={base(ms.get('Path'))!r} subs={subs}")


def stream_text(sid):
    c, body, _ = plex.call("GET", f"/library/streams/{sid}", raw=True)
    lines = [l for l in body.decode(errors="replace").splitlines() if l and "-->" not in l and not l.strip().isdigit()]
    return lines[0] if lines else f"<{c}>"


def dump_plex():
    sid = plex.sections()["Movies"]["key"]
    movies = plex.leaves(sid, 1)
    print(f"PLEX movies={len(movies)}")
    for lm in movies:
        mt = plex.meta(lm["ratingKey"])
        print(f"  item rk={mt['ratingKey']} title={mt.get('title')!r} year={mt.get('year')} editionTitle={mt.get('editionTitle')!r} guid={mt.get('guid')}")
        for md in mt.get("Media", []):
            for p in md.get("Part", []):
                subs = [stream_text(s["id"]) for s in p.get("Stream", []) if s.get("streamType") == 3]
                print(f"     media id={md['id']} res={md.get('videoResolution')} file={base(p['file'])!r} subs={subs}")


if __name__ == "__main__":
    cmd, name = sys.argv[1], sys.argv[2]
    if cmd == "build":
        build(name)
    else:
        print(f"### layout {name}: " + " | ".join(LAYOUTS[name]))
        for f in (dump_jellyfin, dump_emby):
            try:
                f()
            except Exception as e:
                print(f.__name__, "FAILED", repr(e))
