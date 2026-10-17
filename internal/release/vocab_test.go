package release_test

import (
	"testing"

	"github.com/jatassi/slipstream/internal/release"
)

func roundTrips[T ~string](t *testing.T, all []T, parse func(string) (T, error), bad ...string) {
	t.Helper()
	for _, v := range all {
		if got, err := parse(string(v)); err != nil || got != v {
			t.Errorf("parse(%q) = %q, %v", v, got, err)
		}
	}
	for _, b := range bad {
		if _, err := parse(b); err == nil {
			t.Errorf("parse(%q) succeeded", b)
		}
	}
}

func TestVocabularies(t *testing.T) {
	roundTrips(t, release.HDRs(), release.ParseHDR, "", "HDR", "hdr10", "Dolby Vision")
	roundTrips(t, release.VideoCodecs(), release.ParseVideoCodec, "", "h264", "HEVC", "X264")
	roundTrips(t, release.AudioCodecs(), release.ParseAudioCodec, "", "DTS-HD", "EAC3", "truehd")
	roundTrips(t, release.ChannelLayouts(), release.ParseChannels, "", "6.1", "5.1.2", "stereo")
}

func TestVideoCodecOfAProbedStream(t *testing.T) {
	for codec, want := range map[string]release.VideoCodec{
		"h264": "x264", "hevc": "x265", "av1": "AV1", "vp9": "VP9", "mpeg4": "XviD", "mpeg2video": "MPEG2",
	} {
		if got, known := release.VideoCodecOf(codec).Get(); !known || got != want {
			t.Errorf("VideoCodecOf(%q) = %q, %t, want %q", codec, got, known, want)
		}
	}
	for _, unknown := range []string{"", "vc1", "H264", "x264", "prores"} {
		if got, known := release.VideoCodecOf(unknown).Get(); known {
			t.Errorf("VideoCodecOf(%q) = %q, want unknown", unknown, got)
		}
	}
}

func TestParseLanguage(t *testing.T) {
	for in, want := range map[string]release.Language{
		"en": "en", "EN": "en", "eng": "en", "fre": "fr", "fra": "fr", "ger": "de", "ja": "ja",
		"haw": "haw", "original": release.Original,
	} {
		if got, err := release.ParseLanguage(in); err != nil || got != want {
			t.Errorf("ParseLanguage(%q) = %q, %v, want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "zz", "und", "mul", "zxx", "en-US", "english", "Original"} {
		if got, err := release.ParseLanguage(bad); err == nil {
			t.Errorf("ParseLanguage(%q) = %q, want an error", bad, got)
		}
	}
}
