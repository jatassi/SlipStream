package timing_test

import (
	"fmt"
	"math"
	"runtime"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/subtitles/timing"
	"github.com/jatassi/slipstream/internal/subtitles/timing/timingtest"
)

var k = timing.DefaultConstants()

func gate(set timing.Framerates, maxOffset time.Duration) timing.Gate {
	return timing.Gate{Match: float64(timing.DefaultMatchPercent) / 100, MaxOffset: maxOffset, Framerates: set}
}

var standardGate = gate(timing.StandardRatios, 30*time.Second)

// standard is the six ratios besides 1 that StandardRatios tries.
var standard = []float64{24 / 23.976, 25 / 23.976, 25.0 / 24, 23.976 / 24, 23.976 / 25, 24.0 / 25}

type film struct {
	ref  timing.Reference
	cues []timing.Span
}

// newFilm is a two-hour film at 40% speech with an exact map, so a
// recovered offset has no VAD bias.
func newFilm(seed uint64) film {
	s := timingtest.NewSchedule(seed, 2*time.Hour, 0.4)
	return film{s.Reference(), s.Cues(seed + 1)}
}

func within(t *testing.T, what string, got, want, tol time.Duration) {
	t.Helper()
	if d := got - want; d < -tol || d > tol {
		t.Errorf("%s = %v, want %v within %v", what, got, want, tol)
	}
}

func passed(t *testing.T, fit timing.Fit) {
	t.Helper()
	if fit.Verdict != timing.Passed {
		t.Fatalf("verdict %d, want Passed: match %.3f prominence %.3f coverage %.3f, %d segments", fit.Verdict, fit.Match, fit.Prominence, fit.Coverage, len(fit.Segments))
	}
}

func TestOffsets(t *testing.T) {
	g := gate(timing.StandardRatios, 60*time.Second)
	for _, shift := range []time.Duration{2500 * time.Millisecond, -37 * time.Second} {
		f := newFilm(1)
		fit := timing.Align(f.ref, timingtest.Shift(f.cues, shift), g, k)
		passed(t, fit)
		if len(fit.Segments) != 1 || fit.Ratio() != 1 {
			t.Errorf("shift %v: %+v, want one segment at ratio 1", shift, fit.Segments)
		}
		within(t, "offset", fit.Offset(), -shift, timing.Frame)
	}
}

func TestRatios(t *testing.T) {
	for i, r := range standard {
		f := newFilm(uint64(10 + i))
		fit := timing.Align(f.ref, timingtest.Scale(f.cues, r), standardGate, k)
		passed(t, fit)
		if math.Abs(fit.Ratio()*r-1) > 1e-12 {
			t.Errorf("ratio %.6f: found %.6f, want its inverse", r, fit.Ratio())
		}
		within(t, "offset", fit.Offset(), 0, timing.Frame)
	}
	f := newFilm(20)
	fit := timing.Align(f.ref, timingtest.Scale(f.cues, 1/1.0331), gate(timing.RangeRatios, 30*time.Second), k)
	passed(t, fit)
	if math.Abs(fit.Ratio()-1.0331) > 5e-4 {
		t.Errorf("range ratio %.5f, want 1.0331 within 5e-4", fit.Ratio())
	}
}

func TestSplit(t *testing.T) {
	f := newFilm(30)
	at := 45 * time.Minute
	fit := timing.Align(f.ref, timingtest.ShiftAfter(f.cues, at, 12*time.Second), standardGate, k)
	passed(t, fit)
	if len(fit.Segments) != 2 {
		t.Fatalf("segments %+v, want two", fit.Segments)
	}
	within(t, "first offset", fit.Segments[0].Offset, 0, timing.Frame)
	within(t, "second offset", fit.Segments[1].Offset, -12*time.Second, timing.Frame)
	if first := f.cues[fit.Segments[1].First].Start; first < at-time.Minute || first > at+time.Minute {
		t.Errorf("second segment starts at %v, want about %v", first, at)
	}
}

