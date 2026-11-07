package cardigann

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/dlclark/regexp2"
)

// value is one template variable: null, a string or a list of strings, the
// three kinds Prowlarr's variables dictionary holds.
type value struct {
	kind valueKind
	text string
	list []string
}

type valueKind int

const (
	nullKind valueKind = iota
	textKind
	listKind
)

// nullValue is the null value. A null renders empty and is false in an if.
var nullValue = value{}

func textValue(s string) value { return value{kind: textKind, text: s} }

func listValue(items []string) value { return value{kind: listKind, list: items} }

func (v value) IsNull() bool { return v.kind == nullKind }

// String is v's text: empty for null and for a list.
func (v value) String() string { return v.text }

// asString is C#'s `value as string`: a list reads as null.
func (v value) asString() (string, bool) {
	return v.text, v.kind == textKind
}

func (v value) blank() bool {
	s, ok := v.asString()
	return !ok || strings.TrimSpace(s) == ""
}

// scope holds the variables a template reads, keyed by their full name as a
// template writes it, such as ".Config.username" or ".Query.Keywords".
type scope map[string]value

func (s scope) lookup(name string) (value, error) {
	v, ok := s[name]
	if !ok {
		return nullValue, fmt.Errorf("template variable %s isn't set", name)
	}
	return v, nil
}

// modifier rewrites each value a template substitutes, as Prowlarr's
// TemplateTextModifier does when it URL-encodes a path.
type modifier func(string) string

// template is a Cardigann template, parsed once. Eval gives Prowlarr's
// result without scanning substituted text again.
type template struct {
	src   string
	nodes []tnode
}

// Source is the template as written.
func (t *template) Source() string {
	if t == nil {
		return ""
	}
	return t.src
}

// Eval renders the template against scope. A variable missing from scope is
// an error, as it is in Prowlarr; a null renders empty. A nil template
// renders empty.
func (t *template) Eval(scope scope, mod modifier) (string, error) {
	if t == nil {
		return "", nil
	}
	var b strings.Builder
	st := evalState{scope: scope, mod: mod}
	if err := st.run(t.nodes, &b); err != nil {
		return "", fmt.Errorf("evaluate %q: %w", t.src, err)
	}
	return b.String(), nil
}

type evalState struct {
	scope scope
	mod   modifier
	elem  *string
	index int
	inRng bool
}

func (st *evalState) modify(s string) string {
	if st.mod == nil {
		return s
	}
	return st.mod(s)
}

func (st *evalState) run(nodes []tnode, b *strings.Builder) error {
	for _, n := range nodes {
		if err := n.eval(st, b); err != nil {
			return err
		}
	}
	return nil
}

type tnode interface {
	eval(st *evalState, b *strings.Builder) error
}

type textNode string

func (n textNode) eval(_ *evalState, b *strings.Builder) error {
	b.WriteString(string(n))
	return nil
}

type substNode struct{ expr expr }

func (n substNode) eval(st *evalState, b *strings.Builder) error {
	v, err := n.expr.value(st.scope)
	if err != nil {
		return err
	}
	if v.kind == listKind {
		return errors.New("a list can't be substituted as text")
	}
	b.WriteString(st.modify(v.text))
	return nil
}

type ifNode struct {
	cond      expr
	then, els []tnode
}

func (n ifNode) eval(st *evalState, b *strings.Builder) error {
	v, err := n.cond.value(st.scope)
	if err != nil {
		return err
	}
	if truthy(v) {
		return st.run(n.then, b)
	}
	return st.run(n.els, b)
}

func truthy(v value) bool {
	switch v.kind {
	case textKind:
		return strings.TrimSpace(v.text) != ""
	case listKind:
		return len(v.list) > 0
	case nullKind:
	}
	return false
}

type rangeNode struct {
	variable string
	body     []tnode
}

func (n rangeNode) eval(st *evalState, b *strings.Builder) error {
	v, err := st.scope.lookup(n.variable)
	if err != nil {
		return err
	}
	if v.kind != listKind {
		return fmt.Errorf("range over %s, which isn't a list", n.variable)
	}
	saved := *st
	defer func() { *st = saved }()
	for i, item := range v.list {
		st.elem, st.index, st.inRng = &item, i, true
		if err := st.run(n.body, b); err != nil {
			return err
		}
	}
	return nil
}

type elemNode struct{}

func (elemNode) eval(st *evalState, b *strings.Builder) error {
	if st.elem == nil {
		return errors.New("{{.}} is outside a range")
	}
	b.WriteString(st.modify(*st.elem))
	return nil
}

type indexNode struct{}

func (indexNode) eval(st *evalState, b *strings.Builder) error {
	if !st.inRng {
		return errors.New("a range index is outside a range")
	}
	b.WriteString(strconv.Itoa(st.index))
	return nil
}

