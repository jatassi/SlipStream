#!/usr/bin/env python3
"""Read-only inventory of /Volumes/Media Downloads and Plex. Uses os.stat only, never opens files."""
import collections
import datetime
import json
import os
import re
import sys

MEDIA = "/Volumes/Media"
OUT = os.path.dirname(os.path.abspath(__file__))
VIDEO_EXT = {".mkv", ".mp4", ".avi", ".m4v", ".ts", ".mov", ".wmv"}
CLIENTS = [
    ("radarr", os.path.join(MEDIA, "Downloads/radarr"), "movie"),
    ("sonarr", os.path.join(MEDIA, "Downloads/sonarr"), "series"),
    ("SlipStream-Movies", os.path.join(MEDIA, "Downloads/SlipStream/Movies"), "movie"),
    ("SlipStream-Series", os.path.join(MEDIA, "Downloads/SlipStream/Series"), "series"),
]
PLEX_MOVIES = os.path.join(MEDIA, "Plex/Movies")
PLEX_SHOWS = os.path.join(MEDIA, "Plex/Shows")

YEAR_RE = re.compile(r"^(?:19|20)\d{2}$")
RES_RE = re.compile(r"^\d{3,4}[pi]$", re.I)
EP_TOKEN_RE = re.compile(r"^S(\d{1,2})((?:E\d{1,3})+)(?:-E?(\d{1,3}))?", re.I)
SEASON_TOKEN_RE = re.compile(r"^S(\d{1,2})(?:-S(\d{1,2}))?(?:-\d{3,4}p)?$", re.I)
SOURCE_WORDS = {
    "uhd", "bluray", "blu-ray", "web", "web-dl", "webrip", "remux", "hdtv", "hybrid",
    "repack", "proper", "extended", "theatrical", "unrated", "remastered", "complete",
    "series", "webscreener", "dual", "yg",
}


def iso(ts):
    return datetime.datetime.fromtimestamp(ts, datetime.timezone.utc).strftime("%Y-%m-%dT%H:%M:%S.%fZ")


def compact(title):
    t = title.lower().replace("&", "and")
    t = t.replace("'", "").replace("’", "")
    return re.sub(r"[^a-z0-9]+", "", t)


def join_title(tokens):
    out = []
    i = 0
    while i < len(tokens):
        j = i
        while j < len(tokens) and re.fullmatch(r"[A-Z]", tokens[j]):
            j += 1
        if j - i >= 2:
            out.append(".".join(tokens[i:j]) + ".")
            i = j
        else:
            out.append(tokens[i])
            i += 1
    return " ".join(out)


def parse_release(name):
    s = re.sub(r"[\[\]()]", " ", name)
    tokens = [t for t in re.split(r"[\s.]+", s) if t]
    marker = None
    season_info = None
    for i, t in enumerate(tokens):
        m = EP_TOKEN_RE.match(t)
        if m:
            marker = i
            eps = re.findall(r"E(\d{1,3})", m.group(2), re.I)
            nums = [int(x) for x in eps]
            if m.group(3):
                nums = list(range(nums[-1], int(m.group(3)) + 1))
            season_info = ("episode", [int(m.group(1))], nums)
            break
        m = SEASON_TOKEN_RE.match(t)
        if m:
            marker = i
            a = int(m.group(1))
            b = int(m.group(2)) if m.group(2) else a
            seasons = list(range(a, b + 1))
            season_info = ("series-pack" if len(seasons) > 1 else "season-pack", seasons, [])
            break
        if t.lower() == "season" and i + 1 < len(tokens) and tokens[i + 1].isdigit():
            marker = i
            season_info = ("season-pack", [int(tokens[i + 1])], [])
            break
        if RES_RE.match(t):
            marker = i
            break
    head = tokens[:marker] if marker is not None else tokens
    year = None
    year_idx = None
    for i, t in enumerate(head):
        if YEAR_RE.match(t) and i > 0:
            year, year_idx = int(t), i
    if year_idx is not None:
        title_tokens = head[:year_idx]
    else:
        title_tokens = list(head)
        if marker is None or season_info is None:
            while title_tokens and title_tokens[-1].lower() in SOURCE_WORDS and len(title_tokens) > 1:
                title_tokens.pop()
        if marker is not None and season_info is not None:
            for t in tokens[marker + 1:]:
                if YEAR_RE.match(t):
                    year = int(t)
                    break
                if RES_RE.match(t):
                    break
    if season_info:
        kind, seasons, eps = season_info
    else:
        kind, seasons, eps = "movie", [], []
    return {
        "title": join_title(title_tokens),
        "year": year,
        "kind": kind,
        "seasons": seasons,
        "name_episodes": eps,
    }


