package sim_test

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/jatassi/slipstream/fakes/internal/mediafix"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/internal/media"
)

func TestTheWalkProbesEachFileOfAReleaseOfFiles(t *testing.T) {
	sd := &scenario.MediaVideo{Width: 1920, Height: 1080, Seconds: 8, HDR: "sdr"}
	hdr := &scenario.MediaVideo{Width: 3840, Height: 2160, Seconds: 9, HDR: "hdr10"}
	r := scenario.Release{Name: "Lantern.S01", TVDB: &scenario.ReleaseEpisode{Series: 1001, Season: 1}, Files: []scenario.ReleaseFile{
		{Path: "Lantern.S01E01.mkv", Video: sd, Size: 1 << 20},
		{Path: "Season 01/Lantern.S01E02.mkv", Video: hdr, Size: 1 << 20},
		{Path: "Lantern.S01E03.mkv", Unreadable: true, Size: 1 << 20},
		{Path: "Lantern.nfo", Size: 120},
	}}
	p := sim.ReleaseProbes(r)
	in := filepath.Join(t.TempDir(), "downloads")
	for _, f := range r.Files[:2] {
		got, err := p.Probe(t.Context(), filepath.Join(in, "Lantern.S01", filepath.FromSlash(f.Path)))
		if want := mediafix.Want(f.Media()); err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("probe of %s = %+v, %v; want %+v", f.Path, got, err, want)
		}
	}
	var unreadable *media.UnreadableError
	if _, err := p.Probe(t.Context(), filepath.Join(in, "Lantern.S01", "Lantern.S01E03.mkv")); !errors.As(err, &unreadable) {
		t.Errorf("probe of the unreadable episode = %v, want an UnreadableError", err)
	}
	if _, err := p.Probe(t.Context(), filepath.Join(in, "Lantern.S01", "Lantern.S01E01.mkv.part")); err == nil || errors.As(err, &unreadable) {
		t.Errorf("probe of a file the release doesn't list = %v, want no probe", err)
	}
	if _, err := p.Probe(t.Context(), filepath.Join(in, "Lantern.S01", "Lantern.nfo")); err == nil || errors.As(err, &unreadable) {
		t.Errorf("probe of the nfo = %v, want no probe", err)
	}

	one := sim.ReleaseProbes(scenario.Release{Name: "Orchard.2021", TMDBID: 1, Video: &scenario.Video{Width: 1280, Height: 720, Seconds: 6}})
	got, err := one.Probe(t.Context(), filepath.Join(in, "Orchard.2021.mkv"))
	if err != nil || got.Video.Width != 1280 || got.Duration.Seconds() != 6 {
		t.Errorf("probe of a release of one video = %+v, %v; want its 1280 wide, 6 second video", got, err)
	}
}
