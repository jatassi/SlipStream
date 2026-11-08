package targets_test

import (
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/formats"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/targets"
)

const mib = 1 << 20

func must[T any](t *testing.T) func(T, error) T {
	return func(v T, err error) T {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
}

func filters(t *testing.T, rules ...any) targets.Filters {
	t.Helper()
	var f targets.Filters
	if len(rules)%3 != 0 {
		t.Fatalf("%d rules; each is an attribute, a mode and values", len(rules))
	}
	for i := 0; i+2 < len(rules); i += 3 {
		a, aok := rules[i].(targets.Attribute)
		m, mok := rules[i+1].(targets.Mode)
		v, vok := rules[i+2].([]string)
		if !aok || !mok || !vok {
			t.Fatalf("rule %d is not an attribute, a mode and values", i/3)
		}
		if err := f.Set(a, m, v); err != nil {
			t.Fatal(err)
		}
	}
	return f
}

// hd ranks discs above web at 1080p, stops at discs, and holds WEB-DL to
// 10 to 40 MiB a minute.
func hd(t *testing.T, f targets.Filters) targets.Target {
	t.Helper()
	return must[targets.Target](t)(targets.NewTarget(targets.Spec{
		Scores:  targets.Scores{Increment: 1},
		Name:    "1080p",
		Tiers:   [][]release.Quality{{release.Bluray1080p, release.Remux1080p}, {release.WEBDL1080p, release.WEBRip1080p}},
		Cutoff:  release.Remux1080p,
		Filters: f,
		Sizes:   map[release.Quality]targets.Size{release.WEBDL1080p: {Min: 10, Max: new(40.0)}},
	}))
}

func webdl(size int64, runtime time.Duration) release.Facts {
	f := release.Facts{Quality: release.Known(release.WEBDL1080p)}
	if size > 0 {
		f.Size = release.Known(size)
	}
	if runtime > 0 {
		f.Coverage.Runtime = release.Known(runtime)
	}
	return f
}

type verdict struct {
	rank      int // -1 when unranked
	meets     bool
	grabbable bool
	reasons   []targets.Reason
}

func verdictOf(v targets.Verdict) verdict {
	out := verdict{rank: -1, meets: v.Meets(), grabbable: v.Grabbable(), reasons: v.Reasons()}
	if r, ok := v.Rank(); ok {
		out.rank = int(r)
	}
	return out
}

var notChecked = targets.Reason{Code: targets.SizeNotChecked}

func TestJudgeRanksQualitiesAndChecksSizesPerMinuteOfCoveredRuntime(t *testing.T) {
	target := hd(t, targets.Filters{})
	for name, c := range map[string]struct {
		facts release.Facts
		want  verdict
	}{
		"an unknown quality": {release.Facts{}, verdict{-1, false, false, []targets.Reason{{Code: targets.QualityNotInTarget}}}},
		"a quality the Target lacks": {
			release.Facts{Quality: release.Known(release.HDTV720p)},
			verdict{-1, false, false, []targets.Reason{{Code: targets.QualityNotInTarget, Value: "hdtv-720p"}}},
		},
		"no size":                    {webdl(0, 100*time.Minute), verdict{1, true, true, []targets.Reason{notChecked}}},
		"no runtime":                 {webdl(2000*mib, 0), verdict{1, true, true, []targets.Reason{notChecked}}},
		"inside the range":           {webdl(2000*mib, 100*time.Minute), verdict{1, true, true, nil}},
		"at the minimum":             {webdl(1000*mib, 100*time.Minute), verdict{1, true, true, nil}},
		"below the minimum":          {webdl(999*mib, 100*time.Minute), verdict{1, true, false, []targets.Reason{{Code: targets.SizeBelow}}}},
		"above the maximum":          {webdl(4001*mib, 100*time.Minute), verdict{1, true, false, []targets.Reason{{Code: targets.SizeAbove}}}},
		"a two-Item pack in range":   {webdl(7000*mib, 200*time.Minute), verdict{1, true, true, nil}},
		"a two-Item pack over range": {webdl(9000*mib, 200*time.Minute), verdict{1, true, false, []targets.Reason{{Code: targets.SizeAbove}}}},
		"a floor with no maximum": {
			release.Facts{Quality: release.Known(release.Bluray1080p), Size: release.Known(int64(100000 * mib)), Coverage: release.Coverage{Runtime: release.Known(100 * time.Minute)}},
			verdict{0, true, true, nil},
		},
		"under a floor": {
			release.Facts{Quality: release.Known(release.Bluray1080p), Size: release.Known(int64(5000 * mib)), Coverage: release.Coverage{Runtime: release.Known(100 * time.Minute)}},
			verdict{0, true, false, []targets.Reason{{Code: targets.SizeBelow}}},
		},
	} {
		if got := verdictOf(targets.Judge(target, targets.Scored{Facts: c.facts}, nil)); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: Judge = %+v, want %+v", name, got, c.want)
		}
	}
}

