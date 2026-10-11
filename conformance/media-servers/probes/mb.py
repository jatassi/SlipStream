"""Jellyfin and Emby client (shared MediaBrowser lineage)."""
import json, os, sys, time, urllib.parse
from lib import req, creds

SERVERS = {"jellyfin": "http://localhost:8096", "emby": "http://localhost:8097"}


class MB:
    def __init__(self, kind):
        self.kind = kind
        self.base = SERVERS[kind]
        self.token = None
        self.uid = None
        self.devid = "conformance-probe-%d" % os.getpid()

    def auth_header(self):
        v = f'MediaBrowser Client="conformance", Device="probe", DeviceId="{self.devid}", Version="1.0"'
        if self.token:
            v += f', Token="{self.token}"'
        h = {"Authorization": v, "X-Emby-Authorization": v}
        if self.token:
            h["X-Emby-Token"] = self.token
        return h

    def call(self, method, path, body=None, raw=False, **q):
        url = self.base + path
        if q:
            url += ("&" if "?" in url else "?") + urllib.parse.urlencode(q)
        return req(method, url, body, self.auth_header(), raw=raw)

    def setup(self):
        user, pw = creds()
        c, info, _ = self.call("GET", "/System/Info/Public")
        print(self.kind, "public info", c, info.get("Version"), "wizard done:", info.get("StartupWizardCompleted"))
        if not info.get("StartupWizardCompleted"):
            print(self.call("POST", "/Startup/Configuration", {"UICulture": "en-US", "MetadataCountryCode": "US", "PreferredMetadataLanguage": "en"})[:2])
            print(self.call("GET", "/Startup/User")[:2])
            print(self.call("POST", "/Startup/User", {"Name": user, "Password": pw})[:2])
            print(self.call("POST", "/Startup/RemoteAccess", {"EnableRemoteAccess": True, "EnableAutomaticPortMapping": False})[:2])
            print(self.call("POST", "/Startup/Complete")[:2])
        self.login()

    def login(self):
        user, pw = creds()
        c, r, _ = self.call("POST", "/Users/AuthenticateByName", {"Username": user, "Pw": pw, "Password": pw})
        assert c == 200, (c, r)
        self.token = r["AccessToken"]
        self.uid = r["User"]["Id"]

    def items(self, **q):
        q.setdefault("Recursive", "true")
        q.setdefault("Fields", "Path,MediaSources,MediaStreams")
        c, r, _ = self.call("GET", f"/Users/{self.uid}/Items", **q)
        return r.get("Items", []) if c == 200 else []

    def item(self, iid):
        c, r, _ = self.call("GET", f"/Users/{self.uid}/Items/{iid}")
        return r if c == 200 else None

    def media_updated(self, path, update):
        return self.call("POST", "/Library/Media/Updated", {"Updates": [{"Path": path, "UpdateType": update}]})


if __name__ == "__main__":
    m = MB(sys.argv[1])
    m.setup()
    print("token ok", bool(m.token))


def lib_options(kind, path, ctype):
    t = "Movie" if ctype == "movies" else "Series"
    o = {
        "EnableRealtimeMonitor": False,
        "EnablePhotos": False,
        "EnableChapterImageExtraction": False,
        "ExtractChapterImagesDuringLibraryScan": False,
        "EnableTrickplayImageExtraction": False,
        "ExtractTrickplayImagesDuringLibraryScan": False,
        "EnableEmbeddedTitles": False,
        "EnableInternetProviders": False,
        "SaveLocalMetadata": False,
        "PathInfos": [{"Path": path}],
        "TypeOptions": [{"Type": t, "MetadataFetchers": [], "ImageFetchers": []}]
        + ([{"Type": "Season", "MetadataFetchers": [], "ImageFetchers": []}, {"Type": "Episode", "MetadataFetchers": [], "ImageFetchers": []}] if t == "Series" else []),
    }
    return o


def setup_libs(m):
    c, vf, _ = m.call("GET", "/Library/VirtualFolders")
    have = {v["Name"] for v in vf}
    for name, ctype, path in [("Movies", "movies", "/media/movies"), ("Shows", "tvshows", "/media/tv")]:
        if name in have:
            continue
        r = m.call("POST", "/Library/VirtualFolders", {"LibraryOptions": lib_options(m.kind, path, ctype)},
                   name=name, collectionType=ctype, refreshLibrary="false", paths=path)
        print("create", name, r[:2])
    c, vf, _ = m.call("GET", "/Library/VirtualFolders")
    for v in vf:
        lo = v.get("LibraryOptions", {})
        print(v["Name"], v.get("Locations"), "realtime:", lo.get("EnableRealtimeMonitor"), "id:", v.get("ItemId"))
    c, tasks, _ = m.call("GET", "/ScheduledTasks")
    for t in tasks:
        if t.get("Key") == "RefreshLibrary" or t.get("Name") in ("Scan media library", "Scan Media Library"):
            print("task", t["Name"], t["Id"], "triggers before:", t.get("Triggers"))
            print("clear triggers", m.call("POST", f"/ScheduledTasks/{t['Id']}/Triggers", [])[:2])
            c, t2, _ = m.call("GET", f"/ScheduledTasks/{t['Id']}")
            print("triggers after:", t2.get("Triggers"))
            return t["Id"]


def scan_running(m, tid):
    c, t, _ = m.call("GET", f"/ScheduledTasks/{tid}")
    return t.get("State") == "Running"
