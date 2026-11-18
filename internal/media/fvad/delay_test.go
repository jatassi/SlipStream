//go:build cgo

package fvad_test

import (
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/media/speech"
)

func TestDelayedAudio(t *testing.T) {
	const (
		length = 30 * time.Second
		delay  = 1500 * time.Millisecond
	)
	runs := schedule(3, length)
	plain, err := mapper(t).Map(t.Context(), writeWAV(t, runs, length), -1)
	if err != nil {
		t.Fatal(err)
	}
	late, err := mapper(t).Map(t.Context(), delayed(t, runs, length, delay), -1)
	if err != nil {
		t.Fatal(err)
	}
	if late.Header.Start != delay || plain.Header.Start != 0 {
		t.Fatalf("Start = %v and %v, want %v and 0", late.Header.Start, plain.Header.Start, delay)
	}
	shift := int(delay / speech.Frame)
	for i := range late.Header.Frames - shift {
		if late.Speech(i+shift) == plain.Speech(i) || plain.Speech(i-1) != plain.Speech(i) || plain.Speech(i+1) != plain.Speech(i) {
			continue
		}
		t.Errorf("frame %d reads speech %v late and %v on time, more than a frame from an edge", i, late.Speech(i+shift), plain.Speech(i))
	}
	for i := range shift {
		if late.Speech(i) {
			t.Fatalf("frame %d, before the audio starts, reads speech", i)
		}
	}
}