func TestJudgeAppliesEveryFilterModeAndPassesUnknownFacts(t *testing.T) {
	failed := func(a targets.Attribute, v string) []targets.Reason {
		return []targets.Reason{{Code: targets.FilterFailed, Attribute: a, Value: v}, notChecked}
	}
	passed := []targets.Reason{notChecked}
	with := func(edit func(*release.Facts)) release.Facts {
		f := webdl(0, 0)
		edit(&f)
		return f
	}
	hdr := func(v ...release.HDR) func(*release.Facts) {
		return func(f *release.Facts) { f.HDR = release.Known(v) }
	}
	langs := func(v ...release.Language) func(*release.Facts) {
		return func(f *release.Facts) { f.Languages = release.Known(v) }
	}
	original := func(o release.Language, v ...release.Language) func(*release.Facts) {
		return func(f *release.Facts) { f.Languages, f.Coverage.Original = release.Known(v), release.Known(o) }
	}
	for name, c := range map[string]struct {
		rules []any
		facts release.Facts
		want  []targets.Reason
	}{
		"any passes everything":           {nil, with(hdr("DV")), passed},
		"only, a value in the set":        {[]any{targets.HDR, targets.Only, []string{"HDR10", "DV"}}, with(hdr("DV")), passed},
		"only, one of several in the set": {[]any{targets.HDR, targets.Only, []string{"HDR10"}}, with(hdr("DV", "HDR10")), passed},
		"only, none in the set":           {[]any{targets.HDR, targets.Only, []string{"HDR10"}}, with(hdr("SDR", "HLG")), failed(targets.HDR, "SDR")},
		"only, known to have none":        {[]any{targets.HDR, targets.Only, []string{"HDR10"}}, with(hdr()), failed(targets.HDR, "")},
		"only, unknown":                   {[]any{targets.HDR, targets.Only, []string{"HDR10"}}, webdl(0, 0), passed},
		"not, none in the set":            {[]any{targets.HDR, targets.Not, []string{"DV"}}, with(hdr("HDR10")), passed},
		"not, one in the set":             {[]any{targets.HDR, targets.Not, []string{"DV"}}, with(hdr("HDR10", "DV")), failed(targets.HDR, "DV")},
		"not, unknown":                    {[]any{targets.HDR, targets.Not, []string{"DV"}}, webdl(0, 0), passed},
		"video only, the codec": {
			[]any{targets.VideoCodec, targets.Only, []string{"x265"}},
			with(func(f *release.Facts) { f.Video = release.Known[release.VideoCodec]("x265") }), passed,
		},
		"video only, another codec": {
			[]any{targets.VideoCodec, targets.Only, []string{"x265"}},
			with(func(f *release.Facts) { f.Video = release.Known[release.VideoCodec]("x264") }), failed(targets.VideoCodec, "x264"),
		},
		"video not, the codec": {
			[]any{targets.VideoCodec, targets.Not, []string{"XviD"}},
			with(func(f *release.Facts) { f.Video = release.Known[release.VideoCodec]("XviD") }), failed(targets.VideoCodec, "XviD"),
		},
		"video, unknown": {[]any{targets.VideoCodec, targets.Only, []string{"x265"}}, webdl(0, 0), passed},
		"audio not, a codec present": {
			[]any{targets.AudioCodec, targets.Not, []string{"MP3"}},
			with(func(f *release.Facts) { f.Audio = release.Known([]release.AudioCodec{"AAC", "MP3"}) }), failed(targets.AudioCodec, "MP3"),
		},
		"channels only, a layout present": {
			[]any{targets.Channels, targets.Only, []string{"5.1", "7.1"}},
			with(func(f *release.Facts) { f.Channels = release.Known([]release.Channels{"2.0", "5.1"}) }), passed,
		},
		"channels only, none present": {
			[]any{targets.Channels, targets.Only, []string{"5.1"}},
			with(func(f *release.Facts) { f.Channels = release.Known([]release.Channels{"2.0"}) }), failed(targets.Channels, "2.0"),
		},
		"language only, by any ISO 639 code": {[]any{targets.Language, targets.Only, []string{"eng"}}, with(langs("en")), passed},
		"language only, another":             {[]any{targets.Language, targets.Only, []string{"en"}}, with(langs("fr")), failed(targets.Language, "fr")},
		"language only original, unknown":    {[]any{targets.Language, targets.Only, []string{"original"}}, with(langs("fr")), passed},
		"language only original, it":         {[]any{targets.Language, targets.Only, []string{"original"}}, with(original("fr", "fr")), passed},
		"language only original, not it":     {[]any{targets.Language, targets.Only, []string{"original"}}, with(original("fr", "en")), failed(targets.Language, "en")},
		"language only original or english":  {[]any{targets.Language, targets.Only, []string{"original", "en"}}, with(original("fr", "en")), passed},
		"language not original, it":          {[]any{targets.Language, targets.Not, []string{"original"}}, with(original("fr", "en", "fr")), failed(targets.Language, "fr")},
		"language not original, unknown":     {[]any{targets.Language, targets.Not, []string{"original"}}, with(langs("fr")), passed},
		"language, unknown":                  {[]any{targets.Language, targets.Only, []string{"en"}}, webdl(0, 0), passed},
	} {
		v := targets.Judge(hd(t, filters(t, c.rules...)), targets.Scored{Facts: c.facts}, nil)
		if got := v.Reasons(); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: reasons = %+v, want %+v", name, got, c.want)
		}
		if meets := len(c.want) == 1; v.Meets() != meets || v.Grabbable() != meets {
			t.Errorf("%s: Meets = %t, Grabbable = %t, want %t", name, v.Meets(), v.Grabbable(), meets)
		}
	}
}