def parse_group(name):
    if "-" not in name:
        return None
    cand = name.rsplit("-", 1)[1].strip().rstrip(")").strip(" .")
    if not cand:
        return None
    cand = cand.split()[-1].strip(" .)")
    if not cand or "." in cand:
        return None
    return cand


EP_IN_FILE_RE = re.compile(r"S(\d{1,3})((?:E\d{1,3})+)(?:-E?(\d{1,3}))?", re.I)


def episodes_from_name(fname):
    m = EP_IN_FILE_RE.search(fname)
    if not m:
        return []
    s = int(m.group(1))
    nums = [int(x) for x in re.findall(r"E(\d{1,3})", m.group(2), re.I)]
    if m.group(3):
        nums = list(range(nums[-1], int(m.group(3)) + 1))
    return ["S%02dE%02d" % (s, n) for n in nums]


def stat_file(path):
    st = os.stat(path)
    return st.st_size, st.st_mtime


def is_sample(rel_parts, fname):
    if any(p.lower() in ("sample", "samples") for p in rel_parts[:-1]):
        return True
    return "sample" in fname.lower()


def build_releases():
    releases = []
    for client, base, ctype in CLIENTS:
        for entry in sorted(os.listdir(base)):
            if entry == ".DS_Store":
                continue
            top = os.path.join(base, entry)
            if os.path.isdir(top):
                release_name = entry
                video_paths, incomplete, other = [], [], 0
                for root, dirs, files in os.walk(top):
                    dirs.sort()
                    for f in sorted(files):
                        if f == ".DS_Store":
                            continue
                        p = os.path.join(root, f)
                        ext = os.path.splitext(f)[1].lower()
                        if ext in VIDEO_EXT:
                            video_paths.append(p)
                        elif f.lower().endswith(tuple(e + ".part" for e in VIDEO_EXT)):
                            incomplete.append(p)
                        else:
                            other += 1
                is_dir = True
            else:
                ext = os.path.splitext(entry)[1]
                if ext.lower() in VIDEO_EXT:
                    release_name = entry[: -len(ext)]
                    video_paths, incomplete, other = [top], [], 0
                elif entry.lower().endswith(tuple(e + ".part" for e in VIDEO_EXT)):
                    release_name = entry
                    video_paths, incomplete, other = [], [top], 0
                else:
                    release_name = entry
                    video_paths, incomplete, other = [], [], 1
                is_dir = False
            parsed = parse_release(release_name)
            files = []
            for p in video_paths:
                size, mt = stat_file(p)
                rel_parts = os.path.relpath(p, top if is_dir else os.path.dirname(top)).split(os.sep)
                fname = rel_parts[-1]
                eps = episodes_from_name(fname)
                files.append({
                    "path": p,
                    "size_bytes": size,
                    "mtime": iso(mt),
                    "mtime_epoch": mt,
                    "episode": eps[0] if eps else None,
                    "episodes": eps if len(eps) > 1 else None,
                    "is_sample": is_sample(rel_parts, fname),
                })
            real = [f for f in files if not f["is_sample"]]
            kind = parsed["kind"]
            if len(parsed["seasons"]) == 1:
                season = parsed["seasons"][0]
            else:
                season = None
            if kind == "episode" and len(real) == 1 and not real[0]["episode"]:
                real[0]["episode"] = "S%02dE%02d" % (season, parsed["name_episodes"][0]) if parsed["name_episodes"] else None
            ep_ids = []
            for f in real:
                for e in (f["episodes"] or ([f["episode"]] if f["episode"] else [])):
                    if e not in ep_ids:
                        ep_ids.append(e)
            for n in parsed["name_episodes"]:
                if season is not None:
                    e = "S%02dE%02d" % (season, n)
                    if e not in ep_ids:
                        ep_ids.append(e)
            ep_ids.sort()
            if real:
                earliest = min(f["mtime_epoch"] for f in real)
            else:
                earliest = os.stat(top).st_mtime
            releases.append({
                "release_name": release_name,
                "client_dir": client,
                "entry_type": "folder" if is_dir else "file",
                "entry_name": entry,
                "kind": kind,
                "title": parsed["title"],
                "year": parsed["year"],
                "season": season,
                "seasons": parsed["seasons"],
                "episodes": ep_ids,
                "group": parse_group(release_name),
                "files": files,
                "incomplete_files": incomplete,
                "non_video_file_count": other,
                "total_size_bytes": sum(f["size_bytes"] for f in real),
                "earliest_mtime": iso(earliest),
                "earliest_mtime_epoch": earliest,
                "client_expects": ctype,
            })
    return releases


