// Ported from Prowlarr's src/NzbDrone.Core/Prowlarr.Core.csproj
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package dom

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/andybalholm/cascadia"
	"golang.org/x/net/html"
)

// Selector is a compiled CSS selector group with AngleSharp's semantics
// where they differ from cascadia's: :contains is case-sensitive and keeps
// the last word of an unquoted argument, :has may start with a
// combinator, :scope is the element queried, and an attribute value may
// be unquoted.
type Selector struct {
	src    string
	m      cascadia.Matcher
	scoped bool
}

const (
	scopeAttr = "data-slipstream-scope"
	hasAttr   = "data-slipstream-has"
)

var selectorCache = struct {
	sync.Mutex
	m map[string]*Selector
}{m: map[string]*Selector{}}

// Compile compiles css, reusing an earlier compile of the same text.
func Compile(css string) (*Selector, error) {
	selectorCache.Lock()
	s, ok := selectorCache.m[css]
	selectorCache.Unlock()
	if ok {
		return s, nil
	}
	rewritten, err := rewriteSelector(css)
	if err != nil {
		return nil, fmt.Errorf("compile the selector %q: %w", css, err)
	}
	m, err := compileGroup(rewritten)
	if err != nil {
		return nil, fmt.Errorf("compile the selector %q: %w", css, err)
	}
	s = &Selector{src: css, m: m, scoped: strings.Contains(css, ":scope")}
	selectorCache.Lock()
	if len(selectorCache.m) > 4096 {
		selectorCache.m = map[string]*Selector{}
	}
	selectorCache.m[css] = s
	selectorCache.Unlock()
	return s, nil
}

// MustCompile is Compile for a selector written in code, which panics
// when css doesn't compile.
func MustCompile(css string) *Selector {
	s, err := Compile(css)
	if err != nil {
		panic(err)
	}
	return s
}

func (s *Selector) String() string { return s.src }

func (s *Selector) withScope(n *html.Node, f func()) {
	if !s.scoped || n.Type != html.ElementNode {
		f()
		return
	}
	n.Attr = append(n.Attr, html.Attribute{Key: scopeAttr})
	defer func() { n.Attr = n.Attr[:len(n.Attr)-1] }()
	f()
}

// Match reports whether n itself matches, AngleSharp's Matches.
func (s *Selector) Match(n *html.Node) bool {
	var ok bool
	s.withScope(n, func() { ok = s.m.Match(n) })
	return ok
}

// First is the first descendant of n that matches, in document order,
// AngleSharp's QuerySelector.
func (s *Selector) First(n *html.Node) *html.Node {
	var out *html.Node
	s.withScope(n, func() { out = cascadia.Query(n, s.m) })
	return out
}

// All is every descendant of n that matches, in document order,
// AngleSharp's QuerySelectorAll.
func (s *Selector) All(n *html.Node) []*html.Node {
	var out []*html.Node
	s.withScope(n, func() { out = cascadia.QueryAll(n, s.m) })
	return out
}

var (
	containsArg   = regexp.MustCompile(`(?i):contains\(`)
	unquotedValue = regexp.MustCompile(`\[\s*([-\w]+)\s*([~|^$*]?=)\s*([^\]"'\s]+)\s*\]`)
)

// rewriteSelector turns AngleSharp's dialect into cascadia's: :contains
// becomes a case-sensitive :matches of the literal text, an unquoted
// attribute value is quoted, and :scope becomes a marker attribute set on
// the queried element.
func rewriteSelector(css string) (string, error) {
	css = unquotedValue.ReplaceAllString(css, `[$1$2"$3"]`)
	css = strings.ReplaceAll(css, ":scope", "["+scopeAttr+"]")
	var b strings.Builder
	for {
		loc := containsArg.FindStringIndex(css)
		if loc == nil {
			b.WriteString(css)
			return typesInCompound(b.String()), nil
		}
		b.WriteString(css[:loc[0]])
		text, rest, err := containsText(css[loc[1]:])
		if err != nil {
			return "", err
		}
		b.WriteString(":matches(" + literalPattern(text) + ")")
		css = rest
	}
}

// containsText reads a :contains argument up to its closing parenthesis:
// a quoted string, or the last word of unquoted text as AngleSharp keeps
// it.
func containsText(s string) (text, rest string, err error) {
	s = strings.TrimLeft(s, " \t")
	if s != "" && (s[0] == '"' || s[0] == '\'') {
		text, n, err := cssString(s)
		if err != nil {
			return "", "", err
		}
		after := strings.TrimLeft(s[n:], " \t")
		if !strings.HasPrefix(after, ")") {
			return "", "", errors.New(":contains has no closing )")
		}
		return text, after[1:], nil
	}
	end := strings.IndexByte(s, ')')
	if end < 0 {
		return "", "", errors.New(":contains has no closing )")
	}
	words := strings.Fields(s[:end])
	if len(words) == 0 {
		return "", s[end+1:], nil
	}
	return words[len(words)-1], s[end+1:], nil
}

