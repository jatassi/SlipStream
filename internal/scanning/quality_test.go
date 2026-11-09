package scanning_test

import (
	"testing"

	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/scanning"
)

func TestQualityOf(t *testing.T) {
	tests := []struct {
		rel     string
		probed  release.Resolution
		quality release.Quality
		cut     release.CutKey
	}{
		{"Heat (1995)/Heat.1995.Bluray-1080p.mkv", release.R1080p, release.Bluray1080p, release.Standard},
		{"Heat (1995)/Heat.1995.Bluray-1080p.mkv", release.R2160p, release.Bluray2160p, release.Standard},
		{"Heat (1995) WEBDL/Heat.mkv", release.R1080p, release.WEBDL1080p, release.Standard},
		{"Show/Show.S01.1080p.BluRay-GRP/Show.S01E01.mkv", release.R1080p, release.Bluray1080p, release.Standard},
		{"Heat (1995) Remux/Heat WEBRip.mkv", release.R1080p, release.WEBRip1080p, release.Standard},
		{"Heat (1995)/Heat.mkv", release.R1080p, release.HDTV1080p, release.Standard},
		{"Heat.1995.DVDRip.mkv", release.R480p, release.DVDQuality, release.Standard},
		{"Blade Runner (1982)/Blade Runner (1982) {edition-Final Cut} Bluray-1080p.mkv", release.R1080p, release.Bluray1080p, "final-cut"},
		{"Blade Runner (1982) {edition-Final Cut}/Blade Runner.mkv", release.R1080p, release.HDTV1080p, release.Standard},
	}
	for _, tt := range tests {
		read, q := scanning.QualityOf(tt.rel, release.TitleOnly{}, release.BuiltIn(), tt.probed)
		cut := read.Facts.Cut
		if q != tt.quality || cut != tt.cut {
			t.Errorf("QualityOf(%q, %s) = %s, %s; want %s, %s", tt.rel, tt.probed, q, cut, tt.quality, tt.cut)
		}
	}
}

func TestRequalified(t *testing.T) {
	tests := []struct {
		name   string
		was    release.Quality
		rel    string
		probed release.Resolution
		want   release.Quality
	}{
		{"the same class keeps the quality", release.Bluray1080p, "Heat/Heat WEBDL.mkv", release.R1080p, release.Bluray1080p},
		{"a new class keeps the source", release.Bluray1080p, "Heat/Heat WEBDL.mkv", release.R2160p, release.Bluray2160p},
		{"576p is the SD class", release.WEBRip480p, "Heat/Heat.mkv", release.R576p, release.WEBRip480p},
		{"a source the class lacks reads the path", release.DVDQuality, "Heat/Heat WEBDL.mkv", release.R1080p, release.WEBDL1080p},
		{"a cinema source never changes", release.CAM, "Heat/Heat.mkv", release.R1080p, release.CAM},
	}
	for _, tt := range tests {
		if got := scanning.Requalified(tt.was, tt.rel, release.TitleOnly{}, release.BuiltIn(), tt.probed); got != tt.want {
			t.Errorf("%s: Requalified(%s, %q, %s) = %s, want %s", tt.name, tt.was, tt.rel, tt.probed, got, tt.want)
		}
	}
}
