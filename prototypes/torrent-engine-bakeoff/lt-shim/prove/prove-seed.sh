#!/bin/bash
# usage: [TRACKER=path] prove-seed.sh BIN FIXTURES TRKPORT LISTENPORT PREFIX OUTDIR HOLD_S
set -u
BIN=$1 FX=$2 TP=$3 LP=$4 PFX=$5 OUT=$6 HOLD=${7:-5}
mkdir -p "$OUT"; rm -f "$OUT"/trk.jsonl "$OUT"/status.jsonl
"${TRACKER:-$(dirname "$0")/../../tracker/tracker}" --log "$OUT/trk.jsonl" --interval 1800 --min-interval 900 --listen 127.0.0.1:$TP > "$OUT/trk.out" 2>&1 &
TRK=$!
sleep 0.5
"$BIN" --mode seed --fixtures "$FX" --listen-port "$LP" --peer-id-prefix "$PFX" --status "$OUT/status.jsonl" 2> "$OUT/engine.err" &
ENG=$!
sleep "$HOLD"
kill -TERM $ENG; wait $ENG; echo "engine exit=$?"
kill -TERM $TRK; wait $TRK
cat "$OUT/status.jsonl"
python3 -c "
import json,collections,sys
L=[json.loads(l) for l in open(sys.argv[1])]
c=collections.Counter(l['event'] or 'none' for l in L)
print('announces', dict(c))
for ev in ('started','stopped'):
    print(ev, 'unique infohashes', len({l['infohash'] for l in L if l['event']==ev}))
print('peer_ids', sorted({l['peer_id'][:8] for l in L}), 'ua', sorted({l['ua'] for l in L}), 'rejected', sum(l['rejected'] for l in L), 'violations', sum(len(l['violations']) for l in L))
" "$OUT/trk.jsonl"
