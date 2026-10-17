package targets_test

import (
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/targets"
)

func keys(tiers [][]release.Quality) [][]string {
	out := make([][]string, len(tiers))
	for i, tier := range tiers {
		for _, q := range tier {
			out[i] = append(out[i], q.String())
		}
	}
	return out
}

func TestEachPresetLaysOutEveryQualityBestFirstWithCinemaAtTheBottom(t *testing.T) {
	for p, want := range map[targets.Preset][][]string{
		targets.AnythingBetter: {
			{"remux-2160p"}, {"bluray-2160p"}, {"webdl-2160p"}, {"webrip-2160p"}, {"hdtv-2160p"},
			{"remux-1080p"}, {"bluray-1080p"}, {"webdl-1080p"}, {"webrip-1080p"}, {"hdtv-1080p"},
			{"bluray-720p"}, {"webdl-720p"}, {"webrip-720p"}, {"hdtv-720p"},
			{"webrip-480p"}, {"dvd"}, {"sdtv"}, {"cam"},
		},
		targets.DiscOrResolution: {
			{"bluray-2160p", "remux-2160p"}, {"hdtv-2160p", "webrip-2160p", "webdl-2160p"},
			{"bluray-1080p", "remux-1080p"}, {"hdtv-1080p", "webrip-1080p", "webdl-1080p"},
			{"bluray-720p"}, {"hdtv-720p", "webrip-720p", "webdl-720p"},
			{"dvd"}, {"sdtv", "webrip-480p"}, {"cam"},
		},
		targets.ResolutionOnly: {
			{"hdtv-2160p", "webrip-2160p", "webdl-2160p", "bluray-2160p", "remux-2160p"},
			{"hdtv-1080p", "webrip-1080p", "webdl-1080p", "bluray-1080p", "remux-1080p"},
			{"hdtv-720p", "webrip-720p", "webdl-720p", "bluray-720p"},
			{"sdtv", "dvd", "webrip-480p"}, {"cam"},
		},
	} {
		if got := keys(p.Tiers()); !reflect.DeepEqual(got, want) {
			t.Errorf("%s tiers = %v, want %v", p, got, want)
		}
	}
	if got := targets.Presets(); !reflect.DeepEqual(got, []targets.Preset{targets.DiscOrResolution, targets.AnythingBetter, targets.ResolutionOnly}) {
		t.Errorf("Presets = %v, want the default first", got)
	}
}

func TestLayoutKeepsAPresetToTheChosenQualitiesWithTheCutoffAtTopOrBottom(t *testing.T) {
	var hd []release.Quality
	for _, q := range release.Qualities() {
		if r := q.Info().Resolution; r == release.R1080p || r == release.R720p {
			hd = append(hd, q)
		}
	}
	starter := targets.Layout(targets.DiscOrResolution, hd, true)
	if got, want := keys(starter.Tiers), [][]string{
		{"bluray-1080p", "remux-1080p"}, {"hdtv-1080p", "webrip-1080p", "webdl-1080p"}, {"bluray-720p"}, {"hdtv-720p", "webrip-720p", "webdl-720p"},
	}; !reflect.DeepEqual(got, want) || starter.Cutoff != release.Bluray1080p {
		t.Fatalf("disc or resolution over 720p and 1080p = %v cutoff %s, want %v cutoff bluray-1080p", got, starter.Cutoff, want)
	}
	for p, cutoff := range map[targets.Preset]release.Quality{
		targets.AnythingBetter: release.HDTV720p, targets.DiscOrResolution: release.HDTV720p, targets.ResolutionOnly: release.HDTV720p,
	} {
		s := targets.Layout(p, hd, false)
		target := must[targets.Target](t)(targets.NewTarget(s))
		if s.Cutoff != cutoff || target.Cutoff != len(target.Tiers)-1 {
			t.Errorf("%s with no upgrades: cutoff %s at tier %d of %d, want %s at the bottom", p, s.Cutoff, target.Cutoff, len(target.Tiers), cutoff)
		}
		for _, tier := range target.Tiers {
			for _, a := range tier {
				if a.Size != targets.Floor(a.Quality) {
					t.Errorf("%s lays %s out at %+v, want its floor", p, a.Quality, a.Size)
				}
			}
		}
	}
	if s := targets.Layout(targets.ResolutionOnly, []release.Quality{release.SDTV, release.CAM}, true); !reflect.DeepEqual(keys(s.Tiers), [][]string{{"sdtv"}, {"cam"}}) {
		t.Fatalf("resolution only over SDTV and CAM = %v, want CAM below", keys(s.Tiers))
	}
}

