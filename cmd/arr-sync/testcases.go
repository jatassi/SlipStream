package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"
)

// fixture is one upstream test file's [TestCase] and [TestCaseSource]
// rows, by test method, in source order.
type fixture struct {
	Methods []method
}

type method struct {
	Name   string     `json:"name"`
	Params []param    `json:"params"`
	Ignore string     `json:"ignore,omitempty"`
	Cases  []testCase `json:"cases"`
	source string
}

type param struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Default any    `json:"default,omitempty"`
}

type testCase struct {
	Args  []any          `json:"args"`
	Named map[string]any `json:"named,omitempty"`
}

type attribute struct {
	name  string
	args  []any
	named map[string]any
	line  int
}

func readFixture(s *source) (fixture, error) {
	r := &fixtureReader{cursor: &cursor{source: s}, sources: map[string][]testCase{}, memberDepth: -1}
	for r.i < len(r.toks) {
		if err := r.step(); err != nil {
			return r.f, err
		}
	}
	for i, m := range r.f.Methods {
		if m.source == "" {
			continue
		}
		rows, ok := r.sources[m.source]
		if !ok {
			return r.f, s.errorf(0, "%s reads cases from %s, which the file doesn't declare as object[]", m.Name, m.source)
		}
		r.f.Methods[i].Cases = append(r.f.Methods[i].Cases, rows...)
	}
	return r.f, nil
}

// fixtureReader walks a test class: attributes gather until the member
// they mark, and object[] fields become named sources of cases.
type fixtureReader struct {
	*cursor
	f                  fixture
	sources            map[string][]testCase
	pending            []attribute
	depth, memberDepth int
	classNext          bool
}

func (r *fixtureReader) step() error {
	switch {
	case r.is(0, "{"):
		r.depth++
		if r.classNext {
			r.memberDepth, r.classNext = r.depth, false
		}
		r.i++
	case r.is(0, "}"):
		r.depth--
		r.i++
	case r.depth != r.memberDepth:
		r.classNext = r.classNext || r.is(0, "class")
		r.i++
	case r.is(0, "["):
		attrs, err := r.attributes()
		if err != nil {
			return err
		}
		r.pending = append(r.pending, attrs...)
	default:
		line := r.peek(0).line
		m, isMethod, err := r.member(r.sources)
		if err != nil {
			return err
		}
		pending := r.pending
		r.pending = nil
		if !isMethod {
			return nil
		}
		if err := m.attach(r.cursor, pending, line); err != nil {
			return err
		}
		if m.Cases != nil || m.source != "" {
			r.f.Methods = append(r.f.Methods, m)
		}
	}
	return nil
}

func (m *method) attach(c *cursor, attrs []attribute, line int) error {
	for _, a := range attrs {
		switch a.name {
		case "TestCase":
			m.Cases = append(m.Cases, testCase{Args: append([]any{}, a.args...), Named: a.named})
		case "TestCaseSource":
			name, ok := a.args[0].(string)
			if len(a.args) != 1 || !ok {
				return c.errorf(a.line, "TestCaseSource with arguments cmd/arr-sync can't read")
			}
			m.source = name
		case "Ignore":
			m.Ignore = "Ignore"
			if len(a.args) > 0 {
				m.Ignore, _ = a.args[0].(string)
			}
		case "Explicit":
			m.Ignore = "Explicit"
		}
	}
	if m.Cases == nil && m.source == "" {
		return nil
	}
	for _, tc := range m.Cases {
		if len(tc.Args) > len(m.Params) && !m.variadic() {
			return c.errorf(line, "%s has a case with %d arguments for %d parameters", m.Name, len(tc.Args), len(m.Params))
		}
	}
	return nil
}

func (m *method) variadic() bool {
	return len(m.Params) > 0 && strings.HasPrefix(m.Params[len(m.Params)-1].Type, "params ")
}

