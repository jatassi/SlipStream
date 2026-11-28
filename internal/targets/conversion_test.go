package targets_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	conv "github.com/jatassi/slipstream/internal/encode"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/nodes/nodestest"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
)

func settings() conv.Settings {
	s := conv.Defaults()
	s.Kbps = 6000
	return s
}

// madeSpec is a Bluray Target of any size named name whose files are made with s.
func madeSpec(name string, s *conv.Settings) targets.Spec {
	spec := targets.Spec{
		Name: name, Tiers: [][]release.Quality{{release.Bluray2160p, release.Bluray1080p}}, Cutoff: release.Bluray1080p, Scores: targets.Scores{Increment: 1},
		Sizes: map[release.Quality]targets.Size{release.Bluray2160p: {}, release.Bluray1080p: {}},
	}
	if s != nil {
		spec.Conversion = &targets.Conversion{Settings: *s}
	}
	return spec
}

func ptr[T any](v T) *T { return &v }

func TestAMadeFromRowNeedsConversionSettingsAndADownloadedSourceAboveIt(t *testing.T) {
	ctx := t.Context()
	s := newShelf(t, 0)
	create := func(spec targets.Spec) targets.Target {
		return must[targets.Target](t)(s.goals.CreateTarget(ctx, identity.Admin{}, spec))
	}
	src, made, also := create(madeSpec("Source", nil)), create(madeSpec("Made", ptr(settings()))), create(madeSpec("Also", ptr(settings())))
	down := func(id targets.TargetID) targets.Entry { return targets.Entry{Target: id, Sourcing: targets.Download} }
	derive := func(id, from targets.TargetID) targets.Entry {
		return targets.Entry{Target: id, Sourcing: targets.Derive, From: &from}
	}
	after := func(id, from targets.TargetID, grace time.Duration) targets.Entry {
		return targets.Entry{Target: id, Sourcing: targets.DownloadThenDerive, From: &from, Grace: grace}
	}
	for i, c := range []struct {
		name    string
		entries []targets.Entry
		want    error
	}{
		{"made from a download above", []targets.Entry{down(src.ID), derive(made.ID, src.ID)}, nil},
		{"made from a download-then-derive row above", []targets.Entry{down(src.ID), after(made.ID, src.ID, time.Hour), derive(also.ID, made.ID)}, nil},
		{"made from a row below", []targets.Entry{derive(made.ID, src.ID), down(src.ID)}, targets.ErrSourceNotAbove},
		{"made from a made row", []targets.Entry{down(src.ID), derive(made.ID, src.ID), derive(also.ID, made.ID)}, targets.ErrSourceNotAbove},
		{"made from a Target not in the Profile", []targets.Entry{derive(made.ID, src.ID)}, targets.ErrSourceNotAbove},
		{"made with no Conversion settings", []targets.Entry{down(made.ID), derive(src.ID, made.ID)}, targets.ErrNoConversion},
		{"downloaded with a source", []targets.Entry{down(src.ID), {Target: made.ID, Sourcing: targets.Download, From: &src.ID}}, targets.ErrBadSourcing},
		{"made with no source", []targets.Entry{down(src.ID), {Target: made.ID, Sourcing: targets.Derive}}, targets.ErrBadSourcing},
		{"made with a wait", []targets.Entry{down(src.ID), {Target: made.ID, Sourcing: targets.Derive, From: &src.ID, Grace: time.Hour}}, targets.ErrBadSourcing},
		{"an unknown sourcing", []targets.Entry{{Target: src.ID, Sourcing: "copied"}}, targets.ErrBadSourcing},
		{"a wait of 1 hour", []targets.Entry{down(src.ID), after(made.ID, src.ID, time.Hour)}, nil},
		{"a wait of 90 days", []targets.Entry{down(src.ID), after(made.ID, src.ID, 90*24*time.Hour)}, nil},
		{"a wait under 1 hour", []targets.Entry{down(src.ID), after(made.ID, src.ID, time.Hour-time.Second)}, targets.ErrBadGrace},
		{"a wait over 90 days", []targets.Entry{down(src.ID), after(made.ID, src.ID, 90*24*time.Hour+time.Second)}, targets.ErrBadGrace},
		{"a wait of part of a second", []targets.Entry{down(src.ID), after(made.ID, src.ID, time.Hour+time.Millisecond)}, targets.ErrBadGrace},
	} {
		spec := targets.ProfileSpec{Name: "P" + string(rune('a'+i)), ContentType: "cinema", Entries: c.entries, Editions: targets.AnyCut()}
		p, err := s.goals.CreateProfile(ctx, identity.Admin{}, spec)
		if !errors.Is(err, c.want) {
			t.Errorf("%s: err = %v, want %v", c.name, err, c.want)
			continue
		}
		if err != nil {
			continue
		}
		read := must[targets.Profile](t)(s.goals.GetProfile(ctx, identity.Admin{}, p.ID))
		if !reflect.DeepEqual(read.Entries, c.entries) {
			t.Errorf("%s: read back %+v, want %+v", c.name, read.Entries, c.entries)
		}
	}
}

