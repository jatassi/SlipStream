package cardigann

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"sync"
	"unicode"

	"github.com/andybalholm/cascadia"
	"golang.org/x/net/html"
)

// parseHTML parses an HTML response, as AngleSharp's HtmlParser does.
func parseHTML(body string) (*html.Node, error) {
	doc, err := html.Parse(strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("parse the HTML response: %w", err)
	}
	return doc, nil
}

// textContent is the text of n's descendant text nodes, as AngleSharp's
// TextContent.
func textContent(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}

// attr is the value of n's attribute, named in any case.
func attr(n *html.Node, name string) (string, bool) {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, name) {
			return a.Val, true
		}
	}
	return "", false
}

// selector is a compiled CSS selector group with AngleSharp's semantics
// where they differ from cascadia's: :contains is case-sensitive and keeps
// the last word of an unquoted argument, :scope is the element queried,
// and an attribute value may be unquoted.
type selector struct {
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
	m map[string]*selector
}{m: map[string]*selector{}}

// compileSelector compiles css, reusing an earlier compile of the same
// text.
func compileSelector(css string) (*selector, error) {
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
	s = &selector{src: css, m: m, scoped: strings.Contains(css, ":scope")}
	selectorCache.Lock()
	if len(selectorCache.m) > 4096 {
		selectorCache.m = map[string]*selector{}
	}
	selectorCache.m[css] = s
	selectorCache.Unlock()
	return s, nil
}

func (s *selector) String() string { return s.src }

func (s *selector) withScope(n *html.Node, f func()) {
	if !s.scoped || n.Type != html.ElementNode {
		f()
		return
	}
	n.Attr = append(n.Attr, html.Attribute{Key: scopeAttr})
	defer func() { n.Attr = n.Attr[:len(n.Attr)-1] }()
	f()
}

// Match reports whether n itself matches, AngleSharp's Matches.
func (s *selector) Match(n *html.Node) bool {
	var ok bool
	s.withScope(n, func() { ok = s.m.Match(n) })
	return ok
}

// First is the first descendant of n that matches, in document order,
// AngleSharp's QuerySelector.
func (s *selector) First(n *html.Node) *html.Node {
	var out *html.Node
	s.withScope(n, func() { out = cascadia.Query(n, s.m) })
	return out
}

// All is every descendant of n that matches, in document order,
// AngleSharp's QuerySelectorAll.
func (s *selector) All(n *html.Node) []*html.Node {
	var out []*html.Node
	s.withScope(n, func() { out = cascadia.QueryAll(n, s.m) })
	return out
}

// querySelector is CardigannBase.QuerySelector: a leading :root is dropped
// and the rest runs from the top element of n's document.
func querySelector(n *html.Node, css string) (*html.Node, error) {
	if rest, ok := strings.CutPrefix(css, ":root"); ok {
		css = strings.TrimSpace(rest)
		for n.Parent != nil && n.Parent.Type == html.ElementNode {
			n = n.Parent
		}
	}
	s, err := compileSelector(css)
	if err != nil {
		return nil, err
	}
	return s.First(n), nil
}

// Select is HandleSelector (CardigannBase.cs:129-204) over an HTML or XML
// row: text when set, else the element the selector picks (the row itself
// when it matches), with remove, case, attribute and the filters. A
// selector that finds nothing gives Null when required is false.
func (b *selectorBlock) Select(row *html.Node, env env, required bool) (value, error) {
	if b.Text != nil {
		return b.textValue(env)
	}
	selection := row
	if b.Selector != nil {
		css, err := b.Selector.Eval(env.Scope, nil)
		if err != nil {
			return nullValue, err
		}
		if selection, err = pick(row, css); err != nil {
			return nullValue, err
		}
		if selection == nil {
			return missing(required, fmt.Errorf("the selector %q matches nothing", css))
		}
	}
	if err := removeMatches(selection, b.Remove); err != nil {
		return nullValue, err
	}
	value, found, err := b.pickValue(selection, env)
	if err != nil {
		return nullValue, err
	}
	if !found {
		return missing(required, fmt.Errorf("no value for the selector %q", b.Selector.Source()))
	}
	return b.Filters.Apply(textValue(strings.TrimFunc(value, unicode.IsSpace)), env)
}

func (b *selectorBlock) textValue(env env) (value, error) {
	s, err := b.Text.Eval(env.Scope, nil)
	if err != nil {
		return nullValue, err
	}
	return b.Filters.Apply(textValue(s), env)
}

func missing(required bool, err error) (value, error) {
	if required {
		return nullValue, err
	}
	return nullValue, nil
}

