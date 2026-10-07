#!/bin/bash
# PROTOTYPE tool: shoot every review view of the given directions. tools/matrix.sh <out-dir> A B C
out="$1"; shift
views=(wide:home wide:library wide:series wide:work/halcyon-drift wide:work/brightwater wide:activity wide:nodes wide:requests phone:home phone:library phone:work/brightwater phone:work/halcyon-drift/actions phone:activity phone:search phone:more portal:home portal:search portal:work/arcadia-station portal:request/r3 portal:requests portal-wide:home portal-wide:work/halcyon-drift portal-wide:request/r3)
for k in "$@"; do
  specs=(); for v in "${views[@]}"; do specs+=("$k:$v"); done
  node "$(dirname "$0")/shoot.mjs" --out "$out/$k" --wait 1300 "${specs[@]}" | grep -v '\.png$'
done