type reReplaceNode struct {
	variable    string
	re          *regex
	replacement string
}

func (n reReplaceNode) eval(st *evalState, b *strings.Builder) error {
	v, err := st.scope.lookup(n.variable)
	if err != nil {
		return err
	}
	if v.kind == listKind {
		return fmt.Errorf("re_replace over %s, which is a list", n.variable)
	}
	out, err := n.re.Replace(v.text, n.replacement)
	if err != nil {
		return err
	}
	b.WriteString(st.modify(out))
	return nil
}

type joinNode struct {
	variable, sep string
}

func (n joinNode) eval(st *evalState, b *strings.Builder) error {
	v, err := st.scope.lookup(n.variable)
	if err != nil {
		return err
	}
	if v.kind != listKind {
		return fmt.Errorf("join over %s, which isn't a list", n.variable)
	}
	b.WriteString(st.modify(strings.Join(v.list, n.sep)))
	return nil
}

type expr interface {
	value(scope scope) (value, error)
}

type varExpr string

func (e varExpr) value(scope scope) (value, error) { return scope.lookup(string(e)) }

type litExpr string

func (e litExpr) value(scope) (value, error) { return textValue(string(e)), nil }

type logicFn int

const (
	fnAnd logicFn = iota
	fnOr
	fnEq
	fnNe
)

var logicFns = map[string]logicFn{"and": fnAnd, "or": fnOr, "eq": fnEq, "ne": fnNe}

type callExpr struct {
	fn   logicFn
	args []expr
}

// value follows CardigannBase.cs:417-486. and returns the first operand
// that is blank, or the last; or returns the first that isn't blank, or the
// last. Both skip string literals. eq and ne compare two operands, a list
// reading as null.
func (e callExpr) value(scope scope) (value, error) {
	switch e.fn {
	case fnAnd, fnOr:
		return e.andOr(scope, e.fn == fnAnd)
	case fnEq, fnNe:
		a, err := e.operand(scope, 0)
		if err != nil {
			return nullValue, err
		}
		c, err := e.operand(scope, 1)
		if err != nil {
			return nullValue, err
		}
		if (a == c) == (e.fn == fnEq) {
			return textValue("True"), nil
		}
		return nullValue, nil
	}
	return nullValue, nil
}

type operand struct {
	text string
	ok   bool
}

func (e callExpr) operand(scope scope, i int) (operand, error) {
	v, err := e.args[i].value(scope)
	if err != nil {
		return operand{}, err
	}
	s, ok := v.asString()
	return operand{text: s, ok: ok}, nil
}

func (e callExpr) andOr(scope scope, isAnd bool) (value, error) {
	var result value
	for _, a := range e.args {
		if _, lit := a.(litExpr); lit {
			continue
		}
		v, err := a.value(scope)
		if err != nil {
			return nullValue, err
		}
		result = v
		if v.blank() == isAnd {
			break
		}
	}
	return result, nil
}

func parseTemplate(src string) (*template, error) {
	p := tparser{src: src}
	nodes, end, err := p.parseNodes(0)
	if err != nil {
		return nil, fmt.Errorf("parse template %q: %w", src, err)
	}
	if end != "" {
		return nil, fmt.Errorf("parse template %q: {{ %s }} has no block to close", src, end)
	}
	return &template{src: src, nodes: nodes}, nil
}

type tparser struct {
	src string
	pos int
}

// parseNodes reads nodes until the input ends or an {{ else }} or {{ end }}
// closes the block, which it returns.
func (p *tparser) parseNodes(rangeDepth int) ([]tnode, string, error) {
	var nodes []tnode
	for p.pos < len(p.src) {
		open := strings.Index(p.src[p.pos:], "{{")
		if open < 0 {
			nodes = append(nodes, textNode(p.src[p.pos:]))
			p.pos = len(p.src)
			break
		}
		if open > 0 {
			nodes = append(nodes, textNode(p.src[p.pos:p.pos+open]))
		}
		p.pos += open
		n, closer, err := p.parseAction(rangeDepth)
		if err != nil {
			return nil, "", err
		}
		if closer != "" {
			return nodes, closer, nil
		}
		nodes = append(nodes, n)
	}
	return nodes, "", nil
}

