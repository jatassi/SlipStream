package cues

import (
	"bytes"
	"embed"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/language"
)

//go:embed testdata
var testdata embed.FS

const limit = 1 << 20

func read(t testing.TB, name string) []byte {
	t.Helper()
	b, err := testdata.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func lang(t testing.TB, s string) language.Code {
	t.Helper()
	c, err := language.Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func parse(t testing.TB, b []byte) Doc {
	t.Helper()
	d, err := Parse(b, lang(t, "en"), 0, limit)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestKeepsBytes(t *testing.T) {
	cases := []struct {
		file   string
		format Format
		ext    Ext
		times  []Times
	}{
		{"crlf.srt", SRT, "srt", []Times{
			{Start: ms(1000), End: ms(3500)},
			{Start: ms(4250), End: ms(6000)},
			{Start: ms(7000), End: ms(8000), Skip: true},
			{Start: ms(62003), End: ms(64100)},
			{Start: ms(70000), End: ms(72000), Skip: true},
		}},
		{"fonts.ass", ASS, "ass", []Times{
			{Start: ms(1000), End: ms(3500)},
			{Start: ms(4250), End: ms(6000)},
			{Start: ms(7000), End: ms(9000), Skip: true},
			{Start: ms(62100), End: ms(64100)},
		}},
		{"classic.ssa", SSA, "ssa", []Times{
			{Start: ms(1000), End: ms(2500)},
			{Start: ms(3000), End: ms(4000)},
		}},
	}
	for _, c := range cases {
		t.Run(c.file, func(t *testing.T) {
			d := parse(t, read(t, c.file))
			if d.Format() != c.format || d.From() != c.format || d.Ext() != c.ext {
				t.Fatalf("format %v from %v ext %q; want %v and %q", d.Format(), d.From(), d.Ext(), c.format, c.ext)
			}
			if got := d.Times(); !equalTimes(got, c.times) {
				t.Fatalf("times %v; want %v", got, c.times)
			}
			keepsBytes(t, read(t, c.file), d)
		})
	}
}

// keepsBytes asks that retiming d to its own times writes in again, and
// that a +2 s retime changes only the stamps.
func keepsBytes(t *testing.T, in []byte, d Doc) {
	t.Helper()
	same, err := d.Retimed(d.Times())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(same.Bytes(), in) {
		t.Fatalf("Retimed(Times()) changed the bytes:\n%s", same.Bytes())
	}
	shifted := make([]Times, d.Len())
	for i, tm := range d.Times() {
		shifted[i] = Times{Start: tm.Start + 2*time.Second, End: tm.End + 2*time.Second}
	}
	moved, err := d.Retimed(shifted)
	if err != nil {
		t.Fatal(err)
	}
	if !SameText(d, moved) || bytes.Equal(moved.Bytes(), in) {
		t.Fatal("a +2 s retime changed bytes outside the stamps, or no bytes")
	}
	again := parse(t, moved.Bytes())
	if !equalStarts(again.Times(), shifted) || !SameText(again, moved) {
		t.Fatalf("the retimed bytes parse to %v; want %v", again.Times(), shifted)
	}
}

func TestKeepsFonts(t *testing.T) {
	d := parse(t, read(t, "fonts.ass"))
	shifted := make([]Times, d.Len())
	for i, tm := range d.Times() {
		shifted[i] = Times{Start: tm.Start + time.Hour, End: tm.End + time.Hour}
	}
	moved, err := d.Retimed(shifted)
	if err != nil {
		t.Fatal(err)
	}
	out := string(moved.Bytes())
	for _, kept := range []string{
		"[Fonts]\nfontname: CustomFont_0.ttf\nM(\\'!!!!!!!!@!%!!!@!!!Q`!!$!-1T5\"0",
		"Dialogue: 0,9:99:99.99,this is inside a font blob and stays",
		"[Graphics]\nfilename: logo.png",
		"Comment: 0,0:00:02.00,0:00:03.00,Default,,0,0,0,,translator note",
		"Dialogue: 0,1:00:04.25,1:00:06.00,Default,Neil,0,0,0,,{\\t(0,500,\\fscx120)}Out, and away,\\Nfor good.",
		"{\\move(100,100,500,500)}",
	} {
		if !strings.Contains(out, kept) {
			t.Errorf("retimed ASS lost %q", kept)
		}
	}
}

func TestRetimedUnit(t *testing.T) {
	cases := []struct {
		file string
		unit time.Duration
	}{{"crlf.srt", time.Millisecond}, {"fonts.ass", 10 * time.Millisecond}, {"classic.ssa", 10 * time.Millisecond}}
	for _, c := range cases {
		t.Run(c.file, func(t *testing.T) {
			d := parse(t, read(t, c.file))
			if d.Unit() != c.unit {
				t.Fatalf("unit %v; want %v", d.Unit(), c.unit)
			}
			ts := make([]Times, d.Len())
			for i := range ts {
				at := time.Duration(i*7919+13) * c.unit
				ts[i] = Times{Start: at, End: at + 3*c.unit}
			}
			ts[0] = Times{}
			moved, err := d.Retimed(ts)
			if err != nil {
				t.Fatal(err)
			}
			if got := parse(t, moved.Bytes()).Times(); !equalStarts(got, ts) {
				t.Fatalf("round trip %v; want %v", got, ts)
			}
			refusesOffUnit(t, d, ts)
		})
	}
}

// refusesOffUnit asks that Retimed refuses a time off d's unit, below
// zero or ending early, and a count that isn't d's.
func refusesOffUnit(t *testing.T, d Doc, ts []Times) {
	t.Helper()
	u := d.Unit()
	for _, bad := range []Times{
		{Start: u + time.Millisecond/2, End: 2 * u},
		{Start: u, End: 2*u + u/2 + time.Microsecond},
		{Start: -u, End: u},
		{Start: 2 * u, End: u},
	} {
		off := append([]Times(nil), ts...)
		off[1] = bad
		if _, err := d.Retimed(off); !errors.Is(err, ErrStamp) {
			t.Errorf("Retimed with %v: %v; want ErrStamp", bad, err)
		}
	}
	if _, err := d.Retimed(ts[1:]); err == nil {
		t.Error("Retimed with one time too few wrote the Doc")
	}
}

func TestParseRefuses(t *testing.T) {
	cases := map[string][]byte{
		"no cues":       []byte("hello\nthere\n"),
		"ends early":    []byte("1\n00:00:05,000 --> 00:00:04,000\nback\n"),
		"over the cap":  bytes.Repeat([]byte("a"), limit+1),
		"no frame rate": []byte("{10}{20}no rate\n"),
		"empty ASS":     []byte("[Script Info]\nScriptType: v4.00+\n\n[Events]\nFormat: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text\nDialogue: 0,bad,0:00:01.00,Default,,0,0,0,,x\n"),
	}
	for name, b := range cases {
		var u Unreadable
		if _, err := Parse(b, lang(t, "en"), 0, limit); !errors.As(err, &u) {
			t.Errorf("%s: %v; want Unreadable", name, err)
		}
	}
}

func ms(n int) time.Duration { return time.Duration(n) * time.Millisecond }

func equalTimes(a, b []Times) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalStarts(a, b []Times) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Start != b[i].Start || a[i].End != b[i].End {
			return false
		}
	}
	return true
}

func FuzzParse(f *testing.F) {
	for _, name := range []string{
		"crlf.srt", "fonts.ass", "classic.ssa", "sample.vtt", "sample.ttml", "sample.smi",
		"sample.microdvd.sub", "sample.mpl2.txt", "sample.subviewer.sub", "sample.tmplayer.txt",
	} {
		f.Add(read(f, name))
	}
	en := lang(f, "en")
	f.Fuzz(func(t *testing.T, b []byte) {
		d, err := Parse(b, en, 25, limit)
		if err != nil {
			return
		}
		again, err := Parse(d.Bytes(), en, 25, limit)
		if err != nil {
			t.Fatalf("Bytes don't parse again: %v\n%q", err, d.Bytes())
		}
		if !equalTimes(again.Times(), d.Times()) || again.Format() != d.Format() {
			t.Fatalf("Bytes parse again to %v; want %v", again.Times(), d.Times())
		}
		if same, err := d.Retimed(d.Times()); err != nil || !bytes.Equal(same.Bytes(), d.Bytes()) {
			t.Fatalf("Retimed(Times()) changed the Doc: %v", err)
		}
	})
}
