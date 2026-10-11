import sys, json
from mb import MB
m = MB(sys.argv[1]); m.login()
for it in m.items(IncludeItemTypes="Movie,Episode"):
    print(it["Type"], it["Name"], it.get("Id"), "path:", it.get("Path"))
    for ms in it.get("MediaSources", []):
        subs = [(s.get("Index"), s.get("Language"), s.get("IsExternal"), s.get("Path")) for s in ms.get("MediaStreams", []) if s["Type"] == "Subtitle"]
        print("   MS id=%s name=%r path=%s subs=%s" % (ms.get("Id"), ms.get("Name"), ms.get("Path"), subs))
