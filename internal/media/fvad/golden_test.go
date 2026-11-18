//go:build cgo

package fvad_test

import (
	"bytes"
	"encoding/binary"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/media/fvad"
	"github.com/jatassi/slipstream/internal/media/speech/speechtest"
)

var update = flag.Bool("update", false, "write the golden files")

const goldenLength = 30 * time.Second

func golden(t *testing.T, name string, got []byte) {
	t.Helper()
	p := filepath.Join("testdata", name)
	if *update {
		if err := os.WriteFile(p, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("%s differs from what this build makes (%d bytes, want %d); run with -update only if libfvad or the format changed on purpose", p, len(got), len(want))
	}
}

func TestGoldenRawPCM(t *testing.T) {
	pcm := speechtest.PCM(schedule(7, goldenLength), goldenLength)
	vad, err := fvad.New(fvad.Mode, speechtest.Rate)
	if err != nil {
		t.Fatal(err)
	}
	defer vad.Close()
	var lines strings.Builder
	frame := make([]int16, speechtest.Rate/100)
	for at := 0; at+2*len(frame) <= len(pcm); at += 2 * len(frame) {
		for i := range frame {
			frame[i] = int16(binary.LittleEndian.Uint16(pcm[at+2*i:])) //nolint:gosec // the sample's two's-complement bits
		}
		speech, err := vad.Speech(frame)
		if err != nil {
			t.Fatal(err)
		}
		lines.WriteByte("01"[btoi(speech)])
		if (at/(2*len(frame))+1)%100 == 0 {
			lines.WriteByte('\n')
		}
	}
	golden(t, "golden.bits", []byte(lines.String()))
}

func TestGoldenMap(t *testing.T) {
	m, err := mapper(t).Map(t.Context(), writeWAV(t, schedule(7, goldenLength), goldenLength), -1)
	if err != nil {
		t.Fatal(err)
	}
	golden(t, "golden.map", m.Encode())
}

func TestVADRefuses(t *testing.T) {
	if _, err := fvad.New(4, speechtest.Rate); err == nil {
		t.Error("mode 4 made a detector")
	}
	if _, err := fvad.New(fvad.Mode, 22050); err == nil {
		t.Error("22.05 kHz made a detector")
	}
	vad, err := fvad.New(fvad.Mode, speechtest.Rate)
	if err != nil {
		t.Fatal(err)
	}
	defer vad.Close()
	if _, err := vad.Speech(make([]int16, 100)); err == nil {
		t.Error("a 100-sample frame read")
	}
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}
