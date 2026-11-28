package encode_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/encode"
	"github.com/jatassi/slipstream/internal/language"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/release"
)

func raw() encode.Raw {
	return encode.Raw{
		Codec: "hevc", MaxHeight: 1080, Kbps: 8000, Depth: 10, Range: "hdr10", KeepDV: true,
		Curve: "bt2390", Audio: "copy", DropCommentary: true, Container: "mkv",
	}
}

func TestParse(t *testing.T) {
	cases := []struct {
		name    string
		edit    func(r *encode.Raw)
		refused string
	}{
		{"every field known", func(*encode.Raw) {}, ""},
		{"the lowest bitrate", func(r *encode.Raw) { r.Kbps = 100 }, ""},
		{"the highest bitrate", func(r *encode.Raw) { r.Kbps = 200_000 }, ""},
		{"a bitrate below 100", func(r *encode.Raw) { r.Kbps = 99 }, "Kbps"},
		{"a bitrate above 200,000", func(r *encode.Raw) { r.Kbps = 200_001 }, "Kbps"},
		{"10 bit H.264", func(r *encode.Raw) { r.Codec, r.KeepDV = "h264", false }, "Depth"},
		{"8 bit H.264", func(r *encode.Raw) { r.Codec, r.Depth, r.KeepDV = "h264", 8, false }, ""},
		{"Dolby Vision kept in SDR", func(r *encode.Raw) { r.Range = "sdr" }, "KeepDV"},
		{"Dolby Vision kept in H.264", func(r *encode.Raw) { r.Codec, r.Depth = "h264", 8 }, "KeepDV"},
		{"Dolby Vision kept from the source range", func(r *encode.Raw) { r.Range = "source" }, ""},
		{"an unknown codec", func(r *encode.Raw) { r.Codec = "vp9" }, "Codec"},
		{"an unknown height", func(r *encode.Raw) { r.MaxHeight = 1440 }, "MaxHeight"},
		{"an unknown depth", func(r *encode.Raw) { r.Depth = 12 }, "Depth"},
		{"an unknown range", func(r *encode.Raw) { r.Range = "hlg" }, "Range"},
		{"an unknown curve", func(r *encode.Raw) { r.Curve = "aces" }, "Curve"},
		{"an unknown audio choice", func(r *encode.Raw) { r.Audio = "opus" }, "Audio"},
		{"an unknown container", func(r *encode.Raw) { r.Container = "avi" }, "Container"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := raw()
			c.edit(&r)
			s, err := encode.Parse(r)
			field := ""
			if fe, ok := errors.AsType[*encode.FieldError](err); ok {
				field = fe.Field
			} else if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if field != c.refused {
				t.Fatalf("refused %q, want %q", field, c.refused)
			}
			if c.refused == "" && (string(s.Codec) != r.Codec || s.Kbps != r.Kbps || s.KeepDV != r.KeepDV) {
				t.Fatalf("Parse(%+v) = %+v", r, s)
			}
		})
	}
}

func TestShapeKey(t *testing.T) {
	cases := []struct {
		s    encode.Settings
		want string
	}{
		{encode.Settings{Codec: encode.H264, Depth: 8, Range: encode.SDR, Curve: encode.BT2390, Kbps: 4000, MaxHeight: 720}, "h264.8.sdr.bt2390"},
		{encode.Settings{Codec: encode.HEVC, Depth: 10, Range: encode.HDR10, KeepDV: true, Curve: encode.Hable}, "hevc.10.hdr10.hable.dv"},
	}
	for _, c := range cases {
		if got := c.s.Shape().String(); got != c.want {
			t.Errorf("%+v keys as %q, want %q", c.s, got, c.want)
		}
	}
}

func video(codec string, depth int, hdr ...release.HDR) media.Record {
	if hdr == nil {
		hdr = []release.HDR{"SDR"}
	}
	return media.Record{Version: media.Version, Video: media.Video{Codec: codec, Width: 3840, Height: 2160, BitDepth: depth, HDR: hdr}}
}

func dv(profile, compatibility int, hdr ...release.HDR) media.Record {
	r := video("hevc", 10, hdr...)
	r.DolbyVision = &media.DolbyVision{Profile: profile, Compatibility: compatibility}
	return r
}

