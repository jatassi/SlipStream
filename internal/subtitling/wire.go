package subtitling

import (
	"errors"
	"time"

	"github.com/jatassi/slipstream/internal/calls"
	"github.com/jatassi/slipstream/internal/language"
	subtitlesv1 "github.com/jatassi/slipstream/internal/pb/slipstream/subtitles/v1"
	"github.com/jatassi/slipstream/internal/subtitles/cues"
	"github.com/jatassi/slipstream/internal/subtitles/timing"
)

var outcomeKinds = map[calls.OutcomeKind]subtitlesv1.OutcomeKind{
	calls.OK:          subtitlesv1.OutcomeKind_OUTCOME_KIND_OK,
	calls.NotFound:    subtitlesv1.OutcomeKind_OUTCOME_KIND_NOT_FOUND,
	calls.Refused:     subtitlesv1.OutcomeKind_OUTCOME_KIND_REFUSED,
	calls.Limited:     subtitlesv1.OutcomeKind_OUTCOME_KIND_LIMITED,
	calls.Capped:      subtitlesv1.OutcomeKind_OUTCOME_KIND_CAPPED,
	calls.Unreachable: subtitlesv1.OutcomeKind_OUTCOME_KIND_UNREACHABLE,
}

func outcomesToWire(os []calls.Outcome) []*subtitlesv1.Outcome {
	out := make([]*subtitlesv1.Outcome, len(os))
	for i, o := range os {
		w := &subtitlesv1.Outcome{Provider: string(o.Provider), Kind: outcomeKinds[o.Kind], Words: o.Words, AtUnixMicro: o.At.UnixMicro()}
		if !o.Until.IsZero() {
			w.UntilUnixMicro = o.Until.UnixMicro()
		}
		if a := o.Allowance; a != nil {
			w.Allowance = &subtitlesv1.Allowance{Downloads: windowToWire(a.Downloads), Searches: windowToWire(a.Searches)}
		}
		out[i] = w
	}
	return out
}

func windowToWire(w *calls.Window) *subtitlesv1.Window {
	if w == nil {
		return nil
	}
	return &subtitlesv1.Window{Used: int64(w.Used), Limit: int64(w.Limit), Remaining: int64(w.Remaining), ResetsUnixMicro: w.Resets.UnixMicro(), Exact: w.Exact}
}

func outcomesOf(ws []*subtitlesv1.Outcome) []calls.Outcome {
	out := make([]calls.Outcome, len(ws))
	for i, w := range ws {
		o := calls.Outcome{Provider: calls.ID(w.GetProvider()), Words: w.GetWords(), At: time.UnixMicro(w.GetAtUnixMicro())}
		for k, v := range outcomeKinds {
			if v == w.GetKind() {
				o.Kind = k
			}
		}
		if u := w.GetUntilUnixMicro(); u != 0 {
			o.Until = time.UnixMicro(u)
		}
		if a := w.GetAllowance(); a != nil {
			o.Allowance = &calls.Allowance{Downloads: windowOf(a.GetDownloads()), Searches: windowOf(a.GetSearches())}
		}
		out[i] = o
	}
	return out
}

func windowOf(w *subtitlesv1.Window) *calls.Window {
	if w == nil {
		return nil
	}
	return &calls.Window{Used: int(w.GetUsed()), Limit: int(w.GetLimit()), Remaining: int(w.GetRemaining()), Resets: time.UnixMicro(w.GetResetsUnixMicro()), Exact: w.GetExact()}
}

var kinds = map[Kind]subtitlesv1.Subject_Kind{Feature: subtitlesv1.Subject_KIND_FEATURE, Episode: subtitlesv1.Subject_KIND_EPISODE}

func subjectToWire(s Subject) *subtitlesv1.Subject {
	ids := map[string]string{}
	for k, v := range map[string]string{"imdb": s.IMDb, "tmdb": s.TMDB, "tvdb": s.TVDB} {
		if v != "" {
			ids[k] = v
		}
	}
	return &subtitlesv1.Subject{Kind: kinds[s.Kind], Ids: ids, Title: s.Title, Year: int32(s.Year), Season: int32(s.Season), Episodes: int32s(s.Episodes)} //nolint:gosec // G115: a year and a season number are far under 2³¹
}

