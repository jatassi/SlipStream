package cues

import (
	"bytes"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

var (
	assDefault = []string{"layer", "start", "end", "style", "name", "marginl", "marginr", "marginv", "effect", "text"}
	ssaDefault = []string{"marked", "start", "end", "style", "name", "marginl", "marginr", "marginv", "effect", "text"}
	assTime    = regexp.MustCompile(`^(\d{1,3}):(\d{1,2}):(\d{1,2})(?:[.,](\d{1,3}))?$`)
	assMarkup  = regexp.MustCompile(`\{[^}]*\}|\\[Nnh]`)
)

// parseASS reads the Dialogue lines of text's [Events] section by its
// Format line and rewrites each Start and End in the canonical
// H:MM:SS.cc. Every other byte stays: [Script Info], styles, [Fonts] and
// [Graphics] blobs, Comment lines and inline override tags.
func parseASS(text []byte, f Format) (Doc, error) {
	d := Doc{format: f, from: f}
	var bodies []string
	out := make([]byte, 0, len(text))
	format := assDefault
	if f == SSA {
		format = ssaDefault
	}
	section := ""
	at := 0
	for lineAt := 0; lineAt < len(text); {
		end := len(text)
		if nl := bytes.IndexByte(text[lineAt:], '\n'); nl >= 0 {
			end = lineAt + nl
		}
		line := strings.TrimRight(string(text[lineAt:end]), "\r")
		next := end + 1
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]"):
			section = strings.ToLower(trimmed)
		case section != "[events]":
		case hasField(trimmed, "format"):
			format = format[:0:0]
			for _, name := range strings.Split(trimmed[len("format:"):], ",") {
				format = append(format, strings.ToLower(strings.TrimSpace(name)))
			}
		case hasField(trimmed, "dialogue"):
			c, body, err := assCue(lineAt, line, format)
			if err != nil {
				return Doc{}, err
			}
			out = append(out, text[at:c.startAt[0]]...)
			s, e := c.startAt[1], c.endAt
			c.startAt = [2]int{len(out), 0}
			out = append(out, stampASS(c.start)...)
			c.startAt[1] = len(out)
			out = append(out, text[s:e[0]]...)
			c.endAt[0] = len(out)
			out = append(out, stampASS(c.end)...)
			c.endAt[1] = len(out)
			at = e[1]
			d.cues = append(d.cues, c)
			bodies = append(bodies, body)
		}
		lineAt = next
	}
	if len(d.cues) == 0 {
		return Doc{}, noCues()
	}
	out = append(out, text[at:]...)
	d.text = out
	markSkips(d.cues, bodies, assMarkup)
	return d, nil
}

func hasField(line, name string) bool {
	return len(line) > len(name) && line[len(name)] == ':' && strings.EqualFold(line[:len(name)], name)
}

// assCue reads one Dialogue line at lineAt, its stamps' ranges in text
// and its Text field.
func assCue(lineAt int, line string, format []string) (cue, string, error) {
	fields := assFields(line, len(format))
	field := func(name string) ([2]int, bool) {
		i := slices.Index(format, name)
		if i < 0 || i >= len(fields) {
			return [2]int{}, false
		}
		return fields[i], true
	}
	sr, okS := field("start")
	er, okE := field("end")
	if !okS || !okE || sr[0] > er[0] {
		return cue{}, "", Unreadable{fmt.Sprintf("a Dialogue line has no Start and End: %q", line)}
	}
	start, err := parseASSTime(line[sr[0]:sr[1]])
	if err != nil {
		return cue{}, "", err
	}
	end, err := parseASSTime(line[er[0]:er[1]])
	if err != nil {
		return cue{}, "", err
	}
	if end < start {
		return cue{}, "", endsEarly(start, end)
	}
	body := ""
	if tr, ok := field("text"); ok {
		body = line[tr[0]:tr[1]]
	}
	return cue{
		start:   start,
		end:     end,
		startAt: [2]int{lineAt + sr[0], lineAt + sr[1]},
		endAt:   [2]int{lineAt + er[0], lineAt + er[1]},
	}, body, nil
}

// assFields are the byte ranges of a Dialogue line's n fields after its
// colon, spaces trimmed. The last field takes the rest of the line, commas
// included, and a short line has fewer.
func assFields(line string, n int) [][2]int {
	at := strings.IndexByte(line, ':') + 1
	fields := make([][2]int, 0, n)
	for len(fields) < n {
		to := len(line)
		if comma := strings.IndexByte(line[at:], ','); comma >= 0 && len(fields) < n-1 {
			to = at + comma
		}
		from, end := at, to
		for from < end && line[from] == ' ' {
			from++
		}
		for end > from && line[end-1] == ' ' {
			end--
		}
		fields = append(fields, [2]int{from, end})
		if to == len(line) {
			break
		}
		at = to + 1
	}
	return fields
}

// parseASSTime reads H:MM:SS.cc, rounding a millisecond fraction to the
// centisecond the canonical stamp writes.
func parseASSTime(s string) (time.Duration, error) {
	m := assTime.FindStringSubmatch(s)
	if m == nil {
		return 0, Unreadable{fmt.Sprintf("%q is not an ASS time", s)}
	}
	h, _ := strconv.Atoi(m[1])
	mi, _ := strconv.Atoi(m[2])
	sec, _ := strconv.Atoi(m[3])
	frac := m[4] + strings.Repeat("0", 3-len(m[4]))
	ms, _ := strconv.Atoi(frac)
	t := time.Duration(h)*time.Hour + time.Duration(mi)*time.Minute + time.Duration(sec)*time.Second + time.Duration(ms)*time.Millisecond
	return t.Round(10 * time.Millisecond), nil
}

func stampASS(t time.Duration) []byte {
	cs := t.Milliseconds() / 10
	return fmt.Appendf(nil, "%d:%02d:%02d.%02d", cs/360000, cs/6000%60, cs/100%60, cs%100)
}
