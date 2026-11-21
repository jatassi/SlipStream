// Package speech is the Speech Map: one bit per 10 ms frame of a file's
// audio, set where a voice detector heard speech, and its encoding. It
// does no IO: a Mapper builds maps, and the Node keeps their bytes.
package speech

import (
	"context"
	"errors"
	"fmt"
	"math/bits"
	"time"
)

// Frame is the time one bit covers.
const Frame = 10 * time.Millisecond

// MaxFrames is the most frames a map holds: 12 hours.
const MaxFrames = 12 * 3600 * 100

// Mapper builds a file's Speech Map from one audio stream, an ordinal or -1
// for the default track, else the first.
type Mapper interface {
	Map(ctx context.Context, path string, stream int) (Map, error)
}

var (
	ErrNoAudio = errors.New("the file has no audio stream")
	// ErrNodeFault is a Node that can't build maps at all, such as one
	// whose ffmpeg doesn't start.
	ErrNodeFault = errors.New("this Node can't scan audio")
	ErrFrames    = errors.New("the bits don't cover the frames")
)

// DecodeError is audio the decoder couldn't read. First is the first line
// of its error output.
type DecodeError struct{ First string }

func (e *DecodeError) Error() string { return "decode the audio: " + e.First }

// Header is what a map was built from. A reader whose Format, VAD, Mode,
// Rate, FrameMS or Recipe differs reads the map as stale.
type Header struct {
	Format  int
	VAD     string
	Mode    int
	Rate    int
	FrameMS int
	// Stream is the audio ordinal the map heard.
	Stream int
	Recipe string
	// Start is how late the stream starts, which the map begins with as
	// silence.
	Start  time.Duration
	Frames int
	// Identity is the file's identity when the map was built, set by the
	// Node that keeps it.
	Identity []byte
}

// Map is one bit per frame.
type Map struct {
	Header Header
	bits   []uint64
}

// New is the map of h.Frames frames whose speech is bits, frame i at bit
// i%64 of word i/64. Bits past Frames must be clear.
func New(h Header, words []uint64) (Map, error) {
	if h.Frames < 0 || h.Frames > MaxFrames || len(words) != wordsOf(h.Frames) {
		return Map{}, fmt.Errorf("%w: %d words for %d frames", ErrFrames, len(words), h.Frames)
	}
	if r := h.Frames % 64; r != 0 && words[len(words)-1]>>r != 0 {
		return Map{}, fmt.Errorf("%w: speech past frame %d", ErrFrames, h.Frames)
	}
	return Map{Header: h, bits: words}, nil
}

func wordsOf(frames int) int { return (frames + 63) / 64 }

func (m Map) Speech(frame int) bool {
	return frame >= 0 && frame < m.Header.Frames && m.bits[frame/64]&(1<<(frame%64)) != 0
}

// Bits are the map's words, frame i at bit i%64 of word i/64. Callers
// don't change them.
func (m Map) Bits() []uint64 { return m.bits }

// Density is the share of frames that hold speech.
func (m Map) Density() float64 {
	if m.Header.Frames == 0 {
		return 0
	}
	set := 0
	for _, w := range m.bits {
		set += bits.OnesCount64(w)
	}
	return float64(set) / float64(m.Header.Frames)
}

func (m Map) Duration() time.Duration { return time.Duration(m.Header.Frames) * Frame }

// Builder sets a map's bits frame by frame, in order.
type Builder struct {
	frames int
	words  []uint64
}

func (b *Builder) Add(speech bool) {
	if b.frames%64 == 0 {
		b.words = append(b.words, 0)
	}
	if speech {
		b.words[len(b.words)-1] |= 1 << (b.frames % 64)
	}
	b.frames++
}

func (b *Builder) Frames() int { return b.frames }

// Map is the map of the frames added, under h with its Frames set.
func (b *Builder) Map(h Header) (Map, error) {
	h.Frames = b.frames
	return New(h, b.words)
}
