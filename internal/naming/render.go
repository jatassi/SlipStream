package naming

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// style is how values and episode blocks print under a Scheme's
// settings.
type style struct {
	colon  Colon
	custom string
	multi  MultiEpisode
}

// separators are the characters the collapse joins and the segment end
// drops, Sonarr's [- ._].
const separators = " -._"

const ellipsis = "..."

// render renders b for s and f. With several Units each episode block
// renders them in the multi-episode style, an Item-scope title joins
// theirs, and every other Item-scope token reads the first. The caller
// applies segment to the whole segment, a file's extension included, so a
// cut's "..." before the extension survives.
func (st style) render(b bound, s Subject, f FileFacts) string {
	return collapse(st.pieces(b, s, f))
}

func (st style) pieces(b bound, s Subject, f FileFacts) []piece {
	ps := make([]piece, 0, len(b.parts))
	at := 0
	if len(s.Units) > 1 {
		one := func(r *boundRef, u Unit) piece { return st.piece(r, Subject{Work: s.Work, Units: []Unit{u}}, 0, f) }
		for _, sp := range b.blockSpans() {
			ps = st.run(ps, b.parts[at:sp.from], s, f)
			ps = append(ps, multi(blockOf(b.parts, sp), st.multi, s.Units, one)...)
			at = sp.to
		}
	}
	return st.run(ps, b.parts[at:], s, f)
}

// run appends the pieces of parts outside any episode block.
func (st style) run(ps []piece, parts []part, s Subject, f FileFacts) []piece {
	for _, p := range parts {
		switch p := p.(type) {
		case literal:
			ps = append(ps, piece{text: string(p), literal: true})
		case *boundRef:
			ps = append(ps, st.piece(p, s, 0, f))
		}
	}
	return ps
}

// piece is r's value for one Unit of s with its prefix, suffix and tag,
// or an empty piece when the value prints nothing.
func (st style) piece(r *boundRef, s Subject, unit int, f FileFacts) piece {
	v := st.value(r, valueOf(r.entry, s, unit, f), f)
	if strings.TrimSpace(v) == "" {
		return piece{empty: true}
	}
	switch {
	case r.tag != noTag:
		return piece{text: "{" + r.tag.word() + "-" + r.prefix + v + r.suffix + "}"}
	case r.entry.tag != noTag:
		v = "{" + r.entry.tag.word() + "-" + v + "}"
	}
	return piece{text: r.prefix + v + r.suffix}
}

// valueOf is e's value for one Unit of s, or for an Item-scope title over
// several Units, their titles joined.
func valueOf(e *entry, s Subject, unit int, f FileFacts) Value {
	if len(s.Units) < 2 || e.Scope != ItemScope || e.Kind != KindTitle {
		return e.read(s, unit, f)
	}
	base := e
	if e.base != nil {
		base = e.base
	}
	titles := make([]Value, len(s.Units))
	for i := range s.Units {
		titles[i] = base.read(s, i, f)
	}
	if e.form == nil {
		return joinTitles(titles, nil)
	}
	year := e.yearAt(s, 0, f)
	derive := func(t string) string { return e.derive(t, year) }
	if strings.HasPrefix(e.form.suffix, "Clean") {
		return joinTitles(titles, derive)
	}
	joined := joinTitles(titles, nil)
	if joined.kind != textValue {
		return joined
	}
	return Text(derive(joined.text))
}

// value is v through r's pipeline: its kind's form and r's modifier, the
// colon mode, the cross-OS map, r's separator and r's casing.
func (st style) value(r *boundRef, v Value, f FileFacts) string {
	s := form(r, v, f)
	s = st.colons(s)
	s = crossOS(s)
	if r.sep != 0 {
		s = strings.ReplaceAll(s, " ", string(r.sep))
	}
	switch r.casing {
	case lower:
		s = strings.ToLower(s)
	case upper:
		s = strings.ToUpper(s)
	case asIs:
	}
	return s
}

// form is v as text, with r's modifier applied.
func form(r *boundRef, v Value, f FileFacts) string {
	switch v.kind {
	case textValue:
		if c, ok := r.mod.(cut); ok {
			return cutText(v.text, c)
		}
		return v.text
	case numberValue:
		if p, ok := r.mod.(padding); ok {
			return fmt.Sprintf("%0*d", int(p), v.n)
		}
		return strconv.FormatInt(v.n, 10)
	case dateValue:
		return strings.ReplaceAll(v.date.String(), "-", " ")
	case listValue:
		return r.entry.list.print(v.list, r.mod, f)
	}
	return ""
}

// cutText keeps at most c.bytes of s, the ellipsis included, at a whole
// character. Spaces and dots where it cuts go, as Sonarr's Truncate does.
func cutText(s string, c cut) string {
	if len(s) <= c.bytes {
		return s
	}
	room, mark := c.bytes-len(ellipsis), ellipsis
	if room <= 0 {
		room, mark = c.bytes, ""
	}
	if c.fromEnd {
		i := len(s) - room
		for !utf8.RuneStart(s[i]) {
			i++
		}
		return mark + strings.TrimLeft(s[i:], " .")
	}
	i := room
	for !utf8.RuneStart(s[i]) {
		i--
	}
	return strings.TrimRight(s[:i], " .") + mark
}

