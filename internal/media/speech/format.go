package speech

import (
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/klauspost/compress/zstd"
)

// FormatVersion is the encoding's version, which the header carries.
const FormatVersion = 1

const (
	magic     = "SSMP"
	headerCap = 4 << 10
)

var ErrCorrupt = errors.New("read the speech map")

// Encode is "SSMP", the header's length as a uvarint, the header's fields
// in order (integers as varints, text and bytes length-prefixed), then the
// zstd of the bits' little-endian words.
func (m Map) Encode() []byte {
	h := m.Header
	var head []byte
	head = binary.AppendUvarint(head, uint64(h.Format)) //nolint:gosec // a FormatVersion
	head = appendBytes(head, []byte(h.VAD))
	head = binary.AppendVarint(head, int64(h.Mode))
	head = binary.AppendVarint(head, int64(h.Rate))
	head = binary.AppendVarint(head, int64(h.FrameMS))
	head = binary.AppendVarint(head, int64(h.Stream))
	head = appendBytes(head, []byte(h.Recipe))
	head = binary.AppendVarint(head, int64(h.Start))
	head = binary.AppendUvarint(head, uint64(h.Frames)) //nolint:gosec // New refuses negative frames
	head = appendBytes(head, h.Identity)
	raw := make([]byte, 8*len(m.bits))
	for i, w := range m.bits {
		binary.LittleEndian.PutUint64(raw[8*i:], w)
	}
	out := binary.AppendUvarint([]byte(magic), uint64(len(head)))
	out = append(out, head...)
	e, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedDefault), zstd.WithEncoderConcurrency(1), zstd.WithSingleSegment(true))
	if err != nil {
		panic(fmt.Sprintf("speech: zstd encoder: %v", err))
	}
	defer func() { _ = e.Close() }()
	return e.EncodeAll(raw, out)
}

func appendBytes(b, v []byte) []byte { return append(binary.AppendUvarint(b, uint64(len(v))), v...) }

// Decode reads an encoded map. It refuses a bad magic, a header longer
// than 4 KiB, more than MaxFrames frames, bits of another size than the
// frames need, and speech past the last frame.
func Decode(b []byte) (Map, error) {
	rest, ok := cutPrefix(b, magic)
	if !ok {
		return Map{}, fmt.Errorf("%w: not a speech map", ErrCorrupt)
	}
	n, size := binary.Uvarint(rest)
	if size <= 0 || n > headerCap || n > uint64(len(rest)-size) { //nolint:gosec // size is at most len(rest)
		return Map{}, fmt.Errorf("%w: a header of %d bytes", ErrCorrupt, n)
	}
	r := reader{b: rest[size : size+int(n)]}
	h := Header{
		Format:  int(r.uvarint()), //nolint:gosec // a version to compare, never a size
		VAD:     string(r.bytes()),
		Mode:    r.int(),
		Rate:    r.int(),
		FrameMS: r.int(),
		Stream:  r.int(),
		Recipe:  string(r.bytes()),
		Start:   time.Duration(r.varint()),
	}
	frames := r.uvarint()
	h.Identity = r.bytes()
	switch {
	case r.err != nil:
		return Map{}, fmt.Errorf("%w: %w", ErrCorrupt, r.err)
	case len(r.b) != 0:
		return Map{}, fmt.Errorf("%w: %d bytes after the header", ErrCorrupt, len(r.b))
	case frames > MaxFrames:
		return Map{}, fmt.Errorf("%w: %d frames is more than %d", ErrCorrupt, frames, MaxFrames)
	}
	h.Frames = int(frames)
	want := 8 * wordsOf(h.Frames)
	d, err := zstd.NewReader(nil, zstd.WithDecoderConcurrency(1), zstd.WithDecoderMaxMemory(uint64(want)+1<<10)) //nolint:gosec // want is at most 8*wordsOf(MaxFrames)
	if err != nil {
		return Map{}, fmt.Errorf("%w: %w", ErrCorrupt, err)
	}
	defer d.Close()
	raw, err := d.DecodeAll(rest[size+int(n):], make([]byte, 0, want))
	switch {
	case err != nil:
		return Map{}, fmt.Errorf("%w: %w", ErrCorrupt, err)
	case len(raw) != want:
		return Map{}, fmt.Errorf("%w: %d bytes of bits for %d frames", ErrCorrupt, len(raw), h.Frames)
	}
	words := make([]uint64, want/8)
	for i := range words {
		words[i] = binary.LittleEndian.Uint64(raw[8*i:])
	}
	m, err := New(h, words)
	if err != nil {
		return Map{}, fmt.Errorf("%w: %w", ErrCorrupt, err)
	}
	return m, nil
}

func cutPrefix(b []byte, p string) ([]byte, bool) {
	if len(b) < len(p) || string(b[:len(p)]) != p {
		return nil, false
	}
	return b[len(p):], true
}

var errShort = errors.New("the header ends early")

type reader struct {
	b   []byte
	err error
}

func (r *reader) uvarint() uint64 {
	if r.err != nil {
		return 0
	}
	v, n := binary.Uvarint(r.b)
	if n <= 0 {
		r.err = errShort
		return 0
	}
	r.b = r.b[n:]
	return v
}

func (r *reader) varint() int64 {
	if r.err != nil {
		return 0
	}
	v, n := binary.Varint(r.b)
	if n <= 0 {
		r.err = errShort
		return 0
	}
	r.b = r.b[n:]
	return v
}

func (r *reader) int() int { return int(r.varint()) }

func (r *reader) bytes() []byte {
	n := r.uvarint()
	if r.err != nil {
		return nil
	}
	if n > uint64(len(r.b)) {
		r.err = errShort
		return nil
	}
	v := r.b[:n:n]
	r.b = r.b[n:]
	return v
}
