//go:build cgo

package fvad_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/media/fvad"
	"github.com/jatassi/slipstream/internal/media/speech"
)

func TestNoAudio(t *testing.T) {
	video := filepath.Join(t.TempDir(), "silent.mkv")
	cmd := exec.CommandContext(t.Context(), bundled(t, "ffmpeg"), "-v", "error", "-f", "lavfi", "-i", "color=c=black:s=64x64:r=24:d=1",
		"-c:v", "libx264", "-preset", "ultrafast", video)
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("encode: %v: %s", err, b)
	}
	for _, stream := range []int{-1, 0} {
		if _, err := mapper(t).Map(t.Context(), video, stream); !errors.Is(err, speech.ErrNoAudio) {
			t.Errorf("stream %d of a video-only file = %v, want ErrNoAudio", stream, err)
		}
	}
	wav := writeWAV(t, nil, 1)
	if _, err := mapper(t).Map(t.Context(), wav, 1); !errors.Is(err, speech.ErrNoAudio) {
		t.Errorf("the second stream of a one-stream file = %v, want ErrNoAudio", err)
	}
}

func TestDecodeFails(t *testing.T) {
	whole, err := os.ReadFile(writeWAV(t, schedule(1, goldenLength), goldenLength))
	if err != nil {
		t.Fatal(err)
	}
	cut := filepath.Join(t.TempDir(), "cut.wav")
	if err := os.WriteFile(cut, whole[:20], 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = mapper(t).Map(t.Context(), cut, -1)
	if d, ok := errors.AsType[*speech.DecodeError](err); !ok || d.First == "" {
		t.Fatalf("a truncated file = %v, want a DecodeError with the tool's first line", err)
	}
}

func TestAudioThatDecodesToNothingFails(t *testing.T) {
	video := filepath.Join(t.TempDir(), "no-packets.mkv")
	cmd := exec.CommandContext(t.Context(), bundled(t, "ffmpeg"), "-v", "error", "-f", "lavfi", "-i", "color=c=black:s=64x64:r=24:d=10",
		"-f", "lavfi", "-i", "sine=frequency=440:sample_rate=48000:duration=10",
		"-c:v", "libx264", "-preset", "ultrafast", "-c:a", "aac", "-bsf:a", "noise=drop=1", video)
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("encode: %v: %s", err, b)
	}
	m, err := mapper(t).Map(t.Context(), video, -1)
	if d, ok := errors.AsType[*speech.DecodeError](err); !ok || d.First == "" {
		t.Fatalf("an audio stream that decodes to nothing = %d frames, %v; want a DecodeError", m.Header.Frames, err)
	}
}

func TestNoFFmpeg(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "ffmpeg")
	wav := writeWAV(t, schedule(1, goldenLength), goldenLength)
	for _, d := range []media.Decoder{{FFmpeg: missing, FFprobe: bundled(t, "ffprobe")}, {FFmpeg: bundled(t, "ffmpeg"), FFprobe: missing}} {
		if _, err := (fvad.Mapper{Decoder: d}).Map(t.Context(), wav, -1); !errors.Is(err, speech.ErrNodeFault) {
			t.Errorf("Map with %+v = %v, want ErrNodeFault", d, err)
		}
	}
}