func subjectOf(w *subtitlesv1.Subject) (Subject, error) {
	s := Subject{IMDb: w.GetIds()["imdb"], TMDB: w.GetIds()["tmdb"], TVDB: w.GetIds()["tvdb"], Title: w.GetTitle(), Year: int(w.GetYear()), Season: int(w.GetSeason()), Episodes: ints(w.GetEpisodes())}
	for k, v := range kinds {
		if v == w.GetKind() {
			s.Kind = k
		}
	}
	if s.Kind == 0 {
		return Subject{}, errors.New("the subject has no kind")
	}
	return s, nil
}

func queryToWire(q Query) *subtitlesv1.Query {
	return &subtitlesv1.Query{Language: q.Language.String(), Forced: q.Forced}
}

func queryOf(s *subtitlesv1.Subject, w *subtitlesv1.Query) (Query, error) {
	subject, err := subjectOf(s)
	if err != nil {
		return Query{}, err
	}
	lang, err := language.Parse(w.GetLanguage())
	if err != nil {
		return Query{}, err
	}
	return Query{Subject: subject, Language: lang, Forced: w.GetForced()}, nil
}

func offerToWire(o Offer) *subtitlesv1.Offer {
	return &subtitlesv1.Offer{
		Provider: string(o.Provider), Key: o.Key, Releases: o.Releases, Language: o.Language.String(), Forced: o.Forced, Hi: o.HI, Ai: o.AI,
		Pack: o.Pack, Season: int32(o.Season), Episodes: int32s(o.Episodes), Downloads: int64(o.Downloads), Rating: o.Rating, Image: o.Image, Page: o.Page, //nolint:gosec // G115: a season number is far under 2³¹
	}
}

func offerOf(w *subtitlesv1.Offer) (Offer, error) {
	lang, err := language.Parse(w.GetLanguage())
	if err != nil {
		return Offer{}, err
	}
	return Offer{
		Provider: calls.ID(w.GetProvider()), Key: w.GetKey(), Releases: w.GetReleases(), Language: lang, Forced: w.GetForced(), HI: w.GetHi(), AI: w.GetAi(),
		Pack: w.GetPack(), Season: int(w.GetSeason()), Episodes: ints(w.GetEpisodes()), Downloads: int(w.GetDownloads()), Rating: w.GetRating(), Image: w.GetImage(), Page: w.GetPage(),
	}, nil
}

var framerates = map[timing.Framerates]subtitlesv1.Gate_Framerates{
	timing.NoRatios:       subtitlesv1.Gate_FRAMERATES_NONE,
	timing.StandardRatios: subtitlesv1.Gate_FRAMERATES_STANDARD,
	timing.RangeRatios:    subtitlesv1.Gate_FRAMERATES_RANGE,
}

func gateToWire(g timing.Gate) *subtitlesv1.Gate {
	return &subtitlesv1.Gate{MatchPercent: int32(g.Match*100 + 0.5), MaxOffsetMs: g.MaxOffset.Milliseconds(), Framerates: framerates[g.Framerates]}
}

func gateOf(w *subtitlesv1.Gate) (timing.Gate, error) {
	g := timing.Gate{Match: float64(w.GetMatchPercent()) / 100, MaxOffset: time.Duration(w.GetMaxOffsetMs()) * time.Millisecond}
	for k, v := range framerates {
		if v == w.GetFramerates() {
			g.Framerates = k
		}
	}
	if g.Framerates == 0 {
		return timing.Gate{}, errors.New("the gate has no framerates")
	}
	return g, nil
}

func constantsToWire(k timing.Constants) *subtitlesv1.Constants {
	return &subtitlesv1.Constants{
		SplitPenalty: k.SplitPenalty, LengthPenalty: k.LengthPenalty, GuardMaxMs: k.GuardMax.Milliseconds(), Prominence: k.Prominence,
		ProminenceGapMs: k.ProminenceGap.Milliseconds(), Coverage: k.Coverage, GoldenTolerance: k.GoldenTolerance, MaxSegments: int32(k.MaxSegments), //nolint:gosec // G115: MaxSegments is a handful
		MinSegmentMs: k.MinSegment.Milliseconds(), CheckSlackMs: k.CheckSlack.Milliseconds(),
	}
}