// attributes reads one [A, B(args), C(x, Name = y)] group.
func (c *cursor) attributes() ([]attribute, error) {
	c.i++
	var attrs []attribute
	err := c.list("]", func() error {
		a := attribute{name: c.peek(0).text, line: c.peek(0).line}
		c.i++
		if c.is(0, "(") {
			c.i++
			if err := c.list(")", func() error {
				if c.peek(0).kind == tIdent && c.is(1, "=") {
					name := c.peek(0).text
					c.i += 2
					v, err := c.value()
					if a.named == nil {
						a.named = map[string]any{}
					}
					a.named[name] = v
					return err
				}
				v, err := c.value()
				a.args = append(a.args, v)
				return err
			}); err != nil {
				return err
			}
		}
		attrs = append(attrs, a)
		return nil
	})
	return attrs, err
}

// member reads one class member. A method returns its name and
// parameters, and an object[] field becomes a named source of cases.
func (c *cursor) member(sources map[string][]testCase) (method, bool, error) {
	start := c.i
	for {
		switch {
		case c.i >= len(c.toks):
			return method{}, false, nil
		case c.is(0, ";"):
			c.i++
			return method{}, false, nil
		case c.is(0, "{"):
			c.skipBalanced()
			return method{}, false, nil
		case c.is(0, "(") && c.i > start && c.peek(-1).kind == tIdent:
			m := method{Name: c.peek(-1).text}
			c.i++
			params, err := c.params()
			if err != nil {
				return m, false, err
			}
			m.Params = params
			c.skipBody()
			return m, true, nil
		case c.is(0, "="):
			name := c.peek(-1).text
			isSource := c.i-start >= 4 && c.is(-4, "object", "[", "]")
			c.i++
			if !isSource {
				c.skipBody()
				return method{}, false, nil
			}
			rows, err := c.sourceRows()
			sources[name] = rows
			return method{}, false, err
		}
		c.i++
	}
}

func (c *cursor) skipBody() {
	for c.i < len(c.toks) {
		switch {
		case c.is(0, "{"):
			c.skipBalanced()
			if !c.is(0, ";") && !c.is(0, ",") && !c.is(0, ")") {
				return
			}
		case c.is(0, "(") || c.is(0, "["):
			c.skipBalanced()
		case c.is(0, ";"):
			c.i++
			return
		default:
			c.i++
		}
	}
}

func (c *cursor) params() ([]param, error) {
	var params []param
	err := c.list(")", func() error {
		var typ []string
		for depth := 0; ; c.i++ {
			t := c.peek(0)
			switch t.text {
			case "<", "[":
				depth++
			case ">", "]":
				depth--
			}
			if depth == 0 && t.kind == tIdent && (c.is(1, ",") || c.is(1, ")") || c.is(1, "=")) {
				break
			}
			if c.i >= len(c.toks) {
				return c.errorf(t.line, "unterminated parameter list")
			}
			typ = append(typ, t.text)
		}
		p := param{Name: c.peek(0).text, Type: joinType(typ)}
		c.i++
		if c.is(0, "=") {
			c.i++
			v, err := c.value()
			if err != nil {
				return err
			}
			p.Default = v
		}
		params = append(params, p)
		return nil
	})
	return params, err
}

func joinType(parts []string) string {
	var b strings.Builder
	for i, p := range parts {
		if i > 0 && isWord(p) && isWord(parts[i-1]) {
			b.WriteByte(' ')
		}
		b.WriteString(p)
	}
	return b.String()
}

func isWord(s string) bool { return s != "" && isAlnum(s[0]) }

