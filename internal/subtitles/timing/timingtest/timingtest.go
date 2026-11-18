// Package timingtest builds Timing sync fixtures from a seeded speech
// schedule: the map a file would have, how a VAD hears it, the cues a
// person timed to it, and the transforms that put a subtitle out of time.
package timingtest

import (
	"math"
	"math/rand/v2"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/subtitles/timing"
)

// Schedule is where speech falls in a file: sorted, disjoint runs.
type Schedule struct {
	Length time.Duration
	Runs   []Run
}

type Run struct{ Start, End time.Duration }

// NewSchedule is speech runs of log-normal length from 0.3 to 4 s, spaced
// by exponential gaps so that about density of length is speech.
func NewSchedule(seed uint64, length time.Duration, density float64) Schedule {
	r := source(seed)
	const minGap = 150 * time.Millisecond
	meanRun := 1.45 * float64(time.Second)
	meanGap := meanRun*(1-density)/density - float64(minGap)
	s := Schedule{Length: length}
	at := minGap + time.Duration(r.ExpFloat64()*meanGap)
	for {
		long := math.Exp(math.Log(1.2)+0.6*r.NormFloat64()) * float64(time.Second)
		end := at + time.Duration(min(max(long, 0.3e9), 4e9))
		if end > length {
			return s
		}
		s.Runs = append(s.Runs, Run{at, end})
		at = end + minGap + time.Duration(r.ExpFloat64()*meanGap)
	}
}

// Reference is the schedule's exact Speech Map.
func (s Schedule) Reference() timing.Reference {
	frames := int((s.Length + timing.Frame - 1) / timing.Frame)
	bits := make([]uint64, (frames+63)/64)
	for _, run := range s.Runs {
		to := min(int((run.End+timing.Frame-1)/timing.Frame), frames)
		for f := int(run.Start / timing.Frame); f < to; f++ {
			bits[f/64] |= 1 << (f % 64)
		}
	}
	return timing.NewReference(bits, frames)
}

// Density is the share of the schedule that is speech.
func (s Schedule) Density() float64 {
	var on time.Duration
	for _, run := range s.Runs {
		on += run.End - run.Start
	}
	return float64(on) / float64(s.Length)
}

// VAD is how a voice detector mishears a schedule.
type VAD struct {
	// Miss is the share of speech runs it doesn't hear.
	Miss float64
	// Music is about the share of the file it hears as speech where there
	// is none, in runs of 0.2 to 2.5 s.
	Music float64
}

// Typical is libfvad's mode 3 on a film's mixed audio as Timing sync
// expects it. Harsh is a loud score under quiet speech.
var (
	Typical = VAD{Miss: 0.08, Music: 0.04}
	Harsh   = VAD{Miss: 0.3, Music: 0.15}
)

// Heard is the schedule as v hears it. Each run it hears starts up to
// 30 ms late and holds 20 to 90 ms past its end, libfvad's mode-3
// hangover.
func (s Schedule) Heard(seed uint64, v VAD) Schedule {
	r := source(seed)
	out := Schedule{Length: s.Length}
	for _, run := range s.Runs {
		if r.Float64() < v.Miss {
			continue
		}
		out.Runs = append(out.Runs, Run{
			Start: run.Start + uniform(r, 0, 30*time.Millisecond),
			End:   run.End + uniform(r, 20*time.Millisecond, 90*time.Millisecond),
		})
	}
	meanGap := 1.35 * float64(time.Second) / v.Music
	for at := time.Duration(0); v.Music > 0; {
		at += time.Duration(r.ExpFloat64() * meanGap)
		end := at + uniform(r, 200*time.Millisecond, 2500*time.Millisecond)
		if end > s.Length {
			break
		}
		out.Runs = append(out.Runs, Run{at, end})
	}
	slices.SortFunc(out.Runs, func(a, b Run) int { return int(a.Start - b.Start) })
	merged := out.Runs[:0]
	for _, run := range out.Runs {
		if n := len(merged); n > 0 && run.Start <= merged[n-1].End {
			merged[n-1].End = max(merged[n-1].End, run.End)
			continue
		}
		merged = append(merged, run)
	}
	out.Runs = merged
	return out
}

// Cues is a subtitle a person timed to the schedule: a cue per run, some
// runs merged and some split, each edge off by up to 200 ms either way,
// a credit line first and last, and a music cue in some long gaps. The
// jitter has zero mean, so the true timing is where the cues overlap the
// speech best.
func (s Schedule) Cues(seed uint64) []timing.Span {
	r := source(seed)
	const jitter = 200 * time.Millisecond
	cues := []timing.Span{{Start: 500 * time.Millisecond, End: 3 * time.Second, Skip: true}}
	runs := s.Runs
	for i := 0; i < len(runs); i++ {
		run := runs[i]
		if i+1 < len(runs) && runs[i+1].Start-run.End < 1500*time.Millisecond && r.Float64() < 0.15 {
			i++
			run.End = runs[i].End
		}
		parts := []Run{run}
		if run.End-run.Start > 2500*time.Millisecond && r.Float64() < 0.15 {
			mid := (run.Start + run.End) / 2
			parts = []Run{{run.Start, mid - 50*time.Millisecond}, {mid + 50*time.Millisecond, run.End}}
		}
		for _, p := range parts {
			start := max(p.Start+uniform(r, -jitter, jitter), 0)
			end := max(p.End+uniform(r, -jitter, jitter), start+200*time.Millisecond)
			cues = append(cues, timing.Span{Start: start, End: end})
		}
		if i+1 < len(runs) && runs[i+1].Start-run.End > 4*time.Second && r.Float64() < 0.1 {
			cues = append(cues, timing.Span{Start: run.End + time.Second, End: runs[i+1].Start - time.Second, Skip: true})
		}
	}
	return append(cues, timing.Span{Start: s.Length - 5*time.Second, End: s.Length - 2*time.Second, Skip: true})
}

// Shift moves every cue by d.
func Shift(spans []timing.Span, d time.Duration) []timing.Span {
	return ShiftAfter(spans, math.MinInt64, d)
}

// ShiftAfter moves the cues starting at or after at by d, as an ad break
// or an added scene does.
func ShiftAfter(spans []timing.Span, at, d time.Duration) []timing.Span {
	out := slices.Clone(spans)
	for i := range out {
		if out[i].Start >= at {
			out[i].Start += d
			out[i].End += d
		}
	}
	return out
}

// Scale multiplies every time by ratio, as a framerate conversion does.
func Scale(spans []timing.Span, ratio float64) []timing.Span {
	out := slices.Clone(spans)
	for i := range out {
		out[i].Start = time.Duration(math.Round(ratio * float64(out[i].Start)))
		out[i].End = time.Duration(math.Round(ratio * float64(out[i].End)))
	}
	return out
}

// Drop removes about share of the cues.
func Drop(spans []timing.Span, seed uint64, share float64) []timing.Span {
	r := source(seed)
	return slices.DeleteFunc(slices.Clone(spans), func(timing.Span) bool { return r.Float64() < share })
}

func uniform(r *rand.Rand, lo, hi time.Duration) time.Duration {
	return lo + time.Duration(r.Float64()*float64(hi-lo))
}

func source(seed uint64) *rand.Rand { return rand.New(&splitmix{seed}) } //nolint:gosec // seeded on purpose so fixtures replay

// splitmix is SplitMix64, a seeded source that needs no process-wide
// randomness.
type splitmix struct{ x uint64 }

func (s *splitmix) Uint64() uint64 {
	s.x += 0x9e3779b97f4a7c15
	z := s.x
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return z ^ (z >> 31)
}
