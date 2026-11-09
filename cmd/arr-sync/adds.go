package main

// addsEntry is one if-statement of an upstream method that adds the enum
// member value when any of words holds.
type addsEntry struct {
	words []string
	value string
}

// addsKinds are the tests an adds method's if-statements make, by the
// member the test calls: text.Contains("word") and match.Groups["word"].
var addsKinds = []struct{ call, open, close string }{
	{"Contains", "(", ")"},
	{"Groups", "[", "]"},
}

// readAdds reads each "if (test || test) { list.Add(Enum.Member); }" of
// method, as one table per kind of test, named Method.Kind. An Add with no
// test before it, such as a fallback, isn't read.
func readAdds(s *source, method string) ([]regexDecl, error) {
	body, err := methodBody(s, method)
	if err != nil {
		return nil, err
	}
	tables := make([][]addsEntry, len(addsKinds))
	kind, words := -1, []string(nil)
	for ; body.i < len(body.toks); body.i++ {
		if k, word, ok := addsTest(body); ok {
			if kind >= 0 && kind != k {
				return nil, s.errorf(body.peek(0).line, "an if-statement in %s mixes %s and %s", method, addsKinds[kind].call, addsKinds[k].call)
			}
			kind, words = k, append(words, word)
		}
		if body.is(0, "if") && kind >= 0 {
			return nil, s.errorf(body.peek(0).line, "a test in %s adds nothing", method)
		}
		if member, ok := added(body); ok {
			if kind >= 0 {
				tables[kind] = append(tables[kind], addsEntry{words: words, value: member})
			}
			kind, words = -1, nil
		}
	}
	var decls []regexDecl
	for k, entries := range tables {
		if len(entries) > 0 {
			decls = append(decls, regexDecl{name: method + "." + addsKinds[k].call, adds: entries})
		}
	}
	if len(decls) == 0 {
		return nil, s.errorf(0, "method %s adds nothing under a test", method)
	}
	return decls, nil
}

// methodBody is a cursor over the tokens of method's body, braces included.
func methodBody(s *source, method string) (*cursor, error) {
	c := &cursor{source: s}
	for c.i < len(c.toks) && !declares(c, method) {
		c.i++
	}
	if c.i >= len(c.toks) {
		return nil, s.errorf(0, "no method %s", method)
	}
	c.i++
	c.skipBalanced()
	if !c.is(0, "{") {
		return nil, s.errorf(c.peek(0).line, "method %s has no body", method)
	}
	start := c.i
	c.skipBalanced()
	return &cursor{source: &source{file: s.file, toks: s.toks[start:c.i]}}, nil
}

// added is the member of "Add(Enum.Member)" when c is at one.
func added(c *cursor) (string, bool) {
	ok := c.is(0, "Add", "(") && c.peek(2).kind == tIdent && c.is(3, ".") && c.peek(4).kind == tIdent && c.is(5, ")")
	return c.peek(4).text, ok
}

// addsTest is the kind and the word of the test c is at.
func addsTest(c *cursor) (kind int, word string, ok bool) {
	for k, ak := range addsKinds {
		if c.is(0, ak.call, ak.open) && c.peek(2).kind == tString && c.is(3, ak.close) {
			return k, c.peek(2).text, true
		}
	}
	return 0, "", false
}

// declares is whether c is at the declaration of method: its name after
// a return type, which ends in a name or a generic's closing bracket.
func declares(c *cursor, method string) bool {
	if c.i == 0 || c.peek(0).kind != tIdent || c.peek(0).text != method || !c.is(1, "(") {
		return false
	}
	before := c.toks[c.i-1]
	return before.kind == tIdent || before.text == ">"
}
