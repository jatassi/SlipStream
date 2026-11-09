package naming

import (
	"cmp"
	"regexp"
	"slices"
	"strings"
	"unicode"
)

// blockSpan finds a bound template's episode blocks: the block runs from
// the first SeqOuter or SeqInner ref to the last SeqInner ref, literals
// inside it included, and each SeqAlone ref is a block of its own. The
// block is empty when the template has no Inner ref, as a level template
// naming only its Outer.
func blockSpan(parts []part) (block span, alone []int) {
	first := -1
	for i, p := range parts {
		r, ok := p.(*boundRef)
		if !ok {
			continue
		}
		switch r.entry.Seq {
		case SeqOuter:
			if first < 0 {
				first = i
			}
		case SeqInner:
			if first < 0 {
				first = i
			}
			block = span{from: first, to: i + 1}
		case SeqAlone:
			alone = append(alone, i)
		}
	}
	return block, alone
}

// blockSpans are the spans multi renders for b, in part order.
func (b *bound) blockSpans() []span {
	var spans []span
	if b.block.to > b.block.from {
		spans = append(spans, b.block)
	}
	for _, i := range b.alone {
		spans = append(spans, span{from: i, to: i + 1})
	}
	slices.SortFunc(spans, func(x, y span) int { return cmp.Compare(x.from, y.from) })
	return spans
}

// episodeBlock is one episode block with the template text around it that
// a style repeats.
type episodeBlock struct {
	parts []part
	// lead is the letters that end the literal before the block, as "S"
	// in " - S{season:00}".
	lead string
	// before is the separator run before lead, as " - ", which Duplicate
	// repeats. A block with none repeats "-", as Sonarr does for an
	// absolute number.
	before string
}

// blockOf is the block parts[s] reads as.
func blockOf(parts []part, s span) episodeBlock {
	b := episodeBlock{parts: parts[s.from:s.to], before: "-"}
	if s.from == 0 {
		return b
	}
	l, ok := parts[s.from-1].(literal)
	if !ok {
		return b
	}
	text := string(l)
	rest := strings.TrimRightFunc(text, unicode.IsLetter)
	b.lead = text[len(rest):]
	if rest != "" && strings.Trim(rest, sepChars) == "" {
		b.before = rest
	}
	return b
}

// multi renders b for the units of one file, sorted by outer then inner,
// in style. One unit renders plain. Units crossing an outer render as
// Duplicate with each unit's own outer, and a gap in inner renders as
// Repeat, so 1 and 3 never read as 1 to 3. An Alone block renders Extend
// for Repeat and Scene and a range for both range styles. A block any unit
// has no number for renders as that unit's, so an absolute number one
// episode lacks prints for none, as in Sonarr. value renders one ref for
// one unit.
func multi(b episodeBlock, style MultiEpisode, units []Unit, value func(*boundRef, Unit) piece) []piece {
	outer, inner, at := b.refs()
	nums := make([]numbered, len(units))
	for i, u := range units {
		nums[i] = numbered{unit: u}
		if outer != nil {
			nums[i].outer, _ = number(outer, u)
		}
		var ok bool
		if nums[i].inner, ok = number(inner, u); !ok {
			return b.render(u, value)
		}
	}
	slices.SortStableFunc(nums, func(x, y numbered) int {
		return cmp.Or(cmp.Compare(x.outer, y.outer), cmp.Compare(x.inner, y.inner))
	})
	first, last := nums[0], nums[len(nums)-1]
	pieces := b.render(first.unit, value)
	if len(nums) == 1 {
		return pieces
	}

	switch {
	case first.outer != last.outer:
		style = Duplicate
	case gapped(nums):
		style = Repeat
	}
	if inner.entry.Seq == SeqAlone {
		style = aloneStyle[style]
	}

	between := b.lead
	if at > 0 {
		if l, ok := b.parts[at-1].(literal); ok {
			between = string(l)
		}
	}
	for _, n := range nums[1:] {
		switch style {
		case Extend:
			pieces = append(pieces, joint("-"), value(inner, n.unit))
		case Duplicate:
			pieces = append(pieces, joint(b.before+b.lead))
			pieces = append(pieces, b.render(n.unit, value)...)
		case Repeat:
			pieces = append(pieces, joint(between), value(inner, n.unit))
		case Scene:
			pieces = append(pieces, joint("-"+between), value(inner, n.unit))
		case Range:
			return append(pieces, joint("-"), value(inner, last.unit))
		case PrefixedRange:
			return append(pieces, joint("-"+between), value(inner, last.unit))
		}
	}
	return pieces
}