def parse_folder(folder):
    m = re.match(r"^(.*) \((\d{4})\)$", folder)
    if m:
        return m.group(1), int(m.group(2))
    return folder, None


def build_library():
    lib = []
    empty_dirs = []
    for root, dirs, files in os.walk(PLEX_MOVIES):
        dirs.sort()
        rel = os.path.relpath(root, PLEX_MOVIES)
        if rel != "." and not dirs and not files:
            empty_dirs.append(root)
        if rel == ".":
            continue
        folder = rel.split(os.sep)[0]
        for f in sorted(files):
            if os.path.splitext(f)[1].lower() not in VIDEO_EXT:
                continue
            p = os.path.join(root, f)
            size, mt = stat_file(p)
            title, year = parse_folder(folder)
            base = os.path.splitext(f)[0]
            suffix, prefix_ok = None, True
            if base.startswith(folder):
                rest = base[len(folder):]
                if rest.startswith(" - "):
                    suffix = rest[3:]
                elif rest:
                    suffix = rest
            else:
                prefix_ok = False
            lib.append({
                "path": p, "size_bytes": size, "mtime": iso(mt), "mtime_epoch": mt,
                "content_type": "movie", "folder": folder, "title": title, "year": year,
                "season": None, "episode": None, "episodes": None,
                "plex_quality_suffix": suffix, "filename_matches_folder": prefix_ok,
            })
    for root, dirs, files in os.walk(PLEX_SHOWS):
        dirs.sort()
        rel = os.path.relpath(root, PLEX_SHOWS)
        if rel != "." and not dirs and not files:
            empty_dirs.append(root)
        if rel == ".":
            continue
        parts = rel.split(os.sep)
        folder = parts[0]
        season_dir = parts[1] if len(parts) > 1 else None
        for f in sorted(files):
            if os.path.splitext(f)[1].lower() not in VIDEO_EXT:
                continue
            p = os.path.join(root, f)
            size, mt = stat_file(p)
            title, year = parse_folder(folder)
            eps = episodes_from_name(f)
            m = EP_IN_FILE_RE.search(f)
            file_title = f[: m.start()].rstrip(" -") if m else None
            suffix = None
            if m:
                tail = os.path.splitext(f[m.end():])[0]
                if tail.startswith(" - "):
                    suffix = tail[3:]
                elif tail:
                    suffix = tail
            lib.append({
                "path": p, "size_bytes": size, "mtime": iso(mt), "mtime_epoch": mt,
                "content_type": "series", "folder": folder, "title": title, "year": year,
                "file_title": file_title, "season_dir": season_dir,
                "season": int(m.group(1)) if m else None,
                "episode": eps[0] if eps else None,
                "episodes": eps if len(eps) > 1 else None,
                "plex_quality_suffix": suffix, "filename_matches_folder": True,
            })
    return lib, empty_dirs


