package cardigann

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	"golang.org/x/net/html"

	"github.com/jatassi/slipstream/internal/prowlarr/dom"
)

// querySelector is CardigannBase.QuerySelector: a leading :root is dropped
// and the rest runs from the top element of n's document.
func querySelector(n *html.Node, css string) (*html.Node, error) {
	if rest, ok := strings.CutPrefix(css, ":root"); ok {
		css = strings.TrimSpace(rest)
		for n.Parent != nil && n.Parent.Type == html.ElementNode {
			n = n.Parent
		}
	}
	s, err := dom.Compile(css)
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
	s, err := dom.Compile(css)
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
	s, err := dom.Compile(css)
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
		v, ok := dom.Attr(n, b.Attribute)
		return v, ok, nil
	}
	return dom.TextContent(n), true, nil
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

// Match is MatchSelector (CardigannRequestGenerator.cs:992-1027) for the
// download layer: the first element of doc the templated selector picks,
// with real CSS :root, then its attribute or text and the filters. It
// reports false when nothing matches; a missing attribute is an error.
func (f *selectorField) Match(doc *html.Node, env env) (value, bool, error) {
	css, err := f.Selector.Eval(env.Scope, nil)
	if err != nil {
		return nullValue, false, err
	}
	s, err := dom.Compile(css)
	if err != nil {
		return nullValue, false, err
	}
	el := s.First(doc)
	if el == nil {
		return nullValue, false, nil
	}
	value := dom.TextContent(el)
	if f.Attribute != "" {
		v, ok := dom.Attr(el, f.Attribute)
		if !ok {
			return nullValue, false, fmt.Errorf("the attribute %q isn't set", f.Attribute)
		}
		value = v
	}
	out, err := f.Filters.Apply(textValue(value), env)
	return out, true, err
}
