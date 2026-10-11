#!/usr/bin/env bash
set -u
R=$(cd "$(dirname "$0")/.." && pwd)
OUT=${1:-/home/user/results/seed}
for rep in 1 2 3; do
  for n in 1000 3000; do
    for e in anacrolix rain lt-go; do
      "$R/run/seed.sh" $e $n 60 180 "$OUT/$e-$n-$rep" >"$OUT.log.tmp" 2>&1
      tail -1 "$OUT.log.tmp" >>"$OUT.results.jsonl"
    done
  done
done
"$R/run/seed.sh" lt-rust 1000 60 180 "$OUT/lt-rust-1000-1" 2>&1 | tail -1 >>"$OUT.results.jsonl"
echo done >>"$OUT.results.jsonl"