func TestTurningOffSettingsOrReorderingRefusesWhileARowIsMadeFromThem(t *testing.T) {
	ctx := t.Context()
	s := newShelf(t, 0)
	src := must[targets.Target](t)(s.goals.CreateTarget(ctx, identity.Admin{}, madeSpec("Source", nil)))
	made := must[targets.Target](t)(s.goals.CreateTarget(ctx, identity.Admin{}, madeSpec("Made", ptr(settings()))))
	rows := []targets.Entry{{Target: src.ID, Sourcing: targets.Download}, {Target: made.ID, Sourcing: targets.Derive, From: &src.ID}}
	var profiles []targets.Profile
	for _, name := range []string{"Shows", "Movies"} {
		spec := targets.ProfileSpec{Name: name, ContentType: "cinema", Entries: rows, Editions: targets.AnyCut()}
		profiles = append(profiles, must[targets.Profile](t)(s.goals.CreateProfile(ctx, identity.Admin{}, spec)))
	}
	reordered := targets.ProfileSpec{Name: "Movies", ContentType: "cinema", Entries: []targets.Entry{rows[1], rows[0]}, Editions: targets.AnyCut()}
	for _, c := range []struct {
		name   string
		change targets.Change
		want   []string
	}{
		{"settings turned off", targets.UpdateTarget{ID: made.ID, Spec: madeSpec("Made", nil)}, []string{"Movies", "Shows"}},
		{"a reorder", targets.UpdateProfile{ID: profiles[1].ID, Spec: reordered}, []string{"Movies"}},
	} {
		var inUse *targets.SourceInUseError
		if err := s.apply(t, c.change); !errors.As(err, &inUse) || !reflect.DeepEqual(inUse.Profiles, c.want) {
			t.Errorf("%s: err = %v, want a source in use by %v", c.name, err, c.want)
		}
	}
}