func TestClassOf(t *testing.T) {
	cases := []struct {
		name string
		r    media.Record
		want encode.Class
	}{
		{"H.264 SDR", video("h264", 8), encode.ClassH264x8},
		{"HEVC 8 bit SDR", video("hevc", 8), encode.ClassHEVCx8},
		{"HEVC 10 bit SDR", video("hevc", 10), encode.ClassHEVCx10},
		{"AV1 8 bit SDR", video("av1", 8), encode.ClassAV1x8},
		{"AV1 10 bit SDR", video("av1", 10), encode.ClassAV1x10},
		{"HDR10", video("hevc", 10, "HDR10"), encode.ClassHDR10},
		{"HDR10+", video("hevc", 10, "HDR10+", "HDR10"), encode.ClassHDR10},
		{"AV1 HDR10", video("av1", 10, "HDR10"), encode.ClassHDR10},
		{"HLG", video("hevc", 10, "HLG"), encode.ClassHLG},
		{"Dolby Vision 5", dv(5, 0, "DV"), encode.ClassDV5},
		{"Dolby Vision 8.1", dv(8, 1, "DV", "HDR10"), encode.ClassDV81},
		{"Dolby Vision 8.4", dv(8, 4, "DV", "HLG"), encode.ClassDV81},
		{"Dolby Vision 7", dv(7, 6, "DV", "HDR10"), encode.ClassDV81},
		{"VC-1", video("vc1", 8), encode.ClassDecodeSDR},
		{"MPEG-2", video("mpeg2video", 8), encode.ClassDecodeSDR},
		{"MPEG-2 HDR10", video("mpeg2video", 10, "HDR10"), encode.ClassDecodeHDR10},
		{"10 bit H.264", video("h264", 10), encode.ClassDecodeSDR},
	}
	for _, c := range cases {
		if got := encode.ClassOf(c.r); got != c.want {
			t.Errorf("%s classes as %q, want %q", c.name, got, c.want)
		}
	}
}

type read struct {
	quality    release.Quality
	modifier   bool
	video      release.VideoCodec
	resolution release.Resolution
	hdr        []release.HDR
}

func readOf(r release.Read) read {
	q, _ := r.Facts.Quality.Get()
	_, mod := r.Facts.Modifier.Get()
	v, _ := r.Facts.Video.Get()
	res, _ := r.Facts.Resolution.Get()
	hdr, _ := r.Facts.HDR.Get()
	return read{q, mod, v, res, hdr}
}

func source(q release.Quality, res release.Resolution, codec release.VideoCodec, hdr ...release.HDR) release.Read {
	return release.Read{Release: "Heat.1995.REPACK.Directors.Cut.BluRay.REMUX-FraMeSToR", Facts: release.Facts{
		Quality: release.Known(q), Source: release.Known(release.OriginBluray), Modifier: release.Known(release.ModRemux),
		Resolution: release.Known(res), Video: release.Known(codec), HDR: release.Known(hdr),
		Group: release.Known("FraMeSToR"), Revision: release.Revision{Version: 2, Repack: true}, Edition: release.Known("Directors Cut"),
	}}
}

func TestPredictAndRewrite(t *testing.T) {
	sdr1080 := encode.Settings{Codec: encode.H264, MaxHeight: 1080, Kbps: 8000, Depth: 8, Range: encode.SDR, Curve: encode.BT2390, Audio: encode.CopyAudio, Container: encode.MKV}
	uhd := video("hevc", 10, "HDR10")
	uhd.Bitrate = 60_000_000
	fhd := video("h264", 8)
	fhd.Width, fhd.Height, fhd.Bitrate = 1920, 1080, 30_000_000
	madeFHD := video("h264", 8)
	madeFHD.Width, madeFHD.Height = 1920, 1080
	cases := []struct {
		name string
		got  func() release.Read
		want read
	}{
		{
			"a 2160p Remux made into 1080p",
			func() release.Read {
				return encode.Predict(source(release.Remux2160p, release.R2160p, "x265", "HDR10"), uhd, sdr1080)
			},
			read{release.Bluray1080p, false, "x264", release.R1080p, []release.HDR{"SDR"}},
		},
		{
			"a 1080p Remux whose video is copied",
			func() release.Read {
				return encode.Predict(source(release.Remux1080p, release.R1080p, "x264", "SDR"), fhd, sdr1080)
			},
			read{release.Remux1080p, true, "x264", release.R1080p, []release.HDR{"SDR"}},
		},
		{
			"the probe of a 2160p Remux made into 1080p",
			func() release.Read {
				return encode.Rewrite(source(release.Remux2160p, release.R2160p, "x265", "HDR10"), madeFHD, false)
			},
			read{release.Bluray1080p, false, "x264", release.R1080p, []release.HDR{"SDR"}},
		},
		{
			"the probe of a copied 1080p Remux",
			func() release.Read {
				return encode.Rewrite(source(release.Remux1080p, release.R1080p, "x264", "SDR"), madeFHD, true)
			},
			read{release.Remux1080p, true, "x264", release.R1080p, []release.HDR{"SDR"}},
		},
		{
			"the probe of a 1080p Remux re-encoded at the same codec and size",
			func() release.Read {
				return encode.Rewrite(source(release.Remux1080p, release.R1080p, "x264", "SDR"), madeFHD, false)
			},
			read{release.Bluray1080p, false, "x264", release.R1080p, []release.HDR{"SDR"}},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := c.got()
			if got := readOf(r); got.quality != c.want.quality || got.modifier != c.want.modifier || got.video != c.want.video ||
				got.resolution != c.want.resolution || !slices.Equal(got.hdr, c.want.hdr) {
				t.Errorf("reads as %+v, want %+v", got, c.want)
			}
			group, _ := r.Facts.Group.Get()
			edition, _ := r.Facts.Edition.Get()
			if group != "FraMeSToR" || r.Facts.Revision != (release.Revision{Version: 2, Repack: true}) || edition != "Directors Cut" {
				t.Errorf("group %q, revision %v, edition %q, want the source's", group, r.Facts.Revision, edition)
			}
			if r.Release != "" {
				t.Errorf("a made file has the release name %q", r.Release)
			}
		})
	}
}

