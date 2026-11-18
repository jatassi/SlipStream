package speech_test

import (
	"encoding/binary"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/klauspost/compress/zstd"

	"github.com/jatassi/slipstream/internal/media/speech"
	"github.com/jatassi/slipstream/internal/media/speech/speechtest"
)

func sample(t *testing.T) speech.Map {
	t.Helper()
	m, err := speechtest.Map([]speechtest.Run{{Start: time.Second, End: 2500 * time.Millisecond}, {Start: 4 * time.Second, End: 4010 * time.Millisecond}}, 6*time.Second+30*time.Millisecond, 1)
	if err != nil {
		t.Fatal(err)
	}
	m.Header.Start = 1500 * time.Millisecond
	m.Header.Identity = []byte{1, 2, 3}
	return m
}

func TestEncodeDecode(t *testing.T) {
	m := sample(t)
	got, err := speech.Decode(m.Encode())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Header, m.Header) || !reflect.DeepEqual(got.Bits(), m.Bits()) {
		t.Fatalf("Decode(Encode) = %+v, want %+v", got, m)
	}
	if got.Header.Frames != 603 || !got.Speech(100) || got.Speech(250) || !got.Speech(400) || got.Speech(401) || got.Speech(603) {
		t.Fatalf("the round trip reads other speech: %d frames", got.Header.Frames)
	}
	if d := got.Density(); d != 151.0/603 {
		t.Errorf("Density = %v, want 151/603", d)
	}
	if got.Duration() != 6030*time.Millisecond {
		t.Errorf("Duration = %v", got.Duration())
	}
}

func TestNewRefuses(t *testing.T) {
	if _, err := speech.New(speech.Header{Frames: 65}, []uint64{0}); !errors.Is(err, speech.ErrFrames) {
		t.Errorf("one word for 65 frames = %v, want ErrFrames", err)
	}
	if _, err := speech.New(speech.Header{Frames: 3}, []uint64{1 << 3}); !errors.Is(err, speech.ErrFrames) {
		t.Errorf("speech past the last frame = %v, want ErrFrames", err)
	}
}

func TestDecodeRefuses(t *testing.T) {
	good := sample(t).Encode()
	long := append(binary.AppendUvarint([]byte("SSMP"), 5000), make([]byte, 5000)...)
	pastEnd := make([]byte, 8*10)
	pastEnd[len(pastEnd)-1] = 0x80
	cases := map[string][]byte{
		"no magic":         append([]byte("XXXX"), good[4:]...),
		"short":            good[:3],
		"a 5000-byte head": long,
		"cut bits":         good[:len(good)-4],
		"extra bits":       encoded(603, make([]byte, 8*11)),
		"missing bits":     encoded(603, make([]byte, 8*9)),
		"speech past end":  encoded(603, pastEnd),
		"too many frames":  encoded(speech.MaxFrames+1, nil),
		"not zstd":         append(encoded(64, nil)[:len(encoded(64, nil))-9], "not zstd!"...),
	}
	for name, b := range cases {
		if _, err := speech.Decode(b); !errors.Is(err, speech.ErrCorrupt) {
			t.Errorf("%s: Decode = %v, want ErrCorrupt", name, err)
		}
	}
	if _, err := speech.Decode(encoded(603, make([]byte, 8*10))); err != nil {
		t.Errorf("a well-formed silent map = %v", err)
	}
}

// encoded is a map's encoding of frames with zero header fields and raw
// as its bits.
func encoded(frames int, raw []byte) []byte {
	var head []byte
	head = binary.AppendUvarint(head, 0)
	head = binary.AppendUvarint(head, 0)
	for range 4 {
		head = binary.AppendVarint(head, 0)
	}
	head = binary.AppendUvarint(head, 0)
	head = binary.AppendVarint(head, 0)
	head = binary.AppendUvarint(head, uint64(frames)) //nolint:gosec // the cases' frames are non-negative
	head = binary.AppendUvarint(head, 0)
	out := append(binary.AppendUvarint([]byte("SSMP"), uint64(len(head))), head...)
	e, err := zstd.NewWriter(nil, zstd.WithSingleSegment(true))
	if err != nil {
		panic(err)
	}
	return e.EncodeAll(raw, out)
}

func FuzzDecode(f *testing.F) {
	f.Add(encoded(603, make([]byte, 8*10)))
	f.Add(encoded(1, []byte{1, 0, 0, 0, 0, 0, 0, 0}))
	f.Add([]byte("SSMP\x00"))
	f.Fuzz(func(t *testing.T, b []byte) {
		m, err := speech.Decode(b)
		if err != nil {
			return
		}
		again, err := speech.Decode(m.Encode())
		if err != nil || !reflect.DeepEqual(again.Bits(), m.Bits()) || again.Header.Frames != m.Header.Frames {
			t.Fatalf("a decoded map doesn't survive a round trip: %v", err)
		}
	})
}
