package media_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/media/mediatest"
	"github.com/jatassi/slipstream/internal/release"
)

type payloads map[bencode.InfoHash]string

func (p payloads) Path(h bencode.InfoHash, index int) (string, error) {
	path, ok := p[h]
	if !ok || index != 0 {
		return "", errors.New("the torrent is not in this Node's client")
	}
	return path, nil
}

func TestProberAnswersTheLongestMatchingKey(t *testing.T) {
	near, far := media.Record{Container: "near"}, media.Record{Container: "far"}
	p := mediatest.Prober{Of: map[string]media.Record{"Heat.mkv": far, "Heat (1995)/Heat.mkv": near}}
	for path, want := range map[string]string{
		"/media/movies/Heat (1995)/Heat.mkv": "near",
		"/media/movies/Other/Heat.mkv":       "far",
		"/media/movies/Other/MyHeat.mkv":     "",
	} {
		got, err := p.Probe(context.Background(), path)
		if got.Container != want || (want == "") != (err != nil) {
			t.Errorf("Probe(%q) = %q, %v; want %q", path, got.Container, err, want)
		}
	}
}

func TestOldRecordsReadAsVersionZero(t *testing.T) {
	for _, old := range []string{
		`{"duration":6000000000,"container":"matroska,webm","codec":"h264","width":1920,"height":1080}`,
		`{"width":1920,"height":1080,"codec":"h264"}`,
	} {
		var r media.Record
		if err := json.Unmarshal([]byte(old), &r); err != nil {
			t.Fatal(err)
		}
		if r.Version != 0 || r.Codec != "h264" || r.Width != 1920 || r.Height != 1080 {
			t.Errorf("%s reads as %+v, want version 0 with h264 1920x1080", old, r)
		}
		p := r.Probed()
		res, _ := p.Resolution.Get()
		video, _ := p.Video.Get()
		_, hdr := p.HDR.Get()
		_, audio := p.Audio.Get()
		_, channels := p.Channels.Get()
		_, languages := p.Languages.Get()
		if res != release.R1080p || video != "x264" || hdr || audio || channels || languages {
			t.Errorf("%s is probed as %+v, want only 1080p x264 known", old, p)
		}
	}
}

func TestRecordKeepsTheOldKeysAtTheTop(t *testing.T) {
	b, err := json.Marshal(media.Record{Version: media.Version, Container: "mp4", Duration: time.Second, Video: media.Video{Codec: "hevc", Width: 1280, Height: 720}})
	if err != nil {
		t.Fatal(err)
	}
	var top map[string]any
	if err := json.Unmarshal(b, &top); err != nil {
		t.Fatal(err)
	}
	if top["codec"] != "hevc" || top["width"] != 1280.0 || top["height"] != 720.0 || top["container"] != "mp4" || top["duration"] != 1e9 || top["version"] != 1.0 {
		t.Fatalf("record JSON = %s, want codec, width, height, container, duration and version at the top", b)
	}
}

func TestProbedInTheReleaseVocabularies(t *testing.T) {
	r := media.Record{
		Version: media.Version,
		Video:   media.Video{Codec: "hevc", Width: 3840, Height: 1600, HDR: []release.HDR{"HDR10+", "HDR10"}},
		Audio: []media.Audio{
			{Codec: "dts", Profile: "DTS-HD MA + DTS:X", Channels: 8, Language: "en"},
			{Codec: "dts", Profile: "DTS", Channels: 6, Language: "en"},
			{Codec: "pcm_s24le", Channels: 2, Language: "fr"},
			{Codec: "vorbis", Channels: 3, Language: "fr"},
		},
	}
	p := r.Probed()
	if res, _ := p.Resolution.Get(); res != release.R2160p {
		t.Errorf("resolution = %v, want 2160p", res)
	}
	if v, _ := p.Video.Get(); v != "x265" {
		t.Errorf("video = %v, want x265", v)
	}
	for name, got := range map[string][2]any{
		"hdr":       {fact(p.HDR), []release.HDR{"HDR10+", "HDR10"}},
		"audio":     {fact(p.Audio), []release.AudioCodec{"DTS-HD MA", "DTS", "PCM"}},
		"channels":  {fact(p.Channels), []release.Channels{"7.1", "5.1", "2.0"}},
		"languages": {fact(p.Languages), []release.Language{"en", "fr"}},
	} {
		if !equal(got[0], got[1]) {
			t.Errorf("%s = %v, want %v", name, got[0], got[1])
		}
	}

	r.Audio = append(r.Audio, media.Audio{Codec: "aac", Channels: 2})
	if _, known := r.Probed().Languages.Get(); known {
		t.Error("languages are known with an untagged track, want unknown")
	}
	r.Audio = nil
	if got, known := r.Probed().Audio.Get(); !known || len(got) != 0 {
		t.Errorf("audio of a file without audio = %v, %v; want known and empty", got, known)
	}
}

func fact[T any](f release.Fact[T]) any {
	v, known := f.Get()
	if !known {
		return nil
	}
	return v
}
