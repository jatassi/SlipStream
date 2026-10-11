#!/usr/bin/env python3
import argparse, collections, json, statistics
from bisect import bisect_left

ap = argparse.ArgumentParser()
ap.add_argument("log")
ap.add_argument("--since", type=float, default=None)
ap.add_argument("--json", action="store_true")
a = ap.parse_args()

rows = []
for l in open(a.log):
    l = l.strip()
    if not l:
        continue
    try:
        r = json.loads(l)
    except ValueError:
        continue
    if a.since is None or r["t"] >= a.since:
        rows.append(r)

listeners = collections.Counter(r["listener"] for r in rows)
events = collections.Counter(r["event"] or "(none)" for r in rows)
viol = collections.Counter(v for r in rows for v in r.get("violations", []))
rejected = sum(1 for r in rows if r.get("rejected"))
uas = collections.Counter(r["ua"] for r in rows)
pfx = collections.Counter(r["peer_id"][:8] for r in rows)

per = collections.defaultdict(list)
for r in rows:
    if not r["event"] and not r.get("rejected"):
        per[(r["infohash"], r["peer_id_hex"])].append(r["t"])
gaps_by_ih = collections.defaultdict(list)
for (ih, _), ts in per.items():
    ts.sort()
    gaps_by_ih[ih] += [b - x for x, b in zip(ts, ts[1:])]
gap_stats = {ih: {"n": len(g), "min": min(g), "median": statistics.median(g), "max": max(g)}
             for ih, g in gaps_by_ih.items() if g}
allgaps = [x for g in gaps_by_ih.values() for x in g]
overall = {"n": len(allgaps), "min": min(allgaps), "median": statistics.median(allgaps), "max": max(allgaps)} if allgaps else None

ts = sorted(r["t"] for r in rows)
peak = 0
j = 0
for i, t in enumerate(ts):
    while ts[j] < t - 1.0:
        j += 1
    peak = max(peak, i - j + 1)

res = {
    "total": len(rows),
    "rejected": rejected,
    "per_listener": dict(listeners),
    "per_event": dict(events),
    "distinct_infohashes": len({r["infohash"] for r in rows}),
    "gap_overall_noevent": overall,
    "gap_per_infohash_noevent": gap_stats,
    "violations": dict(viol),
    "peak_announces_1s": peak,
    "peer_id_prefixes": dict(pfx),
    "user_agents": dict(uas),
}
if a.json:
    print(json.dumps(res, indent=1))
else:
    print(f"total announces: {res['total']} (rejected {rejected})")
    print(f"distinct infohashes: {res['distinct_infohashes']}")
    print(f"peak announces in 1s: {peak}")
    print("per listener:", res["per_listener"])
    print("per event:", res["per_event"])
    print("violations:", res["violations"])
    print("no-event gap overall (s):", overall)
    top = sorted(gap_stats.items(), key=lambda kv: kv[1]["min"])[:5]
    print(f"no-event gap per infohash ({len(gap_stats)} with >=2 announces; 5 smallest min):")
    for ih, s in top:
        print(f"  {ih[:12]} n={s['n']} min={s['min']:.2f} median={s['median']:.2f} max={s['max']:.2f}")
    print("peer_id prefixes:", res["peer_id_prefixes"])
    print("user agents:", res["user_agents"])
