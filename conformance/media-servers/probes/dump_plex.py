from plex import *
for name, typ in [("Movies", 1), ("Shows", 4)]:
    sid = sections()[name]["key"]
    for m in leaves(sid, typ):
        print(describe(meta(m["ratingKey"])))