// print filters items by m and prints what is left.
func (l listForm) print(items []string, m mod, f FileFacts) string {
	switch l {
	case nameList:
		if ff, ok := m.(formatFilter); ok {
			items = slices.DeleteFunc(slices.Clone(items), func(s string) bool { return slices.Contains(ff.names, s) == ff.drop })
		}
		return strings.Join(items, " ")
	case pickList:
		if one, ok := m.(oneFormat); ok && slices.Contains(items, string(one)) {
			return string(one)
		}
		return ""
	case langList, audioLangList:
	}
	kept := items
	if lf, ok := m.(langs); ok {
		kept = lf.filter(items)
	}
	if original, _ := f.Facts.Coverage.Original.Get(); l == audioLangList && len(kept) == 1 && strings.EqualFold(kept[0], string(original)) {
		return ""
	}
	if len(kept) == 0 {
		return ""
	}
	return "[" + strings.Join(kept, "+") + "]"
}

// filter is Sonarr's language filter: the kept codes in the modifier's
// order, or the codes not dropped in the list's, then "--" when a
// trailing "+" asked to mark that one was left out.
func (l langs) filter(codes []string) []string {
	has := func(list []string, c string) bool {
		return slices.ContainsFunc(list, func(x string) bool { return strings.EqualFold(x, c) })
	}
	var out []string
	if l.drop {
		for _, c := range codes {
			if !has(l.codes, c) {
				out = append(out, c)
			}
		}
	} else {
		for _, c := range l.codes {
			if has(codes, c) {
				out = append(out, strings.ToUpper(c))
			}
		}
	}
	if l.mark && len(out) != len(codes) {
		out = append(out, "--")
	}
	return out
}

// colons prints each ":" of s by the colon mode. A space the replacement
// starts or ends with joins a space beside the colon, so no mode doubles a
// space.
func (st style) colons(s string) string {
	if !strings.Contains(s, ":") {
		return s
	}
	var b strings.Builder
	last := byte(0)
	for i := 0; i < len(s); i++ {
		if s[i] != ':' {
			b.WriteByte(s[i])
			last = s[i]
			continue
		}
		spaced := i+1 < len(s) && s[i+1] == ' '
		rep := st.replacement(spaced)
		if last == ' ' {
			rep = strings.TrimPrefix(rep, " ")
		}
		b.WriteString(rep)
		if rep != "" {
			last = rep[len(rep)-1]
		}
		if spaced && last == ' ' {
			i++
		}
	}
	return b.String()
}

// replacement is what one colon prints. Smart prints ": " as " - " and
// any other ":" as "-".
func (st style) replacement(spaced bool) string {
	switch st.colon {
	case ColonDelete:
		return ""
	case ColonDash:
		return "-"
	case ColonSpaceDash:
		return " -"
	case ColonSpaceDashSpace:
		return " - "
	case ColonSmart:
		if spaced {
			return " - "
		}
		return "-"
	case ColonCustom:
		return st.custom
	}
	panic(fmt.Sprintf("naming: colon mode %d", st.colon))
}

// crossOSChars is Sonarr's map for the characters a name can't hold on
// Windows, macOS or Linux.
var crossOSChars = strings.NewReplacer(`\`, "+", "/", "+", "*", "-", "?", "!", "<", "", ">", "", "|", "", `"`, "")

func crossOS(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, crossOSChars.Replace(s))
}

// collapse joins pieces into one segment. Where empty pieces leave two
// literals meeting, the separator run ending the left one and the run
// starting the right one become the longer of the two, the left on a tie,
// and the separators ending the last literal go. Nothing inside a value
// changes.
func collapse(ps []piece) string {
	for i := 0; i < len(ps); {
		if !ps[i].empty {
			i++
			continue
		}
		j := i
		for j < len(ps) && ps[j].empty {
			j++
		}
		if i > 0 && j < len(ps) && ps[i-1].literal && ps[j].literal && meet(&ps[i-1], &ps[j]) {
			continue
		}
		i = j
	}
	trimEnd(ps)
	var b strings.Builder
	for _, p := range ps {
		b.WriteString(p.text)
	}
	return b.String()
}

// meet keeps the longer of the separator runs where l and r meet. It
// reports that r is left empty, so the gap reaches the piece after it.
func meet(l, r *piece) bool {
	lrun := len(l.text) - len(strings.TrimRight(l.text, separators))
	rrun := len(r.text) - len(strings.TrimLeft(r.text, separators))
	if lrun >= rrun {
		r.text = r.text[rrun:]
	} else {
		l.text = l.text[:len(l.text)-lrun]
	}
	r.empty = r.text == ""
	return r.empty
}

// trimEnd drops the separators ending the last literals.
func trimEnd(ps []piece) {
	for k := len(ps) - 1; k >= 0; k-- {
		if ps[k].empty {
			continue
		}
		if !ps[k].literal {
			return
		}
		if ps[k].text = strings.TrimRight(ps[k].text, separators); ps[k].text != "" {
			return
		}
	}
}

// segment is s as a path segment: leading spaces and trailing spaces and
// dots go, and a Windows device name gets "_" before it.
func segment(s string) string {
	s = strings.TrimRight(strings.TrimLeft(s, " "), " .")
	if reserved(s) {
		s = "_" + s
	}
	return s
}

var devices = regexp.MustCompile(`(?i)^(CON|PRN|AUX|NUL|COM[1-9]|LPT[1-9])$`)

// reserved holds whatever follows the device name, as Windows does:
// NUL.txt and NUL .txt open the device too.
func reserved(s string) bool {
	base, _, _ := strings.Cut(s, ".")
	return devices.MatchString(strings.TrimRight(base, " "))
}

func (t tag) word() string {
	for _, w := range tagWords {
		if w.tag == t {
			return w.word
		}
	}
	panic(fmt.Sprintf("naming: tag %d has no word", t))
}
