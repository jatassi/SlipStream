package naming

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	prefixChars = " -._[({"
	suffixChars = " -._)]"
	sepChars    = " ._-"
	modChars    = " ,+-"
	// maxWidth is the largest pad or cut: a segment holds 255 bytes.
	maxWidth = 255
)

var tagWords = []struct {
	word string
	tag  tag
}{
	{"edition", editionTag},
	{"imdb", imdbTag},
	{"imdbid", imdbidTag},
	{"tmdb", tmdbTag},
	{"tmdbid", tmdbidTag},
	{"tvdb", tvdbTag},
	{"tvdbid", tvdbidTag},
}

// Parse reads text in Sonarr's token grammar with Radarr's movie
// extensions, knowing no catalog. A token is "{", a prefix from
// " -._[({", a name, an optional ":" modifier, a suffix from " -._)]}"
// and "}". A "{" in the prefix needs a "}" in the suffix, so {{Edition}}
// prints its value in braces. A tag is {edition-{X}}, {imdb-{X}},
// {imdbid-{X}}, {tmdb-{X}}, {tmdbid-{X}}, {tvdb-{X}} or {tvdbid-{X}}.
// There is no brace escape, and "/" and "\" never parse. A refusal is a
// *ParseError.
func Parse(text string) (Template, error) {
	if i := strings.IndexAny(text, `/\`); i >= 0 {
		c := text[i : i+1]
		return Template{}, refuse(i, "", "a template can't contain "+quote(c)+" because it names one folder or file", "Remove the "+quote(c)+".")
	}
	p := parser{src: text}
	start := 0
	for p.i < len(text) {
		switch text[p.i] {
		case '}':
			return Template{}, refuse(p.i, "", `the "}" closes no token`, "Remove it, or make it a token's suffix, as in {{Edition}}.")
		case '{':
			p.literal(start)
			r, err := p.ref()
			if err != nil {
				return Template{}, err
			}
			p.parts = append(p.parts, r)
			start = p.i
		default:
			p.i++
		}
	}
	p.literal(start)
	return Template{text: text, parts: p.parts}, nil
}

// KeyOf is name as matched.
func KeyOf(name string) Key {
	return Key(strings.Map(func(r rune) rune {
		if strings.ContainsRune(sepChars, r) {
			return -1
		}
		return unicode.ToLower(r)
	}, name))
}

// refuse makes a *ParseError whose sentence is cause with its first
// letter raised, then fix.
func refuse(pos int, token, cause, fix string) *ParseError {
	r, n := utf8.DecodeRuneInString(cause)
	return &ParseError{Pos: pos, Token: token, Sentence: string(unicode.ToUpper(r)) + cause[n:] + ". " + fix}
}

type parser struct {
	src   string
	i     int
	parts []part
}

func (p *parser) literal(start int) {
	if start < p.i {
		p.parts = append(p.parts, literal(p.src[start:p.i]))
	}
}

// ref reads the token or tag whose "{" is at p.i.
func (p *parser) ref() (*ref, error) {
	at := p.i
	for _, t := range tagWords {
		if strings.HasPrefix(p.src[at+1:], t.word+"-{") {
			return p.tagged(at, t.tag, at+1+len(t.word)+1)
		}
	}
	return p.token(at)
}

func (p *parser) tagged(at int, t tag, inner int) (*ref, error) {
	for _, w := range tagWords {
		if strings.HasPrefix(p.src[inner+1:], w.word+"-{") {
			text := p.through(at)
			return nil, refuse(at, text, text+" has a tag inside a tag", "A tag holds one token, as in {edition-{Edition}}.")
		}
	}
	r, err := p.token(inner)
	if err != nil {
		return nil, err
	}
	switch {
	case p.i == len(p.src):
		return nil, p.unclosed(at)
	case p.src[p.i] != '}':
		text := p.through(at)
		return nil, refuse(p.i, text, text+" has "+quote(p.char())+" after its inner token", `Close the tag with "}" right after it, as in {edition-{Edition}}.`)
	}
	p.i++
	r.text, r.at, r.tag = p.src[at:p.i], at, t
	return r, nil
}

// token reads "{" prefix name ":" mod suffix "}" from at.
func (p *parser) token(at int) (*ref, error) {
	p.i = at + 1
	prefix := p.run(prefixChars)
	r := &ref{at: at, prefix: prefix}
	if err := p.name(at, r); err != nil {
		return nil, err
	}
	if err := p.modifier(at, r); err != nil {
		return nil, err
	}
	if err := p.suffix(at, r, strings.Count(prefix, "{")); err != nil {
		return nil, err
	}
	r.text = p.src[at:p.i]
	return r, nil
}

func (p *parser) run(chars string) string {
	start := p.i
	for p.i < len(p.src) && strings.IndexByte(chars, p.src[p.i]) >= 0 {
		p.i++
	}
	return p.src[start:p.i]
}

// name reads words of letters and digits joined by one separator each.
// A run of separators that no word follows is the suffix's.
func (p *parser) name(at int, r *ref) error {
	start := p.i
	if p.i == len(p.src) {
		return p.unclosed(at)
	}
	if !isAlnum(p.src[p.i]) {
		text := p.through(at)
		if p.src[p.i] == '}' {
			return refuse(at, text, text+" names no token", "Write a token name between the braces, such as {Target}.")
		}
		return refuse(p.i, text, text+" has "+quote(p.char())+" where its name starts", `Start the name with a letter or digit. Only a space, "-", ".", "_", "[", "(" or "{" can come before it.`)
	}
	for {
		for p.i < len(p.src) && isAlnum(p.src[p.i]) {
			p.i++
		}
		end := p.i
		between := p.run(sepChars)
		if between == "" || p.i == len(p.src) || !isAlnum(p.src[p.i]) {
			p.i = end
			break
		}
		if len(between) > 1 {
			text := p.through(at)
			return refuse(end, text, text+" has "+quote(between)+" between two words of its name", `Separate the words with one space, ".", "-" or "_".`)
		}
		if between[0] != ' ' && r.sep != 0 && r.sep != between[0] {
			text := p.through(at)
			return refuse(end, text, text+" separates the words of its name with both "+quote(string(r.sep))+" and "+quote(between), "Use one of them.")
		}
		if between[0] != ' ' {
			r.sep = between[0]
		}
	}
	r.name = p.src[start:p.i]
	r.key = KeyOf(r.name)
	r.casing = casingOf(r.name)
	return nil
}

// modifier reads ":" and the modifier text, leaving a trailing space or
// "-" to the suffix as Sonarr does.
func (p *parser) modifier(at int, r *ref) error {
	if p.i == len(p.src) || p.src[p.i] != ':' {
		return nil
	}
	p.i++
	start := p.i
	for p.i < len(p.src) && (isAlnum(p.src[p.i]) || strings.IndexByte(modChars, p.src[p.i]) >= 0) {
		p.i++
	}
	for p.i > start && (p.src[p.i-1] == ' ' || p.src[p.i-1] == '-') {
		p.i--
	}
	if start == p.i {
		text := p.through(at)
		return refuse(start-1, text, text+` has nothing after ":"`, `Add a modifier, such as :00 or :30, or remove the ":".`)
	}
	m, cause, fix := modOf(p.src[start:p.i])
	if cause != "" {
		text := p.through(at)
		return refuse(start, text, text+" "+cause, fix)
	}
	r.mod = m
	return nil
}

// modOf reads modifier text: all zeros pad, other digits cut, an
// optional "-" before them cutting from the end, and anything else is
// words. A refusal is the cause's words after the token, and the fix.
func modOf(text string) (m mod, cause, fix string) {
	digits, fromEnd := strings.CutPrefix(text, "-")
	if digits == "" || strings.TrimLeft(digits, "0123456789") != "" {
		return words{text: digits, drop: fromEnd}, "", ""
	}
	n, err := strconv.Atoi(digits)
	switch {
	case n == 0 && !fromEnd && len(digits) > maxWidth:
		return nil, fmt.Sprintf("pads to more than %d digits", maxWidth), fmt.Sprintf("Use %d zeros or fewer.", maxWidth)
	case n == 0 && !fromEnd:
		return padding(len(digits)), "", ""
	case err != nil || n > maxWidth:
		return nil, fmt.Sprintf("cuts to more than %d bytes", maxWidth), fmt.Sprintf("Use a number from 1 to %d, the most a name can hold.", maxWidth)
	case n == 0:
		return nil, "cuts the value to 0 bytes", fmt.Sprintf("Use a number from 1 to %d.", maxWidth)
	}
	return cut{bytes: n, fromEnd: fromEnd}, "", ""
}

// suffix reads the suffix and the closing brace. Each "{" of the prefix
// takes one "}" into the suffix first.
func (p *parser) suffix(at int, r *ref, open int) error {
	start := p.i
	for p.i < len(p.src) {
		switch c := p.src[p.i]; {
		case c == '}' && open == 0:
			r.suffix = p.src[start:p.i]
			p.i++
			return nil
		case c == '}':
			open--
		case c == '{':
			return p.unclosed(at)
		case strings.IndexByte(suffixChars, c) < 0:
			text := p.through(at)
			return refuse(p.i, text, text+" can't contain "+quote(p.char())+" there", `After the name, a token can have a ":" modifier and then only a space, "-", ".", "_", ")", "]" or "}".`)
		}
		p.i++
	}
	return p.unclosed(at)
}

func (p *parser) unclosed(at int) *ParseError {
	text := strings.TrimRight(p.src[at:p.i], " ")
	return refuse(at, text, text+" has no closing brace", `Add "}" to close it.`)
}

// through is the token from at as typed: up to and including the next
// "}" from p.i, or to the end of the template.
func (p *parser) through(at int) string {
	if i := strings.IndexByte(p.src[p.i:], '}'); i >= 0 {
		return p.src[at : p.i+i+1]
	}
	return p.src[at:]
}

// quote quotes s as a sentence does, with no escapes.
func quote(s string) string { return `"` + s + `"` }

// char is the character at p.i, whole.
func (p *parser) char() string {
	_, n := utf8.DecodeRuneInString(p.src[p.i:])
	return p.src[p.i : p.i+n]
}

func isAlnum(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}

// casingOf is Radarr's rule: a name whose letters are all lowercase
// lowers its value, all uppercase uppers it.
func casingOf(name string) casing {
	switch {
	case strings.ToLower(name) == name:
		return lower
	case strings.ToUpper(name) == name:
		return upper
	}
	return asIs
}

// langs reads w as language codes of letters joined by "+". A trailing
// "+" marks that a dropped language prints "--". ok is false for an
// empty code or one with other characters.
func (w words) langs() (langs, bool) {
	text, mark := strings.CutSuffix(w.text, "+")
	codes := strings.Split(text, "+")
	for _, c := range codes {
		if c == "" || strings.TrimFunc(c, func(r rune) bool { return r < utf8.RuneSelf && unicode.IsLetter(r) }) != "" {
			return langs{}, false
		}
	}
	return langs{codes: codes, drop: w.drop, mark: mark}, true
}

// formats reads w as format names joined by ",". ok is false for an empty
// name.
func (w words) formats() (formatFilter, bool) {
	names := strings.Split(w.text, ",")
	for _, n := range names {
		if strings.TrimSpace(n) == "" {
			return formatFilter{}, false
		}
	}
	return formatFilter{names: names, drop: w.drop}, true
}

// one reads w as one format's name. ok is false for a list or a drop.
func (w words) one() (oneFormat, bool) {
	if w.drop || strings.Contains(w.text, ",") {
		return "", false
	}
	return oneFormat(w.text), true
}
