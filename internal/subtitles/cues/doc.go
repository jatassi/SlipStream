// Package cues reads subtitle bytes in any common format and encoding and
// keeps them as SRT, ASS or SSA in UTF-8 without a BOM. A Doc keeps every
// byte outside its timestamps, so retiming it never touches cue text
// (ADR 0008).
package cues

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/language"
)

type Format uint8

const (
	SRT Format = iota + 1
	ASS
	SSA
	WebVTT
	MicroDVD
	SubViewer
	MPL2
	TMPlayer
	SAMI
	TTML
	VobSub
	PGS
)

var formatNames = []string{
	SRT: "SRT", ASS: "ASS", SSA: "SSA", WebVTT: "WebVTT", MicroDVD: "MicroDVD", SubViewer: "SubViewer",
	MPL2: "MPL2", TMPlayer: "TMPlayer", SAMI: "SAMI", TTML: "TTML", VobSub: "VobSub", PGS: "PGS",
}

func (f Format) String() string {
	if int(f) < len(formatNames) && formatNames[f] != "" {
		return formatNames[f]
	}
	return fmt.Sprintf("Format(%d)", f)
}

// Ext is what a Sidecar's tail ends in: "srt", "ass" or "ssa".
type Ext string

var exts = map[Format]Ext{SRT: "srt", ASS: "ass", SSA: "ssa"}

// Times is one cue's time. Skip marks a cue that carries no speech signal
// (empty after markup, wholly bracketed or music symbols only, or a credit
// line first or last), which Timing sync lets ride a neighbour's offset.
type Times struct {
	Start, End time.Duration
	Skip       bool
}

// Doc is one parsed subtitle in UTF-8 without a BOM, in SRT, ASS or SSA.
type Doc struct {
	format Format
	from   Format
	text   []byte
	cues   []cue
}

// cue is one cue's times and the byte ranges of its two stamps in text.
type cue struct {
	start, end     time.Duration
	startAt, endAt [2]int
	skip           bool
}

var (
	ErrImage = errors.New("image subtitles can't be used")
	// ErrStamp is a time Retimed can't write in the Doc's format: below
	// zero, off its unit, or an end before its start.
	ErrStamp = errors.New("the time can't be written in this subtitle format")
)

// Unreadable is a subtitle Parse can't read, with what was wrong.
type Unreadable struct{ Cause string }

func (u Unreadable) Error() string { return "the subtitle can't be read: " + u.Cause }

// Parse sniffs b by its bytes, decodes its encoding for lang, parses it,
// and converts every format but SRT, ASS and SSA to SRT. fps is the
// video's frame rate, for MicroDVD. Zero takes the file's {1}{1}23.976
// line or refuses. Bytes over limit refuse.
func Parse(b []byte, lang language.Code, fps float64, limit int) (Doc, error) {
	if len(b) > limit {
		return Doc{}, Unreadable{fmt.Sprintf("it's %d bytes, over the %d byte limit", len(b), limit)}
	}
	if image(b) != 0 {
		return Doc{}, ErrImage
	}
	text, _, err := Decode(b, lang)
	if err != nil {
		return Doc{}, Unreadable{err.Error()}
	}
	f := sniff(text)
	if f == SRT {
		return parseSRT(text, SRT)
	}
	if f == ASS || f == SSA {
		return parseASS(text, f)
	}
	convert, ok := converters[f]
	if !ok {
		return Doc{}, Unreadable{"its bytes are in no subtitle format"}
	}
	items, err := convert(string(text), lang, fps)
	if err != nil {
		return Doc{}, Unreadable{err.Error()}
	}
	return parseSRT(writeSRT(items), f)
}

// Format is SRT, ASS or SSA: what the Doc's bytes are.
func (d Doc) Format() Format { return d.format }

// From is what the bytes Parse read were.
func (d Doc) From() Format { return d.from }

func (d Doc) Len() int { return len(d.cues) }

func (d Doc) Ext() Ext { return exts[d.format] }

// Bytes is the Doc in UTF-8 without a BOM.
func (d Doc) Bytes() []byte { return bytes.Clone(d.text) }

// Unit is the step of the format's stamps: 1 ms for SRT and 10 ms for
// ASS and SSA.
func (d Doc) Unit() time.Duration {
	if d.format == SRT {
		return time.Millisecond
	}
	return 10 * time.Millisecond
}

// Times is each cue's time in source order, with the skip rules applied.
func (d Doc) Times() []Times {
	ts := make([]Times, len(d.cues))
	for i, c := range d.cues {
		ts[i] = Times{Start: c.start, End: c.end, Skip: c.skip}
	}
	return ts
}

// Retimed writes ts as each cue's stamps, in order, and keeps every other
// byte. It writes the times as given: one below zero, off Unit or ending
// before it starts is ErrStamp. Skip is ignored. ASS inline \t and \move
// times stay as written.
func (d Doc) Retimed(ts []Times) (Doc, error) {
	if len(ts) != len(d.cues) {
		return Doc{}, fmt.Errorf("retime %d cues with %d times", len(d.cues), len(ts))
	}
	unit := d.Unit()
	for _, t := range ts {
		if t.Start < 0 || t.End < t.Start || t.Start%unit != 0 || t.End%unit != 0 {
			return Doc{}, fmt.Errorf("%w: %v to %v", ErrStamp, t.Start, t.End)
		}
	}
	stamp := stampSRT
	if d.format != SRT {
		stamp = stampASS
	}
	out := Doc{format: d.format, from: d.from, cues: make([]cue, len(d.cues))}
	text := make([]byte, 0, len(d.text))
	at := 0
	for i, c := range d.cues {
		n := cue{start: ts[i].Start, end: ts[i].End, skip: c.skip}
		text = append(text, d.text[at:c.startAt[0]]...)
		n.startAt[0] = len(text)
		text = append(text, stamp(n.start)...)
		n.startAt[1] = len(text)
		text = append(text, d.text[c.startAt[1]:c.endAt[0]]...)
		n.endAt[0] = len(text)
		text = append(text, stamp(n.end)...)
		n.endAt[1] = len(text)
		at = c.endAt[1]
		out.cues[i] = n
	}
	text = append(text, d.text[at:]...)
	out.text = text
	return out, nil
}

// SameText reports whether a and b are one format with the same cues and
// every byte outside their stamps equal: the check's text compare.
func SameText(a, b Doc) bool {
	if a.format != b.format || len(a.cues) != len(b.cues) {
		return false
	}
	ga, gb := a.gaps(), b.gaps()
	for i := range ga {
		if !bytes.Equal(ga[i], gb[i]) {
			return false
		}
	}
	return true
}

// gaps are the byte runs between stamps, first and last included.
func (d Doc) gaps() [][]byte {
	gs := make([][]byte, 0, 2*len(d.cues)+1)
	at := 0
	for _, c := range d.cues {
		gs = append(gs, d.text[at:c.startAt[0]], d.text[c.startAt[1]:c.endAt[0]])
		at = c.endAt[1]
	}
	return append(gs, d.text[at:])
}

func noCues() error { return Unreadable{"it has no cues"} }

func endsEarly(start, end time.Duration) error {
	return Unreadable{fmt.Sprintf("a cue ends at %v, before it starts at %v", end, start)}
}