func TestPredictRange(t *testing.T) {
	s := func(r encode.Range, keepDV bool) encode.Settings {
		return encode.Settings{Codec: encode.HEVC, MaxHeight: 1080, Kbps: 8000, Depth: 10, Range: r, KeepDV: keepDV, Curve: encode.BT2390}
	}
	cases := []struct {
		name string
		m    media.Record
		s    encode.Settings
		want []release.HDR
	}{
		{"HDR10 to SDR", video("hevc", 10, "HDR10"), s(encode.SDR, false), []release.HDR{"SDR"}},
		{"SDR stays SDR in an HDR10 range", video("hevc", 10), s(encode.HDR10, false), []release.HDR{"SDR"}},
		{"HDR10+ loses its dynamic metadata", video("hevc", 10, "HDR10+", "HDR10"), s(encode.SourceRange, false), []release.HDR{"HDR10"}},
		{"HLG keeps HLG from the source range", video("hevc", 10, "HLG"), s(encode.SourceRange, false), []release.HDR{"HLG"}},
		{"HLG becomes HDR10", video("hevc", 10, "HLG"), s(encode.HDR10, false), []release.HDR{"HDR10"}},
		{"Dolby Vision 8.1 kept", dv(8, 1, "DV", "HDR10"), s(encode.HDR10, true), []release.HDR{"DV", "HDR10"}},
		{"Dolby Vision 8.1 dropped", dv(8, 1, "DV", "HDR10"), s(encode.SourceRange, false), []release.HDR{"HDR10"}},
		{"Dolby Vision 7 comes out as HDR10", dv(7, 6, "DV", "HDR10"), s(encode.SourceRange, true), []release.HDR{"HDR10"}},
	}
	for _, c := range cases {
		hdr, _ := encode.Predict(release.Read{}, c.m, c.s).Facts.HDR.Get()
		if !slices.Equal(hdr, c.want) {
			t.Errorf("%s makes %v, want %v", c.name, hdr, c.want)
		}
	}
}

func TestKbps(t *testing.T) {
	s := encode.Settings{Codec: encode.H264, MaxHeight: 1080, Kbps: 8000, Depth: 8, Range: encode.SDR}
	cases := []struct {
		name         string
		w, h         int
		bitrate      int64
		wantW, wantH int
		want         int
	}{
		{"a 1080p source", 1920, 1080, 40_000_000, 1920, 1080, 8000},
		{"a 2160p source scaled down", 3840, 2160, 60_000_000, 1920, 1080, 8000},
		{"a cropped 2160p source", 3840, 1600, 60_000_000, 1920, 800, 5926},
		{"a 720p source", 1280, 720, 40_000_000, 1280, 720, 3556},
		{"a 4:3 source", 1440, 1080, 40_000_000, 1440, 1080, 6000},
		{"a source below the bitrate", 1920, 1080, 5_000_000, 1920, 1080, 5000},
		{"a source of no stated bitrate", 1920, 1080, 0, 1920, 1080, 8000},
		{"an odd width", 1919, 1080, 40_000_000, 1918, 1080, 7992},
	}
	for _, c := range cases {
		m := media.Record{Bitrate: c.bitrate, Video: media.Video{Width: c.w, Height: c.h}}
		w, h := encode.Out(m, s)
		if got := encode.Kbps(m, s); w != c.wantW || h != c.wantH || got != c.want {
			t.Errorf("%s: %dx%d at %d kbps, want %dx%d at %d", c.name, w, h, got, c.wantW, c.wantH, c.want)
		}
	}
}