def title_agrees(a, b):
    ka, kb = compact(a), compact(b)
    if not ka or not kb:
        return False
    if ka == kb:
        return True
    short, long_ = (ka, kb) if len(ka) <= len(kb) else (kb, ka)
    return len(short) >= 4 and (long_.endswith(short) or long_.startswith(short))


def match_library(lib, releases):
    by_size = collections.defaultdict(list)
    for r in releases:
        for f in r["files"]:
            if f["is_sample"]:
                continue
            by_size[f["size_bytes"]].append((r, f))
    for lf in lib:
        cands = by_size.get(lf["size_bytes"], [])
        lf["matched_release"] = None
        lf["matched_download_path"] = None
        lf["match_title_agrees"] = None
        lf["match_year_agrees"] = None
        lf["match_ambiguous"] = False
        lf["match_candidates"] = None
        if not cands:
            continue

        def agree(c):
            r, f = c
            if not title_agrees(lf["title"], r["title"]):
                return False
            if lf["content_type"] == "series" and lf["episode"]:
                if f["episode"] and f["episode"] not in (f["episodes"] or [f["episode"]]):
                    return False
            return True

        good = [c for c in cands if agree(c)]
        pool = good if good else cands
        pool = sorted(pool, key=lambda c: (c[1]["mtime_epoch"], c[1]["path"]))
        r, f = pool[0]
        lf["matched_release"] = r["release_name"]
        lf["matched_download_path"] = f["path"]
        lf["match_title_agrees"] = bool(good)
        if lf["year"] and r["year"]:
            lf["match_year_agrees"] = lf["year"] == r["year"]
        if len(pool) > 1:
            lf["match_ambiguous"] = True
        if len(cands) > 1:
            lf["match_candidates"] = [{"release_name": c[0]["release_name"], "path": c[1]["path"],
                                       "title_agrees": c in good} for c in cands]


