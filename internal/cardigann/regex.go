package cardigann

import (
	"fmt"
	"strings"
	"time"

	"github.com/dlclark/regexp2"
)

// regexTimeout bounds one match, so a pattern that backtracks without end
// fails its field instead of holding the search.
const regexTimeout = time.Second

// regex is a .NET regular expression run by regexp2, with .NET's
// replacement syntax.
type regex struct {
	re *regexp2.Regexp
}

// compileRegex compiles a .NET pattern. regexp2 lacks .NET's \p{IsBlock}
// classes, so each is rewritten to its code point range first.
func compileRegex(pattern string) (*regex, error) {
	rewritten, err := rewriteBlocks(pattern)
	if err != nil {
		return nil, err
	}
	re, err := regexp2.Compile(rewritten, regexp2.None)
	if err != nil {
		return nil, fmt.Errorf("compile regex %q: %w", pattern, err)
	}
	re.MatchTimeout = regexTimeout
	return &regex{re: re}, nil
}

// Group1 is the first group of the first match, empty when nothing
// matches: .NET's Match(s).Groups[1].Value.
func (r *regex) Group1(s string) (string, error) {
	m, err := r.re.FindStringMatch(s)
	if err != nil {
		return "", fmt.Errorf("match %q: %w", r.re.String(), err)
	}
	if m == nil {
		return "", nil
	}
	g := m.GroupByNumber(1)
	if g == nil {
		return "", nil
	}
	return g.String(), nil
}

// Replace replaces every match, reading $1, ${name} and $$ in replacement
// as .NET does.
func (r *regex) Replace(s, replacement string) (string, error) {
	out, err := r.re.Replace(s, replacement, -1, -1)
	if err != nil {
		return "", fmt.Errorf("replace %q: %w", r.re.String(), err)
	}
	return out, nil
}

func (r *regex) MatchString(s string) (bool, error) {
	ok, err := r.re.MatchString(s)
	if err != nil {
		return false, fmt.Errorf("match %q: %w", r.re.String(), err)
	}
	return ok, nil
}

// blocks are .NET's named Unicode blocks, as \p{IsName} spells them.
var blocks = map[string][2]rune{
	"BasicLatin":                     {0x0000, 0x007F},
	"Latin-1Supplement":              {0x0080, 0x00FF},
	"LatinExtended-A":                {0x0100, 0x017F},
	"LatinExtended-B":                {0x0180, 0x024F},
	"IPAExtensions":                  {0x0250, 0x02AF},
	"Greek":                          {0x0370, 0x03FF},
	"GreekandCoptic":                 {0x0370, 0x03FF},
	"Cyrillic":                       {0x0400, 0x04FF},
	"CyrillicSupplement":             {0x0500, 0x052F},
	"Armenian":                       {0x0530, 0x058F},
	"Hebrew":                         {0x0590, 0x05FF},
	"Arabic":                         {0x0600, 0x06FF},
	"Devanagari":                     {0x0900, 0x097F},
	"Thai":                           {0x0E00, 0x0E7F},
	"Georgian":                       {0x10A0, 0x10FF},
	"HangulJamo":                     {0x1100, 0x11FF},
	"GeneralPunctuation":             {0x2000, 0x206F},
	"CJKSymbolsandPunctuation":       {0x3000, 0x303F},
	"Hiragana":                       {0x3040, 0x309F},
	"Katakana":                       {0x30A0, 0x30FF},
	"CJKUnifiedIdeographsExtensionA": {0x3400, 0x4DBF},
	"CJKUnifiedIdeographs":           {0x4E00, 0x9FFF},
	"HangulSyllables":                {0xAC00, 0xD7AF},
	"CJKCompatibilityIdeographs":     {0xF900, 0xFAFF},
	"HalfwidthandFullwidthForms":     {0xFF00, 0xFFEF},
}

func rewriteBlocks(pattern string) (string, error) {
	if !strings.Contains(pattern, "{Is") {
		return pattern, nil
	}
	var b strings.Builder
	inClass := false
	for i := 0; i < len(pattern); i++ {
		c := pattern[i]
		switch {
		case c == '\\' && i+1 < len(pattern):
			n, err := writeEscape(&b, pattern[i:], inClass)
			if err != nil {
				return "", fmt.Errorf("compile regex %q: %w", pattern, err)
			}
			i += n - 1
		case c == '[' && !inClass:
			inClass = true
			b.WriteByte(c)
			if strings.HasPrefix(pattern[i+1:], "^") {
				b.WriteByte('^')
				i++
			}
			if strings.HasPrefix(pattern[i+1:], "]") {
				b.WriteByte(']')
				i++
			}
		case c == ']' && inClass:
			inClass = false
			b.WriteByte(c)
		default:
			b.WriteByte(c)
		}
	}
	return b.String(), nil
}

// writeEscape copies the escape that starts s, rewriting \p{IsName} and
// \P{IsName}, and reports how many bytes it read.
func writeEscape(b *strings.Builder, s string, inClass bool) (int, error) {
	if len(s) < 5 || (s[1] != 'p' && s[1] != 'P') || !strings.HasPrefix(s[2:], "{Is") {
		b.WriteString(s[:2])
		return 2, nil
	}
	end := strings.IndexByte(s, '}')
	if end < 0 {
		b.WriteString(s[:2])
		return 2, nil
	}
	name := s[5:end]
	r, ok := blocks[name]
	if !ok {
		return 0, fmt.Errorf("the Unicode block %s isn't known", name)
	}
	span := fmt.Sprintf(`\u%04X-\u%04X`, r[0], r[1])
	negate := s[1] == 'P'
	switch {
	case inClass && negate:
		return 0, fmt.Errorf("\\P{Is%s} inside a character class can't be rewritten", name)
	case inClass:
		b.WriteString(span)
	case negate:
		b.WriteString("[^" + span + "]")
	default:
		b.WriteString("[" + span + "]")
	}
	return end + 1, nil
}