// TestPercentiles holds Timing sync to alass's measured accuracy: after
// sync, 50% of cue starts within 50 ms of where the subtitle had them
// before it was put out of time, 80% within 100 ms, 90% within 400 ms and
// 95% within 800 ms. The maps are as a VAD hears the films.
func TestPercentiles(t *testing.T) {
	transforms := []func(i int, cues []timing.Span) []timing.Span{
		func(i int, c []timing.Span) []timing.Span {
			return timingtest.Shift(c, time.Duration(i%50-25)*time.Second)
		},
		func(i int, c []timing.Span) []timing.Span {
			return timingtest.Shift(timingtest.Scale(c, standard[i%len(standard)]), time.Duration(i%20-10)*time.Second)
		},
		func(i int, c []timing.Span) []timing.Span {
			return timingtest.ShiftAfter(timingtest.Shift(c, 3*time.Second), time.Duration(30+i)*time.Minute, -9*time.Second)
		},
		func(i int, c []timing.Span) []timing.Span {
			return timingtest.Shift(c, -time.Duration(i%25)*time.Second)
		},
	}
	const films = 40
	errs := make([][]time.Duration, films)
	forEach(films, func(i int, seed uint64) {
		s := timingtest.NewSchedule(100+seed, 2*time.Hour, 0.3+0.2*float64(i%5)/4)
		truth := s.Cues(200 + seed)
		if i%4 == 3 {
			truth = timingtest.Drop(truth, seed, 0.2)
		}
		wrong := transforms[i%4](i, truth)
		fit := timing.Align(s.Heard(300+seed, timingtest.Typical).Reference(), wrong, standardGate, k)
		for c, sp := range timing.Apply(wrong, fit, time.Millisecond) {
			if !sp.Skip {
				errs[i] = append(errs[i], (sp.Start - truth[c].Start).Abs())
			}
		}
	})
	all := slices.Concat(errs...)
	for _, p := range []struct {
		within time.Duration
		share  float64
	}{{50 * time.Millisecond, 0.5}, {100 * time.Millisecond, 0.8}, {400 * time.Millisecond, 0.9}, {800 * time.Millisecond, 0.95}} {
		n := 0
		for _, e := range all {
			if e <= p.within {
				n++
			}
		}
		share := float64(n) / float64(len(all))
		t.Logf("%.1f%% of %d cue starts within %v", 100*share, len(all), p.within)
		if share < p.share {
			t.Errorf("%.1f%% within %v, want at least %.0f%%", 100*share, p.within, 100*p.share)
		}
	}
}

func TestFailures(t *testing.T) {
	verdict := func(t *testing.T, fit timing.Fit, want ...timing.Verdict) {
		t.Helper()
		if !slices.Contains(want, fit.Verdict) {
			t.Errorf("verdict %d, want one of %v: match %.3f prominence %.3f coverage %.3f", fit.Verdict, want, fit.Match, fit.Prominence, fit.Coverage)
		}
	}
	t.Run("an unrelated schedule", func(t *testing.T) {
		f := newFilm(40)
		other := newFilm(41)
		verdict(t, timing.Align(f.ref, other.cues, standardGate, k), timing.LowMatch, timing.Ambiguous)
	})
	t.Run("a file twice the subtitle's length", func(t *testing.T) {
		s := timingtest.NewSchedule(42, 2*time.Hour, 0.4)
		half := slices.DeleteFunc(s.Cues(43), func(sp timing.Span) bool { return sp.End > time.Hour })
		verdict(t, timing.Align(s.Reference(), half, standardGate, k), timing.Uncovered)
	})
	for _, d := range []float64{0.3, 0.4, 0.5} {
		t.Run(fmt.Sprintf("random placement at %.0f%% speech", 100*d), func(t *testing.T) {
			s := timingtest.NewSchedule(uint64(44+100*d), 2*time.Hour, d)
			random := timingtest.NewSchedule(uint64(45+100*d), 2*time.Hour, 0.4).Cues(46)
			verdict(t, timing.Align(s.Heard(47, timingtest.Typical).Reference(), random, standardGate, k), timing.LowMatch)
		})
	}
	t.Run("a shift past the bound", func(t *testing.T) {
		f := newFilm(48)
		fit := timing.Align(f.ref, timingtest.Shift(f.cues, 45*time.Second), standardGate, k)
		verdict(t, fit, timing.PastOffset)
		within(t, "offset", fit.Offset(), -45*time.Second, timing.Frame*5)
	})
	t.Run("no ratios for a converted file", func(t *testing.T) {
		f := newFilm(49)
		verdict(t, timing.Align(f.ref, timingtest.Scale(f.cues, 25/23.976), gate(timing.NoRatios, 30*time.Second), k), timing.LowMatch)
	})
	t.Run("silence", func(t *testing.T) {
		f := newFilm(50)
		verdict(t, timing.Align(timing.NewReference(make([]uint64, 1000), 64000), f.cues, standardGate, k), timing.NoSignal)
	})
	t.Run("no voiced cue", func(t *testing.T) {
		f := newFilm(51)
		verdict(t, timing.Align(f.ref, []timing.Span{{Start: time.Second, End: 2 * time.Second, Skip: true}}, standardGate, k), timing.NoSignal)
	})
}