// aloneStyle is how an Alone block, which has no letter between its
// numbers, prints each style.
var aloneStyle = map[MultiEpisode]MultiEpisode{
	Extend:        Extend,
	Duplicate:     Duplicate,
	Repeat:        Extend,
	Scene:         Extend,
	Range:         Range,
	PrefixedRange: Range,
}

type numbered struct {
	unit         Unit
	outer, inner int64
}

func gapped(nums []numbered) bool {
	for i := 1; i < len(nums); i++ {
		if nums[i].inner != nums[i-1].inner+1 {
			return true
		}
	}
	return false
}

// refs are b's first Outer ref (nil when it has none), its last Inner or
// its Alone ref, and that ref's index in b.parts.
func (b episodeBlock) refs() (outer, inner *boundRef, at int) {
	for i, p := range b.parts {
		r, ok := p.(*boundRef)
		if !ok {
			continue
		}
		switch r.entry.Seq {
		case SeqOuter:
			if outer == nil {
				outer = r
			}
		case SeqInner, SeqAlone:
			inner, at = r, i
		}
	}
	return outer, inner, at
}

func (b episodeBlock) render(u Unit, value func(*boundRef, Unit) piece) []piece {
	pieces := make([]piece, len(b.parts))
	for i, p := range b.parts {
		switch p := p.(type) {
		case literal:
			pieces[i] = piece{text: string(p), literal: true}
		case *boundRef:
			pieces[i] = value(p, u)
		}
	}
	return pieces
}

func joint(text string) piece { return piece{text: text, literal: true} }

func number(r *boundRef, u Unit) (int64, bool) {
	v := r.entry.read(Subject{Units: []Unit{u}}, 0, FileFacts{})
	return v.n, v.kind == numberValue
}

// multiPart is Sonarr's MultiPartCleanupRegex: a trailing "(1)", "Part 1"
// or "Pt. 1", a colon before it included.
var multiPart = regexp.MustCompile(`(?i):?\s?(?:\(\d+\)|(?:Part|Pt\.?)\s?\d+)$`)

// joinTitles is an Item-scope title over the several Items of one file,
// as Sonarr joins episode titles: each trimmed of trailing " .?", a
// trailing "(n)" or "Part n" stripped, duplicates dropped, joined with
// " + ". clean is nil for a Title form. For a CleanTitle form it cleans
// each title after the strip, which needs the parentheses it removes, and
// the titles join with " and ". When every title strips to nothing they
// join unstripped, so "Part 1" and "Part 2" keep their numbers, and one
// title is never stripped.
func joinTitles(values []Value, clean func(string) string) Value {
	var raw, stripped []string
	for _, v := range values {
		if v.kind != textValue {
			continue
		}
		t := strings.TrimRight(v.text, " .?")
		raw = append(raw, t)
		stripped = append(stripped, strings.TrimSpace(multiPart.ReplaceAllString(t, "")))
	}
	titles := stripped
	if len(raw) == 1 || strings.Join(stripped, "") == "" {
		titles = raw
	}
	titles = unique(titles)
	sep := " + "
	if clean != nil {
		sep = " and "
		for i, t := range titles {
			titles[i] = clean(t)
		}
	}
	return Text(strings.Join(titles, sep))
}

func unique(s []string) []string {
	var out []string
	for _, v := range s {
		if !slices.Contains(out, v) {
			out = append(out, v)
		}
	}
	return out
}