func TestNewTargetRefusesWhatNoTargetCanBe(t *testing.T) {
	spec := func(edit func(*targets.Spec)) targets.Spec {
		s := targets.Spec{Name: "1080p", Tiers: [][]release.Quality{{release.WEBDL1080p}, {release.HDTV1080p}}, Cutoff: release.WEBDL1080p}
		edit(&s)
		return s
	}
	for name, c := range map[string]struct {
		spec targets.Spec
		want error
	}{
		"no tiers":                 {spec(func(s *targets.Spec) { s.Tiers = nil }), targets.ErrBadTiers},
		"an empty tier":            {spec(func(s *targets.Spec) { s.Tiers = append(s.Tiers, nil) }), targets.ErrBadTiers},
		"a quality twice":          {spec(func(s *targets.Spec) { s.Tiers[1] = append(s.Tiers[1], release.WEBDL1080p) }), targets.ErrBadTiers},
		"a cutoff it doesn't take": {spec(func(s *targets.Spec) { s.Cutoff = release.Bluray1080p }), targets.ErrQualityNotInTarget},
		"a size it doesn't take":   {spec(func(s *targets.Spec) { s.Sizes = map[release.Quality]targets.Size{release.CAM: {}} }), targets.ErrQualityNotInTarget},
		"a negative minimum":       {spec(func(s *targets.Spec) { s.Sizes = map[release.Quality]targets.Size{release.HDTV1080p: {Min: -1}} }), targets.ErrBadSize},
		"a maximum below its floor": {spec(func(s *targets.Spec) {
			s.Sizes = map[release.Quality]targets.Size{release.HDTV1080p: {Min: 5, Max: new(4.0)}}
		}), targets.ErrBadSize},
	} {
		if _, err := targets.NewTarget(c.spec); !errors.Is(err, c.want) {
			t.Errorf("%s: err = %v, want %v", name, err, c.want)
		}
	}

	target := must[targets.Target](t)(targets.NewTarget(spec(func(s *targets.Spec) {
		s.Cutoff = release.HDTV1080p
		s.Sizes = map[release.Quality]targets.Size{release.HDTV1080p: {Min: 5, Max: new(60.0)}}
	})))
	want := [][]targets.Accepted{
		{{Quality: release.WEBDL1080p, Size: targets.Size{Min: 12.5}}},
		{{Quality: release.HDTV1080p, Size: targets.Size{Min: 5, Max: new(60.0)}}},
	}
	if got := target.Tiers; target.Cutoff != 1 || len(got) != 2 || !slices.Equal(got[0], want[0]) || got[1][0].Size.Min != 5 || *got[1][0].Size.Max != 60 {
		t.Fatalf("NewTarget = %+v cutoff %d, want %+v cutoff 1", got, target.Cutoff, want)
	}
}

func TestFiltersRefuseValuesTheirModeOrAttributeDoesNotTake(t *testing.T) {
	var f targets.Filters
	for name, c := range map[string]struct {
		a      targets.Attribute
		m      targets.Mode
		values []string
	}{
		"any with a value":          {targets.HDR, targets.Any, []string{"DV"}},
		"only with none":            {targets.HDR, targets.Only, nil},
		"a value twice":             {targets.Language, targets.Not, []string{"en", "eng"}},
		"a value of another kind":   {targets.VideoCodec, targets.Only, []string{"TrueHD"}},
		"a language that is none":   {targets.Language, targets.Only, []string{"und"}},
		"an attribute that is none": {"bitrate", targets.Only, []string{"1"}},
	} {
		if err := f.Set(c.a, c.m, c.values); !errors.Is(err, targets.ErrBadFilter) {
			t.Errorf("%s: err = %v, want ErrBadFilter", name, err)
		}
	}
	if _, err := targets.ParseMode("prefer"); !errors.Is(err, targets.ErrBadFilter) {
		t.Errorf("ParseMode(prefer) err = %v, want ErrBadFilter", err)
	}
}