func TestJudgeListsEveryFailureInOrder(t *testing.T) {
	target := hd(t, filters(t, targets.HDR, targets.Only, []string{"SDR"}, targets.Language, targets.Not, []string{"de"}))
	facts := release.Facts{
		Quality: release.Known(release.CAM), HDR: release.Known([]release.HDR{"DV"}), Languages: release.Known([]release.Language{"de"}),
	}
	want := []targets.Reason{
		{Code: targets.QualityNotInTarget, Value: "cam"},
		{Code: targets.FilterFailed, Attribute: targets.HDR, Value: "DV"},
		{Code: targets.FilterFailed, Attribute: targets.Language, Value: "de"},
	}
	if got := targets.Judge(target, targets.Scored{Facts: facts}, nil).Reasons(); !reflect.DeepEqual(got, want) {
		t.Fatalf("reasons = %+v, want %+v", got, want)
	}
}

var (
	fa = formats.ID{1}
	fb = formats.ID{2}
	fc = formats.ID{3}
)

// scoredSpec is a Target over tiers with cutoff, scoring fa 10, fb 25 and
// fc -10.
func scoredSpec(tiers [][]release.Quality, cutoff release.Quality, minimum, until, increment int, p targets.Propers) targets.Spec {
	return targets.Spec{
		Name: "scored", Tiers: tiers, Cutoff: cutoff, Propers: p,
		Scores: targets.Scores{Formats: map[formats.ID]int{fa: 10, fb: 25, fc: -10}, Minimum: minimum, UpgradeUntil: until, Increment: increment},
	}
}