// pick is the row itself when it matches css, else the first match below
// it.
func pick(row *html.Node, css string) (*html.Node, error) {
	s, err := compileSelector(css)
	if err != nil {
		return nil, err
	}
	if s.Match(row) {
		return row, nil
	}
	return querySelector(row, css)
}

func removeMatches(n *html.Node, css string) error {
	if css == "" {
		return nil
	}
	s, err := compileSelector(css)
	if err != nil {
		return err
	}
	for _, m := range s.All(n) {
		if m.Parent != nil {
			m.Parent.RemoveChild(m)
		}
	}
	return nil
}

func (b *selectorBlock) pickValue(n *html.Node, env env) (text string, ok bool, err error) {
	switch {
	case b.Case != nil:
		for _, c := range b.Case {
			found, err := pick(n, c.Key)
			if err != nil {
				return "", false, err
			}
			if found != nil {
				v, err := c.Value.Eval(env.Scope, nil)
				return v, true, err
			}
		}
		return "", false, nil
	case b.Attribute != "":
		v, ok := attr(n, b.Attribute)
		return v, ok, nil
	}
	return textContent(n), true, nil
}

// SelectJSON is HandleJsonSelector (CardigannBase.cs:206-275) over a JSON
// object. Leading dots of the selector are dropped; the caller passes the
// parent row when the selector starts with "..".
func (b *selectorBlock) SelectJSON(obj *jsonToken, env env, required bool) (value, error) {
	if b.Text != nil {
		return b.textValue(env)
	}
	value := nullValue
	if b.Selector != nil {
		sel, err := b.Selector.Eval(env.Scope, nil)
		if err != nil {
			return nullValue, err
		}
		sel = sel[len(b.Selector.Source())-len(strings.TrimLeft(b.Selector.Source(), ".")):]
		token, err := selectJSONToken(obj, sel)
		if err != nil {
			return nullValue, err
		}
		if token == nil {
			return missing(required, fmt.Errorf("the selector %q matches nothing", sel))
		}
		if value, err = jsonValue(token, env); err != nil {
			return nullValue, err
		}
	}
	if b.Case != nil {
		var err error
		if value, err = b.jsonCase(value, env); err != nil {
			return nullValue, err
		}
		if value.IsNull() {
			return missing(required, errors.New("no case matches the value"))
		}
	}
	if !value.IsNull() {
		value = textValue(strings.TrimFunc(value.text, unicode.IsSpace))
	}
	return b.Filters.Apply(value, env)
}

func selectJSONToken(obj *jsonToken, sel string) (*jsonToken, error) {
	path, ok, err := obj.fieldSelector(sel)
	if err != nil || !ok {
		return nil, err
	}
	return obj.SelectToken(path)
}

// jsonValue is an array's items joined with commas, else the token's
// Value<string>().
func jsonValue(token *jsonToken, env env) (value, error) {
	if token.kind != jsonArray {
		return token.Text(env.Clock.Location)
	}
	parts := make([]string, len(token.items))
	for i, item := range token.items {
		parts[i] = item.String()
	}
	return textValue(strings.Join(parts, ",")), nil
}

// jsonCase replaces the value with the first case whose key equals it or
// is "*". A value no key matches stays as it is.
func (b *selectorBlock) jsonCase(value value, env env) (value, error) {
	for _, c := range b.Case {
		if (!value.IsNull() && value.text == c.Key) || c.Key == "*" {
			s, err := c.Value.Eval(env.Scope, nil)
			if err != nil {
				return nullValue, err
			}
			return textValue(s), nil
		}
	}
	return value, nil
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

// Match is MatchSelector (CardigannRequestGenerator.cs:992-1027) for the
// download layer: the first element of doc the templated selector picks,
// with real CSS :root, then its attribute or text and the filters. It
// reports false when nothing matches; a missing attribute is an error.
func (f *selectorField) Match(doc *html.Node, env env) (value, bool, error) {
	css, err := f.Selector.Eval(env.Scope, nil)
	if err != nil {
		return nullValue, false, err
	}
	s, err := compileSelector(css)
	if err != nil {
		return nullValue, false, err
	}
	el := s.First(doc)
	if el == nil {
		return nullValue, false, nil
	}
	value := textContent(el)
	if f.Attribute != "" {
		v, ok := attr(el, f.Attribute)
		if !ok {
			return nullValue, false, fmt.Errorf("the attribute %q isn't set", f.Attribute)
		}
		value = v
	}
	out, err := f.Filters.Apply(textValue(value), env)
	return out, true, err
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