// sourceRows reads { new object[] { a, b }, ... } as cases.
func (c *cursor) sourceRows() ([]testCase, error) {
	if err := c.expect("{"); err != nil {
		return nil, err
	}
	var rows []testCase
	err := c.list("}", func() error {
		v, err := c.value()
		if err != nil {
			return err
		}
		args, ok := v.([]any)
		if !ok {
			return c.errorf(c.peek(0).line, "a case source row that isn't new object[] { ... }")
		}
		rows = append(rows, testCase{Args: args})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return rows, c.expect(";")
}

// value reads one C# expression as JSON data: literals as themselves,
// arrays as arrays, Type.Member as {"ref": ...}, x.Call(...) as
// {"call": ..., "on": x, "args": [...]} and new T(...) as {"new": ...}.
func (c *cursor) value() (any, error) {
	v, err := c.primary()
	for err == nil && c.is(0, ".") && c.peek(1).kind == tIdent && c.is(2, "(") {
		call := map[string]any{"call": c.peek(1).text, "on": v}
		c.i += 3
		var args []any
		err = c.list(")", func() error {
			a, err := c.value()
			args = append(args, a)
			return err
		})
		if len(args) > 0 {
			call["args"] = args
		}
		v = call
	}
	return v, err
}

func (c *cursor) primary() (any, error) {
	t := c.peek(0)
	switch {
	case t.kind == tString, t.kind == tChar:
		c.i++
		return t.text, nil
	case t.kind == tNumber:
		c.i++
		return number(c, t, "")
	case c.is(0, "-") && c.peek(1).kind == tNumber:
		c.i += 2
		return number(c, c.peek(-1), "-")
	case c.is(0, "true"), c.is(0, "false"):
		c.i++
		return t.text == "true", nil
	case c.is(0, "null"):
		c.i++
		return nil, nil
	case c.is(0, "nameof", "("):
		name := c.peek(2).text
		c.i += 3
		return name, c.expect(")")
	case c.is(0, "new"):
		c.i++
		return c.construction()
	case t.kind == tIdent:
		name := []string{t.text}
		c.i++
		for c.is(0, ".") && c.peek(1).kind == tIdent && !c.is(2, "(") {
			name = append(name, c.peek(1).text)
			c.i += 2
		}
		return map[string]any{"ref": strings.Join(name, ".")}, nil
	}
	return nil, c.errorf(t.line, "a value written as %q, a form cmd/arr-sync can't read", t.text)
}

func (c *cursor) construction() (any, error) {
	var typ []string
	for !c.is(0, "[") && !c.is(0, "{") && !c.is(0, "(") {
		typ = append(typ, c.peek(0).text)
		c.i++
	}
	switch {
	case c.is(0, "[", "]", "{"):
		c.i += 2
	case c.is(0, "[") && c.peek(1).kind == tNumber && c.is(2, "]"):
		if c.peek(1).text != "0" {
			return nil, c.errorf(c.peek(0).line, "an array sized %s with no values", c.peek(1).text)
		}
		c.i += 3
		return []any{}, nil
	case c.is(0, "("):
		c.i++
		var args []any
		err := c.list(")", func() error {
			a, err := c.value()
			args = append(args, a)
			return err
		})
		return map[string]any{"new": joinType(typ), "args": args}, err
	}
	if err := c.expect("{"); err != nil {
		return nil, err
	}
	values := []any{}
	err := c.list("}", func() error {
		v, err := c.value()
		values = append(values, v)
		return err
	})
	return values, err
}

func number(c *cursor, t token, sign string) (any, error) {
	text := strings.ToLower(t.text)
	if hex, ok := strings.CutPrefix(text, "0x"); ok {
		n, err := strconv.ParseInt(hex, 16, 64)
		if err != nil {
			return nil, c.errorf(t.line, "number %s: %v", t.text, err)
		}
		return json.Number(sign + strconv.FormatInt(n, 10)), nil
	}
	text = strings.TrimRight(text, "mfdlu")
	if n, err := strconv.ParseInt(sign+text, 10, 64); err == nil {
		return json.Number(strconv.FormatInt(n, 10)), nil
	}
	f, err := strconv.ParseFloat(sign+text, 64)
	if err != nil {
		return nil, c.errorf(t.line, "number %s: %v", t.text, err)
	}
	return json.Number(strconv.FormatFloat(f, 'f', -1, 64)), nil
}

// organizerCases writes <out>/<upstream>/<Fixture>.json for each pinned
// naming test file: one case per row of each ported method. It refuses a
// [TestCase] method the pins neither port nor skip, so a resync shows
// every new upstream case. It returns the cases per written file.
func organizerCases(t *tree, p pins, cos map[string]checkout) (map[string]int, error) {
	t.own(p.Organizer.Out)
	counts := map[string]int{}
	for _, of := range p.Organizer.Files {
		co := cos[of.From]
		s, err := co.source(of.File)
		if err != nil {
			return nil, err
		}
		f, err := readFixture(s)
		if err != nil {
			return nil, err
		}
		rows, err := of.cases(f)
		if err != nil {
			return nil, fmt.Errorf("%s %s: %w", co.Name, of.File, err)
		}
		if len(rows) == 0 {
			continue
		}
		written := path.Join(p.Organizer.Out, of.From, stem(of.File)+".json")
		var b bytes.Buffer
		fmt.Fprintf(&b, "{\n  \"upstream\": %s,\n  \"commit\": %s,\n  \"file\": %s,\n  \"cases\": [", quoteJSON(co.Name), quoteJSON(co.Commit), quoteJSON(of.File))
		for i, r := range rows {
			line, err := marshal(r)
			if err != nil {
				return nil, err
			}
			if i > 0 {
				b.WriteString(",")
			}
			b.WriteString("\n    " + line)
		}
		b.WriteString("\n  ]\n}\n")
		t.add(written, b.Bytes())
		counts[written] = len(rows)
	}
	return counts, nil
}

// namingCase is one row of a ported method as a naming case.
type namingCase struct {
	Template string           `json:"template"`
	Values   map[string]any   `json:"values,omitempty"`
	Items    []map[string]any `json:"items,omitempty"`
	Settings map[string]any   `json:"settings,omitempty"`
	Want     string           `json:"want"`
	Source   string           `json:"source"`
}

func (of organizerFile) cases(f fixture) ([]namingCase, error) {
	seen := map[string]bool{}
	var rows []namingCase
	for _, m := range f.Methods {
		seen[m.Name] = true
		om, ported := of.Methods[m.Name]
		if _, skipped := of.Skip[m.Name]; skipped || len(m.Cases) == 0 {
			continue
		}
		if !ported {
			return nil, fmt.Errorf("[TestCase] method %s is neither ported nor skipped in %s", m.Name, pinsFile)
		}
		for i, c := range m.Cases {
			row, err := om.row(m, c)
			if err != nil {
				return nil, fmt.Errorf("%s row %d: %w", m.Name, i+1, err)
			}
			row.Source = fmt.Sprintf("%s %s #%d", path.Base(of.File), m.Name, i+1)
			rows = append(rows, row)
		}
	}
	for name := range of.Methods {
		if !seen[name] {
			return nil, fmt.Errorf("%s ports method %s, which the file doesn't have", pinsFile, name)
		}
	}
	for name := range of.Skip {
		if !seen[name] {
			return nil, fmt.Errorf("%s skips method %s, which the file doesn't have", pinsFile, name)
		}
	}
	return rows, nil
}

func (om organizerMethod) row(m method, c testCase) (namingCase, error) {
	a := args{m: m, c: c}
	row := namingCase{Template: a.text(om.Template), Want: a.text(om.Want), Values: a.set(om.Values), Settings: a.set(om.Settings)}
	for _, item := range om.Items {
		row.Items = append(row.Items, a.set(item))
	}
	return row, a.err
}

// args fills ${name} from one row's arguments, keeping the first error.
type args struct {
	m   method
	c   testCase
	err error
}

var (
	placeholder = regexp.MustCompile(`\$\{(\w+)\}`)
	only        = regexp.MustCompile(`^\$\{(\w+)\}$`)
)

func (a *args) arg(name string) any {
	for i, p := range a.m.Params {
		if p.Name != name {
			continue
		}
		if i < len(a.c.Args) {
			return a.c.Args[i]
		}
		return p.Default
	}
	if a.err == nil {
		a.err = fmt.Errorf("no parameter is named %s", name)
	}
	return nil
}

func (a *args) text(s string) string {
	return placeholder.ReplaceAllStringFunc(s, func(m string) string {
		v := a.value(a.arg(m[2 : len(m)-1]))
		if v == nil {
			return ""
		}
		return fmt.Sprint(v)
	})
}

// value is an argument as a case holds it: an enum member by its name.
func (a *args) value(v any) any {
	if r, ok := v.(map[string]any); ok {
		ref, ok := r["ref"].(string)
		if !ok && a.err == nil {
			a.err = fmt.Errorf("an argument written as %v, which a naming case can't hold", r)
		}
		return ref[strings.LastIndexByte(ref, '.')+1:]
	}
	return v
}

func (a *args) set(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		s, ok := v.(string)
		switch {
		case !ok:
			out[k] = v
		case only.MatchString(s):
			if v := a.value(a.arg(s[2 : len(s)-1])); v != nil {
				out[k] = v
			}
		default:
			out[k] = a.text(s)
		}
	}
	return out
}
