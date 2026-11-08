package release_test

import (
	"testing"

	"github.com/jatassi/slipstream/internal/release"
)

func TestDescribe(t *testing.T) {
	for _, c := range []struct {
		name string
		size release.Fact[int64]
		want string
	}{
		{"The.Quiet.Orchard.2021.1080p.WEB-DL.DDP5.1.Atmos.DV.HDR10.x265-FLUX", release.Known(int64(6_657_199_308)), "WEBDL-1080p · Dolby Vision · DD+ 5.1 · FLUX · 6.2 GB"},
		{"The.Quiet.Orchard.2021.2160p.UHD.BluRay.REMUX.HDR.HEVC.TrueHD.7.1.Atmos-FGT", release.Known(int64(64_424_509_440)), "Remux-2160p · HDR10 · TrueHD 7.1 · FGT · 60 GB"},
		{"The.Quiet.Orchard.2021.720p.HDTV.AAC-GRP", release.Known(int64(734_003_200)), "HDTV-720p · AAC · GRP · 700 MB"},
		{"The.Quiet.Orchard.2021.1080p.BluRay.5.1.x264", release.Fact[int64]{}, "Bluray-1080p · 5.1"},
		{"The Quiet Orchard", release.Known(int64(512)), "512 bytes"},
	} {
		r := release.Title(c.name, release.TitleOnly{}, release.BuiltIn())
		r.Facts.Size = c.size
		if got := release.Describe(r); got != c.want {
			t.Errorf("Describe(%s) = %q, want %q", c.name, got, c.want)
		}
	}
}
