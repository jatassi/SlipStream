package timing_test

import (
	"errors"
	"math"
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/subtitles/timing"
	"github.com/jatassi/slipstream/internal/subtitles/timing/timingtest"
)

// held is what a holder answers: its fit and the cues it retimed.
type held struct {
	ref      timing.Reference
	original []timing.Span
	fit      timing.Fit
	retimed  []timing.Span
	gate     timing.Gate
	q        time.Duration
}

func hold(t *testing.T, ref timing.Reference, original []timing.Span, g timing.Gate, q time.Duration) held {
	t.Helper()
	fit := timing.Align(ref, original, g, k)
	passed(t, fit)
	return held{ref, original, fit, timing.Apply(original, fit, q), g, q}
}

func (h held) claimed() []int {
	firsts := make([]int, len(h.fit.Segments))
	for i, s := range h.fit.Segments {
		firsts[i] = s.First
	}
	return firsts
}

func (h held) judge(retimed []timing.Span, claimed []int) (timing.Fit, error) {
	return timing.Judge(h.ref, h.original, retimed, claimed, h.fit.Prominence, h.gate, k, h.q)
}

func forged(t *testing.T, what string, err error) {
	t.Helper()
	if !errors.Is(err, timing.ErrForged) {
		t.Errorf("%s: err %v, want ErrForged", what, err)
	}
}

// TestJudgeQuantized runs Align, Apply at the stamp unit and Judge, as a
// holder and the Controller do, for SRT's 1 ms and ASS's 10 ms. Every
// honest timing passes with the Controller's Match within 0.005 of the
// holder's, and each forgery is refused.
func TestJudgeQuantized(t *testing.T) {
	for _, q := range []time.Duration{time.Millisecond, 10 * time.Millisecond} {
		cases := honest(t, q)
		for i, h := range cases {
			got, err := h.judge(h.retimed, h.claimed())
			if err != nil {
				t.Fatalf("q %v case %d (%d segments, ratio %.5f): honest timing refused: %v", q, i, len(h.fit.Segments), h.fit.Ratio(), err)
			}
			if got.Verdict != timing.Passed || math.Abs(got.Match-h.fit.Match) > 0.005 {
				t.Errorf("q %v case %d: verdict %d match %.4f, want Passed within 0.005 of %.4f", q, i, got.Verdict, got.Match, h.fit.Match)
			}
			for s, seg := range got.Segments {
				if claim := h.fit.Segments[s]; seg.First != claim.First || math.Abs(seg.Ratio-claim.Ratio) > 1e-6 || (seg.Offset-claim.Offset).Abs() > q {
					t.Errorf("q %v case %d segment %d: refit %+v, claimed %+v", q, i, s, seg, claim)
				}
			}
		}
		refused(t, q, cases[2], cases[len(cases)-2])
	}
}

// honest is a holder's answer for each standard ratio, a Range ratio, a
// mid-file split and a −3 s head that clamps three cues, in that order.
func honest(t *testing.T, q time.Duration) []held {
	t.Helper()
	var cases []held
	for i, r := range append([]float64{1}, standard...) {
		f := newFilm(uint64(200 + i))
		cases = append(cases, hold(t, f.ref, timingtest.Shift(timingtest.Scale(f.cues, r), 6*time.Second), standardGate, q))
	}
	f := newFilm(210)
	cases = append(cases, hold(t, f.ref, timingtest.Scale(f.cues, 1/1.0331), gate(timing.RangeRatios, 30*time.Second), q))
	f = newFilm(211)
	split := hold(t, f.ref, timingtest.ShiftAfter(f.cues, 50*time.Minute, -11*time.Second), standardGate, q)
	if n := len(split.fit.Segments); n != 2 {
		t.Fatalf("split: %d segments, want 2", n)
	}
	f = newFilm(212)
	head := slices.Concat([]timing.Span{
		{Start: 200 * time.Millisecond, End: 900 * time.Millisecond},
		{Start: time.Second, End: 1800 * time.Millisecond},
		{Start: 2100 * time.Millisecond, End: 2900 * time.Millisecond},
	}, timingtest.Shift(f.cues, 3*time.Second))
	negative := hold(t, f.ref, head, standardGate, q)
	if clamped := negative.retimed[2].Start; clamped != 0 {
		t.Fatalf("negative head: third cue starts at %v, want clamped to 0", clamped)
	}
	return append(cases, split, negative)
}