func TestContainsAsksEveryFileMeetingTheInnerTargetToMeetTheOuter(t *testing.T) {
	target := func(tiers [][]release.Quality, sizes map[release.Quality]targets.Size, rules ...any) targets.Target {
		return must[targets.Target](t)(targets.NewTarget(targets.Spec{Name: "t", Tiers: tiers, Cutoff: tiers[0][0], Sizes: sizes, Filters: filters(t, rules...)}))
	}
	web := [][]release.Quality{{release.WEBDL1080p, release.WEBRip1080p}}
	wide := [][]release.Quality{{release.Bluray1080p}, {release.WEBDL1080p, release.WEBRip1080p, release.HDTV1080p}}
	small := func(max float64) map[release.Quality]targets.Size {
		return map[release.Quality]targets.Size{release.WEBDL1080p: {Min: 12.5, Max: new(max)}, release.WEBRip1080p: {Min: 12.5, Max: new(max)}}
	}
	for name, c := range map[string]struct {
		outer, inner targets.Target
		want         bool
	}{
		"fewer qualities":           {target(wide, nil), target(web, nil), true},
		"more qualities":            {target(web, nil), target(wide, nil), false},
		"itself":                    {target(wide, nil), target(wide, nil), true},
		"a smaller maximum":         {target(web, small(40)), target(web, small(30)), true},
		"no maximum inside one":     {target(web, small(40)), target(web, nil), false},
		"a lower minimum":           {target(web, nil), target(web, map[release.Quality]targets.Size{release.WEBDL1080p: {Min: 1}}), false},
		"a filter inside any":       {target(web, nil), target(web, nil, targets.HDR, targets.Only, []string{"SDR"}), true},
		"any inside a filter":       {target(web, nil, targets.HDR, targets.Only, []string{"SDR"}), target(web, nil), false},
		"only a subset":             {target(web, nil, targets.HDR, targets.Only, []string{"SDR", "HDR10"}), target(web, nil, targets.HDR, targets.Only, []string{"SDR"}), true},
		"only a superset":           {target(web, nil, targets.HDR, targets.Only, []string{"SDR"}), target(web, nil, targets.HDR, targets.Only, []string{"SDR", "HDR10"}), false},
		"not a superset":            {target(web, nil, targets.AudioCodec, targets.Not, []string{"MP3"}), target(web, nil, targets.AudioCodec, targets.Not, []string{"MP3", "AAC"}), true},
		"not a subset":              {target(web, nil, targets.AudioCodec, targets.Not, []string{"MP3", "AAC"}), target(web, nil, targets.AudioCodec, targets.Not, []string{"MP3"}), false},
		"a codec only, outside not": {target(web, nil, targets.VideoCodec, targets.Not, []string{"XviD"}), target(web, nil, targets.VideoCodec, targets.Only, []string{"x265"}), true},
		"a codec only, inside not":  {target(web, nil, targets.VideoCodec, targets.Not, []string{"x265"}), target(web, nil, targets.VideoCodec, targets.Only, []string{"x265"}), false},
		"a codec not, inside only": {
			target(web, nil, targets.VideoCodec, targets.Only, []string{"x264", "x265", "AV1", "VP9"}),
			target(web, nil, targets.VideoCodec, targets.Not, []string{"XviD", "MPEG2"}), true,
		},
		"a codec not, past only": {
			target(web, nil, targets.VideoCodec, targets.Only, []string{"x264", "x265"}),
			target(web, nil, targets.VideoCodec, targets.Not, []string{"XviD", "MPEG2"}), false,
		},
		"languages only, outside not": {target(web, nil, targets.Language, targets.Not, []string{"de"}), target(web, nil, targets.Language, targets.Only, []string{"en"}), false},
	} {
		if got := targets.Contains(c.outer, c.inner); got != c.want {
			t.Errorf("%s: Contains = %t, want %t", name, got, c.want)
		}
	}
}
