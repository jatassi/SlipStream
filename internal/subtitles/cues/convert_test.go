package cues

import (
	"bytes"
	"io"
	"regexp"
	"strings"
	"testing"

	"github.com/asticode/go-astisub"
)

func TestConvert(t *testing.T) {
	cases := []struct {
		file string
		lang string
		fps  float64
		from Format
		want string
	}{
		{"sample.vtt", "en", 0, WebVTT, "" +
			"1\n00:00:01,000 --> 00:00:03,500\nMy name is <i>Neil</i>.\n\n" +
			"2\n00:00:04,250 --> 00:00:06,000\nDon't let yourself & anyone\nget attached.\n\n" +
			"3\n01:00:07,000 --> 01:00:08,000\nHi there\n\n"},
		{"sample.ttml", "en", 0, TTML, "" +
			"1\n00:00:01,000 --> 00:00:03,500\nMy name is\nNeil.\n\n" +
			"2\n00:00:04,250 --> 00:00:06,000\n<i>Don't</i> let yourself\n\n" +
			"3\n00:00:07,000 --> 00:00:08,000\nget attached.\n\n" +
			"4\n00:01:02,200 --> 00:01:04,000\nThirty seconds flat.\n\n"},
		{"offsets.ttml", "en", 0, TTML, "1\n00:01:11,500 --> 00:01:13,500\nInside a timed div.\n\n"},
		{"sample.smi", "en", 0, SAMI, "" +
			"1\n00:00:01,000 --> 00:00:03,500\nHello\nthere\n\n" +
			"2\n00:00:04,250 --> 00:00:06,000\n<i>Don't</i> let yourself & anyone\n\n"},
		{"sample.smi", "ko", 0, SAMI, "1\n00:00:01,000 --> 00:00:03,500\n안녕하세요\n\n"},
		{"sample.microdvd.sub", "en", 0, MicroDVD, "" +
			"1\n00:00:01,001 --> 00:00:03,504\nMy name is\nNeil.\n\n" +
			"2\n00:00:04,254 --> 00:00:06,006\n<i>Don't let yourself</i>\nget attached.\n\n" +
			"3\n00:00:07,007 --> 00:00:08,008\nThirty seconds.\n\n"},
		{"sample.microdvd.sub", "en", 25, MicroDVD, "" +
			"1\n00:00:00,960 --> 00:00:03,360\nMy name is\nNeil.\n\n" +
			"2\n00:00:04,080 --> 00:00:05,760\n<i>Don't let yourself</i>\nget attached.\n\n" +
			"3\n00:00:06,720 --> 00:00:07,680\nThirty seconds.\n\n"},
		{"sample.mpl2.txt", "en", 0, MPL2, "" +
			"1\n00:00:01,000 --> 00:00:03,500\nMy name is\nNeil.\n\n" +
			"2\n00:00:04,200 --> 00:00:06,000\n<i>Don't let yourself</i>\nget attached.\n\n"},
		{"sample.subviewer.sub", "en", 0, SubViewer, "" +
			"1\n00:00:01,000 --> 00:00:03,500\nMy name is\nNeil.\n\n" +
			"2\n00:00:04,250 --> 00:00:06,000\nDon't let yourself\n\n"},
		{"sample.tmplayer.txt", "en", 0, TMPlayer, "" +
			"1\n00:00:01,000 --> 00:00:04,000\nMy name is\nNeil.\n\n" +
			"2\n00:00:04,000 --> 00:00:06,000\nDon't let yourself\n\n" +
			"3\n00:00:07,000 --> 00:00:12,000\nget attached.\n\n"},
	}
	for _, c := range cases {
		t.Run(c.file+"/"+c.lang, func(t *testing.T) {
			d, err := Parse(read(t, c.file), lang(t, c.lang), c.fps, limit)
			if err != nil {
				t.Fatal(err)
			}
			if d.Format() != SRT || d.From() != c.from || d.Ext() != "srt" {
				t.Fatalf("format %v from %v; want SRT from %v", d.Format(), d.From(), c.from)
			}
			if got := string(d.Bytes()); got != c.want {
				t.Fatalf("converted to\n%s\nwant\n%s", got, c.want)
			}
		})
	}
}

// TestConvertOracle reads each format go-astisub reads with go-astisub
// too, and asks for the same cue times and text.
func TestConvertOracle(t *testing.T) {
	cases := []struct {
		file   string
		oracle func(io.Reader) (*astisub.Subtitles, error)
	}{
		{"sample.vtt", astisub.ReadFromWebVTT},
		{"sample.ttml", astisub.ReadFromTTML},
		{"crlf.srt", astisub.ReadFromSRT},
		{"fonts.ass", astisub.ReadFromSSA},
		{"classic.ssa", astisub.ReadFromSSA},
	}
	for _, c := range cases {
		t.Run(c.file, func(t *testing.T) {
			in := read(t, c.file)
			want, err := c.oracle(bytes.NewReader(in))
			if err != nil {
				t.Fatal(err)
			}
			agrees(t, parse(t, in), want)
		})
	}
}

func agrees(t *testing.T, d Doc, want *astisub.Subtitles) {
	t.Helper()
	got := d.Times()
	if len(got) != len(want.Items) {
		t.Fatalf("%d cues; go-astisub reads %d", len(got), len(want.Items))
	}
	read := astisub.ReadFromSRT
	if d.Format() != SRT {
		read = astisub.ReadFromSSA
	}
	ours, err := read(bytes.NewReader(d.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	for i, it := range want.Items {
		if got[i].Start != it.StartAt || got[i].End != it.EndAt {
			t.Errorf("cue %d at %v to %v; go-astisub reads %v to %v", i, got[i].Start, got[i].End, it.StartAt, it.EndAt)
		}
		if g, w := plain(ours.Items[i]), plain(it); g != w {
			t.Errorf("cue %d reads %q; go-astisub reads %q", i, g, w)
		}
	}
}

var spaces = regexp.MustCompile(`\s+`)

func plain(it *astisub.Item) string {
	var lines []string
	for _, l := range it.Lines {
		for _, li := range l.Items {
			lines = append(lines, li.Text)
		}
	}
	return strings.TrimSpace(spaces.ReplaceAllString(anyTag.ReplaceAllString(strings.Join(lines, " "), ""), " "))
}