// refused forges h's and split's answers.
func refused(t *testing.T, q time.Duration, h, split held) {
	t.Helper()
	moved := slices.Clone(h.retimed)
	c := len(moved) / 2
	exact := h.fit.Segments[0].Ratio*float64(h.original[c].Start) + float64(h.fit.Segments[0].Offset)
	away := q + k.CheckSlack + time.Millisecond
	if float64(moved[c].Start) < exact {
		away = -away
	}
	moved[c].Start += away
	forged(t, "a cue moved past the slack", errOf(h.judge(moved, h.claimed())))

	clamped := slices.Clone(split.retimed)
	clamped[split.fit.Segments[1].First+5].Start = 0
	forged(t, "a clamped cue in the second segment", errOf(split.judge(clamped, split.claimed())))

	nine := make([]int, 9)
	for s := range nine {
		nine[s] = s * len(h.original) / 9
	}
	forged(t, "nine segments", errOf(h.judge(h.retimed, nine)))

	off := h.fit
	span := float64(h.original[len(h.original)-2].End - h.original[1].Start)
	off.Segments = []timing.Segment{{Ratio: h.fit.Ratio() + 2*2*float64(q)/span, Offset: h.fit.Offset()}}
	forged(t, "a ratio 2δ off the set", errOf(h.judge(timing.Apply(h.original, off, q), h.claimed())))
}

func TestJudge(t *testing.T) {
	q := time.Millisecond
	f := newFilm(300)
	h := hold(t, f.ref, timingtest.Shift(f.cues, 4*time.Second), standardGate, q)
	retime := func(segs ...timing.Segment) []timing.Span {
		return timing.Apply(h.original, timing.Fit{Segments: segs}, q)
	}

	forged(t, "an offset past the bound", errOf(h.judge(retime(timing.Segment{Ratio: 1, Offset: 31 * time.Second}), []int{0})))
	forged(t, "a ratio off the set", errOf(h.judge(retime(timing.Segment{Ratio: 1.02, Offset: -4 * time.Second}), []int{0})))
	off := slices.Clone(h.retimed)
	off[100].End += 500 * time.Millisecond
	forged(t, "a cue off its line", errOf(h.judge(off, []int{0})))
	short := len(h.original) - 10
	forged(t, "a segment shorter than MinSegment", errOf(h.judge(h.retimed, []int{0, short})))
	forged(t, "a first segment past the first cue", errOf(h.judge(h.retimed, []int{1})))
	forged(t, "a cue count that differs", errOf(h.judge(h.retimed[1:], []int{0})))

	noRatios := h
	noRatios.gate = gate(timing.NoRatios, 30*time.Second)
	forged(t, "a standard ratio where none is allowed", errOf(noRatios.judge(retime(timing.Segment{Ratio: 25.0 / 24, Offset: -4 * time.Second}), []int{0})))

	wrong, err := h.judge(retime(timing.Segment{Ratio: 1, Offset: 16 * time.Second}), []int{0})
	if err != nil {
		t.Fatalf("a consistent timing at the wrong offset: %v", err)
	}
	if wrong.Verdict != timing.LowMatch || wrong.Match >= h.fit.Match-0.3 {
		t.Errorf("a timing at the wrong offset scored %.3f with verdict %d, want the map's own low Match", wrong.Match, wrong.Verdict)
	}
}

func errOf(_ timing.Fit, err error) error { return err }
