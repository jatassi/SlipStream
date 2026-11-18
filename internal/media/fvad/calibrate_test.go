//go:build cgo

package fvad_test

import (
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/media/speech"
	"github.com/jatassi/slipstream/internal/media/speech/speechtest"
)

// TestCalibrateBursts is the bar every map test stands on: WebRTC's speech
// test frame, through the bundled ffmpeg and libfvad mode 3, reads as
// speech inside each burst, and silence reads as silence once the
// hangover passes.
func TestCalibrateBursts(t *testing.T) {
	const length = 120 * time.Second
	runs := schedule(1, length)
	m, err := mapper(t).Map(t.Context(), writeWAV(t, runs, length), -1)
	if err != nil {
		t.Fatal(err)
	}
	if want := int(length / speech.Frame); m.Header.Frames != want {
		t.Fatalf("the map has %d frames, want %d", m.Header.Frames, want)
	}
	truth, err := speechtest.Map(runs, length, 0)
	if err != nil {
		t.Fatal(err)
	}
	var burst, heard, quiet, falseSpeech, speechRead int
	for i := range m.Header.Frames {
		at := time.Duration(i) * speech.Frame
		if m.Speech(i) {
			speechRead++
		}
		switch {
		case truth.Speech(i):
			burst++
			if m.Speech(i) {
				heard++
			}
		case !after(runs, at, hangover):
			quiet++
			if m.Speech(i) {
				falseSpeech++
				t.Errorf("frame %d at %v reads speech, %v past every burst (%v)", i, at, hangover, since(runs, at))
			}
		}
	}
	recall := float64(heard) / float64(burst)
	precision := float64(heard) / float64(speechRead)
	t.Logf("%d bursts, density %.3f: recall %.4f (%d of %d burst frames), precision %.4f (%d of %d speech frames), %d of %d quiet frames read speech",
		len(runs), truth.Density(), recall, heard, burst, precision, heard, speechRead, falseSpeech, quiet)
	if recall < 0.95 {
		t.Errorf("recall %.4f, want at least 0.95", recall)
	}
}

// after is whether t lies at most d after the end of a run.
func after(runs []speechtest.Run, t, d time.Duration) bool {
	for _, r := range runs {
		if t >= r.End && t <= r.End+d {
			return true
		}
	}
	return false
}

func since(runs []speechtest.Run, t time.Duration) time.Duration {
	best := time.Duration(1 << 62)
	for _, r := range runs {
		if t >= r.End && t-r.End < best {
			best = t - r.End
		}
	}
	return best
}
