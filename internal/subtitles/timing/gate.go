package timing

import "math"

// Score is the gate's numbers for spans as they stand, with no search:
// raw is the share a of voiced cue frames that fall on speech, match is
// (a − c) / (1 − c) where c is the map's density, the share random
// placement scores, and coverage is the share of the map's speech span
// that the voiced cues span. Align and Judge both score through it, and it
// serves an untimed upload's Details.
func Score(ref Reference, spans []Span) (raw, match, coverage float64) {
	on, total := 0, 0
	lo, hi := math.MaxInt, math.MinInt
	for _, sp := range spans {
		from, to, ok := frames(sp, 1)
		if !ok {
			continue
		}
		on += ref.Overlap(from, to)
		total += to - from
		lo, hi = min(lo, from), max(hi, to)
	}
	c := ref.Density()
	if total == 0 || c >= 1 {
		return 0, 0, 0
	}
	raw = float64(on) / float64(total)
	match = (raw - c) / (1 - c)
	if ref.last > ref.first {
		covered := min(hi, ref.last) - max(lo, ref.first)
		coverage = float64(max(covered, 0)) / float64(ref.last-ref.first)
	}
	return raw, match, coverage
}

// verdict is the gate on a scored fit. The settings come first, then the
// constants inside the gate.
func verdict(f Fit, g Gate, k Constants) Verdict {
	switch {
	case f.Match < g.Match:
		return LowMatch
	case f.Prominence > k.Prominence:
		return Ambiguous
	case f.Coverage < k.Coverage:
		return Uncovered
	}
	return Passed
}
