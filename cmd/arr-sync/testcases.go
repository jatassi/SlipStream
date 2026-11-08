package main

import (
	"encoding/json"
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
