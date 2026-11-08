package mediafix_test

import (
	"bytes"
	"crypto/sha256"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/internal/mediafix"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/media"
)

func builder(t *testing.T) (*mediafix.Builder, mediafix.Tools) {
	t.Helper()
	tools, err := mediafix.NativeTools()
	if err != nil {
		t.Fatal(err)
	}
	b, err := mediafix.NewBuilder(t.Context(), tools, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return b, tools
}

func file(name string, v scenario.MediaVideo) scenario.MediaFile {
	return scenario.MediaFile{Path: name + ".mkv", Video: v}
}

// tracks holds every codec at every channel count it encodes, an untagged
// track and every subtitle disposition.
var tracks = scenario.MediaFile{
	Path:  "Tracks (2020)/Tracks (2020).mkv",
	Video: scenario.MediaVideo{Width: 640, Height: 360, Seconds: 2, HDR: "sdr"},
	Audio: []scenario.MediaAudio{
		{Codec: "truehd", Channels: 6, Language: "eng", Default: true, Title: "Director's Commentary"},
		{Codec: "eac3", Channels: 6, Language: "ger"},
		{Codec: "ac3", Channels: 2, Language: "spa"},
		{Codec: "aac", Channels: 1, Language: "jpn"},
		{Codec: "aac", Channels: 6, Language: "jpn"},
		{Codec: "aac", Channels: 8},
		{Codec: "flac", Channels: 8, Language: "fre"},
	},
	Subtitles: []scenario.MediaSubtitle{{Language: "eng", Default: true}, {Language: "fre", Forced: true}, {Language: "ita", Default: true, Forced: true}, {}},
}

func TestBuildReadsAsWant(t *testing.T) {
	b, tools := builder(t)
	files := []scenario.MediaFile{tracks, file("1080p dv8", scenario.MediaVideo{Width: 1920, Height: 1080, Seconds: 1, HDR: "dv8"})}
	for _, hdr := range scenario.HDRKinds {
		files = append(files, file(hdr, scenario.MediaVideo{Width: 320, Height: 180, Seconds: 1, HDR: hdr}))
	}
	for _, f := range files {
		t.Run(f.Path, func(t *testing.T) {
			built, err := b.Build(t.Context(), f)
			if err != nil {
				t.Fatal(err)
			}
			got, err := media.FFprobe{Path: tools.FFprobe}.Probe(t.Context(), built)
			if err != nil {
				t.Fatal(err)
			}
			if got.Bitrate <= 0 {
				t.Errorf("Probe read a bit rate of %d", got.Bitrate)
			}
			got.Bitrate, got.Duration = 0, got.Duration.Round(time.Second)
			if want := mediafix.Want(f); !reflect.DeepEqual(got, want) {
				t.Fatalf("Probe = %+v\nwant %+v", got, want)
			}
		})
	}
}

func TestBuildIsTheSameOnEveryRun(t *testing.T) {
	files := []scenario.MediaFile{
		tracks,
		file("hdr10plus", scenario.MediaVideo{Width: 320, Height: 180, Seconds: 2, HDR: "hdr10plus"}),
		file("dv5", scenario.MediaVideo{Width: 320, Height: 180, Seconds: 2, HDR: "dv5"}),
		file("dv8", scenario.MediaVideo{Width: 320, Height: 180, Seconds: 2, HDR: "dv8"}),
	}
	sums := func() [][sha256.Size]byte {
		b, _ := builder(t)
		var out [][sha256.Size]byte
		for _, f := range files {
			built, err := b.Build(t.Context(), f)
			if err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(built)
			if err != nil {
				t.Fatal(err)
			}
			out = append(out, sha256.Sum256(data))
		}
		return out
	}
	if first, second := sums(), sums(); !slices.Equal(first, second) {
		t.Fatalf("two builds hash %x and %x", first, second)
	}
}

// TestCapturesHaveNotDrifted captures again what internal/media's Parse
// tests read and compares it with the checked-in files, so a recipe
// changed without a capture fails here.
func TestCapturesHaveNotDrifted(t *testing.T) {
	b, _ := builder(t)
	dir := t.TempDir()
	for _, c := range mediafix.Captures {
		if err := b.Capture(t.Context(), c, dir); err != nil {
			t.Fatal(err)
		}
	}
	checkedIn := filepath.Join("..", "..", "..", "internal", "media", "testdata", "ffprobe")
	if got, want := names(t, checkedIn), names(t, dir); !slices.Equal(got, want) {
		t.Fatalf("%s holds %v, the captures are %v", checkedIn, got, want)
	}
	for _, name := range names(t, dir) {
		got, err := os.ReadFile(filepath.Join(checkedIn, name))
		if err != nil {
			t.Fatal(err)
		}
		want, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s differs from a fresh capture; run go run ./fakes/internal/mediafix/capture internal/media/testdata/ffprobe", name)
		}
	}
}

func names(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		out = append(out, e.Name())
	}
	return out
}
