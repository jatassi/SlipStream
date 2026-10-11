#!/bin/bash
# usage: [TRACKER=path] prove-life.sh BIN FIXTURES TRKPORT SEEDPORT LIFEPORT INDEX OUTDIR
set -u
BIN=$1 FX=$2 TP=$3 SP=$4 LP=$5 IDX=$6 OUT=$7
rm -rf "$OUT"; mkdir -p "$OUT/dl"
"${TRACKER:-$(dirname "$0")/../../tracker/tracker}" --log "$OUT/trk.jsonl" --interval 1800 --min-interval 900 --listen 127.0.0.1:$TP > "$OUT/trk.out" 2>&1 &
TRK=$!
sleep 0.5
"$BIN" --mode seed --fixtures "$FX" --listen-port "$SP" --peer-id-prefix -SEED01- --status "$OUT/seed.jsonl" 2> "$OUT/seed.err" &
SEED=$!
sleep 2
SECONDS=0; "$BIN" --mode lifecycle --fixtures "$FX" --index "$IDX" --data-dir "$OUT/dl" --listen-port "$LP" --peer-id-prefix -LIFE01- --status "$OUT/life.jsonl" 2> "$OUT/life.err"
echo "lifecycle exit=$? wall=${SECONDS}s"
kill -TERM $SEED; wait $SEED; echo "seed exit=$?"
kill -TERM $TRK; wait $TRK
echo "--- life status"; cat "$OUT/life.jsonl"
echo "--- seed status"; cat "$OUT/seed.jsonl"
echo "--- data"; ls -ls "$OUT/dl"
python3 -c "
import json,sys
L=[json.loads(l) for l in open(sys.argv[1])]
ih=sys.argv[2]
for l in L:
    if l['infohash']==ih: print(round(l['t']-L[0]['t'],1), l['peer_id'][:8], l['event'] or '(none)', 'left', l['left'], 'down', l['downloaded'], 'up', l['uploaded'])
" "$OUT/trk.jsonl" "$(python3 -c "import json,sys;print(json.load(open(sys.argv[1]))[int(sys.argv[2])]['infohash'])" "$FX/manifest.json" "$IDX")"