// cssString reads a quoted CSS string with its backslash escapes and
// reports how many bytes it read.
func cssString(s string) (text string, n int, err error) {
	quote := s[0]
	var b strings.Builder
	for i := 1; i < len(s); i++ {
		switch c := s[i]; c {
		case quote:
			return b.String(), i + 1, nil
		case '\\':
			if i+1 < len(s) {
				i++
				b.WriteByte(s[i])
			}
		default:
			b.WriteByte(c)
		}
	}
	return "", 0, errors.New("a quoted string has no closing quote")
}

// literalPattern is a regexp matching text literally, with no bare
// parenthesis or bracket, because cascadia finds the end of :matches by
// counting them.
func literalPattern(text string) string {
	return strings.NewReplacer(`\(`, `\x28`, `\)`, `\x29`, `\[`, `\x5b`, `\]`, `\x5d`).Replace(regexp.QuoteMeta(text))
}

// typesInCompound rewrites a type selector that follows a pseudo-class or
// an attribute selector in one compound, as in a:not(.x)a, which AngleSharp
// reads and cascadia rejects, to the equivalent :not(:not(a)).
func typesInCompound(css string) string {
	var b strings.Builder
	var quote byte
	for i := 0; i < len(css); i++ {
		c := css[i]
		b.WriteByte(c)
		switch {
		case c == '\\' && i+1 < len(css):
			i++
			b.WriteByte(css[i])
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case (c == ')' || c == ']') && i+1 < len(css) && isNameStart(css[i+1]):
			end := i + 1
			for end < len(css) && isNameChar(css[end]) {
				end++
			}
			b.WriteString(":not(:not(" + css[i+1:end] + "))")
			i = end - 1
		}
	}
	return b.String()
}

func isNameStart(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '_'
}

func isNameChar(c byte) bool {
	return isNameStart(c) || c >= '0' && c <= '9' || c == '-'
}

// compileGroup parses a selector group. A part that ends in a :has whose
// argument starts with a combinator, as in div:has(~ p), which cascadia
// rejects, matches through relativeHas.
func compileGroup(css string) (cascadia.Matcher, error) {
	parts := splitTopLevel(css)
	if !slices.ContainsFunc(parts, func(p string) bool { _, _, ok := cutRelativeHas(p); return ok }) {
		return parseGroup(css)
	}
	group := make(anyOf, 0, len(parts))
	for _, part := range parts {
		base, rel, ok := cutRelativeHas(part)
		if !ok {
			m, err := parseGroup(part)
			if err != nil {
				return nil, err
			}
			group = append(group, m)
			continue
		}
		b, err := parseGroup(base)
		if err != nil {
			return nil, err
		}
		r, err := parseGroup("[" + hasAttr + "] " + rel)
		if err != nil {
			return nil, err
		}
		group = append(group, relativeHas{base: b, rel: r})
	}
	return group, nil
}

func parseGroup(css string) (cascadia.SelectorGroup, error) {
	g, err := cascadia.ParseGroup(css)
	if err != nil {
		return nil, fmt.Errorf("parse %q: %w", css, err)
	}
	return g, nil
}

// splitTopLevel splits a selector group at the commas outside parentheses,
// brackets and quotes.
func splitTopLevel(css string) []string {
	var parts []string
	depth, start := 0, 0
	var quote byte
	for i := 0; i < len(css); i++ {
		c := css[i]
		switch {
		case c == '\\':
			i++
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '(' || c == '[':
			depth++
		case c == ')' || c == ']':
			depth--
		case c == ',' && depth == 0:
			parts = append(parts, css[start:i])
			start = i + 1
		}
	}
	return append(parts, css[start:])
}

// cutRelativeHas splits a part ending in :has(<combinator> rel) into the
// part before the :has and the combinator with rel.
func cutRelativeHas(part string) (base, rel string, ok bool) {
	part = strings.TrimSpace(part)
	i := strings.LastIndex(part, ":has(")
	if i < 0 || !strings.HasSuffix(part, ")") {
		return "", "", false
	}
	rel = strings.TrimSpace(part[i+len(":has(") : len(part)-1])
	if rel == "" || !strings.ContainsRune(">+~", rune(rel[0])) || len(splitTopLevel(rel)) > 1 {
		return "", "", false
	}
	base = strings.TrimSpace(part[:i])
	if base == "" || strings.ContainsRune(">+~", rune(base[len(base)-1])) || part[i-1] == ' ' {
		base += " *"
	}
	return base, rel, true
}

// relativeHas matches an element that base matches and that has an
// element related to it as rel says, the element itself marked by hasAttr.
type relativeHas struct {
	base, rel cascadia.SelectorGroup
}

func (h relativeHas) Match(n *html.Node) bool {
	if n.Type != html.ElementNode || !h.base.Match(n) {
		return false
	}
	root := n
	if n.Parent != nil {
		root = n.Parent
	}
	n.Attr = append(n.Attr, html.Attribute{Key: hasAttr})
	defer func() { n.Attr = n.Attr[:len(n.Attr)-1] }()
	return cascadia.Query(root, h.rel) != nil
}

type anyOf []cascadia.Matcher

func (a anyOf) Match(n *html.Node) bool {
	return slices.ContainsFunc(a, func(m cascadia.Matcher) bool { return m.Match(n) })
}
