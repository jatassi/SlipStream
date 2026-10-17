package release_test

import (
	"testing"

	"github.com/jatassi/slipstream/internal/release"
)

func TestResolutionOf(t *testing.T) {
	for _, c := range []struct {
		width, height int
		want          release.Resolution
	}{
		{3840, 2160, release.R2160p},
		{3840, 1600, release.R2160p},
		{1920, 1080, release.R1080p},
		{1920, 800, release.R1080p},
		{1440, 1080, release.R1080p},
		{1280, 720, release.R720p},
		{1280, 536, release.R720p},
		{1024, 576, release.R576p},
		{720, 576, release.R576p},
		{720, 480, release.R480p},
		{640, 360, release.R480p},
	} {
		if got := release.ResolutionOf(c.width, c.height); got != c.want {
			t.Errorf("ResolutionOf(%d, %d) = %s, want %s", c.width, c.height, got, c.want)
		}
	}
}

func TestParseResolution(t *testing.T) {
	for _, r := range resolutions {
		if got, err := release.ParseResolution(r.String()); err != nil || got != r {
			t.Errorf("ParseResolution(%q) = %v, %v", r.String(), got, err)
		}
	}
	for _, bad := range []string{"", "4k", "1080", "1080P"} {
		if _, err := release.ParseResolution(bad); err == nil {
			t.Errorf("ParseResolution(%q) succeeded", bad)
		}
	}
}
