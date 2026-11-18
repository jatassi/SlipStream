package cues

import (
	"html"
	"regexp"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/language"
)

// converter reads one format's text as SRT items.
type converter func(text string, lang language.Code, fps float64) ([]item, error)

var converters = map[Format]converter{
	WebVTT:    fromWebVTT,
	MicroDVD:  fromMicroDVD,
	SubViewer: fromSubViewer,
	MPL2:      fromMPL2,
	TMPlayer:  fromTMPlayer,
	SAMI:      fromSAMI,
	TTML:      fromTTML,
}

// lastShown is how long the last cue of a format that writes no end
// times (TMPlayer, SAMI) stays up.
const lastShown = 5 * time.Second

var (
	keptTag  = regexp.MustCompile(`(?i)^</?([ibu])(\.[^>]*)?>$`)
	anyTag   = regexp.MustCompile(`<[^>]*>`)
	newlines = regexp.MustCompile(`\r?\n`)
)

// keepStyle keeps <i>, <b> and <u> (classes dropped), removes every other
// tag and unescapes entities, as SRT carries text.
func keepStyle(s string) string {
	s = anyTag.ReplaceAllStringFunc(s, func(tag string) string {
		if m := keptTag.FindStringSubmatch(tag); m != nil {
			if strings.HasPrefix(tag, "</") {
				return "</" + strings.ToLower(m[1]) + ">"
			}
			return "<" + strings.ToLower(m[1]) + ">"
		}
		return ""
	})
	return strings.ReplaceAll(html.UnescapeString(s), " ", " ")
}

// endAtNext gives each item the next item's start as its end, and the
// last one lastShown. Items with no text only end the one before.
func endAtNext(starts []time.Duration, texts []string) []item {
	var items []item
	for i, t := range texts {
		end := starts[i] + lastShown
		if i+1 < len(starts) {
			end = starts[i+1]
		}
		if strings.TrimSpace(t) != "" {
			items = append(items, item{start: starts[i], end: end, text: t})
		}
	}
	return items
}
