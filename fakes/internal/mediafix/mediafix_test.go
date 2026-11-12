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
			if why := mediafix.Lacks(f.Video.HDR); why != "" {
				t.Skip(why)
			}
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
// changed without a capture fails here. The commands must match byte for
// byte, and each capture must parse to the same record but for its bit
// rate, since another target's build of an encoder may write other bytes.
func TestCapturesHaveNotDrifted(t *testing.T) {
	b, _ := builder(t)
	fresh := t.TempDir()
	for _, c := range mediafix.Captures {
		if err := b.Capture(t.Context(), c, fresh); err != nil {
			t.Fatal(err)
		}
	}
	checkedIn := filepath.Join("..", "..", "..", "internal", "media", "testdata", "ffprobe")
	if got, want := names(t, checkedIn), names(t, fresh); !slices.Equal(got, want) {
		t.Fatalf("%s holds %v, the captures are %v", checkedIn, got, want)
	}
	const again = "run go run ./fakes/internal/mediafix/capture internal/media/testdata/ffprobe"
	for _, c := range mediafix.Captures {
		if got, want := read(t, checkedIn, c.Name+".cmd"), read(t, fresh, c.Name+".cmd"); !bytes.Equal(got, want) {
			t.Errorf("%s.cmd differs from the recipe's commands; %s", c.Name, again)
		}
		if why := mediafix.Lacks(c.File.Video.HDR); why != "" {
			t.Logf("%s isn't compared: %s", c.Name, why)
			continue
		}
		got, want := parse(t, checkedIn, c.Name), parse(t, fresh, c.Name)
		got.Bitrate, want.Bitrate = 0, 0
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s's capture parses as %+v, a fresh one as %+v; %s", c.Name, got, want, again)
		}
	}
}

func read(t *testing.T, dir, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func parse(t *testing.T, dir, name string) media.Record {
	t.Helper()
	r, err := media.Parse(read(t, dir, name+".streams.json"), read(t, dir, name+".frame.json"))
	if err != nil {
		t.Fatal(err)
	}
	return r
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

func TestBuildDistinctMakesFilesOfTheirOwnThatReadAlike(t *testing.T) {
	b, tools := builder(t)
	f := file("episode", scenario.MediaVideo{Width: 320, Height: 180, Seconds: 1, HDR: "sdr"})
	var sums [][sha256.Size]byte
	for _, key := range []string{"Pack/E01.mkv", "Pack/E02.mkv", "Pack/E01.mkv"} {
		built, err := b.BuildDistinct(t.Context(), f, key)
		if err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(built)
		if err != nil {
			t.Fatal(err)
		}
		sums = append(sums, sha256.Sum256(data))
		got, err := media.FFprobe{Path: tools.FFprobe}.Probe(t.Context(), built)
		if err != nil {
			t.Fatal(err)
		}
		got.Bitrate, got.Duration = 0, got.Duration.Round(time.Second)
		if want := mediafix.Want(f); !reflect.DeepEqual(got, want) {
			t.Fatalf("Probe of %s = %+v\nwant %+v", key, got, want)
		}
	}
	if sums[0] == sums[1] || sums[0] != sums[2] {
		t.Fatalf("the files' sums are %x, want two keys' files to differ and one key's to match", sums)
	}
}