def build_works(lib, releases):
    works = {}

    def key_for(ctype, title, year):
        k = compact(title)
        return (ctype, k, year) if ctype == "movie" else (ctype, k)

    def get(ctype, title, year):
        k = key_for(ctype, title, year)
        if k not in works:
            works[k] = {
                "content_type": ctype, "title": title, "year": year, "titles_seen": set(),
                "years_seen": set(), "library_files": 0, "library_bytes": 0, "library_folders": set(),
                "releases_by_name": set(), "releases_by_size": set(), "episodes_present": set(),
                "episodes_in_downloads": set(), "_ep_counts": collections.Counter(),
            }
        w = works[k]
        w["titles_seen"].add(title)
        if year:
            w["years_seen"].add(year)
        return w

    lib_work = {}
    for lf in lib:
        w = get(lf["content_type"], lf["title"], lf["year"])
        lib_work[lf["path"]] = w
        w["library_files"] += 1
        w["library_bytes"] += lf["size_bytes"]
        w["library_folders"].add(lf["folder"])
        if lf["content_type"] == "series":
            for e in (lf["episodes"] or ([lf["episode"]] if lf["episode"] else [])):
                w["episodes_present"].add(e)
                w["_ep_counts"][e] += 1
        else:
            w["_ep_counts"]["movie"] += 1
        if lf["matched_release"]:
            w["releases_by_size"].add(lf["matched_release"])
    for r in releases:
        ctype = "movie" if r["kind"] == "movie" else "series"
        w = get(ctype, r["title"], r["year"])
        w["releases_by_name"].add(r["release_name"])
        for e in r["episodes"]:
            w["episodes_in_downloads"].add(e)

    NUM = {"one": "1", "two": "2", "three": "3", "four": "4", "five": "5", "six": "6", "seven": "7", "eight": "8", "nine": "9", "ten": "10"}

    def alias_key(title):
        words = re.sub(r"[^a-z0-9 ]+", " ", title.lower().replace("'", "").replace("&", "and")).split()
        return "".join(NUM.get(x, x) for x in words)

    for w in works.values():
        w["releases_by_alias"] = set()
        w["aliases"] = set()
    lib_movies = [(k, w) for k, w in works.items() if k[0] == "movie" and w["library_files"]]
    for k, w in list(works.items()):
        if k[0] != "movie" or w["library_files"]:
            continue
        ak = alias_key(w["title"])
        hits = []
        for k2, w2 in lib_movies:
            bk = alias_key(w2["title"])
            short, long_ = (ak, bk) if len(ak) <= len(bk) else (bk, ak)
            if len(short) >= 4 and (long_.endswith(short) or long_.startswith(short)) and k[2] and k2[2] and abs(k[2] - k2[2]) <= 1:
                hits.append((k2, w2))
        if len(hits) == 1:
            k2, w2 = hits[0]
            w2["releases_by_alias"] |= w["releases_by_name"]
            w2["aliases"].add(w["title"])
            if w["year"]:
                w2["years_seen"].add(w["year"])
            del works[k]

    out = []
    movie_years = collections.defaultdict(set)
    for k, w in works.items():
        if k[0] == "movie":
            movie_years[k[1]].add(k[2])
    for k, w in sorted(works.items(), key=lambda kv: (kv[0][0], kv[0][1], kv[0][2] if len(kv[0]) > 2 else 0)):
        titles = sorted(w["titles_seen"])
        years = sorted(w["years_seen"])
        item = {
            "content_type": w["content_type"],
            "work_key": "|".join(str(x) for x in k),
            "title": titles[0],
            "year": (w["year"] if w["content_type"] == "movie" else (years[0] if years else None)),
            "title_spellings": titles,
            "years_seen": years,
            "library_files": w["library_files"],
            "library_bytes": w["library_bytes"],
            "library_folders": sorted(w["library_folders"]),
            "duplicate_folder_variants": sorted(w["library_folders"]) if len(w["library_folders"]) > 1 else [],
            "releases": sorted(w["releases_by_name"] | w["releases_by_size"] | w["releases_by_alias"]),
            "releases_by_alias": sorted(w["releases_by_alias"] - w["releases_by_name"] - w["releases_by_size"]),
            "alias_titles_merged": sorted(w["aliases"]),
            "releases_by_name": sorted(w["releases_by_name"]),
            "releases_by_size_only": sorted(w["releases_by_size"] - w["releases_by_name"] - w["releases_by_alias"]),
        }
        if w["content_type"] == "series":
            item["episodes_present"] = sorted(w["episodes_present"])
            item["episodes_in_downloads"] = sorted(w["episodes_in_downloads"])
        else:
            others = sorted(y for y in movie_years[k[1]] if y != k[2] and y)
            item["same_title_other_years"] = others
        item["multi_version_items"] = sorted(e for e, c in w["_ep_counts"].items() if c > 1)
        out.append(item)
        k_ = None
    return out


