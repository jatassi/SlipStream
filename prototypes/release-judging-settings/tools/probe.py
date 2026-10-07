#!/usr/bin/env python3
"""Resumable MediaInfo probe. Read-only against the media share.

usage: probe.py [--workers N] [--parse-speed X] [--limit N] [--retry-errors] [--timeout S]
"""
import argparse, json, os, re, subprocess, sys, time
from concurrent.futures import ThreadPoolExecutor, as_completed
from datetime import datetime, timezone

HERE = os.path.dirname(os.path.abspath(__file__))
OUT = os.path.join(HERE, "probe.jsonl")
LIST = os.path.join(HERE, "files.txt")
MEDIAINFO = "/opt/homebrew/bin/mediainfo"


def num(v, cast=float):
    if v is None or v == "":
        return None
    try:
        return cast(float(v)) if cast is int else float(v)
    except (TypeError, ValueError):
        m = re.match(r"\d+(\.\d+)?", str(v))
        return cast(float(m.group())) if m else None


def yes(v):
    return None if v is None else str(v).strip().lower() == "yes"


def split_slash(v):
    return [p.strip() for p in v.split(" / ")] if v else []


def norm_compat(c):
    if not c:
        return "none"
    l = c.lower()
    for key, name in (("hdr10", "HDR10"), ("sdr", "SDR"), ("hlg", "HLG"), ("blu-ray", "Blu-ray")):
        if key in l:
            return name
    return c


def derive_hdr(v):
    fmt = v.get("hdr_format") or ""
    comp = v.get("hdr_compat") or ""
    prof = v.get("hdr_profile") or ""
    transfer = (v.get("transfer") or "")
    parts = split_slash(fmt)
    dv_idx = next((i for i, p in enumerate(parts) if p.startswith("Dolby Vision")), None)
    dv_profile = None
    dv_bl = None
    if dv_idx is not None:
        pl = split_slash(prof)
        cl = split_slash(comp)
        p = pl[dv_idx] if dv_idx < len(pl) else prof
        m = re.search(r"(?:dvhe|dvh1|dav1|dvav|dva1)\.(\d+)", p) or re.search(r"(?:dvhe|dvh1|dav1|dvav|dva1)\.(\d+)", prof)
        dv_profile = int(m.group(1)) if m else None
        dv_bl = norm_compat(cl[dv_idx] if dv_idx < len(cl) else (comp or None))
    plus = "SMPTE ST 2094 App 4" in fmt or "HDR10+" in comp
    has_hdr10 = ("PQ" in transfer or "2084" in transfer) or ("HDR10" in comp.replace("HDR10+", ""))
    has_hdr10 = has_hdr10 and dv_profile != 5
    has_hlg = "HLG" in transfer or "HLG" in comp
    return dv_profile, dv_bl, has_hdr10, plus, has_hlg


def parse(path, doc):
    tracks = doc.get("media", {}).get("track", [])
    gen = next((t for t in tracks if t["@type"] == "General"), {})
    vids = [t for t in tracks if t["@type"] == "Video"]
    auds = [t for t in tracks if t["@type"] == "Audio"]
    txts = [t for t in tracks if t["@type"] == "Text"]
    rec = {"container": gen.get("Format"), "duration_s": num(gen.get("Duration"))}
    video = None
    if vids:
        t = vids[0]
        lib = t.get("Encoded_Library_Name") or t.get("Encoded_Library") or t.get("Writing_Library")
        video = {
            "codec": t.get("Format"),
            "profile": t.get("Format_Profile"),
            "width": num(t.get("Width"), int),
            "height": num(t.get("Height"), int),
            "frame_rate": num(t.get("FrameRate")),
            "bit_depth": num(t.get("BitDepth"), int),
            "bitrate_bps": num(t.get("BitRate"), int),
            "hdr_format": t.get("HDR_Format"),
            "hdr_compat": t.get("HDR_Format_Compatibility"),
            "hdr_profile": t.get("HDR_Format_Profile"),
            "transfer": t.get("transfer_characteristics"),
            "color_primaries": t.get("colour_primaries"),
            "encoded_library": lib,
            "encoder_settings_present": bool(t.get("Encoded_Library_Settings")),
        }
        if rec["duration_s"] is None:
            rec["duration_s"] = num(t.get("Duration"))
    rec["video"] = video
    rec["video_track_count"] = len(vids)
    rec["audio"] = [
        {
            "codec": t.get("Format"),
            "commercial": t.get("Format_Commercial_IfAny"),
            "features": t.get("Format_AdditionalFeatures"),
            "channels": num(t.get("Channels"), int),
            "layout": t.get("ChannelLayout"),
            "language": t.get("Language"),
            "title": t.get("Title"),
            "default": yes(t.get("Default")),
            "bitrate_bps": num(t.get("BitRate"), int),
        }
        for t in auds
    ]
    rec["subs"] = [
        {
            "format": t.get("Format"),
            "language": t.get("Language"),
            "title": t.get("Title"),
            "forced": yes(t.get("Forced")),
            "default": yes(t.get("Default")),
        }
        for t in txts
    ]
    dv_profile, dv_bl, has_hdr10, plus, has_hlg = derive_hdr(video or {})
    rec["dv_profile"] = dv_profile
    rec["dv_bl_compat"] = dv_bl
    rec["has_hdr10"] = has_hdr10
    rec["has_hdr10plus"] = plus
    rec["has_hlg"] = has_hlg
    rec["atmos"] = any(
        "JOC" in (a["features"] or "") or "atmos" in (a["commercial"] or "").lower() for a in rec["audio"]
    )
    rec["dts_x"] = any(
        re.search(r"\bXLL X\b|XLL\+X|DTS:X", (a["features"] or "") + " " + (a["commercial"] or "")) is not None
        for a in rec["audio"]
    )
    return rec


