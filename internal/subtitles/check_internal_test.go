package subtitles

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/language"
	"github.com/jatassi/slipstream/internal/subtitles/cues"
	"github.com/jatassi/slipstream/internal/subtitles/timing"
	"github.com/jatassi/slipstream/internal/subtitles/timing/timingtest"
	"github.com/jatassi/slipstream/internal/subtitling"
	"github.com/jatassi/slipstream/internal/targets"
)

// srt writes spans as an SRT subtitle, one numbered line of text each.
func srt(spans []timing.Span) []byte {
	var b strings.Builder
	stamp := func(d time.Duration) string {
		ms := d.Milliseconds()
		return fmt.Sprintf("%02d:%02d:%02d,%03d", ms/3600000, ms/60000%60, ms/1000%60, ms%1000)
	}
	for i, s := range spans {
		fmt.Fprintf(&b, "%d\n%s --> %s\nLine %d of the film.\n\n", i+1, stamp(s.Start), stamp(s.End), i+1)
	}
	return []byte(b.String())
}

// film is a ten-minute file's map and a subtitle 2 s late for it, and an
// honest holder's passing answer for that subtitle.
type film struct {
	ref    timing.Reference
	doc    cues.Doc
	answer subtitling.Fetched
	gate   timing.Gate
	k      timing.Constants
}

