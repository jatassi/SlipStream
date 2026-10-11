"""Restore the base library layout (Heat x3, The Wire S01E01 x2 + E02, Collateral x2 grouping control). Idempotent."""
import os, shutil
from lib import MEDIA, SEED, copy
BASE = {
    "movies/Heat (1995)/Heat (1995) - 1080p.mkv": "1080.mkv",
    "movies/Heat (1995)/Heat (1995) - 4K HDR.mkv": "2160.mkv",
    "movies/Heat (1995)/Heat (1995) {edition-Director's Cut} - 4K HDR.mkv": "2160.mkv",
    "movies/Collateral (2004)/Collateral (2004) - 1080p.mkv": "1080.mkv",
    "movies/Collateral (2004)/Collateral (2004) - 4K HDR.mkv": "2160.mkv",
    "tv/The Wire/Season 01/The Wire - S01E01 - 1080p.mkv": "1080.mkv",
    "tv/The Wire/Season 01/The Wire - S01E01 - Extended 1080p.mkv": "1080.mkv",
    "tv/The Wire/Season 01/The Wire - S01E02 - 720p.mkv": "720.mkv",
}
want = {os.path.join(MEDIA, k) for k in BASE}
for root, dirs, files in os.walk(MEDIA, topdown=False):
    for f in files:
        p = os.path.join(root, f)
        if p not in want:
            os.remove(p); print("removed", p)
    if root != MEDIA and not os.listdir(root):
        os.rmdir(root); print("rmdir", root)
for rel, seed in BASE.items():
    p = os.path.join(MEDIA, rel)
    if not os.path.exists(p):
        copy(os.path.join(SEED, seed), p); print("restored", rel)
for top in ("movies", "tv"):
    os.makedirs(os.path.join(MEDIA, top), exist_ok=True)
