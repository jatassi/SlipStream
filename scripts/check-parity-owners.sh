#!/bin/sh
# Every kept or superseded row in the parity matrix names exactly one spec issue
# and is planned or shipped. Every excepted or out-of-scope row has Status `-`.
set -eu
matrix="${1:-$(dirname "$0")/../docs/parity-matrix.md}"
awk -F' [|] ' '
/^[|] P-[0-9]+ / {
  id = substr($1, 3); disp = $5; cov = $6; status = $7
  sub(/ [|]$/, "", status)
  links = gsub(/github[.]com\/jatassi\/SlipStream\/issues\/[0-9]+\)/, "&", cov)
  if (disp ~ /^(kept|superseded)/) {
    if (links != 1) { print id ": expected one spec in Covered by, found " links; bad++ }
    if (status != "planned" && status != "shipped") { print id ": Status is " status ", expected planned or shipped"; bad++ }
  } else if (disp ~ /^(excepted|out of scope)/) {
    if (status != "-") { print id ": Status is " status ", expected -"; bad++ }
  } else { print id ": unknown disposition " disp; bad++ }
  rows++
}
END { if (bad) exit 1; print rows " rows checked" }
' "$matrix"
