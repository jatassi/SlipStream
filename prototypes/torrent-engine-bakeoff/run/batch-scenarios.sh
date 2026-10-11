#!/usr/bin/env bash
set -u
R=$(cd "$(dirname "$0")/.." && pwd)
OUT=${1:-/home/user/results/sc}
for sc in pacing tiers failure lifecycle pacing-latency; do
  i=0
  for e in anacrolix rain lt-go; do
    TBASE=$((7600 + i * 10)) EBASE=$((20000 + i * 6000)) python3 "$R/run/scenario.py" $e $sc "$OUT/$sc-$e" >"$OUT/$sc-$e.log" 2>&1 &
    i=$((i + 1))
  done
  wait
done
echo done >"$OUT/DONE"
