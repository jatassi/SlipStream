package cues

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/language"
)

var (
	samiClass = regexp.MustCompile(`(?is)\.([a-z0-9_-]+)\s*\{([^}]*)\}`)
	samiLang  = regexp.MustCompile(`(?i)lang\s*:\s*([a-z_-]+)`)
	samiSync  = regexp.MustCompile(`(?i)<sync\b[^>]*?\bstart\s*=\s*["']?(\d{1,9})["']?[^>]*>`)
	samiPara  = regexp.MustCompile(`(?i)<p\b([^>]*)>`)
	samiAttr  = regexp.MustCompile(`(?i)\bclass\s*=\s*["']?([a-z0-9_-]+)`)
	samiBreak = regexp.MustCompile(`(?i)<br\s*/?>`)
	samiEnd   = regexp.MustCompile(`(?is)</(p|sync|body|sami)>.*`)
)

// fromSAMI reads the class of lang from a SAMI file, which can hold
// several languages by class, or its first class when none is lang's.
// Each SYNC ends where the class's next SYNC starts, and an empty one
// (&nbsp;) only ends the one before.
func fromSAMI(text string, lang language.Code, _ float64) ([]item, error) {
	class := samiPick(text, lang)
	syncs := samiSync.FindAllStringSubmatchIndex(text, -1)
	var starts []time.Duration
	var texts []string
	for i, s := range syncs {
		to := len(text)
		if i+1 < len(syncs) {
			to = syncs[i+1][0]
		}
		body, ok := samiParagraph(text[s[1]:to], class)
		if !ok {
			continue
		}
		ms, _ := strconv.Atoi(text[s[2]:s[3]])
		starts = append(starts, time.Duration(ms)*time.Millisecond)
		body = newlines.ReplaceAllString(body, " ")
		texts = append(texts, keepStyle(samiBreak.ReplaceAllString(body, "\n")))
	}
	return endAtNext(starts, texts), nil
}

func samiPick(text string, lang language.Code) string {
	first := ""
	for _, m := range samiClass.FindAllStringSubmatch(text, -1) {
		l := samiLang.FindStringSubmatch(m[2])
		if l == nil {
			continue
		}
		if first == "" {
			first = m[1]
		}
		if c, ok := language.ParseTag(l[1]); ok && c == lang {
			return m[1]
		}
	}
	return first
}

// samiParagraph is the text of the P of class in one SYNC's content, or
// the content itself in a file with no classes.
func samiParagraph(content, class string) (string, bool) {
	paras := samiPara.FindAllStringSubmatchIndex(content, -1)
	if class == "" && len(paras) == 0 {
		return samiEnd.ReplaceAllString(content, ""), true
	}
	for i, p := range paras {
		attr := samiAttr.FindStringSubmatch(content[p[2]:p[3]])
		if class != "" && (len(attr) < 2 || !strings.EqualFold(attr[1], class)) {
			continue
		}
		to := len(content)
		if i+1 < len(paras) {
			to = paras[i+1][0]
		}
		return samiEnd.ReplaceAllString(content[p[1]:to], ""), true
	}
	return "", false
}

// fromTTML reads TTML (DFXP) p elements, with begin and end or dur in
// clock, frame, tick or offset times, inside the begin of their body and
// div. br breaks a line, and an italic span becomes <i>.
func fromTTML(text string, _ language.Code, _ float64) ([]item, error) {
	dec := xml.NewDecoder(strings.NewReader(text))
	r := ttmlReader{times: readTTMLTimes(nil)}
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			return r.items, nil
		}
		if err != nil {
			return nil, fmt.Errorf("its TTML isn't well-formed XML: %w", err)
		}
		switch el := tok.(type) {
		case xml.StartElement:
			if err := r.start(el); err != nil {
				return nil, err
			}
		case xml.EndElement:
			r.end(el)
		case xml.CharData:
			if r.open != nil {
				r.open.WriteString(newlines.ReplaceAllString(string(el), " "))
			}
		}
	}
}

// ttmlReader is one TTML document's state as its tokens stream past: the
// begin offsets of the open body and divs, the p being read and whether
// each open span is italic.
type ttmlReader struct {
	times   ttmlTimes
	offsets []time.Duration
	open    *strings.Builder
	cur     item
	italic  []bool
	items   []item
}