func TestBytes(t *testing.T) {
	s := encode.Settings{Codec: encode.HEVC, MaxHeight: 1080, Kbps: 8000, Depth: 10, Range: encode.SDR, Audio: encode.StereoAAC, Container: encode.MKV}
	m := video("hevc", 10, "HDR10")
	m.Bitrate, m.Duration = 60_000_000, time.Hour
	m.Audio = []media.Audio{{Codec: "truehd", Channels: 8}}
	if got, want := encode.Bytes(m, s), int64((8000+192)*1000/8*3600); got != want {
		t.Errorf("an hour at 8,000 kbps with stereo AAC is %d bytes, want %d", got, want)
	}
}

func codes(t *testing.T, s ...string) []language.Code {
	t.Helper()
	out := make([]language.Code, len(s))
	for i, c := range s {
		var err error
		if out[i], err = language.Parse(c); err != nil {
			t.Fatal(err)
		}
	}
	return out
}

func tracked() media.Record {
	return media.Record{
		Audio: []media.Audio{
			{Codec: "truehd", Channels: 8, Language: "en", Default: true},
			{Codec: "ac3", Channels: 2, Language: "en", Title: "Commentary with the director"},
			{Codec: "dts", Channels: 6, Language: "fr"},
			{Codec: "aac", Channels: 2, Language: "ja"},
			{Codec: "aac", Channels: 2},
		},
		Subtitles: []media.Subtitle{
			{Codec: "subrip", Language: "en"},
			{Codec: "hdmv_pgs_subtitle", Image: true, Language: "en"},
			{Codec: "subrip", Language: "fr"},
			{Codec: "hdmv_pgs_subtitle", Image: true, Language: "de"},
		},
	}
}

func TestTracks(t *testing.T) {
	cases := []struct {
		name      string
		keep      []language.Code
		audio     encode.Audio
		drop      bool
		container encode.Container
		want      encode.Selection
	}{
		{"every language, commentary dropped", nil, encode.CopyAudio, true, encode.MKV, encode.Selection{Audio: []int{0, 2, 3, 4}, Subtitles: []int{0, 1, 2, 3}}},
		{"the kept languages and untagged audio", codes(t, "en", "ja"), encode.CopyAudio, true, encode.MKV, encode.Selection{Audio: []int{0, 3, 4}, Subtitles: []int{0, 1}}},
		{"commentary kept", codes(t, "en"), encode.CopyAudio, false, encode.MKV, encode.Selection{Audio: []int{0, 1, 4}, Subtitles: []int{0, 1}}},
		{"MP4 drops image subtitles", codes(t, "en"), encode.CopyAudio, true, encode.MP4, encode.Selection{Audio: []int{0, 4}, Subtitles: []int{0}}},
		{"a re-encode from the default track", codes(t, "en", "fr"), encode.StereoAAC, true, encode.MKV, encode.Selection{Audio: []int{0}, Subtitles: []int{0, 1, 2}}},
		{"a re-encode from the first kept track", codes(t, "fr"), encode.AC3AndStereoAAC, true, encode.MKV, encode.Selection{Audio: []int{2}, Subtitles: []int{2}}},
	}
	for _, c := range cases {
		s := encode.Settings{Audio: c.audio, DropCommentary: c.drop, Container: c.container}
		got := encode.Tracks(tracked(), s, c.keep)
		if !slices.Equal(got.Audio, c.want.Audio) || !slices.Equal(got.Subtitles, c.want.Subtitles) {
			t.Errorf("%s keeps %+v, want %+v", c.name, got, c.want)
		}
	}
}

func TestFits(t *testing.T) {
	cases := []struct {
		name      string
		keep      []language.Code
		audio     encode.Audio
		container encode.Container
		want      *encode.Unfit
	}{
		{"TrueHD copied into MP4", nil, encode.CopyAudio, encode.MP4, &encode.Unfit{Code: encode.ContainerAudio, Stream: 0}},
		{"DTS copied into MP4", codes(t, "fr"), encode.CopyAudio, encode.MP4, &encode.Unfit{Code: encode.ContainerAudio, Stream: 2}},
		{"AAC copied into MP4", codes(t, "ja"), encode.CopyAudio, encode.MP4, nil},
		{"TrueHD re-encoded for MP4", nil, encode.StereoAAC, encode.MP4, nil},
		{"TrueHD copied into MKV", nil, encode.CopyAudio, encode.MKV, nil},
	}
	for _, c := range cases {
		s := encode.Settings{Audio: c.audio, Container: c.container, DropCommentary: true}
		got := encode.Fits(s, tracked(), c.keep)
		if (got == nil) != (c.want == nil) || (got != nil && *got != *c.want) {
			t.Errorf("%s fits as %+v, want %+v", c.name, got, c.want)
		}
	}
}