func TestConversionProblemsNamesEachWayTheSettingsMakeRefusedFiles(t *testing.T) {
	hd := func(f targets.Filters, sizes map[release.Quality]targets.Size) targets.Spec {
		return targets.Spec{Tiers: [][]release.Quality{{release.Bluray1080p, release.WEBDL1080p}}, Cutoff: release.Bluray1080p, Filters: f, Sizes: sizes}
	}
	with := func(edit func(*conv.Settings)) conv.Settings {
		s := settings()
		edit(&s)
		return s
	}
	upTo := func(lo, hi float64) targets.Size { return targets.Size{Min: lo, Max: &hi} }
	for _, c := range []struct {
		name   string
		target targets.Spec
		s      conv.Settings
		want   []targets.Problem
	}{
		{"settings the Target takes", hd(targets.Filters{}, nil), settings(), nil},
		{
			"HEVC where only H.264 is allowed", hd(filters(t, targets.VideoCodec, targets.Only, []string{"x264"}), nil),
			with(func(s *conv.Settings) { s.Codec = conv.HEVC }),
			[]targets.Problem{{Code: targets.CodecFiltered, Field: "Codec", Facts: map[string]string{"codec": "hevc"}}},
		},
		{
			"stereo AAC where only DTS is allowed", hd(filters(t, targets.AudioCodec, targets.Only, []string{"DTS"}), nil),
			with(func(s *conv.Settings) { s.Audio = conv.StereoAAC }),
			[]targets.Problem{{Code: targets.CodecFiltered, Field: "Audio", Facts: map[string]string{"audio": "stereo_aac"}}},
		},
		{
			"SDR files where only HDR10 is allowed", hd(filters(t, targets.HDR, targets.Only, []string{"HDR10"}), nil),
			with(func(s *conv.Settings) { s.Range = conv.HDR10; s.Codec = conv.HEVC; s.Depth = conv.Bit10 }),
			[]targets.Problem{{Code: targets.RangeFromSDR, Field: "Range", Facts: map[string]string{"range": "hdr10"}}},
		},
		{
			// 6000 kbps is 42.9 MiB per minute; 2 to 20 MiB is 280 to 2796 kbps.
			"a bitrate above every 1080p range", hd(targets.Filters{}, map[release.Quality]targets.Size{release.Bluray1080p: upTo(5, 20), release.WEBDL1080p: upTo(2, 10)}),
			settings(),
			[]targets.Problem{{Code: targets.OutsideSize, Field: "Kbps", Facts: map[string]string{"kbps": "6000", "minKbps": "280", "maxKbps": "2796"}}},
		},
		{
			"MP4 with copied audio", hd(targets.Filters{}, nil), with(func(s *conv.Settings) { s.Container = conv.MP4 }),
			[]targets.Problem{{Code: targets.ContainerAudio, Field: "Audio", Facts: map[string]string{"container": "mp4"}}},
		},
		{
			"MP4 with copied audio where MP4's audio is all that is allowed", hd(filters(t, targets.AudioCodec, targets.Only, []string{"AAC", "DD"}), nil),
			with(func(s *conv.Settings) { s.Container = conv.MP4 }), nil,
		},
		{
			"720p from a 1080p Target", hd(targets.Filters{}, nil), with(func(s *conv.Settings) { s.MaxHeight = conv.P720 }),
			[]targets.Problem{{Code: targets.QualityUnranked, Field: "MaxHeight", Facts: map[string]string{"height": "720"}}},
		},
	} {
		if got := targets.ConversionProblems(c.target, c.s); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: problems = %+v, want %+v", c.name, got, c.want)
		}
	}
}