def main():
    releases = build_releases()
    lib, empty_dirs = build_library()
    match_library(lib, releases)
    works = build_works(lib, releases)

    # strip internal fields
    for r in releases:
        r["files"] = [{k: v for k, v in f.items()} for f in r["files"]]

    # multi-version
    ep_files = collections.defaultdict(list)
    for lf in lib:
        if lf["content_type"] == "movie":
            ep_files[("movie", compact(lf["title"]), lf["year"])].append(lf)
        else:
            for e in (lf["episodes"] or ([lf["episode"]] if lf["episode"] else [None])):
                ep_files[("episode", compact(lf["title"]), e)].append(lf)
    multi = {k: v for k, v in ep_files.items() if len(v) > 1}
    multi_same_folder = {k: v for k, v in multi.items() if len({x["folder"] for x in v}) == 1}
    multi_items = []
    for k, v in sorted(multi.items(), key=lambda kv: (kv[0][0], str(kv[0][1:]))):
        multi_items.append({
            "kind": k[0], "title": v[0]["title"], "year": v[0]["year"] if k[0] == "movie" else None,
            "episode": k[2] if k[0] == "episode" else None,
            "same_folder": k in multi_same_folder,
            "files": [{"path": x["path"], "size_bytes": x["size_bytes"], "plex_quality_suffix": x["plex_quality_suffix"],
                       "matched_release": x["matched_release"]} for x in v],
        })

    group_counter = collections.Counter(r["group"] or "(none)" for r in releases)
    matched = [x for x in lib if x["matched_release"]]
    sizes_dl = collections.Counter(f["size_bytes"] for r in releases for f in r["files"] if not f["is_sample"])
    stats = {
        "releases": len(releases),
        "releases_by_client_dir": dict(collections.Counter(r["client_dir"] for r in releases)),
        "releases_by_kind": dict(collections.Counter(r["kind"] for r in releases)),
        "releases_kind_vs_client_mismatch": [r["release_name"] for r in releases
                                             if (r["kind"] == "movie") != (r["client_expects"] == "movie")],
        "release_video_files_non_sample": sum(1 for r in releases for f in r["files"] if not f["is_sample"]),
        "release_sample_files": sum(1 for r in releases for f in r["files"] if f["is_sample"]),
        "releases_with_no_video_files": [r["release_name"] for r in releases if not [f for f in r["files"] if not f["is_sample"]]],
        "releases_with_incomplete_files": [r["release_name"] for r in releases if r["incomplete_files"]],
        "download_size_collisions_sizes": sum(1 for s, c in sizes_dl.items() if c > 1),
        "releases_without_parsed_year": sum(1 for r in releases if r["year"] is None),
        "releases_without_parsed_group": [r["release_name"] for r in releases if r["group"] is None],
        "library_files": len(lib),
        "library_movie_files": sum(1 for x in lib if x["content_type"] == "movie"),
        "library_series_files": sum(1 for x in lib if x["content_type"] == "series"),
        "library_matched": len(matched),
        "library_unmatched": len(lib) - len(matched),
        "library_matched_by_type": dict(collections.Counter(x["content_type"] for x in matched)),
        "library_unmatched_by_type": dict(collections.Counter(x["content_type"] for x in lib if not x["matched_release"])),
        "matched_title_disagrees": sum(1 for x in matched if not x["match_title_agrees"]),
        "matched_ambiguous": sum(1 for x in matched if x["match_ambiguous"]),
        "works": len(works),
        "works_by_type": dict(collections.Counter(w["content_type"] for w in works)),
        "works_with_library_files": sum(1 for w in works if w["library_files"]),
        "works_download_only": sum(1 for w in works if not w["library_files"]),
        "works_with_duplicate_folder_variants": sum(1 for w in works if w["duplicate_folder_variants"]),
        "multi_version_items": len(multi),
        "multi_version_movies": sum(1 for k in multi if k[0] == "movie"),
        "multi_version_episodes": sum(1 for k in multi if k[0] == "episode"),
        "multi_version_same_folder": len(multi_same_folder),
        "multi_version_across_folder_variants": len(multi) - len(multi_same_folder),
        "library_empty_folders": len(empty_dirs),
        "top_release_groups": [{"group": g, "count": c} for g, c in group_counter.most_common(40)],
        "distinct_release_groups": len(group_counter),
    }

    meta = {"generated_at": iso(datetime.datetime.now().timestamp()), "media_root": MEDIA,
            "notes": [
                "release totals and earliest_mtime exclude sample files; sample files stay in files[] with is_sample=true",
                "mtime is UTC ISO8601; mtime_epoch is seconds since epoch",
                "release title is the parsed title tokens joined by spaces; dots in titles are lost",
                "library episode is the first episode of a multi-episode file; episodes lists all",
            ]}
    with open(os.path.join(OUT, "inventory.json"), "w") as fh:
        json.dump({"meta": meta, "releases": releases, "library": lib, "works": works,
                   "library_empty_folders": empty_dirs, "multi_version_items": multi_items}, fh, indent=1, ensure_ascii=False)
    with open(os.path.join(OUT, "stats.json"), "w") as fh:
        json.dump(stats, fh, indent=1, ensure_ascii=False)
    print(json.dumps({k: v for k, v in stats.items() if k != "top_release_groups"}, indent=1)[:6000])


if __name__ == "__main__":
    main()
