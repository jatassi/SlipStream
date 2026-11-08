package media_test

import (
	"errors"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/release"
)

func equal(a, b any) bool { return reflect.DeepEqual(a, b) }

func capture(t *testing.T, name string) (streams, frame []byte) {
	t.Helper()
	streams, err := os.ReadFile("testdata/ffprobe/" + name + ".streams.json")
	if err != nil {
		t.Fatal(err)
	}
	frame, err = os.ReadFile("testdata/ffprobe/" + name + ".frame.json")
	if err != nil {
		t.Fatal(err)
	}
	return streams, frame
}

var h264 = media.Video{Codec: "h264", Profile: "Constrained Baseline", Width: 320, Height: 180, BitDepth: 8, FrameRate: media.Rate{Num: 24, Den: 1}, HDR: []release.HDR{"SDR"}}

func hevc(hdr []release.HDR, dv *media.DolbyVision) media.Video {
	return media.Video{Codec: "hevc", Profile: "Main 10", Width: 320, Height: 180, BitDepth: 10, FrameRate: media.Rate{Num: 24, Den: 1}, HDR: hdr, DolbyVision: dv}
}

func surround(codec, profile string) media.Audio {
	return media.Audio{Codec: codec, Profile: profile, Atmos: profile != "", Channels: 6, Layout: "5.1(side)", Language: "en"}
}

// TestParseReadsTheBundledFFprobe parses captures of files the bundled
// tools made, each with the commands beside it in testdata/ffprobe.
func TestParseReadsTheBundledFFprobe(t *testing.T) {
	mkv := func(d time.Duration, bitrate int64, v media.Video, audio ...media.Audio) media.Record {
		return media.Record{Version: media.Version, Container: "matroska,webm", Duration: d, Bitrate: bitrate, Video: v, Audio: audio}
	}
	multi := mkv(1021*time.Millisecond, 2330718, h264,
		media.Audio{Codec: "eac3", Channels: 6, Layout: "5.1(side)", Language: "en", Default: true, Title: "Surround"},
		media.Audio{Codec: "aac", Profile: "LC", Channels: 2, Layout: "stereo", Language: "ja"},
		media.Audio{Codec: "truehd", Channels: 6, Layout: "5.1(side)", Language: "fr"},
	)
	multi.Subtitles = []media.Subtitle{{Codec: "subrip", Language: "es", Forced: true}, {Codec: "subrip", Language: "en"}}
	elementary := h264
	elementary.FrameRate = media.Rate{Num: 48, Den: 1}

	for name, want := range map[string]media.Record{
		"sdr-h264":         mkv(1021*time.Millisecond, 721747, h264, media.Audio{Codec: "aac", Profile: "LC", Channels: 2, Layout: "stereo", Language: "en"}),
		"hdr10":            mkv(time.Second, 254256, hevc([]release.HDR{"HDR10"}, nil)),
		"hdr10plus":        mkv(time.Second, 266736, hevc([]release.HDR{"HDR10+", "HDR10"}, nil)),
		"hlg":              mkv(time.Second, 232936, hevc([]release.HDR{"HLG"}, nil)),
		"dv5":              mkv(time.Second, 226232, hevc([]release.HDR{"DV"}, &media.DolbyVision{Profile: 5, Level: 1, Compatibility: 0})),
		"dv8":              mkv(time.Second, 270600, hevc([]release.HDR{"DV", "HDR10"}, &media.DolbyVision{Profile: 8, Level: 1, Compatibility: 1})),
		"multi-audio-subs": multi,
		"no-duration":      {Version: media.Version, Container: "h264", Video: elementary},
		"eac3-atmos":       mkv(time.Second, 1063808, h264, surround("eac3", "Dolby Digital Plus + Dolby Atmos")),
		"truehd-atmos":     mkv(time.Second, 1780160, h264, surround("truehd", "Dolby TrueHD + Dolby Atmos")),
	} {
		t.Run(name, func(t *testing.T) {
			got, err := media.Parse(capture(t, name))
			if err != nil || !equal(got, want) {
				t.Fatalf("Parse = %+v, %v\nwant %+v", got, err, want)
			}
		})
	}
}

func TestParseReadsTheTransferFromTheFirstFrame(t *testing.T) {
	streams := []byte(`{"format":{"format_name":"matroska,webm"},"streams":[{"codec_type":"video","codec_name":"hevc","color_transfer":"unknown"}]}`)
	frame := []byte(`{"frames":[{"color_transfer":"arib-std-b67"}]}`)
	got, err := media.Parse(streams, frame)
	if err != nil || !equal(got.HDR, []release.HDR{"HLG"}) {
		t.Fatalf("Parse = %v, %v; want HLG from the frame", got.HDR, err)
	}
}

func TestParseSkipsCoverArt(t *testing.T) {
	streams := []byte(`{"format":{"format_name":"mov,mp4,m4a,3gp,3g2,mj2"},"streams":[
		{"codec_type":"video","codec_name":"mjpeg","width":600,"height":900,"disposition":{"attached_pic":1}},
		{"codec_type":"video","codec_name":"h264","width":1920,"height":1080}]}`)
	got, err := media.Parse(streams, []byte(`{"frames":[]}`))
	if err != nil || got.Codec != "h264" || got.Width != 1920 {
		t.Fatalf("Parse = %+v, %v; want the h264 stream", got.Video, err)
	}
}

func TestParseRefusesAFileWithoutVideo(t *testing.T) {
	streams := []byte(`{"format":{"format_name":"mp4","duration":"3.5"},"streams":[{"codec_type":"audio","codec_name":"aac"}]}`)
	_, err := media.Parse(streams, []byte(`{}`))
	if unreadable, ok := errors.AsType[*media.UnreadableError](err); !ok || unreadable.Line != "no video stream" {
		t.Fatalf("Parse(audio only) = %v, want ffprobe can't read the file: no video stream", err)
	}
}
