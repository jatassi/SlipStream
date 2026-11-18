package media_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/media/speech/speechtest"
)

func decoder(t *testing.T) media.Decoder {
	return media.Decoder{FFmpeg: bundled(t, "ffmpeg"), FFprobe: bundled(t, "ffprobe")}
}

func TestDecoderStreamsThePCMItIsGiven(t *testing.T) {
	runs := []speechtest.Run{{Start: 200 * time.Millisecond, End: 900 * time.Millisecond}}
	wav := filepath.Join(t.TempDir(), "speech.wav")
	if err := os.WriteFile(wav, speechtest.WAV(runs, 2*time.Second), 0o600); err != nil {
		t.Fatal(err)
	}
	a, err := decoder(t).Audio(t.Context(), wav, -1)
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(a)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	if a.Stream != 0 || a.Start != 0 || !bytes.Equal(got, speechtest.PCM(runs, 2*time.Second)) {
		t.Fatalf("stream %d from %v: %d bytes, want stream 0 from 0 and the %d bytes written", a.Stream, a.Start, len(got), len(speechtest.PCM(runs, 2*time.Second)))
	}
}

func TestDecoderPicksTheDefaultTrack(t *testing.T) {
	video := filepath.Join(t.TempDir(), "two.mkv")
	cmd := exec.CommandContext(t.Context(), bundled(t, "ffmpeg"), "-v", "error",
		"-f", "lavfi", "-i", "sine=frequency=440:sample_rate=48000:duration=1", "-f", "lavfi", "-i", "sine=frequency=880:sample_rate=48000:duration=1",
		"-map", "0", "-map", "1", "-c:a", "flac", "-disposition:a:0", "0", "-disposition:a:1", "default", video)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("encode: %v: %s", err, out)
	}
	for stream, want := range map[int]int{-1: 1, 0: 0, 1: 1} {
		a, err := decoder(t).Audio(t.Context(), video, stream)
		if err != nil {
			t.Fatal(err)
		}
		n, _ := io.Copy(io.Discard, a)
		if err := a.Close(); err != nil || a.Stream != want || n != 2*media.SampleRate {
			t.Errorf("stream %d decoded stream %d, %d bytes, %v; want stream %d and a second of PCM", stream, a.Stream, n, err, want)
		}
	}
	if _, err := decoder(t).Audio(t.Context(), video, 2); !errors.Is(err, media.ErrNoAudio) {
		t.Errorf("the third stream of two = %v, want ErrNoAudio", err)
	}
}

func TestDecoderCloseStopsFFmpeg(t *testing.T) {
	wav := filepath.Join(t.TempDir(), "long.wav")
	if err := os.WriteFile(wav, speechtest.WAV(nil, time.Minute), 0o600); err != nil {
		t.Fatal(err)
	}
	a, err := decoder(t).Audio(t.Context(), wav, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.Read(make([]byte, 1024)); err != nil {
		t.Fatal(err)
	}
	if err := a.Close(); err != nil {
		t.Fatalf("Close before the end = %v, want nil", err)
	}
}