func (r *ttmlReader) base() time.Duration {
	if len(r.offsets) == 0 {
		return 0
	}
	return r.offsets[len(r.offsets)-1]
}

func (r *ttmlReader) start(el xml.StartElement) error {
	switch el.Name.Local {
	case "tt":
		r.times = readTTMLTimes(el.Attr)
	case "body", "div":
		begin, _, _, err := r.times.span(el.Attr)
		if err != nil {
			return err
		}
		r.offsets = append(r.offsets, r.base()+begin)
	case "p":
		begin, end, ok, err := r.times.span(el.Attr)
		if err != nil || !ok {
			return err
		}
		r.cur = item{start: r.base() + begin, end: r.base() + end}
		r.open = &strings.Builder{}
	case "br":
		if r.open != nil {
			r.open.WriteString("\n")
		}
	case "span":
		it := attr(el.Attr, "fontStyle") == "italic"
		r.italic = append(r.italic, it)
		if r.open != nil && it {
			r.open.WriteString("<i>")
		}
	}
	return nil
}

func (r *ttmlReader) end(el xml.EndElement) {
	switch el.Name.Local {
	case "body", "div":
		r.offsets = r.offsets[:len(r.offsets)-1]
	case "p":
		if r.open != nil {
			r.cur.text = r.open.String()
			r.items = append(r.items, r.cur)
			r.open = nil
		}
	case "span":
		if r.italic[len(r.italic)-1] && r.open != nil {
			r.open.WriteString("</i>")
		}
		r.italic = r.italic[:len(r.italic)-1]
	}
}

type ttmlTimes struct {
	tickRate, frameRate float64
}

func readTTMLTimes(attrs []xml.Attr) ttmlTimes {
	t := ttmlTimes{tickRate: 1, frameRate: 30}
	if r, err := strconv.ParseFloat(attr(attrs, "tickRate"), 64); err == nil && r > 0 {
		t.tickRate = r
	}
	if r, err := strconv.ParseFloat(attr(attrs, "frameRate"), 64); err == nil && r > 0 {
		t.frameRate = r
	}
	return t
}

// span is an element's begin and its end, from end or begin plus dur. ok
// is false when it has no end.
func (t ttmlTimes) span(attrs []xml.Attr) (begin, end time.Duration, ok bool, err error) {
	if begin, err = t.time(attr(attrs, "begin")); err != nil {
		return 0, 0, false, err
	}
	if s := attr(attrs, "end"); s != "" {
		end, err = t.time(s)
		return begin, end, err == nil, err
	}
	if s := attr(attrs, "dur"); s != "" {
		dur, err := t.time(s)
		return begin, begin + dur, err == nil, err
	}
	return begin, 0, false, nil
}

var (
	ttmlClock  = regexp.MustCompile(`^(\d{1,3}):(\d{2}):(\d{2})(?:\.(\d{1,3})\d*|:(\d{1,3})(?:\.\d+)?)?$`)
	ttmlOffset = regexp.MustCompile(`^(\d{1,15}(?:\.\d{1,9})?)(h|ms|m|s|f|t)$`)
)

// longest bounds an offset time, so a huge one can't overflow.
const longest = 1000 * time.Hour

func (t ttmlTimes) time(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	if m := ttmlClock.FindStringSubmatch(s); m != nil {
		d := clock([]string{m[1], m[2], m[3], m[4]})
		if m[5] != "" {
			f, _ := strconv.Atoi(m[5])
			d += time.Duration(float64(f) / t.frameRate * float64(time.Second))
		}
		return d.Round(time.Millisecond), nil
	}
	m := ttmlOffset.FindStringSubmatch(s)
	if m == nil {
		return 0, fmt.Errorf("%q is not a TTML time", s)
	}
	v, _ := strconv.ParseFloat(m[1], 64)
	unit := map[string]float64{
		"h": 3600, "m": 60, "s": 1, "ms": 0.001, "f": 1 / t.frameRate, "t": 1 / t.tickRate,
	}[m[2]]
	if v*unit > longest.Seconds() {
		return 0, fmt.Errorf("%q is longer than any video", s)
	}
	return time.Duration(v * unit * float64(time.Second)).Round(time.Millisecond), nil
}

func attr(attrs []xml.Attr, local string) string {
	for _, a := range attrs {
		if a.Name.Local == local {
			return a.Value
		}
	}
	return ""
}