func (p *tparser) parseAction(rangeDepth int) (tnode, string, error) {
	if n, ok, err := p.parseFixedAction(); ok || err != nil {
		return n, "", err
	}
	end := strings.Index(p.src[p.pos:], "}}")
	if end < 0 {
		return nil, "", errors.New("{{ has no closing }}")
	}
	body := strings.TrimSpace(p.src[p.pos+2 : p.pos+end])
	p.pos += end + 2
	word, rest := splitWord(body)
	switch {
	case body == "else" || body == "end":
		return nil, body, nil
	case word == "if" || strings.HasPrefix(body, "if."):
		return p.parseIf(strings.TrimPrefix(body, "if"), rangeDepth)
	case word == "range":
		return p.parseRange(rest, rangeDepth)
	case body == ".":
		if rangeDepth == 0 {
			return nil, "", errors.New("{{.}} is outside a range")
		}
		return elemNode{}, "", nil
	case strings.HasPrefix(body, "$"):
		if rangeDepth == 0 {
			return nil, "", fmt.Errorf("{{%s}} is outside a range", body)
		}
		return indexNode{}, "", nil
	}
	e, err := parseExpr(body)
	if err != nil {
		return nil, "", err
	}
	return substNode{expr: e}, "", nil
}

// fixedActions are the two functions whose string arguments are raw text
// that may hold "}}", so they are read with Prowlarr's own patterns
// (CardigannBase.cs:371, 396).
var fixedActions = []struct {
	re    *regexp2.Regexp
	build func(m *regexp2.Match) (tnode, error)
}{
	{
		re: regexp2.MustCompile(`\G{{\s*re_replace\s+(\..+?)\s+"(.*?)"\s+"(.*?)"\s*}}`, regexp2.None),
		build: func(m *regexp2.Match) (tnode, error) {
			re, err := compileRegex(m.GroupByNumber(2).String())
			if err != nil {
				return nil, err
			}
			return reReplaceNode{variable: m.GroupByNumber(1).String(), re: re, replacement: m.GroupByNumber(3).String()}, nil
		},
	},
	{
		re: regexp2.MustCompile(`\G{{\s*join\s+(\..+?)\s+"(.*?)"\s*}}`, regexp2.None),
		build: func(m *regexp2.Match) (tnode, error) {
			return joinNode{variable: m.GroupByNumber(1).String(), sep: m.GroupByNumber(2).String()}, nil
		},
	},
}

func (p *tparser) parseFixedAction() (tnode, bool, error) {
	for _, a := range fixedActions {
		m, err := a.re.FindStringMatchStartingAt(p.src, p.pos)
		if err != nil {
			return nil, false, fmt.Errorf("read a template function: %w", err)
		}
		if m == nil {
			continue
		}
		n, err := a.build(m)
		if err != nil {
			return nil, false, err
		}
		p.pos += len(m.String())
		return n, true, nil
	}
	return nil, false, nil
}

func (p *tparser) parseIf(cond string, rangeDepth int) (tnode, string, error) {
	e, err := parseExpr(strings.TrimSpace(cond))
	if err != nil {
		return nil, "", err
	}
	if _, lit := e.(litExpr); lit {
		return nil, "", fmt.Errorf("{{ if%s }} tests a string, not a variable", cond)
	}
	then, closer, err := p.parseNodes(rangeDepth)
	if err != nil {
		return nil, "", err
	}
	n := ifNode{cond: e, then: then}
	if closer == "else" {
		n.els, closer, err = p.parseNodes(rangeDepth)
		if err != nil {
			return nil, "", err
		}
	}
	if closer != "end" {
		return nil, "", errors.New("{{ if }} has no {{ end }}")
	}
	return n, "", nil
}

func (p *tparser) parseRange(head string, rangeDepth int) (tnode, string, error) {
	variable := head
	if strings.HasPrefix(head, "$") {
		_, after, ok := strings.Cut(head, ":=")
		if !ok {
			return nil, "", fmt.Errorf("{{ range %s }} has no :=", head)
		}
		variable = strings.TrimSpace(after)
	}
	if !strings.HasPrefix(variable, ".") || strings.ContainsAny(variable, " \t") {
		return nil, "", fmt.Errorf("{{ range %s }} doesn't name one variable", head)
	}
	body, closer, err := p.parseNodes(rangeDepth + 1)
	if err != nil {
		return nil, "", err
	}
	if closer != "end" {
		return nil, "", errors.New("{{ range }} has no {{ end }}")
	}
	return rangeNode{variable: variable, body: body}, "", nil
}

func splitWord(s string) (word, rest string) {
	i := strings.IndexAny(s, " \t\r\n")
	if i < 0 {
		return s, ""
	}
	return s[:i], strings.TrimSpace(s[i:])
}

// parseExpr reads a variable, a string literal or a logic function call.
// Calls nest in prefix order, as Prowlarr's rewriter reduces them: eq and ne
// take two operands and and/or take the rest.
func parseExpr(src string) (expr, error) {
	toks, err := lexExpr(src)
	if err != nil {
		return nil, err
	}
	ep := exprParser{toks: toks}
	e, err := ep.parse()
	if err != nil {
		return nil, err
	}
	if ep.pos != len(toks) {
		return nil, fmt.Errorf("unexpected %q in {{ %s }}", toks[ep.pos].text, src)
	}
	return e, nil
}

type tokKind int

