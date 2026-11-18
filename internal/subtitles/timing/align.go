// Package timing is Timing sync: it times a subtitle's cues against where
// speech falls in a file's Speech Map, and checks a timing someone else
// claims. It is pure, with no IO, clock or randomness (the subtitles-pure
// lint), and it never touches cue text.
//
// The signals, the FFT offset search, the ratio search and its golden-
// section search are ported from ffsubsync (MIT, copyright 2019 Stephen
// Macke). The split-penalty search and its guard-band scoring are alass's
// (GPL-3.0-or-later, copyright 2017 kaegi) by way of ffsubsync's port.
// Their licenses are in internal/notices/ffsubsync and
// internal/notices/alass, and NOTICES.txt reproduces both.
package timing

import (
	"math"
	"slices"
	"time"
)

// Span is one cue's time. Skip marks a cue that carries no speech signal
// (empty after markup, wholly bracketed, only music symbols, or a credit
// line first or last). It is timed with its segment but never scored.
type Span struct {
	Start, End time.Duration
	Skip       bool
}

// Segment maps the cues from First to the next segment's First:
// t' = Ratio·t + Offset.
type Segment struct {
	First  int
	Offset time.Duration
	Ratio  float64
}

// Fit is a timing and its numbers: Match is (Raw − Density) / (1 −
// Density), Prominence is the best correlation peak the timing doesn't use
// over its best, and Coverage is the share of the map's speech span the
// cues span.
type Fit struct {
	Segments   []Segment
	Match      float64
	Raw        float64
	Density    float64
	Prominence float64
	Coverage   float64
	Verdict    Verdict
}

// Offset is the first segment's, for Details.
func (f Fit) Offset() time.Duration {
	if len(f.Segments) == 0 {
		return 0
	}
	return f.Segments[0].Offset
}

// Ratio is the first segment's, for Details.
func (f Fit) Ratio() float64 {
	if len(f.Segments) == 0 {
		return 1
	}
	return f.Segments[0].Ratio
}

type Verdict uint8

const (
	Passed Verdict = iota + 1
	// LowMatch is a Match under the Gate's.
	LowMatch
	// PastOffset is a timing that would pass but shifts further than the
	// Gate's MaxOffset.
	PastOffset
	// Ambiguous is a timing that another peak, ProminenceGap away from
	// every offset it uses, nearly matches, as a wrong episode does.
	Ambiguous
	// Uncovered is cues spanning less of the map's speech than Coverage,
	// as another cut's do.
	Uncovered
	// NoSignal is a map with no speech, or all speech, or no voiced cue.
	NoSignal
)

// Align finds the best timing of spans against ref under g and k and
// judges it. It is the holder's call.
//
//  1. The FFT correlation over twice MaxOffset picks the ratio: 1, each
//     standard ratio, and for RangeRatios a golden-section search over
//     [0.9, 1.1].
//  2. The split search places every cue on the ±MaxOffset grid at that
//     ratio, and segments shorter than MinSegment or past MaxSegments are
//     merged into the neighbor that loses least.
//  3. The gate scores the timing. One that fails while the correlation's
//     best peak lies past MaxOffset and would pass there is PastOffset.
func Align(ref Reference, spans []Span, g Gate, k Constants) Fit {
	c := ref.Density()
	var x signal
	if c == 0 || c >= 1 || !x.raster(spans, 1) {
		return Fit{Density: c, Verdict: NoSignal}
	}
	w := int(g.MaxOffset / Frame)
	corr := newCorrelator(ref, 2*w)
	rho := bestRatio(corr, &x, spans, g.Framerates, k.GoldenTolerance)
	x.raster(spans, rho)
	scores := corr.correlate(x)
	lag, _ := corr.peak(scores)

	gr := grid{ref: ref, w: w, guard: int(k.GuardMax / Frame), lp: k.LengthPenalty, rho: rho}
	cues := gr.cues(spans)
	rs := gr.settle(spans, cues, runs(gr.split(cues, k.SplitPenalty*float64(time.Second/Frame))), k)
	segs := make([]Segment, len(rs))
	for i, r := range rs {
		segs[i] = Segment{First: r.first, Offset: gr.offset(r.j), Ratio: rho}
	}
	fit := scored(ref, spans, segs)
	fit.Prominence = corr.prominence(scores, segs, int(k.ProminenceGap/Frame))
	fit.Verdict = verdict(fit, g, k)
	if fit.Verdict != Passed && (lag > w || lag < -w) {
		past := scored(ref, spans, []Segment{{Offset: time.Duration(lag) * Frame, Ratio: rho}})
		if past.Match >= g.Match {
			past.Prominence = fit.Prominence
			past.Verdict = PastOffset
			return past
		}
	}
	return fit
}

// scored is segs with the gate's numbers on spans retimed at the finest
// stamp.
func scored(ref Reference, spans []Span, segs []Segment) Fit {
	fit := Fit{Segments: segs, Density: ref.Density()}
	fit.Raw, fit.Match, fit.Coverage = Score(ref, Apply(spans, fit, time.Nanosecond))
	return fit
}

// bestRatio is the allowed ratio whose correlation peaks highest.
func bestRatio(corr *correlator, x *signal, spans []Span, set Framerates, tol float64) float64 {
	try := func(rho float64) float64 {
		x.raster(spans, rho)
		_, score := corr.peak(corr.correlate(*x))
		return score
	}
	if set == NoRatios {
		return 1
	}
	best, bestScore := 1.0, math.Inf(-1)
	for _, rho := range standard {
		if s := try(rho); s > bestScore {
			best, bestScore = rho, s
		}
	}
	if set == RangeRatios {
		if rho, s := golden(try, rangeMin, rangeMax, tol); s > bestScore {
			best = rho
		}
	}
	return best
}

// golden is ffsubsync's golden_section_search.gss, maximizing f on [a, b]
// until the bracket is narrower than tol. It returns the best point it
// evaluated and its value.
func golden(f func(float64) float64, a, b, tol float64) (x, y float64) {
	invphi := (math.Sqrt(5) - 1) / 2
	invphi2 := (3 - math.Sqrt(5)) / 2
	x, y = a, math.Inf(-1)
	eval := func(at float64) float64 {
		v := f(at)
		if v > y {
			x, y = at, v
		}
		return v
	}
	h := b - a
	c, d := a+invphi2*h, a+invphi*h
	yc, yd := eval(c), eval(d)
	for h > tol {
		h *= invphi
		if yc > yd {
			d, yd = c, yc
			c = a + invphi2*h
			yc = eval(c)
		} else {
			a, c, yc = c, d, yd
			d = a + invphi*h
			yd = eval(d)
		}
	}
	return x, y
}

// prominence is the best score at a lag more than gap frames from every
// offset the timing uses, over the best score: 1 when nothing correlates.
func (c *correlator) prominence(scores []float64, segs []Segment, gap int) float64 {
	_, best := c.peak(scores)
	if best <= 0 {
		return 1
	}
	far := 0.0
	for k, v := range scores {
		if v <= far {
			continue
		}
		lag := (k - c.reach) * step
		if !slices.ContainsFunc(segs, func(s Segment) bool { return abs(lag-int(s.Offset/Frame)) <= gap }) {
			far = v
		}
	}
	return far / best
}

func abs(n int) int { return max(n, -n) }
