package release_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/release"
)

type stream struct {
	hdr      []release.HDR
	video    release.VideoCodec
	audio    []release.AudioCodec
	channels []release.Channels
}

func streamOf(f release.Facts) stream {
	hdr, _ := f.HDR.Get()
	video, _ := f.Video.Get()
	audio, _ := f.Audio.Get()
	channels, _ := f.Channels.Get()
	return stream{hdr, video, audio, channels}
}

func TestStreamFactsReadAsTRaSHFormatsTestTheTitle(t *testing.T) {
	for name, want := range map[string]stream{
		"The.Quiet.Orchard.2021.1080p.WEB-DL.DDP5.1.Atmos.DV.HDR10.x265-FLUX":         {[]release.HDR{"DV", "HDR10"}, "x265", []release.AudioCodec{"DD+"}, []release.Channels{"5.1"}},
		"The.Quiet.Orchard.2021.2160p.UHD.BluRay.REMUX.HDR.HEVC.TrueHD.7.1.Atmos-FGT": {[]release.HDR{"HDR10"}, "x265", []release.AudioCodec{"TrueHD"}, []release.Channels{"7.1"}},
		"The.Quiet.Orchard.2021.2160p.WEB-DL.DDP5.1.HDR10+.H.265-GRP":                 {[]release.HDR{"HDR10+"}, "x265", []release.AudioCodec{"DD+"}, []release.Channels{"5.1"}},
		"The.Quiet.Orchard.2021.2160p.UHD.BluRay.x265.HLG.DTS-HD.MA.5.1-GRP":          {[]release.HDR{"HLG"}, "x265", []release.AudioCodec{"DTS-HD MA"}, []release.Channels{"5.1"}},
		"The.Quiet.Orchard.2021.2160p.BluRay.SDR.DTS-X.7.1.x265-GRP":                  {[]release.HDR{"SDR"}, "x265", []release.AudioCodec{"DTS-HD MA"}, []release.Channels{"7.1"}},
		"The.Quiet.Orchard.2021.1080p.BluRay.DTS-ES.6.1.x264-GRP":                     {nil, "x264", []release.AudioCodec{"DTS"}, nil},
		"The.Quiet.Orchard.2021.720p.WEB.H264.AAC2.0-GRP":                             {nil, "x264", []release.AudioCodec{"AAC"}, []release.Channels{"2.0"}},
		"The.Quiet.Orchard.2021.1080p.BluRay.AC3.5.1.x264-GRP":                        {nil, "x264", []release.AudioCodec{"DD"}, []release.Channels{"5.1"}},
		"The.Quiet.Orchard.2021.1080p.BluRay.FLAC.1.0.x264-GRP":                       {nil, "x264", []release.AudioCodec{"FLAC"}, []release.Channels{"1.0"}},
		"The.Quiet.Orchard.2021.1080p.WEB-DL.Opus.2.0.AV1-GRP":                        {nil, "AV1", []release.AudioCodec{"Opus"}, []release.Channels{"2.0"}},
		"The.Quiet.Orchard.2021.1080p.WEB-DL.VP9.LPCM-GRP":                            {nil, "VP9", []release.AudioCodec{"PCM"}, nil},
		"The.Quiet.Orchard.2021.DVDRip.XviD.MP3-GRP":                                  {nil, "XviD", []release.AudioCodec{"MP3"}, nil},
		"The.Quiet.Orchard.2021.DVD9.MPEG2.AC3-GRP":                                   {nil, "MPEG2", []release.AudioCodec{"DD"}, nil},
		"The.Quiet.Orchard.2021.1080p.BluRay-GRP":                                     {},
	} {
		if got := streamOf(titleFacts(name)); !reflect.DeepEqual(got, want) {
			t.Errorf("%s reads %+v, want %+v", name, got, want)
		}
	}
}

// masking reads every name's title as the movie type does: the name
// with its title, the words before the year, replaced.
type masking struct{ release.TitleOnly }

func (masking) ReadName(p release.Prepared) release.Name {
	title, rest, _ := strings.Cut(p.Release.String(), ".2019.")
	return release.Name{Titles: []string{title}, Masked: "A.Movie.2019." + rest}
}

func TestStreamFactsReadTheNameWithItsTitleMasked(t *testing.T) {
	name := "Stereo.2019.1080p.WEB-DL.x264-GRP"
	if got, want := streamOf(release.Title(name, masking{}, nil).Facts), (stream{video: "x264"}); !reflect.DeepEqual(got, want) {
		t.Errorf("%s with its title masked reads %+v, want %+v", name, got, want)
	}
	if got := streamOf(titleFacts(name)).channels; !reflect.DeepEqual(got, []release.Channels{"2.0"}) {
		t.Errorf("%s read whole reads channels %v, want the title's 2.0", name, got)
	}
}
