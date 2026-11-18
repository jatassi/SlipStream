package cues

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/language"
)

var (
	subViewerTiming = regexp.MustCompile(`^(\d{1,2}):(\d{2}):(\d{2})\.(\d{1,3}),(\d{1,2}):(\d{2}):(\d{2})\.(\d{1,3})$`)
	tmPlayerLine    = regexp.MustCompile(`^(\d{1,2}):(\d{2}):(\d{2})(?:,\d+)?[:=](.*)$`)
)

// fromSubViewer reads SubViewer 2: a header in brackets, then a timing
// line, its text lines with [br] breaks, and a blank line.
func fromSubViewer(text string, _ language.Code, _ float64) ([]item, error) {
	var items []item
	open := -1
	for l := range strings.Lines(text) {
		l = strings.TrimSpace(l)
		if m := subViewerTiming.FindStringSubmatch(l); m != nil {
			items = append(items, item{start: clock(m[1:5]), end: clock(m[5:9])})
			open = len(items) - 1
			continue
		}
		if open < 0 {
			continue
		}
		if l == "" {
			open = -1
			continue
		}
		line := keepStyle(strings.ReplaceAll(l, "[br]", "\n"))
		items[open].text = strings.TrimPrefix(items[open].text+"\n"+line, "\n")
	}
	return items, nil
}

// fromTMPlayer reads h:mm:ss:text lines (or = for :), with | breaks. The
// format has no end times, so each cue ends where the next line starts.
// TMPlayer+ lines that share a start are one cue.
func fromTMPlayer(text string, _ language.Code, _ float64) ([]item, error) {
	var starts []time.Duration
	var texts []string
	for l := range strings.Lines(text) {
		m := tmPlayerLine.FindStringSubmatch(strings.TrimSpace(l))
		if m == nil {
			continue
		}
		start := clock([]string{m[1], m[2], m[3], ""})
		line := keepStyle(strings.ReplaceAll(m[4], "|", "\n"))
		if n := len(starts); n > 0 && starts[n-1] == start {
			texts[n-1] += "\n" + line
			continue
		}
		starts = append(starts, start)
		texts = append(texts, line)
	}
	return endAtNext(starts, texts), nil
}

// clock reads hours, minutes, seconds and a decimal fraction.
func clock(f []string) time.Duration {
	h, _ := strconv.Atoi(f[0])
	m, _ := strconv.Atoi(f[1])
	s, _ := strconv.Atoi(f[2])
	ms, _ := strconv.Atoi(f[3] + strings.Repeat("0", 3-len(f[3])))
	return time.Duration(h)*time.Hour + time.Duration(m)*time.Minute + time.Duration(s)*time.Second + time.Duration(ms)*time.Millisecond
}