// TestFalsePassRate times 1,000 subtitles of unrelated schedules against
// 22-minute episodes, the shortest files and so the noisiest matches, in
// three speech density bands, and requires under 0.5% of them to pass at
// the shipped default. It times 300 of the episodes' own subtitles under a
// harsh VAD and requires 99% of those to pass. It logs the numbers that
// set the default, and the pass rates other defaults would have.
func TestFalsePassRate(t *testing.T) {
	trials, honest := falsePassTrials()
	bands := []float64{0.3, 0.4, 0.5}
	type result struct{ random, harsh timing.Fit }
	results := make([]result, trials)
	forEach(trials, func(i int, n uint64) {
		seed := 1_000_000 + 10*n
		s := timingtest.NewSchedule(seed, 22*time.Minute, bands[i%3])
		other := timingtest.NewSchedule(seed+1, 22*time.Minute, bands[(i/3)%3])
		results[i].random = timing.Align(s.Heard(seed+2, timingtest.Typical).Reference(), other.Cues(seed+3), standardGate, k)
		if i < honest {
			own := timingtest.Shift(s.Cues(seed+4), time.Duration(i%50-25)*time.Second)
			results[i].harsh = timing.Align(s.Heard(seed+5, timingtest.Harsh).Reference(), own, standardGate, k)
		}
	})
	for b, d := range bands {
		var random, harsh []timing.Fit
		for i := b; i < trials; i += 3 {
			random = append(random, results[i].random)
			if i < honest {
				harsh = append(harsh, results[i].harsh)
			}
		}
		t.Logf("%.0f%% speech: random match p50 %.3f, p99.5 %.3f, max %.3f over %d; harsh VAD min %.3f over %d",
			100*d, quantile(random, 0.5), quantile(random, 0.995), quantile(random, 1), len(random), quantile(harsh, 0), len(harsh))
		for _, at := range []float64{0.25, 0.3, 0.35, 0.4, 0.5} {
			t.Logf("%.0f%% speech at a %.0f%% match: %.2f%% of random and %.2f%% of harsh pass", 100*d, 100*at, 100*passRate(random, at), 100*passRate(harsh, at))
		}
		if r := passRate(random, standardGate.Match); r >= 0.005 {
			t.Errorf("%.0f%% speech: %.2f%% of random subtitles passed, want under 0.5%%", 100*d, 100*r)
		}
		if r := passRate(harsh, standardGate.Match); r < 0.99 {
			t.Errorf("%.0f%% speech: %.2f%% of honest subtitles passed under a harsh VAD, want 99%%", 100*d, 100*r)
		}
	}
}

func TestNegativeHead(t *testing.T) {
	f := newFilm(60)
	head := []timing.Span{
		{Start: 100 * time.Millisecond, End: 800 * time.Millisecond},
		{Start: time.Second, End: 1900 * time.Millisecond},
		{Start: 2 * time.Second, End: 2600 * time.Millisecond, Skip: true},
	}
	late := slices.Concat(head, timingtest.Shift(f.cues, 3*time.Second))
	fit := timing.Align(f.ref, late, standardGate, k)
	passed(t, fit)
	within(t, "offset", fit.Offset(), -3*time.Second, timing.Frame)
	retimed := timing.Apply(late, fit, 10*time.Millisecond)
	if len(retimed) != len(late) {
		t.Fatalf("%d cues retimed, want %d", len(retimed), len(late))
	}
	for i := range head {
		if retimed[i].Start != 0 || retimed[i].End < 10*time.Millisecond || retimed[i].Skip != head[i].Skip {
			t.Errorf("head cue %d retimed to %+v, want a start of 0 and its own Skip", i, retimed[i])
		}
	}
	within(t, "first film cue", retimed[len(head)].Start, f.cues[0].Start, 2*timing.Frame)
}

func TestMemory(t *testing.T) {
	s := timingtest.NewSchedule(70, 2*time.Hour, 0.32)
	ref := s.Reference()
	cues := timingtest.Shift(s.Cues(71), 4*time.Second)
	if len(cues) < 1500 {
		t.Fatalf("%d cues, want a 1,500-cue film", len(cues))
	}
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	fit := timing.Align(ref, cues, standardGate, k)
	runtime.ReadMemStats(&after)
	passed(t, fit)
	if used := after.TotalAlloc - before.TotalAlloc; used > 8<<20 {
		t.Errorf("Align of %d cues allocated %.1f MB, want under 8 MB", len(cues), float64(used)/(1<<20))
	} else {
		t.Logf("Align of %d cues allocated %.1f MB", len(cues), float64(used)/(1<<20))
	}
}

func BenchmarkAlign(b *testing.B) {
	f := newFilm(80)
	cues := timingtest.Shift(f.cues, 7*time.Second)
	for b.Loop() {
		timing.Align(f.ref, cues, standardGate, k)
	}
}

// passRate is the share of fits that pass the gate at match.
func passRate(fits []timing.Fit, match float64) float64 {
	n := 0
	for _, f := range fits {
		if f.Match >= match && f.Prominence <= k.Prominence && f.Coverage >= k.Coverage {
			n++
		}
	}
	return float64(n) / float64(len(fits))
}

func quantile(fits []timing.Fit, q float64) float64 {
	v := make([]float64, len(fits))
	for i, f := range fits {
		v[i] = f.Match
	}
	slices.Sort(v)
	return v[int(q*float64(len(v)-1))]
}

// forEach runs fn for i from 0 to n−1 on every CPU, with i as a seed too.
func forEach(n int, fn func(i int, seed uint64)) {
	type job struct {
		i    int
		seed uint64
	}
	var wg sync.WaitGroup
	next := make(chan job)
	for range runtime.GOMAXPROCS(0) {
		wg.Go(func() {
			for j := range next {
				fn(j.i, j.seed)
			}
		})
	}
	for i, seed := 0, uint64(0); i < n; i, seed = i+1, seed+1 {
		next <- job{i, seed}
	}
	close(next)
	wg.Wait()
}
