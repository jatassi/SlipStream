package clip_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/converting/clip"
	"github.com/jatassi/slipstream/internal/encode"
	"github.com/jatassi/slipstream/internal/media"
)

func tools(t *testing.T) clip.Tools {
	t.Helper()
	var out clip.Tools
	for _, p := range filepath.SplitList(os.Getenv("SLIPSTREAM_NATIVE_TOOLS")) {
		switch name := filepath.Base(p); name {
		case "ffmpeg", "ffmpeg.exe":
			out.FFmpeg = p
		case "ffprobe", "ffprobe.exe":
			out.FFprobe = p
		case "dovi_tool", "dovi_tool.exe":
			out.DoviTool = p
		}
	}
	if out.FFmpeg == "" || out.FFprobe == "" || out.DoviTool == "" {
		t.Fatal("SLIPSTREAM_NATIVE_TOOLS names no ffmpeg, ffprobe or dovi_tool: run the tests with go run ./cmd/release test")
	}
	out.Environ = os.Environ()
	return out
}

// TestEveryClassMakesAClipThatProbesAsItsClass makes the HDR10 class
// without HDR10+, so an HDR10+ source proves on every OS, Windows
// included, whose bundled x265 can't write HDR10+.
func TestEveryClassMakesAClipThatProbesAsItsClass(t *testing.T) {
	tl := tools(t)
	dir := t.TempDir()
	for _, c := range []encode.Class{
		encode.ClassH264x8, encode.ClassHEVCx8, encode.ClassHEVCx10, encode.ClassAV1x8, encode.ClassAV1x10,
		encode.ClassHDR10, encode.ClassHLG, encode.ClassDV5, encode.ClassDV81, encode.ClassDecodeSDR, encode.ClassDecodeHDR10,
	} {
		t.Run(string(c), func(t *testing.T) {
			made, err := clip.Make(t.Context(), tl, c, clip.Trial, dir)
			if err != nil {
				t.Fatal(err)
			}
			got, err := media.FFprobe{Path: tl.FFprobe}.Probe(t.Context(), made)
			if err != nil {
				t.Fatal(err)
			}
			if class := encode.ClassOf(got); class != c {
				t.Errorf("the clip probes as %s: %+v", class, got)
			}
			if got.Width != 320 || got.Height != 180 || got.Duration.Round(time.Second) != 2*time.Second {
				t.Errorf("the clip is %dx%d for %s, want 320x180 for 2s", got.Width, got.Height, got.Duration)
			}
		})
	}
}
