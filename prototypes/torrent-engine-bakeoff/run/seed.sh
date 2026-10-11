#!/usr/bin/env bash
# usage: seed.sh ENGINE N SETTLE_S MEASURE_S OUTDIR
set -euo pipefail
R=$(cd "$(dirname "$0")/.." && pwd)
ENGINE=$1 N=$2 SETTLE=$3 MEASURE=$4 OUT=$5
TPORT=${TPORT:-7500} EPORT=${EPORT:-18000}
FX=${FXROOT:-/home/user/fx}/$N
case $ENGINE in
  anacrolix) BIN=$R/engine-anacrolix/engine-anacrolix ;;
  rain) BIN=$R/engine-rain/engine-rain ;;
  lt-go) BIN=$R/engine-lt-go/engine-lt-go ;;
  lt-rust) BIN=$R/engine-lt-rust/target/release/engine-lt-rust ;;
esac
mkdir -p "$OUT"
[ -f "$FX/manifest.json" ] || "$R/fixtures/fixtures" gen --out "$FX" --count "$N" --piece-len 4194304 --pieces 1024 --announce "http://127.0.0.1:$TPORT/pk/announce" --name-prefix "n$N-" >/dev/null
"$R/tracker/tracker" --log "$OUT/tracker.jsonl" --interval 1800 --min-interval 1800 --listen "127.0.0.1:$TPORT" >"$OUT/tracker.out" 2>&1 &
TPID=$!
sleep 0.5
rm -f "$OUT/status.jsonl"
T_START=$(date +%s.%N)
"$BIN" --mode seed --fixtures "$FX" --listen-port "$EPORT" --status "$OUT/status.jsonl" >"$OUT/engine.out" 2>&1 &
EPID=$!
python3 "$R/measure/sample.py" --pid $EPID --interval 2 --out "$OUT/samples.jsonl" &
SPID=$!
until grep -q all_added "$OUT/status.jsonl" 2>/dev/null; do
  kill -0 $EPID 2>/dev/null || { echo "engine died"; exit 1; }
  sleep 0.5
done
sleep "$SETTLE"
T0=$(date +%s.%N)
sleep "$MEASURE"
T1=$(date +%s.%N)
kill -TERM $EPID
wait $EPID || true
wait $SPID || true
sleep 1
kill -TERM $TPID; wait $TPID || true
python3 - "$OUT" "$T_START" "$T0" "$T1" "$N" "$ENGINE" <<'PY'
import json,sys,subprocess,os
out,ts,t0,t1,n,eng=sys.argv[1],float(sys.argv[2]),float(sys.argv[3]),float(sys.argv[4]),int(sys.argv[5]),sys.argv[6]
R=os.path.dirname(os.path.dirname(os.path.abspath(sys.argv[0]))) if False else None
log=[json.loads(l) for l in open(f"{out}/tracker.jsonl")]
ev=lambda e:len({l["infohash"] for l in log if l["event"]==e and not l["rejected"]})
status=[json.loads(l) for l in open(f"{out}/status.jsonl")]
added=next(s for s in status if s["event"]=="all_added")
first_started=[l["t"] for l in log if l["event"]=="started"]
win=[json.loads(l) for l in open(f"{out}/samples.jsonl")]
w=[s for s in win if t0<=s["t"]<=t1]
tck=os.sysconf("SC_CLK_TCK")
cpu=(w[-1]["cpu_ticks"]-w[0]["cpu_ticks"])/tck/(w[-1]["t"]-w[0]["t"])*100
med=lambda k:sorted(s[k] for s in w)[len(w)//2]/1024
res=dict(engine=eng,n=n,started_ih=ev("started"),stopped_ih=ev("stopped"),announces=len(log),
 rejected=sum(l["rejected"] for l in log),
 started_span_s=round(max(first_started)-min(first_started),2) if first_started else None,
 all_added_s=added.get("elapsed",added.get("elapsed_s")),
 cpu_pct_core=round(cpu,3),rss_mib=round(med("rss_kb"),1),rss_anon_mib=round(med("rss_anon_kb"),1),
 pss_mib=round(med("pss_kb"),1),peak_rss_anon_mib=round(max(s["rss_anon_kb"] for s in win)/1024,1),
 threads=w[-1]["num_threads"],fds=w[-1]["fds"],window_s=round(w[-1]["t"]-w[0]["t"],1))
print(json.dumps(res)); open(f"{out}/result.json","w").write(json.dumps(res))
PY
