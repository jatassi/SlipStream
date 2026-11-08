package media_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/release"
)

// bundled is the bundled tool that go run ./cmd/release test names.
func bundled(t *testing.T, tool string) string {
	t.Helper()
	for _, p := range filepath.SplitList(os.Getenv("SLIPSTREAM_NATIVE_TOOLS")) {
		if name := filepath.Base(p); name == tool || name == tool+".exe" {
			return p
		}
	}
	t.Fatalf("SLIPSTREAM_NATIVE_TOOLS names no %s: run the tests with go run ./cmd/release test", tool)
	return ""
}

func TestFFprobeReadsAFileWithTheBundledBinary(t *testing.T) {
	dir := t.TempDir()
	video := filepath.Join(dir, "Heat (1995).mkv")
	encode := exec.CommandContext(t.Context(), bundled(t, "ffmpeg"), "-v", "error", "-f", "lavfi", "-i", "testsrc2=size=320x180:rate=24", "-frames:v", "1",
		"-vf", "setparams=color_primaries=bt2020:color_trc=arib-std-b67:colorspace=bt2020nc", "-c:v", "libx264", "-preset", "ultrafast", video)
	if out, err := encode.CombinedOutput(); err != nil {
		t.Fatalf("encode: %v: %s", err, out)
	}
	notes := filepath.Join(dir, "notes.mkv")
	if err := os.WriteFile(notes, []byte("not a video"), 0o600); err != nil {
		t.Fatal(err)
	}
	ffprobe := media.FFprobe{Path: bundled(t, "ffprobe")}

	got, err := ffprobe.Probe(t.Context(), video)
	if err != nil || got.Version != media.Version || got.Codec != "h264" || got.Width != 320 || !equal(got.HDR, []release.HDR{"HLG"}) {
		t.Fatalf("Probe = %+v, %v; want a version %d h264 320x180 HLG record", got, err, media.Version)
	}
	_, err = ffprobe.Probe(t.Context(), notes)
	if unreadable, ok := errors.AsType[*media.UnreadableError](err); !ok || unreadable.Line == "" {
		t.Fatalf("Probe of a text file = %v, want ffprobe's first error line", err)
	}
	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := ffprobe.Probe(canceled, video); !errors.Is(err, context.Canceled) {
		t.Fatalf("Probe after Cancel = %v, want context.Canceled", err)
	}
}
