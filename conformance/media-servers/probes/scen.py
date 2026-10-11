"""Scenario helpers shared across servers. Host path <-> container path: media/ <-> /media/."""
import os, time, json
from lib import MEDIA, SEED, copy, poll, Log, trim
import plex
from mb import MB

def host(p):
    return os.path.join(MEDIA, p)

def cont(p):
    return "/media/" + p


class PlexS:
    kind = "plex"
    def __init__(self):
        s = plex.sections()
        self.sid = {"movies": s["Movies"]["key"], "tv": s["Shows"]["key"]}

    def sec(self, rel):
        return self.sid[rel.split("/")[0]]

    def scan_folder(self, rel_file):
        folder = os.path.dirname(cont(rel_file))
        c, _, ms = plex.call("GET", f"/library/sections/{self.sec(rel_file)}/refresh", path=folder)
        return c, ms, f"GET /library/sections/{self.sec(rel_file)}/refresh?path={folder}"

    def refresh_item(self, rk):
        c, _, ms = plex.call("PUT", f"/library/metadata/{rk}/refresh")
        return c, ms, f"PUT /library/metadata/{rk}/refresh"

    def find_part(self, rel_file):
        typ = 1 if rel_file.startswith("movies") else 4
        f = cont(rel_file)
        for m in plex.leaves(self.sec(rel_file), typ):
            for md in m.get("Media", []):
                for p in md.get("Part", []):
                    if p["file"] == f:
                        return {"ratingKey": m["ratingKey"], "media": md["id"], "part": p["id"], "title": m.get("title")}
        return None

    def part_detail(self, rel_file):
        hit = self.find_part(rel_file)
        if not hit:
            return None, None
        m = plex.meta(hit["ratingKey"])
        for md in m.get("Media", []):
            for p in md.get("Part", []):
                if p["file"] == cont(rel_file):
                    return m, p
        return m, None

    def subs(self, rel_file):
        m, p = self.part_detail(rel_file)
        if not p:
            return None
        return [{"id": s["id"], "lang": s.get("languageCode"), "key": s.get("key"), "codec": s.get("codec"), "format": s.get("format")} for s in p.get("Stream", []) if s.get("streamType") == 3]


class MBS:
    def __init__(self, kind):
        self.kind = kind
        self.m = MB(kind); self.m.login()

    def report(self, rel, update):
        c, r, ms = self.m.media_updated(cont(rel), update)
        return c, ms, f'POST /Library/Media/Updated {{"Updates":[{{"Path":"{cont(rel)}","UpdateType":"{update}"}}]}}'

    def find(self, rel_file):
        """Item id whose detail MediaSources include this file (alternate versions have no list entry on Jellyfin)."""
        f = cont(rel_file)
        c, r, _ = self.m.call("GET", "/Items", Recursive="true", IncludeItemTypes="Movie,Episode", Fields="Path")
        if not isinstance(r, dict):
            return None
        for i in r.get("Items", []):
            if i.get("Path") == f:
                return i["Id"]
        d = os.path.dirname(f)
        for i in r.get("Items", []):
            if os.path.dirname(i.get("Path") or "") == d:
                it = self.m.item(i["Id"]) or {}
                if any(s.get("Path") == f for s in it.get("MediaSources", [])):
                    return i["Id"]
        return None

    def source(self, rel_file):
        """The MediaSource for this file, as seen from item detail (merged versions)."""
        iid = self.find(rel_file)
        if not iid:
            return None, None
        it = self.m.item(iid) or {}
        for s in it.get("MediaSources", []):
            if s.get("Path") == cont(rel_file):
                return it, s
        return it, None

    def subs(self, rel_file):
        m, p = self.part_detail(rel_file)
        if not p:
            return None
        return [{"id": s["id"], "lang": s.get("languageCode"), "key": s.get("key"), "codec": s.get("codec"), "format": s.get("format")} for s in p.get("Stream", []) if s.get("streamType") == 3]


class MBS:
    def __init__(self, kind):
        self.kind = kind
        self.m = MB(kind); self.m.login()

    def report(self, rel, update):
        c, r, ms = self.m.media_updated(cont(rel), update)
        return c, ms, f'POST /Library/Media/Updated {{"Updates":[{{"Path":"{cont(rel)}","UpdateType":"{update}"}}]}}'

    def find(self, rel_file):
        f = cont(rel_file)
        c, r, _ = self.m.call("GET", "/Items", Recursive="true", IncludeItemTypes="Movie,Episode", Fields="Path")
        if not isinstance(r, dict):
            return None
        for i in r.get("Items", []):
            if i.get("Path") == f:
                return i["Id"]
        return None

    def source(self, rel_file):
        """The MediaSource for this file, as seen from item detail (merged versions)."""
        iid = self.find(rel_file)
        if not iid:
            return None, None
        it = self.m.item(iid)
        for s in it.get("MediaSources", []):
            if s.get("Path") == cont(rel_file):
                return it, s
        return it, None

    def subs(self, rel_file):
        it, s = self.source(rel_file)
        if not s:
            return None
        return [{"index": x.get("Index"), "lang": x.get("Language"), "external": x.get("IsExternal"), "path": x.get("Path"), "deliveryUrl": x.get("DeliveryUrl"), "codec": x.get("Codec")}
                for x in s.get("MediaStreams", []) if x["Type"] == "Subtitle"]

    def all_subs(self):
        """Every (version path -> external subtitle paths) across all versions of every item."""
        out = {}
        c, r, _ = self.m.call("GET", "/Items", Recursive="true", IncludeItemTypes="Movie,Episode", Fields="Path")
        for i in r.get("Items", []):
            it = self.m.item(i["Id"])
            for s in it.get("MediaSources", []):
                out[s.get("Path")] = sorted(x.get("Path") or f"embedded:{x.get('Index')}" for x in s.get("MediaStreams", []) if x["Type"] == "Subtitle")
        return out


def servers():
    return [PlexS(), MBS("jellyfin"), MBS("emby")]
