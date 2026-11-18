//go:build cgo

package mediafix_test

import (
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/internal/mediafix"
	fakespeech "github.com/jatassi/slipstream/fakes/internal/speech"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/media/fvad"
	"github.com/jatassi/slipstream/internal/media/speech"
	"github.com/jatassi/slipstream/internal/media/speech/speechtest"
)

// hangover is how long libfvad mode 3 holds speech after a burst ends.
const hangover = 90 * time.Millisecond

func TestAScenarioSpeechTrackMapsAsItsSchedule(t *testing.T) {
	b, tools := builder(t)
	sp := scenario.MediaSpeech{Seed: 7, Density: 0.4, Length: 45 * time.Second}
	f := scenario.MediaFile{Path: "talk.mkv", Video: scenario.MediaVideo{Width: 64, Height: 64, Seconds: 1, HDR: "sdr"}, Audio: []scenario.MediaAudio{
		{Codec: "aac", Channels: 2, Language: "fra"},
		{Codec: "pcm_s16le", Channels: 1, Language: "eng", Speech: &sp},
	}}
	built, err := b.Build(t.Context(), f)
	if err != nil {
		t.Fatal(err)
	}
	rec, err := media.FFprobe{Path: tools.FFprobe}.Probe(t.Context(), built)
	if err != nil {
		t.Fatal(err)
	}
	if want := mediafix.Want(f); len(rec.Audio) != 2 || rec.Audio[1] != want.Audio[1] {
		t.Fatalf("audio = %+v, want %+v", rec.Audio, want.Audio)
	}
	got, err := fvad.Mapper{Decoder: media.Decoder{FFmpeg: tools.FFmpeg, FFprobe: tools.FFprobe}}.Map(t.Context(), built, 1)
	if err != nil {
		t.Fatal(err)
	}
	truth, err := fakespeech.Map(sp, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got.Header.Frames != truth.Header.Frames {
		t.Fatalf("the map has %d frames, the schedule's %d", got.Header.Frames, truth.Header.Frames)
	}
	runs := fakespeech.Runs(sp)
	var burst, missed int
	for i := range truth.Header.Frames {
		at := time.Duration(i) * speech.Frame
		switch {
		case truth.Speech(i):
			burst++
			if !got.Speech(i) {
				missed++
			}
		case got.Speech(i) && !slices.ContainsFunc(runs, func(r speechtest.Run) bool { return at > r.Start-speech.Frame && at < r.End+speech.Frame+hangover }):
			t.Errorf("frame %d at %v reads speech outside every burst, the part frames at its edges and its hangover", i, at)
		}
	}
	recall := 1 - float64(missed)/float64(burst)
	t.Logf("the map hears %.4f of %d speech frames", recall, burst)
	if recall < 0.95 {
		t.Errorf("the map hears %.4f of the schedule's %d speech frames, want at least 0.95", recall, burst)
	}
}