var discsOverWeb = [][]release.Quality{{release.Bluray1080p, release.Remux1080p}, {release.WEBDL1080p, release.WEBRip1080p}}

func TestJudgeScoresAReleaseAndRefusesOneBelowTheMinimum(t *testing.T) {
	target := must[targets.Target](t)(targets.NewTarget(scoredSpec(discsOverWeb, release.Remux1080p, 5, 0, 1, targets.PreferAndUpgrade)))
	webDL := release.Facts{Quality: release.Known(release.WEBDL1080p)}
	for name, c := range map[string]struct {
		facts   release.Facts
		formats formats.Set
		score   int
		reasons []targets.Reason
	}{
		"two scored formats":        {webDL, formats.Set{fa, fb}, 35, []targets.Reason{notChecked}},
		"a format no score names":   {webDL, formats.Set{fa, {9}}, 10, []targets.Reason{notChecked}},
		"no format":                 {webDL, nil, 0, []targets.Reason{notChecked, {Code: targets.ScoreBelowMinimum, Value: "0"}}},
		"a negative score":          {webDL, formats.Set{fc}, -10, []targets.Reason{notChecked, {Code: targets.ScoreBelowMinimum, Value: "-10"}}},
		"a quality it doesn't take": {release.Facts{Quality: release.Known(release.CAM)}, nil, 0, []targets.Reason{{Code: targets.QualityNotInTarget, Value: "cam"}, {Code: targets.ScoreBelowMinimum, Value: "0"}}},
	} {
		v := targets.Judge(target, targets.Scored{Facts: c.facts, Formats: c.formats}, nil)
		if v.Score() != c.score || !reflect.DeepEqual(v.Reasons(), c.reasons) {
			t.Errorf("%s: score %d, reasons %+v; want %d, %+v", name, v.Score(), v.Reasons(), c.score, c.reasons)
		}
		if below := c.score < 5; v.Grabbable() == below || v.Meets() != (name != "a quality it doesn't take") {
			t.Errorf("%s: Grabbable = %t, Meets = %t", name, v.Grabbable(), v.Meets())
		}
	}
}

