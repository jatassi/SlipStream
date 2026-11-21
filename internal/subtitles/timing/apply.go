package timing

import (
	"math"
	"time"
)

// Apply retimes spans by fit, segment by segment, to stamps of unit q:
// 1 ms for SRT, 10 ms for ASS and SSA. It is the only place a time is
// rounded or clamped, and Judge undoes exactly this:
//
//   - each time t maps to ρ·t + o and rounds to the nearest multiple of q,
//     halves away from zero;
//   - a start before zero becomes zero, and an end before start + q becomes
//     start + q.
//
// No span is dropped, and Skip is kept.
func Apply(spans []Span, fit Fit, q time.Duration) []Span {
	out := make([]Span, len(spans))
	for s, seg := range fit.Segments {
		end := len(spans)
		if s+1 < len(fit.Segments) {
			end = fit.Segments[s+1].First
		}
		for i := seg.First; i < end; i++ {
			sp := spans[i]
			start := max(stamp(seg.Ratio*float64(sp.Start)+float64(seg.Offset), q), 0)
			out[i] = Span{
				Start: start,
				End:   max(stamp(seg.Ratio*float64(sp.End)+float64(seg.Offset), q), start+q),
				Skip:  sp.Skip,
			}
		}
	}
	return out
}

func stamp(t float64, q time.Duration) time.Duration {
	return time.Duration(math.Round(t/float64(q))) * q
}
