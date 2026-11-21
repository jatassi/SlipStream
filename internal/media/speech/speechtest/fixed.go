// Package speechtest makes speech at known times: the Speech Map a run
// schedule should read as, the PCM bursts a voice detector hears as that
// speech, and a Mapper that answers the schedule's map without decoding.
package speechtest

import (
	"context"
	"encoding/binary"
	"time"

	"github.com/jatassi/slipstream/internal/media/speech"
)

// Rate is the sample rate PCM and WAV write.
const Rate = 16000

// frameSamples is one speech.Frame of samples at Rate.
const frameSamples = Rate / 100

// Run is a span of speech, from Start up to End.
type Run struct{ Start, End time.Duration }

// Fixed is a Mapper that answers the map of Runs over Length for every
// file, as the bits the runs cover, without decoding.
type Fixed struct {
	Runs   []Run
	Length time.Duration
}

func (f Fixed) Map(_ context.Context, _ string, stream int) (speech.Map, error) {
	return Map(f.Runs, f.Length, max(stream, 0))
}

// Map is the map whose frames are speech where a frame's start lies in a
// run.
func Map(runs []Run, length time.Duration, stream int) (speech.Map, error) {
	var b speech.Builder
	frames := int(length / speech.Frame)
	for i := range frames {
		b.Add(in(runs, time.Duration(i)*speech.Frame))
	}
	return b.Map(speech.Header{
		Format: speech.FormatVersion, VAD: "speechtest", Rate: Rate, FrameMS: int(speech.Frame.Milliseconds()), Stream: stream, Recipe: "speechtest",
	})
}

func in(runs []Run, t time.Duration) bool {
	for _, r := range runs {
		if t >= r.Start && t < r.End {
			return true
		}
	}
	return false
}

// PCM is length of mono 16-bit little-endian samples at Rate: silence, and
// in each run WebRTC's speech test frame, sample k of every 10 ms frame
// being k*k, which libfvad hears as speech in every mode.
func PCM(runs []Run, length time.Duration) []byte {
	n := int(length * Rate / time.Second)
	out := make([]byte, 2*n)
	for _, r := range runs {
		from, to := max(0, int(r.Start*Rate/time.Second)), min(n, int(r.End*Rate/time.Second))
		for i := from; i < to; i++ {
			k := i % frameSamples
			binary.LittleEndian.PutUint16(out[2*i:], uint16(int16(k*k))) //nolint:gosec // k*k is below 160², inside int16
		}
	}
	return out
}

// WAV is PCM in a WAV file.
func WAV(runs []Run, length time.Duration) []byte {
	pcm := PCM(runs, length)
	h := make([]byte, 0, 44+len(pcm))
	h = append(h, "RIFF"...)
	h = binary.LittleEndian.AppendUint32(h, uint32(36+len(pcm))) //nolint:gosec // a test WAV is far under 4 GiB
	h = append(h, "WAVEfmt "...)
	h = binary.LittleEndian.AppendUint32(h, 16)
	h = binary.LittleEndian.AppendUint16(h, 1)
	h = binary.LittleEndian.AppendUint16(h, 1)
	h = binary.LittleEndian.AppendUint32(h, Rate)
	h = binary.LittleEndian.AppendUint32(h, 2*Rate)
	h = binary.LittleEndian.AppendUint16(h, 2)
	h = binary.LittleEndian.AppendUint16(h, 16)
	h = append(h, "data"...)
	h = binary.LittleEndian.AppendUint32(h, uint32(len(pcm))) //nolint:gosec // a test WAV is far under 4 GiB
	return append(h, pcm...)
}