const (
	tokVar tokKind = iota
	tokLit
	tokWord
	tokOpen
	tokClose
)

type token struct {
	kind tokKind
	text string
}

func lexExpr(src string) ([]token, error) {
	var toks []token
	for i := 0; i < len(src); {
		c := src[i]
		switch c {
		case ' ', '\t', '\r', '\n':
			i++
		case '(':
			toks = append(toks, token{tokOpen, "("})
			i++
		case ')':
			toks = append(toks, token{tokClose, ")"})
			i++
		case '"':
			end := strings.IndexByte(src[i+1:], '"')
			if end < 0 {
				return nil, fmt.Errorf("unterminated string in {{ %s }}", src)
			}
			toks = append(toks, token{tokLit, src[i+1 : i+1+end]})
			i += end + 2
		default:
			j := i
			for j < len(src) && !strings.ContainsRune(" \t\r\n()", rune(src[j])) {
				j++
			}
			kind := tokWord
			if c == '.' {
				kind = tokVar
			}
			toks = append(toks, token{kind, src[i:j]})
			i = j
		}
	}
	if len(toks) == 0 {
		return nil, errors.New("empty {{ }}")
	}
	return toks, nil
}

type exprParser struct {
	toks []token
	pos  int
}

func (p *exprParser) done() bool { return p.pos >= len(p.toks) || p.toks[p.pos].kind == tokClose }

func (p *exprParser) parse() (expr, error) {
	if p.done() {
		return nil, errors.New("a template function is missing an operand")
	}
	t := p.toks[p.pos]
	p.pos++
	switch t.kind {
	case tokVar:
		return varExpr(t.text), nil
	case tokLit:
		return litExpr(t.text), nil
	case tokOpen:
		e, err := p.parse()
		if err != nil {
			return nil, err
		}
		if p.pos >= len(p.toks) || p.toks[p.pos].kind != tokClose {
			return nil, errors.New("( has no closing )")
		}
		p.pos++
		return e, nil
	case tokWord:
		return p.parseCall(t.text)
	case tokClose:
	}
	return nil, fmt.Errorf("unexpected %q", t.text)
}

func (p *exprParser) parseCall(name string) (expr, error) {
	fn, ok := logicFns[name]
	if !ok {
		return nil, fmt.Errorf("%q isn't a template function", name)
	}
	call := callExpr{fn: fn}
	for !p.done() && (fn == fnAnd || fn == fnOr || len(call.args) < 2) {
		a, err := p.parse()
		if err != nil {
			return nil, err
		}
		call.args = append(call.args, a)
	}
	if len(call.args) < 2 {
		return nil, fmt.Errorf("%s takes two operands or more", name)
	}
	if fn == fnAnd || fn == fnOr {
		if !hasVariable(call.args) {
			return nil, fmt.Errorf("%s has only string operands, which it skips", name)
		}
	}
	return call, nil
}

func hasVariable(args []expr) bool {
	for _, a := range args {
		if _, lit := a.(litExpr); !lit {
			return true
		}
	}
	return false
}

// variants renders t once per choice of if branches, at most limit times,
// with every substituted value as placeholder and each range run once. A
// selector template compiles only if each variant does.
func (t *template) variants(placeholder string, limit int) []string {
	return nodeVariants(t.nodes, placeholder, limit)
}

func nodeVariants(nodes []tnode, placeholder string, limit int) []string {
	out := []string{""}
	for _, n := range nodes {
		var alts []string
		switch n := n.(type) {
		case textNode:
			alts = []string{string(n)}
		case ifNode:
			alts = append(nodeVariants(n.then, placeholder, limit), nodeVariants(n.els, placeholder, limit)...)
		case rangeNode:
			alts = nodeVariants(n.body, placeholder, limit)
		default:
			alts = []string{placeholder}
		}
		var next []string
		for _, prefix := range out {
			for _, alt := range alts {
				if len(next) < limit {
					next = append(next, prefix+alt)
				}
			}
		}
		out = next
	}
	return out
}

func (t *template) variables() []string {
	var names []string
	var walk func([]tnode)
	var walkExpr func(expr)
	walkExpr = func(e expr) {
		switch e := e.(type) {
		case varExpr:
			names = append(names, string(e))
		case callExpr:
			for _, a := range e.args {
				walkExpr(a)
			}
		}
	}
	walk = func(nodes []tnode) {
		for _, n := range nodes {
			switch n := n.(type) {
			case substNode:
				walkExpr(n.expr)
			case ifNode:
				walkExpr(n.cond)
				walk(n.then)
				walk(n.els)
			case rangeNode:
				names = append(names, n.variable)
				walk(n.body)
			case reReplaceNode:
				names = append(names, n.variable)
			case joinNode:
				names = append(names, n.variable)
			}
		}
	}
	walk(t.nodes)
	return names
}
