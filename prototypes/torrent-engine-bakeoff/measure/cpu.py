#!/usr/bin/env python3
import argparse, json, os, statistics

ap = argparse.ArgumentParser()
ap.add_argument("samples")
ap.add_argument("--from", dest="t0", type=float, required=True)
ap.add_argument("--to", dest="t1", type=float, required=True)
ap.add_argument("--json", action="store_true")
a = ap.parse_args()

rows = [json.loads(l) for l in open(a.samples) if l.strip()]
w = [r for r in rows if a.t0 <= r["t"] <= a.t1]
hz = os.sysconf("SC_CLK_TCK")
cpu = None
if len(w) >= 2:
    cpu = 100.0 * (w[-1]["cpu_ticks"] - w[0]["cpu_ticks"]) / hz / (w[-1]["t"] - w[0]["t"])
mib = lambda k: [r[k] / 1024 for r in w]
res = {"samples": len(w), "cpu_pct_of_core": cpu}
if w:
    for k in ("rss", "rss_anon", "rss_file", "pss"):
        v = mib(k + "_kb")
        res[k + "_mib_median"] = statistics.median(v)
        res[k + "_mib_max"] = max(v)
if a.json:
    print(json.dumps(res))
else:
    for k, v in res.items():
        print(f"{k}: {v:.2f}" if isinstance(v, float) else f"{k}: {v}")
