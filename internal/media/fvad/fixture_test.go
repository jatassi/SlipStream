package fvad_test

import (
	"math"
	"math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/media/fvad"
	"github.com/jatassi/slipstream/internal/media/speech/speechtest"
)

// hangover is how long libfvad mode 3 holds speech after a burst ends.
const hangover = 90 * time.Millisecond

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

func mapper(t *testing.T) fvad.Mapper {
	return fvad.Mapper{Decoder: media.Decoder{FFmpeg: bundled(t, "ffmpeg"), FFprobe: bundled(t, "ffprobe")}}
}

// density is how much of a schedule is speech, about a film's share.
const density = 0.4

// schedule is seeded speech over length at about density: runs of 0.3 to
// 4 s, log-normal, apart by exponential gaps, on 10 ms edges.
func schedule(seed uint64, length time.Duration) []speechtest.Run {
	r := rand.New(rand.NewPCG(seed, seed)) //nolint:gosec // a seeded draw the test replays
	const meanRun = 1.4
	gap := meanRun*(1-density)/density - 0.3
	var runs []speechtest.Run
	at := time.Duration(r.ExpFloat64() * gap * float64(time.Second))
	for {
		secs := min(4, max(0.3, math.Exp(r.NormFloat64()*0.6+math.Log(meanRun)-0.18)))
		start := at.Round(10 * time.Millisecond)
		end := (at + time.Duration(secs*float64(time.Second))).Round(10 * time.Millisecond)
		if end > length {
			return runs
		}
		runs = append(runs, speechtest.Run{Start: start, End: end})
		at = end + time.Duration((0.3+r.ExpFloat64()*gap)*float64(time.Second))
	}
}

func writeWAV(t *testing.T, runs []speechtest.Run, length time.Duration) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "speech.wav")
	if err := os.WriteFile(p, speechtest.WAV(runs, length), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// delayed is the WAV of runs muxed with a black video, its audio starting
// delay after the video.
func delayed(t *testing.T, runs []speechtest.Run, length, delay time.Duration) string {
	t.Helper()
	wav := writeWAV(t, runs, length)
	out := filepath.Join(t.TempDir(), "delayed.mkv")
	cmd := exec.CommandContext(t.Context(), bundled(t, "ffmpeg"), "-v", "error", "-y",
		"-f", "lavfi", "-i", "color=c=black:s=64x64:r=24:d="+secs(length+delay),
		"-itsoffset", secs(delay), "-i", wav,
		"-map", "0:v", "-map", "1:a", "-c:v", "libx264", "-preset", "ultrafast", "-threads", "1", "-c:a", "copy",
		"-fflags", "+bitexact", "-flags:v", "+bitexact", "-map_metadata", "-1", out)
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("mux the delayed fixture: %v: %s", err, b)
	}
	return out
}

func secs(d time.Duration) string { return strconv.FormatFloat(d.Seconds(), 'f', -1, 64) }
