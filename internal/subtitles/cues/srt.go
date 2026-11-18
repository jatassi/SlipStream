package cues

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// srtStamp is an SRT timing line's two stamps, tolerating a period before
// the milliseconds, one or two digit fields and coordinates after it.
var srtStamp = regexp.MustCompile(`(?m)^[ \t]*(\d{1,3}):(\d{1,2}):(\d{1,2})[,.](\d{1,3})[ \t]*-->[ \t]*(\d{1,3}):(\d{1,2}):(\d{1,2})[,.](\d{1,3})`)

var srtIndex = regexp.MustCompile(`(?m)^[ \t]*\d+[ \t]*\r?$`)

// parseSRT reads text's cues by their timing lines and rewrites each
// stamp in the canonical HH:MM:SS,mmm. Every other byte stays, CRLF and
// tags included.
func parseSRT(text []byte, from Format) (Doc, error) {
	ms := srtStamp.FindAllSubmatchIndex(text, -1)
	if len(ms) == 0 {
		return Doc{}, noCues()
	}
	d := Doc{format: SRT, from: from, cues: make([]cue, len(ms))}
	bodies := make([]string, len(ms))
	out := make([]byte, 0, len(text))
	at := 0
	for i, m := range ms {
		start := srtTime(text, m[2:10])
		end := srtTime(text, m[10:18])
		if end < start {
			return Doc{}, endsEarly(start, end)
		}
		c := cue{start: start, end: end}
		out = append(out, text[at:m[2]]...)
		c.startAt[0] = len(out)
		out = append(out, stampSRT(start)...)
		c.startAt[1] = len(out)
		out = append(out, text[m[9]:m[10]]...)
		c.endAt[0] = len(out)
		out = append(out, stampSRT(end)...)
		c.endAt[1] = len(out)
		at = m[17]
		d.cues[i] = c
		bodies[i] = srtBody(text, m, ms, i)
	}
	out = append(out, text[at:]...)
	d.text = out
	markSkips(d.cues, bodies, srtMarkup)
	return d, nil
}

// srtBody is the text between cue i's timing line and the next cue's
// index line.
func srtBody(text []byte, m []int, ms [][]int, i int) string {
	from := len(text)
	if nl := bytes.IndexByte(text[m[1]:], '\n'); nl >= 0 {
		from = m[1] + nl + 1
	}
	to := len(text)
	if i+1 < len(ms) {
		to = ms[i+1][0]
		if idx := srtIndex.FindAllIndex(text[from:to], -1); len(idx) > 0 && strings.TrimSpace(string(text[from+idx[len(idx)-1][1]:to])) == "" {
			to = from + idx[len(idx)-1][0]
		}
	}
	if to < from {
		return ""
	}
	return string(text[from:to])
}

func srtTime(text []byte, g []int) time.Duration {
	n := func(k int) int {
		v, _ := strconv.Atoi(string(text[g[2*k]:g[2*k+1]]))
		return v
	}
	frac := string(text[g[6]:g[7]])
	ms, _ := strconv.Atoi(frac + strings.Repeat("0", 3-len(frac)))
	return time.Duration(n(0))*time.Hour + time.Duration(n(1))*time.Minute + time.Duration(n(2))*time.Second + time.Duration(ms)*time.Millisecond
}

func stampSRT(t time.Duration) []byte {
	ms := t.Milliseconds()
	return fmt.Appendf(nil, "%02d:%02d:%02d,%03d", ms/3600000, ms/60000%60, ms/1000%60, ms%1000)
}

var srtMarkup = regexp.MustCompile(`<[^>]*>|\{[^}]*\}`)

// item is one cue of a converted format: its times and its lines.
type item struct {
	start, end time.Duration
	text       string
}

// writeSRT writes items as SRT, dropping blank lines inside a cue (they
// would end it) and cues left with no text.
func writeSRT(items []item) []byte {
	var b bytes.Buffer
	n := 0
	for _, it := range items {
		var lines []string
		for l := range strings.SplitSeq(it.text, "\n") {
			if l = strings.TrimSpace(l); l != "" {
				lines = append(lines, l)
			}
		}
		if len(lines) == 0 {
			continue
		}
		n++
		fmt.Fprintf(&b, "%d\n%s --> %s\n%s\n\n", n, stampSRT(it.start.Round(time.Millisecond)), stampSRT(it.end.Round(time.Millisecond)), strings.Join(lines, "\n"))
	}
	return b.Bytes()
}
