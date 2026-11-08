package main

import (
	"slices"
	"strings"
)

// regexDecl is one Regex or RegexReplace declaration, or an array of
// them: a field, or a local named Method.local.
type regexDecl struct {
	name    string
	replace bool
	array   bool
	items   []regexItem
	// list, when set, makes the declaration a []string table of a
	// field's string literals instead of a pattern.
	list []string
}

type regexItem struct {
	text    string
	options []string
	with    string
}

var staticRegexCalls = []string{"Match", "IsMatch", "Matches", "Replace", "Split"}

// readRegexes reads every regular expression s declares, in order. A
// pattern written inline in a call fails unless its method is skipped.
func readRegexes(s *source, skipMethods []string) ([]regexDecl, error) {
	c := &cursor{source: s}
	r := regexReader{cursor: c, values: map[string]string{}, memberDepth: -1}
	var decls []regexDecl
	for c.i < len(c.toks) {
		d, ok, err := r.step(skipMethods)
		if err != nil {
			return nil, err
		}
		if ok {
			decls = append(decls, d)
		}
	}
	return decls, nil
}

type regexReader struct {
	*cursor
	values      map[string]string
	depth       int
	memberDepth int
	classNext   bool
	lastCall    string
	method      string
}

func (r *regexReader) step(skipMethods []string) (regexDecl, bool, error) {
	t := r.peek(0)
	switch {
	case t.kind == tPunct && t.text == "{":
		r.depth++
		if r.classNext {
			r.memberDepth, r.classNext = r.depth, false
		} else if r.depth == r.memberDepth+1 {
			r.method = r.lastCall
		}
	case t.kind == tPunct && t.text == "}":
		if r.depth == r.memberDepth+1 {
			r.method = ""
		}
		r.depth--
	case r.depth == r.memberDepth && r.is(0, "=>"):
		r.method = r.lastCall
	case r.depth == r.memberDepth && r.is(0, ";"):
		r.method = ""
	case t.kind != tIdent:
	case t.text == "class":
		r.classNext = true
	case r.depth == r.memberDepth && r.is(1, "("):
		r.lastCall = t.text
	case t.text == "new" && (r.is(1, "Regex") || r.is(1, "RegexReplace")):
		return regexDecl{}, false, r.errorf(t.line, "a regular expression outside a field or local declaration")
	case t.text == "Regex" && r.is(1, ".") && slices.Contains(staticRegexCalls, r.peek(2).text) && !slices.Contains(skipMethods, r.method):
		return regexDecl{}, false, r.errorf(t.line, "Regex.%s with an inline pattern in %s. List the method under skip in pins.json", r.peek(2).text, r.method)
	default:
		return r.declaration()
	}
	r.i++
	return regexDecl{}, false, nil
}

// declaration reads "Type Name = init;" where Type is Regex, RegexReplace,
// their arrays, var holding one of them, or string.
func (r *regexReader) declaration() (regexDecl, bool, error) {
	typ := r.peek(0).text
	array := r.is(1, "[", "]")
	at := 1
	if array {
		at = 3
	}
	nameTok := r.peek(at)
	if nameTok.kind != tIdent || !r.is(at+1, "=") {
		r.i++
		return regexDecl{}, false, nil
	}
	isRegex := typ == "Regex" || typ == "RegexReplace"
	isVar := typ == "var" && r.is(at+2, "new") && (r.is(at+3, "Regex") || r.is(at+3, "RegexReplace"))
	if typ == "string" && !array {
		start := r.i
		r.i += at + 2
		if v, err := r.stringExpr(";"); err == nil {
			r.values[nameTok.text] = v
		}
		r.i = start + 1
		return regexDecl{}, false, nil
	}
	if !isRegex && !isVar {
		r.i++
		return regexDecl{}, false, nil
	}
	if isVar {
		typ = r.peek(at + 3).text
	}
	name := nameTok.text
	if r.method != "" {
		name = r.method + "." + name
	}
	r.i += at + 2
	d := regexDecl{name: name, replace: typ == "RegexReplace", array: array}
	var err error
	if array {
		d.items, err = r.arrayInit()
	} else {
		var it regexItem
		it, err = r.construct(d.replace)
		d.items = []regexItem{it}
		r.values[nameTok.text] = it.text
	}
	if err != nil {
		return regexDecl{}, false, err
	}
	return d, true, r.expect(";")
}