func constantsOf(w *subtitlesv1.Constants) timing.Constants {
	ms := func(v int64) time.Duration { return time.Duration(v) * time.Millisecond }
	return timing.Constants{
		SplitPenalty: w.GetSplitPenalty(), LengthPenalty: w.GetLengthPenalty(), GuardMax: ms(w.GetGuardMaxMs()), Prominence: w.GetProminence(),
		ProminenceGap: ms(w.GetProminenceGapMs()), Coverage: w.GetCoverage(), GoldenTolerance: w.GetGoldenTolerance(), MaxSegments: int(w.GetMaxSegments()),
		MinSegment: ms(w.GetMinSegmentMs()), CheckSlack: ms(w.GetCheckSlackMs()),
	}
}

var timingVerdicts = map[timing.Verdict]subtitlesv1.Timing{
	timing.Passed:     subtitlesv1.Timing_TIMING_PASSED,
	timing.LowMatch:   subtitlesv1.Timing_TIMING_LOW_MATCH,
	timing.PastOffset: subtitlesv1.Timing_TIMING_PAST_OFFSET,
	timing.Ambiguous:  subtitlesv1.Timing_TIMING_AMBIGUOUS,
	timing.Uncovered:  subtitlesv1.Timing_TIMING_UNCOVERED,
	timing.NoSignal:   subtitlesv1.Timing_TIMING_NO_SIGNAL,
}

func fitToWire(f timing.Fit) *subtitlesv1.Fit {
	w := &subtitlesv1.Fit{Match: f.Match, Raw: f.Raw, Density: f.Density, Prominence: f.Prominence, Coverage: f.Coverage, Timing: timingVerdicts[f.Verdict]}
	for _, s := range f.Segments {
		w.Segments = append(w.Segments, &subtitlesv1.Segment{FirstCue: int32(s.First), OffsetMs: s.Offset.Milliseconds(), Ratio: s.Ratio}) //nolint:gosec // G115: a cue index is far under 2³¹
	}
	return w
}

func fitOf(w *subtitlesv1.Fit) timing.Fit {
	f := timing.Fit{Match: w.GetMatch(), Raw: w.GetRaw(), Density: w.GetDensity(), Prominence: w.GetProminence(), Coverage: w.GetCoverage()}
	for k, v := range timingVerdicts {
		if v == w.GetTiming() {
			f.Verdict = k
		}
	}
	for _, s := range w.GetSegments() {
		f.Segments = append(f.Segments, timing.Segment{First: int(s.GetFirstCue()), Offset: time.Duration(s.GetOffsetMs()) * time.Millisecond, Ratio: s.GetRatio()})
	}
	return f
}

var verdicts = map[Verdict]subtitlesv1.Verdict{
	Passed:       subtitlesv1.Verdict_VERDICT_PASSED,
	FailedTiming: subtitlesv1.Verdict_VERDICT_FAILED_TIMING,
	Unreadable:   subtitlesv1.Verdict_VERDICT_UNREADABLE,
	Image:        subtitlesv1.Verdict_VERDICT_IMAGE,
	WrongEpisode: subtitlesv1.Verdict_VERDICT_WRONG_EPISODE,
	TooLarge:     subtitlesv1.Verdict_VERDICT_TOO_LARGE,
	Unfetched:    subtitlesv1.Verdict_VERDICT_UNFETCHED,
}

func verdictOf(w subtitlesv1.Verdict) Verdict {
	for k, v := range verdicts {
		if v == w {
			return k
		}
	}
	return 0
}

var formats = map[cues.Format]string{cues.SRT: "srt", cues.ASS: "ass", cues.SSA: "ssa"}

func formatOf(s string) cues.Format {
	for k, v := range formats {
		if v == s {
			return k
		}
	}
	return 0
}

func int32s(v []int) []int32 {
	out := make([]int32, len(v))
	for i, n := range v {
		out[i] = int32(n) //nolint:gosec // G115: episode numbers are far under 2³¹
	}
	return out
}

func ints(v []int32) []int {
	out := make([]int, len(v))
	for i, n := range v {
		out[i] = int(n)
	}
	return out
}