func newFilm(t *testing.T) film {
	t.Helper()
	sched := timingtest.NewSchedule(7, 10*time.Minute, 0.4)
	f := film{ref: sched.Reference(), k: timing.DefaultConstants(), gate: timing.Gate{Match: 0.3, MaxOffset: 30 * time.Second, Framerates: timing.StandardRatios}}
	doc, err := cues.Parse(srt(timingtest.Shift(sched.Cues(7), 2*time.Second)), english, 0, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	f.doc = doc
	fit := timing.Align(f.ref, subtitling.Spans(doc), f.gate, f.k)
	if fit.Verdict != timing.Passed {
		t.Fatalf("the honest subtitle didn't align: verdict %d", fit.Verdict)
	}
	f.answer = subtitling.Fetched{Key: "c1", Verdict: subtitling.Passed, Format: cues.SRT, Original: doc.Bytes(), Retimed: f.retime(t, fit), Claimed: fit}
	return f
}

var english = func() language.Code { l, _ := language.Parse("en"); return l }()

// retime is the doc's bytes under fit, as a holder writes them.
func (f film) retime(t *testing.T, fit timing.Fit) []byte {
	t.Helper()
	return f.restamp(t, timing.Apply(subtitling.Spans(f.doc), fit, f.doc.Unit()))
}

func (f film) restamp(t *testing.T, spans []timing.Span) []byte {
	t.Helper()
	ts := make([]cues.Times, len(spans))
	for i, s := range spans {
		ts[i] = cues.Times{Start: s.Start, End: s.End}
	}
	d, err := f.doc.Retimed(ts)
	if err != nil {
		t.Fatal(err)
	}
	return d.Bytes()
}

func (f film) asked(offer *subtitling.Offer, group string) asked {
	t := Tail{Language: english}
	return asked{sidecar: Sidecar{Tail: t}, offer: offer, group: group, hi: targets.Allow, bound: boundOf(0, f.ref.Frames(), time.Minute)}
}

var offer = subtitling.Offer{Provider: "subdl", Key: "c1", Releases: []string{"Heat.1995.1080p.BluRay.x264-GROUP"}, Language: english}

func TestTheCheckRefusesEveryAnswerTheMapDoesntSupport(t *testing.T) {
	f := newFilm(t)
	held := subtitling.Spans(parseDoc(t, f.answer.Retimed))
	for _, c := range []struct {
		name  string
		limit int
		bound time.Duration
		edit  func(a *subtitling.Fetched)
	}{
		{name: "the wrong candidate", edit: func(a *subtitling.Fetched) { a.Key = "c2" }},
		{name: "a file over the cap", limit: len(f.answer.Retimed) - 1},
		{name: "the wrong format", edit: func(a *subtitling.Fetched) { a.Format = cues.ASS }},
		{name: "changed text", edit: func(a *subtitling.Fetched) {
			a.Retimed = bytes.Replace(a.Retimed, []byte("Line 9 of"), []byte("Line 9 in"), 1)
		}},
		{name: "a cue past the duration and its margin", bound: time.Minute},
		{name: "a forged segment", edit: func(a *subtitling.Fetched) {
			moved := slices.Clone(held)
			moved[len(moved)/2].Start += f.doc.Unit() + f.k.CheckSlack + 2*time.Millisecond
			a.Retimed = f.restamp(t, moved)
		}},
		{name: "nine segments", edit: func(a *subtitling.Fetched) {
			a.Claimed.Segments = nil
			for s := range 9 {
				a.Claimed.Segments = append(a.Claimed.Segments, timing.Segment{First: s * f.doc.Len() / 9, Ratio: 1})
			}
		}},
		{name: "a ratio off the set", edit: func(a *subtitling.Fetched) {
			fit := timing.Fit{Segments: []timing.Segment{{Ratio: 1.01, Offset: -2 * time.Second}}}
			a.Retimed, a.Claimed.Segments = f.retime(t, fit), fit.Segments
		}},
		{name: "an offset past the bound", edit: func(a *subtitling.Fetched) {
			fit := timing.Fit{Segments: []timing.Segment{{Ratio: 1, Offset: 40 * time.Second}}}
			a.Retimed, a.Claimed.Segments = f.retime(t, fit), fit.Segments
		}},
		{name: "a forged pass the map doesn't support", edit: func(a *subtitling.Fetched) {
			fit := timing.Fit{Segments: []timing.Segment{{Ratio: 1, Offset: 20 * time.Second}}, Match: 0.97}
			a.Retimed, a.Claimed = f.retime(t, fit), fit
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			a, ans, limit := f.asked(&offer, "GROUP"), f.answer, 1<<20
			if c.edit != nil {
				c.edit(&ans)
			}
			if c.limit != 0 {
				limit = c.limit
			}
			if c.bound != 0 {
				a.bound = c.bound
			}
			_, err := check(a, f.ref, ans, f.gate, f.k, limit)
			if !errors.Is(err, timing.ErrForged) {
				t.Fatalf("the check took %s: %v", c.name, err)
			}
		})
	}
}

func parseDoc(t *testing.T, b []byte) cues.Doc {
	t.Helper()
	d, err := cues.Parse(b, english, 0, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestTheCheckStoresTheControllersNumbers(t *testing.T) {
	f := newFilm(t)
	ans := f.answer
	ans.Claimed.Match, ans.Claimed.Raw = 0.99, 0.99
	got, err := check(f.asked(&offer, "GROUP"), f.ref, ans, f.gate, f.k, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	want := percent(f.answer.Claimed.Match)
	if fit := got.Sidecar().Fit; fit == nil || fit.Match < want-1 || fit.Match > want+1 || (fit.Offset+2*time.Second).Abs() > 50*time.Millisecond || fit.Ratio != 1 {
		t.Fatalf("the Sidecar's timing is %+v, want the recomputed %d%% near -2s, not the claimed 99%%", got.Sidecar().Fit, want)
	}
	if !bytes.Equal(got.bytes, f.answer.Retimed) || got.Sidecar().TailText != "en.srt" || got.Sidecar().Source != FromProvider {
		t.Fatalf("the Sidecar is %+v, want the retimed bytes as en.srt from the provider", got.Sidecar())
	}
}

func TestOnlyTheControllersReadingOfTheReleaseMakesASidecarFinal(t *testing.T) {
	f := newFilm(t)
	ai := offer
	ai.AI = true
	hi := offer
	hi.HI = true
	for _, c := range []struct {
		name  string
		offer subtitling.Offer
		group string
		hi    targets.HI
		final bool
	}{
		{"made for the file's group", offer, "GROUP", targets.Allow, true},
		{"made for another group", offer, "OTHER", targets.Allow, false},
		{"a file with no group", offer, "", targets.Allow, false},
		{"machine translated", ai, "GROUP", targets.Allow, false},
		{"hearing impaired the want avoids", hi, "GROUP", targets.Avoid, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			a := f.asked(&c.offer, c.group)
			a.hi = c.hi
			got, err := check(a, f.ref, f.answer, f.gate, f.k, 1<<20)
			if err != nil {
				t.Fatal(err)
			}
			if got.Sidecar().Final != c.final {
				t.Fatalf("final = %t, want %t", got.Sidecar().Final, c.final)
			}
		})
	}
}
