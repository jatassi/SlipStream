package timing

import (
	"errors"
	"math"
	"slices"
	"time"
)

var ErrForged = errors.New("the retimed subtitle doesn't follow its claimed timing")

// Judge is the Controller's check of a holder's timing. It takes the
// original and retimed spans, each parsed by the Controller from the
// bytes, the claimed segments' first cues, and the holder's Prominence,
// which needs the whole correlation to recompute. For each segment it
// refits Apply's line from the cue times, within q + CheckSlack on every
// time, and then scores the retimed spans on the file's own map. The Fit
// is the Controller's: the refitted segments, the recomputed Match and
// Coverage, and their verdict. Every refusal is ErrForged.
func Judge(ref Reference, original, retimed []Span, claimed []int, prominence float64, g Gate, k Constants, q time.Duration) (Fit, error) {
	if len(original) != len(retimed) || len(claimed) == 0 || len(claimed) > k.MaxSegments || claimed[0] != 0 {
		return Fit{}, ErrForged
	}
	segs := make([]Segment, len(claimed))
	for s, first := range claimed {
		end := len(original)
		if s+1 < len(claimed) {
			end = claimed[s+1]
		}
		if first >= end {
			return Fit{}, ErrForged
		}
		seg, err := refit(original[first:end], retimed[first:end], len(claimed) == 1, g, k, q)
		if err != nil {
			return Fit{}, err
		}
		seg.First = first
		segs[s] = seg
	}
	fit := Fit{Segments: segs, Density: ref.Density(), Prominence: prominence}
	fit.Raw, fit.Match, fit.Coverage = Score(ref, retimed)
	fit.Verdict = verdict(fit, g, k)
	return fit, nil
}

// refit recovers one segment's (ρ, o) from its cues by undoing Apply.
//
// A cue whose retimed start is 0 is clamped, and its line must map it to
// zero or before. That refuses a clamped cue in any segment but one that
// starts at the file's head, wherever the cue sits in it, since a
// subtitle's cues need not be in time order.
func refit(original, retimed []Span, only bool, g Gate, k Constants, q time.Duration) (Segment, error) {
	ts, us := unclamped(original, retimed, q)
	if len(ts) < 2 {
		return Segment{}, ErrForged
	}
	span := slices.Max(ts) - slices.Min(ts)
	if span == 0 || (!only && span < float64(k.MinSegment)) {
		return Segment{}, ErrForged
	}
	rho, ok := allowed(slope(ts, us), 2*float64(q)/span, g.Framerates)
	if !ok {
		return Segment{}, ErrForged
	}
	o := 0.0
	for i, t := range ts {
		o += us[i] - rho*t
	}
	o /= float64(len(ts))
	if math.Abs(o) > float64(g.MaxOffset+q) || !onLine(original, retimed, rho, o, q, k.CheckSlack) {
		return Segment{}, ErrForged
	}
	return Segment{Offset: time.Duration(math.Round(o)), Ratio: rho}, nil
}

// unclamped is the (original, retimed) time pairs Apply didn't clamp: the
// starts that aren't 0 and the ends past start + q.
func unclamped(original, retimed []Span, q time.Duration) (ts, us []float64) {
	for i, o := range original {
		r := retimed[i]
		if r.Start != 0 {
			ts, us = append(ts, float64(o.Start)), append(us, float64(r.Start))
		}
		if r.End > r.Start+q {
			ts, us = append(ts, float64(o.End)), append(us, float64(r.End))
		}
	}
	return ts, us
}

// slope is the least-squares slope of us over ts.
func slope(ts, us []float64) float64 {
	tm, um := mean(ts), mean(us)
	var cov, vr float64
	for i, t := range ts {
		cov += (t - tm) * (us[i] - um)
		vr += (t - tm) * (t - tm)
	}
	return cov / vr
}

func mean(v []float64) float64 {
	sum := 0.0
	for _, x := range v {
		sum += x
	}
	return sum / float64(len(v))
}

// onLine is whether every retimed time is Apply's of its original on the
// line t' = rho·t + o, within q + slack.
func onLine(original, retimed []Span, rho, o float64, q, slack time.Duration) bool {
	tol := float64(q + slack)
	for i, sp := range original {
		r := retimed[i]
		start := rho*float64(sp.Start) + o
		switch {
		case r.Start == 0 && start > float64(q)/2+float64(slack):
			return false
		case r.Start != 0 && math.Abs(float64(r.Start)-start) > tol:
			return false
		case math.Abs(float64(r.End)-max(rho*float64(sp.End)+o, float64(r.Start+q))) > tol:
			return false
		}
	}
	return true
}

// allowed is the ratio the set allows for a fitted slope, within delta:
// the stamp unit's worth of slope over the segment's span, twice.
func allowed(fit, delta float64, set Framerates) (float64, bool) {
	switch set {
	case NoRatios:
		return 1, math.Abs(fit-1) <= delta
	case StandardRatios:
		near := standard[0]
		for _, r := range standard {
			if math.Abs(fit-r) < math.Abs(fit-near) {
				near = r
			}
		}
		return near, math.Abs(fit-near) <= delta
	case RangeRatios:
		return fit, fit >= rangeMin-delta && fit <= rangeMax+delta
	}
	return 0, false
}
