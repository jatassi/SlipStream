package cues

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/language"
)

var (
	vttTiming = regexp.MustCompile(`^\s*((?:\d+:)?\d{1,2}:\d{2}\.\d{3})\s+-->\s+((?:\d+:)?\d{1,2}:\d{2}\.\d{3})`)
	vttInline = regexp.MustCompile(`<\d+:\d{2}(:\d{2})?\.\d{3}>`)
	vttRuby   = regexp.MustCompile(`(?s)<rt>.*?</rt>`)
	vttBlank  = regexp.MustCompile(`\n[ \t]*\n`)
)

// fromWebVTT reads WebVTT's cues. Cue settings, NOTE, STYLE and REGION
// blocks, voice names and ruby annotations are markup and drop.
func fromWebVTT(text string, _ language.Code, _ float64) ([]item, error) {
	var items []item
	for _, block := range vttBlank.Split(newlines.ReplaceAllString(text, "\n"), -1) {
		lines := strings.Split(strings.Trim(block, "\n"), "\n")
		for i, l := range lines {
			m := vttTiming.FindStringSubmatch(l)
			if m == nil {
				continue
			}
			start, end := vttTime(m[1]), vttTime(m[2])
			payload := strings.Join(lines[i+1:], "\n")
			payload = vttInline.ReplaceAllString(vttRuby.ReplaceAllString(payload, ""), "")
			items = append(items, item{start: start, end: end, text: keepStyle(payload)})
			break
		}
	}
	return items, nil
}

// vttTime reads a stamp vttTiming matched: [hh:]mm:ss.ttt.
func vttTime(s string) time.Duration {
	parts := strings.Split(s, ":")
	minutes := 0
	for _, p := range parts[:len(parts)-1] {
		n, _ := strconv.Atoi(p)
		minutes = minutes*60 + n
	}
	sec, _ := strconv.ParseFloat(parts[len(parts)-1], 64)
	return time.Duration(minutes)*time.Minute + time.Duration(sec*float64(time.Second)).Round(time.Millisecond)
}
