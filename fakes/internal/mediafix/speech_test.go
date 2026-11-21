package mediafix_test

import (
	"bytes"
	"io"
	"os"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/internal/mediafix"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/media/speech/speechtest"
)

func TestSpeechHoldsItsPCMFromItsDelay(t *testing.T) {
	b, tools := builder(t)
	r := mediafix.SpeechRecipe{
		Runs:   []speechtest.Run{{Start: 300 * time.Millisecond, End: 1200 * time.Millisecond}, {Start: 2 * time.Second, End: 2500 * time.Millisecond}},
		Length: 3 * time.Second, Delay: 750 * time.Millisecond,
	}
	built, err := b.Speech(t.Context(), r)
	if err != nil {
		t.Fatal(err)
	}
	rec, err := media.FFprobe{Path: tools.FFprobe}.Probe(t.Context(), built)
	if err != nil {
		t.Fatal(err)
	}
	if len(rec.Audio) != 1 || rec.Audio[0].Codec != "pcm_s16le" || rec.Audio[0].Channels != 1 {
		t.Fatalf("audio = %+v, want one mono pcm_s16le track", rec.Audio)
	}
	a, err := media.Decoder{FFmpeg: tools.FFmpeg, FFprobe: tools.FFprobe}.Audio(t.Context(), built, -1)
	if err != nil {
		t.Fatal(err)
	}
	pcm, err := io.ReadAll(a)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	pad := int(r.Delay * speechtest.Rate / time.Second * 2)
	if a.Start != r.Delay || len(pcm) != pad+len(speechtest.PCM(r.Runs, r.Length)) || !bytes.Equal(pcm[pad:], speechtest.PCM(r.Runs, r.Length)) || bytes.ContainsFunc(pcm[:pad], func(c rune) bool { return c != 0 }) {
		t.Fatalf("decoded %d bytes from %v, want %d of silence then the %d bytes written, from %v", len(pcm), a.Start, pad, len(speechtest.PCM(r.Runs, r.Length)), r.Delay)
	}

	first, err := os.ReadFile(built)
	if err != nil {
		t.Fatal(err)
	}
	again, _ := builder(t)
	rebuilt, err := again.Speech(t.Context(), r)
	if err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(rebuilt)
	if err != nil || !bytes.Equal(first, second) {
		t.Fatalf("a second build differs: %v", err)
	}
}