func TestJudgeRunsTheUpgradeRulesInOrder(t *testing.T) {
	v1 := release.Revision{Version: 1}
	proper := release.Revision{Version: 2}
	repack := release.Revision{Version: 2, Repack: true}
	realV1 := release.Revision{Version: 1, Real: 1}
	facts := func(q release.Quality, r release.Revision, group string) release.Facts {
		f := release.Facts{Quality: release.Known(q), Revision: r}
		if group != "" {
			f.Group = release.Known(group)
		}
		return f
	}
	held := func(q release.Quality, r release.Revision, group string, score int) *targets.Held {
		return &targets.Held{Facts: facts(q, r, group), Score: score}
	}
	middle := [][]release.Quality{{release.Remux1080p}, {release.Bluray1080p}, {release.WEBDL1080p}}
	for name, c := range map[string]struct {
		target  targets.Spec
		release targets.Scored
		held    *targets.Held
		rule    int
		ok      bool
		refusal targets.ReasonCode
	}{
		"a vacant cell": {
			scoredSpec(discsOverWeb, release.Remux1080p, 0, 0, 1, targets.PreferAndUpgrade),
			targets.Scored{Facts: facts(release.WEBRip1080p, v1, "")}, nil, 1, true, "",
		},
		"over a file the Target doesn't rank": {
			scoredSpec(discsOverWeb, release.Remux1080p, 0, 0, 1, targets.PreferAndUpgrade),
			targets.Scored{Facts: facts(release.WEBRip1080p, v1, "")}, held(release.HDTV720p, v1, "", 0), 1, true, "",
		},
		"over a file of unknown quality": {
			scoredSpec(discsOverWeb, release.Remux1080p, 0, 0, 1, targets.PreferAndUpgrade),
			targets.Scored{Facts: facts(release.WEBRip1080p, v1, "")}, &targets.Held{}, 1, true, "",
		},
		"a proper from another group, past the cutoff": {
			scoredSpec(discsOverWeb, release.Remux1080p, 0, 0, 1, targets.PreferAndUpgrade),
			targets.Scored{Facts: facts(release.Remux1080p, proper, "NTb")}, held(release.Remux1080p, v1, "FLUX", 0), 2, true, "",
		},
		"a REAL over a v2": {
			scoredSpec(discsOverWeb, release.Remux1080p, 0, 0, 1, targets.PreferAndUpgrade),
			targets.Scored{Facts: facts(release.WEBDL1080p, realV1, "")}, held(release.WEBDL1080p, proper, "", 0), 2, true, "",
		},
		"a repack from the file's group in other letters": {
			scoredSpec(discsOverWeb, release.Remux1080p, 0, 0, 1, targets.PreferAndUpgrade),
			targets.Scored{Facts: facts(release.Remux1080p, repack, "flux")}, held(release.Remux1080p, v1, "FLUX", 0), 2, true, "",
		},
		"a repack from another group": {
			scoredSpec(discsOverWeb, release.Remux1080p, 0, 0, 1, targets.PreferAndUpgrade),
			targets.Scored{Facts: facts(release.Remux1080p, repack, "NTb")}, held(release.Remux1080p, v1, "FLUX", 0), 2, false, targets.RepackOtherGroup,
		},
		"a repack over a file of unknown group": {
			scoredSpec(discsOverWeb, release.Remux1080p, 0, 0, 1, targets.PreferAndUpgrade),
			targets.Scored{Facts: facts(release.Remux1080p, repack, "FLUX")}, held(release.Remux1080p, v1, "", 0), 2, false, targets.RepackOtherGroup,
		},
		"a proper under prefer, don't upgrade": {
			scoredSpec(discsOverWeb, release.Remux1080p, 0, 0, 1, targets.PreferNoUpgrade),
			targets.Scored{Facts: facts(release.Remux1080p, proper, "")}, held(release.Remux1080p, v1, "", 0), 7, false, targets.NotAnUpgrade,
		},
		"a proper of another Quality in the tier": {
			scoredSpec(discsOverWeb, release.Remux1080p, 0, 0, 1, targets.PreferAndUpgrade),
			targets.Scored{Facts: facts(release.WEBRip1080p, proper, "")}, held(release.WEBDL1080p, v1, "", 0), 7, false, targets.NotAnUpgrade,
		},
		"a better tier below the cutoff": {
			scoredSpec(discsOverWeb, release.Remux1080p, 0, 0, 1, targets.PreferAndUpgrade),
			targets.Scored{Facts: facts(release.Bluray1080p, v1, "")}, held(release.WEBDL1080p, proper, "", 0), 3, true, "",
		},
		"a better tier over a file at the cutoff": {
			scoredSpec(middle, release.Bluray1080p, 0, 30, 1, targets.PreferAndUpgrade),
			targets.Scored{Facts: facts(release.Remux1080p, v1, ""), Formats: formats.Set{fb}}, held(release.Bluray1080p, v1, "", 0), 4, false, targets.NotAnUpgrade,
		},
		"a worse tier": {
			scoredSpec(discsOverWeb, release.Remux1080p, 0, 0, 1, targets.PreferAndUpgrade),
			targets.Scored{Facts: facts(release.WEBDL1080p, proper, "")}, held(release.Bluray1080p, v1, "", -10), 5, false, targets.NotAnUpgrade,
		},
		"a lower revision of the file's Quality": {
			scoredSpec(discsOverWeb, release.Remux1080p, 0, 30, 1, targets.PreferNoUpgrade),
			targets.Scored{Facts: facts(release.WEBDL1080p, v1, ""), Formats: formats.Set{fb}}, held(release.WEBDL1080p, proper, "", 0), 6, false, targets.NotAnUpgrade,
		},
		"a lower revision under don't prefer, by score": {
			scoredSpec(discsOverWeb, release.Remux1080p, 0, 30, 10, targets.DontPrefer),
			targets.Scored{Facts: facts(release.WEBDL1080p, v1, ""), Formats: formats.Set{fa}}, held(release.WEBDL1080p, proper, "", 0), 8, true, "",
		},
		"a higher revision under don't prefer": {
			scoredSpec(discsOverWeb, release.Remux1080p, 0, 0, 1, targets.DontPrefer),
			targets.Scored{Facts: facts(release.WEBDL1080p, proper, "")}, held(release.WEBDL1080p, v1, "", 0), 7, false, targets.NotAnUpgrade,
		},
		"a file at upgrade-until": {
			scoredSpec(discsOverWeb, release.Remux1080p, 0, 30, 1, targets.PreferAndUpgrade),
			targets.Scored{Facts: facts(release.WEBRip1080p, v1, ""), Formats: formats.Set{fa, fb}}, held(release.WEBDL1080p, v1, "", 30), 7, false, targets.NotAnUpgrade,
		},
		"a file at the cutoff below upgrade-until": {
			scoredSpec(discsOverWeb, release.Remux1080p, 0, 30, 10, targets.PreferAndUpgrade),
			targets.Scored{Facts: facts(release.Bluray1080p, v1, ""), Formats: formats.Set{fb}}, held(release.Remux1080p, v1, "", 10), 8, true, "",
		},
		"a score short of the increment": {
			scoredSpec(discsOverWeb, release.Remux1080p, 0, 30, 10, targets.PreferAndUpgrade),
			targets.Scored{Facts: facts(release.Remux1080p, v1, ""), Formats: formats.Set{fa}}, held(release.Bluray1080p, v1, "", 1), 8, false, targets.NotAnUpgrade,
		},
		"a file scored below upgrade-until 0": {
			scoredSpec(discsOverWeb, release.Remux1080p, 0, 0, 1, targets.PreferAndUpgrade),
			targets.Scored{Facts: facts(release.Remux1080p, v1, "")}, held(release.Remux1080p, v1, "", -10), 8, true, "",
		},
	} {
		v := targets.Judge(must[targets.Target](t)(targets.NewTarget(c.target)), c.release, c.held)
		var refusals []targets.ReasonCode
		for _, r := range v.Reasons() {
			if r.Code != targets.SizeNotChecked {
				refusals = append(refusals, r.Code)
			}
		}
		var want []targets.ReasonCode
		if c.refusal != "" {
			want = []targets.ReasonCode{c.refusal}
		}
		if rule, ok := v.Upgrade(); rule != c.rule || ok != c.ok || !reflect.DeepEqual(refusals, want) || v.Grabbable() != c.ok {
			t.Errorf("%s: rule %d, %t, %v; want rule %d, %t, %v", name, rule, ok, refusals, c.rule, c.ok, want)
		}
	}
}