func TestDefaultKbpsIsTheMiddleOfTheSizeRangeTheMadeFileRanksIn(t *testing.T) {
	size := func(lo float64, hi ...float64) targets.Size {
		s := targets.Size{Min: lo}
		if len(hi) > 0 {
			s.Max = &hi[0]
		}
		return s
	}
	target := func(sizes map[release.Quality]targets.Size) targets.Target {
		var tier []release.Quality
		for _, q := range release.Qualities() {
			if _, ok := sizes[q]; ok {
				tier = append(tier, q)
			}
		}
		return must[targets.Target](t)(targets.NewTarget(targets.Spec{Tiers: [][]release.Quality{tier}, Cutoff: tier[0], Sizes: sizes, Scores: targets.Scores{Increment: 1}}))
	}
	// 1 MiB per minute is 1,048,576 × 8 / 60 / 1000 = 139.81 kbps.
	for _, c := range []struct {
		name   string
		target targets.Target
		at     conv.Height
		tier   release.Origin
		want   int
	}{
		{"the quality it ranks as", target(map[release.Quality]targets.Size{release.Bluray1080p: size(50, 100), release.HDTV1080p: size(30, 40)}), conv.P1080, release.OriginBluray, 10486},
		{"the widest range at the resolution", target(map[release.Quality]targets.Size{release.Bluray1080p: size(50, 100), release.HDTV1080p: size(30, 40)}), conv.P1080, release.OriginWEBDL, 10486},
		{"a range with no maximum is the widest", target(map[release.Quality]targets.Size{release.Bluray1080p: size(50, 100), release.Remux1080p: size(80)}), conv.P1080, release.OriginWEBDL, 13981},
		{"no maximum takes the minimum plus a quarter", target(map[release.Quality]targets.Size{release.Bluray1080p: size(40)}), conv.P1080, release.OriginBluray, 6991},
		{"no quality at the resolution takes its quality's floor", target(map[release.Quality]targets.Size{release.Bluray1080p: size(40)}), conv.P720, release.OriginBluray, 4491},
		{"never below the lowest bitrate", target(map[release.Quality]targets.Size{release.DVDQuality: size(0)}), conv.P480, release.OriginDVD, conv.MinKbps},
	} {
		if got := targets.DefaultKbps(c.target, c.at, c.tier); got != c.want {
			t.Errorf("%s: DefaultKbps = %d, want %d", c.name, got, c.want)
		}
	}
}

func TestTheVersionRisesForEachSettingAndNeverForRunOn(t *testing.T) {
	ctx := t.Context()
	s := newShelf(t, 0)
	node := role.NodeID{UUID: storage.NewUUID()}
	if err := s.lib.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return nodestest.Insert(ctx, tx, node, "Attic")
	}); err != nil {
		t.Fatal(err)
	}
	base := settings()
	target := must[targets.Target](t)(s.goals.CreateTarget(ctx, identity.Admin{}, madeSpec("Made", &base)))
	version := func() int {
		t.Helper()
		read := must[targets.Target](t)(s.goals.GetTarget(ctx, identity.Admin{}, target.ID))
		if read.Conversion == nil {
			t.Fatal("the Target lost its Conversion settings")
		}
		return read.Conversion.Version
	}
	save := func(c *targets.Conversion) {
		t.Helper()
		spec := madeSpec("Made", nil)
		spec.Conversion = c
		if err := s.apply(t, targets.UpdateTarget{ID: target.ID, Spec: spec}); err != nil {
			t.Fatal(err)
		}
	}
	was := version()
	save(&targets.Conversion{Settings: base, RunOn: &node})
	if v := version(); v != was {
		t.Errorf("a Run on change moved the version from %d to %d", was, v)
	}
	for _, edit := range []func(*conv.Settings){
		func(s *conv.Settings) { s.Codec = conv.HEVC },
		func(s *conv.Settings) { s.Depth = conv.Bit10 },
		func(s *conv.Settings) { s.Range = conv.HDR10 },
		func(s *conv.Settings) { s.KeepDV = true },
		func(s *conv.Settings) { s.Curve = conv.Hable },
		func(s *conv.Settings) { s.Kbps = 5000 },
		func(s *conv.Settings) { s.Audio = conv.StereoAAC },
		func(s *conv.Settings) { s.DropCommentary = false },
		func(s *conv.Settings) { s.Container = conv.MP4 },
		func(s *conv.Settings) { s.MaxHeight = conv.P2160 },
	} {
		edit(&base)
		before := version()
		save(&targets.Conversion{Settings: base, RunOn: &node})
		if v := version(); v != before+1 {
			t.Errorf("settings %+v: version %d after %d, want one more", base, v, before)
		}
	}
	last := version()
	save(nil)
	save(&targets.Conversion{Settings: base})
	if v := version(); v <= last {
		t.Errorf("settings turned off and on again: version %d, want above %d", v, last)
	}
}