def probe(path, speed, timeout):
    base = {"path": path}
    try:
        st = os.stat(path)
        base["size_bytes"] = st.st_size
        base["mtime"] = datetime.fromtimestamp(st.st_mtime, timezone.utc).isoformat()
    except OSError as e:
        base["error"] = f"stat: {e}"
        return base
    t0 = time.time()
    try:
        p = subprocess.run(
            [MEDIAINFO, "--Output=JSON", f"--ParseSpeed={speed}", path],
            capture_output=True, text=True, timeout=timeout, errors="replace",
        )
        base["probe_s"] = round(time.time() - t0, 2)
        if p.returncode != 0 or not p.stdout.strip():
            base["error"] = f"mediainfo rc={p.returncode}: {p.stderr.strip()[:200] or 'empty output'}"
            return base
        doc = json.loads(p.stdout)
        if not doc.get("media"):
            base["error"] = "mediainfo: no media in output"
            return base
        base.update(parse(path, doc))
        if base.get("duration_s"):
            base["overall_bitrate_bps"] = int(base["size_bytes"] * 8 / base["duration_s"])
    except subprocess.TimeoutExpired:
        base["probe_s"] = round(time.time() - t0, 2)
        base["error"] = f"timeout after {timeout}s"
    except Exception as e:  # noqa: BLE001
        base["probe_s"] = round(time.time() - t0, 2)
        base["error"] = f"{type(e).__name__}: {e}"
    return base


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--workers", type=int, default=6)
    ap.add_argument("--parse-speed", default="0")
    ap.add_argument("--limit", type=int, default=0)
    ap.add_argument("--timeout", type=int, default=180)
    ap.add_argument("--retry-errors", action="store_true")
    ap.add_argument("--list", default=LIST)
    a = ap.parse_args()

    done, keep = set(), []
    if os.path.exists(OUT):
        with open(OUT) as f:
            for line in f:
                line = line.strip()
                if not line:
                    continue
                try:
                    r = json.loads(line)
                except json.JSONDecodeError:
                    continue
                if a.retry_errors and r.get("error"):
                    continue
                done.add(r["path"])
                keep.append(line)
        if a.retry_errors:
            with open(OUT, "w") as f:
                f.write("\n".join(keep) + ("\n" if keep else ""))
    paths = [l.rstrip("\n") for l in open(a.list) if l.strip()]
    todo = [p for p in paths if p not in done]
    if a.limit:
        todo = todo[: a.limit]
    print(f"todo={len(todo)} already={len(done)} workers={a.workers} speed={a.parse_speed}", flush=True)
    t0 = time.time()
    n = errs = 0
    with open(OUT, "a") as out, ThreadPoolExecutor(a.workers) as ex:
        futs = [ex.submit(probe, p, a.parse_speed, a.timeout) for p in todo]
        for fu in as_completed(futs):
            r = fu.result()
            out.write(json.dumps(r, ensure_ascii=False) + "\n")
            out.flush()
            n += 1
            errs += 1 if r.get("error") else 0
            if n % 25 == 0 or n == len(todo):
                el = time.time() - t0
                print(f"{n}/{len(todo)} errs={errs} {n/el:.2f} files/s elapsed={el:.0f}s", flush=True)
    el = time.time() - t0
    print(f"finished n={n} errs={errs} elapsed={el:.1f}s rate={n/max(el,1e-9):.2f} files/s", flush=True)


if __name__ == "__main__":
    main()