func TestFinalNeedsTheCutoffTierAndUpgradeUntil(t *testing.T) {
	target := must[targets.Target](t)(targets.NewTarget(scoredSpec(discsOverWeb, release.Remux1080p, 0, 30, 1, targets.PreferAndUpgrade)))
	for _, c := range []struct {
		rank  targets.Rank
		score int
		final bool
	}{{0, 30, true}, {0, 29, false}, {1, 100, false}} {
		if got := target.Final(c.rank, c.score); got != c.final {
			t.Errorf("Final(%d, %d) = %t, want %t", c.rank, c.score, got, c.final)
		}
	}
}

func TestARepackFromAnotherGroupNamesTheHeldFilesGroup(t *testing.T) {
	target := must[targets.Target](t)(targets.NewTarget(scoredSpec(discsOverWeb, release.Remux1080p, 0, 0, 1, targets.PreferAndUpgrade)))
	repack := release.Facts{Quality: release.Known(release.Remux1080p), Revision: release.Revision{Version: 2, Repack: true}, Group: release.Known("NTb")}
	held := &targets.Held{Facts: release.Facts{Quality: release.Known(release.Remux1080p), Revision: release.Revision{Version: 1}, Group: release.Known("FKW")}}
	got := targets.Judge(target, targets.Scored{Facts: repack}, held).Reasons()
	if want := (targets.Reason{Code: targets.RepackOtherGroup, Value: "FKW"}); !slices.Contains(got, want) {
		t.Errorf("reasons %+v, want %+v", got, want)
	}
}