func (r *regexReader) arrayInit() ([]regexItem, error) {
	replace := false
	closer := "}"
	switch {
	case r.is(0, "System", ".", "Array", ".", "Empty"), r.is(0, "Array", ".", "Empty"):
		for !r.is(0, ";") {
			r.i++
		}
		return nil, nil
	case r.is(0, "new", "[", "]", "{"):
		r.i += 4
	case r.is(0, "new", "Regex", "[", "]", "{"), r.is(0, "new", "RegexReplace", "[", "]", "{"):
		replace = r.peek(1).text == "RegexReplace"
		r.i += 5
	case r.is(0, "{"):
		r.i++
	case r.is(0, "["):
		closer = "]"
		r.i++
	default:
		return nil, r.errorf(r.peek(0).line, "an array of regular expressions in a form cmd/arr-sync can't read")
	}
	var items []regexItem
	err := r.list(closer, func() error {
		it, err := r.construct(replace || r.is(1, "RegexReplace"))
		items = append(items, it)
		return err
	})
	return items, err
}

// construct reads new Regex(pattern[, options]), new RegexReplace(pattern,
// replacement, options), or the target-typed new(...).
func (r *regexReader) construct(replace bool) (regexItem, error) {
	line := r.peek(0).line
	if err := r.expect("new"); err != nil {
		return regexItem{}, err
	}
	if r.is(0, "Regex") || r.is(0, "RegexReplace") {
		replace = r.peek(0).text == "RegexReplace"
		r.i++
	}
	if err := r.expect("("); err != nil {
		return regexItem{}, err
	}
	var it regexItem
	var err error
	if it.text, err = r.stringExpr(",", ")"); err != nil {
		return it, err
	}
	if replace {
		if err := r.expect(","); err != nil {
			return it, err
		}
		if it.with, err = r.stringExpr(","); err != nil {
			return it, err
		}
	}
	if r.is(0, ",") {
		r.i++
		if it.options, err = r.options(); err != nil {
			return it, err
		}
	}
	if !r.is(0, ")") {
		return it, r.errorf(line, "a regular expression with arguments cmd/arr-sync can't read")
	}
	r.i++
	return it, nil
}

// stringExpr evaluates string terms joined by + up to one of stops at
// depth 0: literals, interpolations of known names, string.Empty and names
// of strings or regular expressions declared earlier.
func (r *regexReader) stringExpr(stops ...string) (string, error) {
	var b strings.Builder
	for {
		if err := r.term(&b); err != nil {
			return "", err
		}
		if r.is(0, "+") {
			r.i++
			continue
		}
		if slices.ContainsFunc(stops, func(s string) bool { return r.is(0, s) }) {
			return b.String(), nil
		}
		return "", r.errorf(r.peek(0).line, "expected the end of a pattern, found %q", r.peek(0).text)
	}
}

func (r *regexReader) term(b *strings.Builder) error {
	t := r.peek(0)
	switch {
	case t.kind == tString:
		b.WriteString(t.text)
	case t.kind == tInterp:
		for _, p := range t.holes {
			if !p.hole {
				b.WriteString(p.text)
				continue
			}
			v, ok := r.values[strings.TrimSuffix(p.text, ".ToString()")]
			if !ok {
				return r.errorf(t.line, "an interpolation of %q, which the file doesn't declare earlier", p.text)
			}
			b.WriteString(v)
		}
	case r.is(0, "string", ".", "Empty"):
		r.i += 2
	case t.kind == tIdent:
		v, ok := r.values[t.text]
		if !ok {
			return r.errorf(t.line, "a reference to %q, which the file doesn't declare earlier", t.text)
		}
		b.WriteString(v)
		if r.is(1, ".", "ToString", "(", ")") {
			r.i += 4
		}
	default:
		return r.errorf(t.line, "a pattern written as %q, a form cmd/arr-sync can't read", t.text)
	}
	r.i++
	return nil
}

func (r *regexReader) options() ([]string, error) {
	var names []string
	for {
		if !r.is(0, "RegexOptions", ".") || r.peek(2).kind != tIdent {
			return nil, r.errorf(r.peek(0).line, "regular expression options written as %q", r.peek(0).text)
		}
		name := r.peek(2).text
		if _, ok := netOptions[name]; !ok {
			return nil, r.errorf(r.peek(0).line, "unknown option RegexOptions.%s", name)
		}
		if name != "None" {
			names = append(names, name)
		}
		r.i += 3
		if !r.is(0, "|") {
			return names, nil
		}
		r.i++
	}
}

var netOptions = map[string]uint32{
	"None":                    0,
	"IgnoreCase":              1,
	"Multiline":               2,
	"ExplicitCapture":         4,
	"Compiled":                8,
	"Singleline":              16,
	"IgnorePatternWhitespace": 32,
	"RightToLeft":             64,
	"ECMAScript":              256,
	"CultureInvariant":        512,
	"NonBacktracking":         1024,
}
