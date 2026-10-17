package targets_test

import (
	"reflect"
	"testing"
	"time"

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
	for i := 0; i < len(rules); i += 3 {
		if err := f.Set(rules[i].(targets.Attribute), rules[i+1].(targets.Mode), rules[i+2].([]string)); err != nil {
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
		if got := verdictOf(targets.Judge(target, c.facts, nil)); !reflect.DeepEqual(got, c.want) {
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
		v := targets.Judge(hd(t, filters(t, c.rules...)), c.facts, nil)
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
	if got := targets.Judge(target, facts, nil).Reasons(); !reflect.DeepEqual(got, want) {
		t.Fatalf("reasons = %+v, want %+v", got, want)
	}
}

func TestJudgeAsksAReleaseToRankAboveTheFileHeldBelowTheCutoff(t *testing.T) {
	target := hd(t, targets.Filters{})
	at := func(q release.Quality) *release.Facts { return &release.Facts{Quality: release.Known(q)} }
	notAnUpgrade := []targets.Reason{notChecked, {Code: targets.NotAnUpgrade}}
	for name, c := range map[string]struct {
		release release.Quality
		held    *release.Facts
		want    verdict
	}{
		"a vacant cell":                       {release.WEBRip1080p, nil, verdict{1, true, true, []targets.Reason{notChecked}}},
		"a better tier":                       {release.Bluray1080p, at(release.WEBDL1080p), verdict{0, true, true, []targets.Reason{notChecked}}},
		"a tie":                               {release.WEBRip1080p, at(release.WEBDL1080p), verdict{1, true, false, notAnUpgrade}},
		"a worse tier":                        {release.WEBDL1080p, at(release.Remux1080p), verdict{1, true, false, notAnUpgrade}},
		"over a file at the cutoff":           {release.Remux1080p, at(release.Bluray1080p), verdict{0, true, false, notAnUpgrade}},
		"over a file the Target doesn't rank": {release.WEBRip1080p, at(release.HDTV720p), verdict{1, true, true, []targets.Reason{notChecked}}},
		"over a file of unknown quality":      {release.WEBRip1080p, &release.Facts{}, verdict{1, true, true, []targets.Reason{notChecked}}},
	} {
		if got := verdictOf(targets.Judge(target, release.Facts{Quality: release.Known(c.release)}, c.held)); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: Judge = %+v, want %+v", name, got, c.want)
		}
	}

	none := must[targets.Target](t)(targets.NewTarget(targets.Layout(targets.ResolutionOnly, []release.Quality{release.WEBDL1080p, release.WEBDL720p}, false)))
	if got := targets.Judge(none, release.Facts{Quality: release.Known(release.WEBDL1080p)}, at(release.WEBDL720p)).Reasons(); !reflect.DeepEqual(got, notAnUpgrade) {
		t.Fatalf("with no upgrades, a better release over a held one: reasons = %+v, want not_an_upgrade", got)
	}
}
