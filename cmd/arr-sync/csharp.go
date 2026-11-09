package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// These readers cover the C# subset that upstream's parser, enum and test
// files declare regular expressions, enums, languages and [TestCase] rows
// in. Anything else inside those fails with its file and line, so nothing
// is copied by hand.

type tokKind int

const (
	tIdent tokKind = iota
	tString
	tInterp
	tNumber
	tChar
	tPunct
)

type token struct {
	kind  tokKind
	text  string // the decoded value of a string, the source text otherwise
	line  int
	holes []piece // an interpolated string's pieces
}

// piece is a run of literal text, or a {hole} naming an expression.
type piece struct {
	text string
	hole bool
}

type source struct {
	file string
	toks []token
}

func (s *source) errorf(line int, format string, args ...any) error {
	return fmt.Errorf("%s:%d: %s", s.file, line, fmt.Sprintf(format, args...))
}

func lex(file, src string) (*source, error) {
	l := lexer{src: strings.TrimPrefix(src, "\ufeff"), line: 1}
	s := &source{file: file}
	for {
		t, ok, err := l.next()
		if err != nil {
			return nil, fmt.Errorf("%s:%d: %w", file, l.line, err)
		}
		if !ok {
			return s, nil
		}
		s.toks = append(s.toks, t)
	}
}

type lexer struct {
	src  string
	i    int
	line int
}

func (l *lexer) next() (token, bool, error) {
	l.skipSpace()
	if l.i >= len(l.src) {
		return token{}, false, nil
	}
	rest := l.src[l.i:]
	line := l.line
	switch {
	case strings.HasPrefix(rest, `"""`):
		return token{}, false, errors.New("a raw string literal, which arr-sync doesn't read")
	case strings.HasPrefix(rest, `@$"`), strings.HasPrefix(rest, `$@"`):
		l.i += 3
		holes, err := l.interpolated(true)
		return token{kind: tInterp, line: line, holes: holes}, true, err
	case strings.HasPrefix(rest, `$"`):
		l.i += 2
		holes, err := l.interpolated(false)
		return token{kind: tInterp, line: line, holes: holes}, true, err
	case strings.HasPrefix(rest, `@"`):
		l.i += 2
		s, err := l.verbatim()
		return token{kind: tString, text: s, line: line}, true, err
	case rest[0] == '"':
		l.i++
		s, err := l.regular('"')
		return token{kind: tString, text: s, line: line}, true, err
	case rest[0] == '\'':
		l.i++
		s, err := l.regular('\'')
		return token{kind: tChar, text: s, line: line}, true, err
	case isIdentStart(rest):
		start := l.i
		if rest[0] == '@' {
			start++
			l.i++
		}
		for l.i < len(l.src) {
			r, n := utf8.DecodeRuneInString(l.src[l.i:])
			if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
				break
			}
			l.i += n
		}
		return token{kind: tIdent, text: l.src[start:l.i], line: line}, true, nil
	case rest[0] >= '0' && rest[0] <= '9':
		start := l.i
		for l.i < len(l.src) && (isAlnum(l.src[l.i]) || l.src[l.i] == '.' && l.i+1 < len(l.src) && l.src[l.i+1] >= '0' && l.src[l.i+1] <= '9') {
			l.i++
		}
		return token{kind: tNumber, text: l.src[start:l.i], line: line}, true, nil
	case strings.HasPrefix(rest, "=>"):
		l.i += 2
		return token{kind: tPunct, text: "=>", line: line}, true, nil
	case strings.HasPrefix(rest, "=="):
		l.i += 2
		return token{kind: tPunct, text: "==", line: line}, true, nil
	}
	_, n := utf8.DecodeRuneInString(rest)
	l.i += n
	return token{kind: tPunct, text: rest[:n], line: line}, true, nil
}

func isIdentStart(s string) bool {
	if s[0] == '@' && len(s) > 1 {
		s = s[1:]
	}
	r, _ := utf8.DecodeRuneInString(s)
	return r == '_' || unicode.IsLetter(r)
}

func isAlnum(c byte) bool {
	return c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '_'
}

func (l *lexer) skipSpace() {
	for l.i < len(l.src) {
		rest := l.src[l.i:]
		switch {
		case rest[0] == '\n':
			l.line++
			l.i++
		case rest[0] == ' ' || rest[0] == '\t' || rest[0] == '\r':
			l.i++
		case strings.HasPrefix(rest, "//"), rest[0] == '#' && l.atLineStart():
			n := strings.IndexByte(rest, '\n')
			if n < 0 {
				n = len(rest)
			}
			l.i += n
		case strings.HasPrefix(rest, "/*"):
			n := strings.Index(rest, "*/")
			if n < 0 {
				n = len(rest) - 2
			}
			l.line += strings.Count(rest[:n], "\n")
			l.i += n + 2
		default:
			return
		}
	}
}

func (l *lexer) atLineStart() bool {
	start := strings.LastIndexByte(l.src[:l.i], '\n') + 1
	return strings.TrimSpace(l.src[start:l.i]) == ""
}

func (l *lexer) verbatim() (string, error) {
	var b strings.Builder
	for l.i < len(l.src) {
		c := l.src[l.i]
		l.i++
		switch {
		case c == '"' && l.i < len(l.src) && l.src[l.i] == '"':
			b.WriteByte('"')
			l.i++
		case c == '"':
			return b.String(), nil
		default:
			if c == '\n' {
				l.line++
			}
			b.WriteByte(c)
		}
	}
	return "", errors.New("a string with no closing quote")
}

func (l *lexer) regular(quote byte) (string, error) {
	var b strings.Builder
	for l.i < len(l.src) {
		c := l.src[l.i]
		l.i++
		switch c {
		case quote:
			return b.String(), nil
		case '\n':
			return "", errors.New("a line break inside a string")
		case '\\':
			if err := l.escape(&b); err != nil {
				return "", err
			}
		default:
			b.WriteByte(c)
		}
	}
	return "", errors.New("a string with no closing quote")
}

var simpleEscapes = map[byte]string{'\'': "'", '"': `"`, '\\': `\`, '0': "\x00", 'a': "\a", 'b': "\b", 'f': "\f", 'n': "\n", 'r': "\r", 't': "\t", 'v': "\v"}

func (l *lexer) escape(b *strings.Builder) error {
	if l.i >= len(l.src) {
		return errors.New("an escape at the end of the file")
	}
	c := l.src[l.i]
	l.i++
	if s, ok := simpleEscapes[c]; ok {
		b.WriteString(s)
		return nil
	}
	digits := map[byte]int{'u': 4, 'U': 8, 'x': 4}[c]
	if digits == 0 {
		return fmt.Errorf(`unknown escape \%c`, c)
	}
	end := l.i
	for end < len(l.src) && end-l.i < digits && strings.IndexByte("0123456789abcdefABCDEF", l.src[end]) >= 0 {
		end++
	}
	r, err := hexRune(l.src[l.i:end])
	if err != nil {
		return err
	}
	l.i = end
	b.WriteRune(r)
	return nil
}

func hexRune(digits string) (rune, error) {
	r, err := strconv.ParseUint(digits, 16, 32)
	if err != nil || r > unicode.MaxRune {
		return 0, fmt.Errorf("the escape %q names no character", digits)
	}
	return rune(r), nil
}

func (l *lexer) interpolated(verbatim bool) ([]piece, error) {
	var pieces []piece
	var b strings.Builder
	for l.i < len(l.src) {
		c := l.src[l.i]
		l.i++
		switch {
		case (c == '{' || c == '}') && l.i < len(l.src) && l.src[l.i] == c:
			b.WriteByte(c)
			l.i++
		case c == '{':
			end := strings.IndexByte(l.src[l.i:], '}')
			if end < 0 {
				return nil, errors.New("an interpolation with no closing brace")
			}
			pieces = append(pieces, piece{text: b.String()}, piece{text: strings.TrimSpace(l.src[l.i : l.i+end]), hole: true})
			b.Reset()
			l.i += end + 1
		case c == '"' && verbatim && l.i < len(l.src) && l.src[l.i] == '"':
			b.WriteByte('"')
			l.i++
		case c == '"':
			return append(pieces, piece{text: b.String()}), nil
		case c == '\\' && !verbatim:
			if err := l.escape(&b); err != nil {
				return nil, err
			}
		default:
			b.WriteByte(c)
		}
	}
	return nil, errors.New("a string with no closing quote")
}

type cursor struct {
	*source
	i int
}

func (c *cursor) peek(n int) token {
	if c.i+n < len(c.toks) {
		return c.toks[c.i+n]
	}
	return token{kind: tPunct, text: "<end of file>", line: c.toks[len(c.toks)-1].line}
}

func (c *cursor) is(n int, texts ...string) bool {
	for k, t := range texts {
		tok := c.peek(n + k)
		if tok.kind != tIdent && tok.kind != tPunct || tok.text != t {
			return false
		}
	}
	return true
}

func (c *cursor) expect(text string) error {
	if !c.is(0, text) {
		return c.errorf(c.peek(0).line, "expected %q, found %q", text, c.peek(0).text)
	}
	c.i++
	return nil
}

func (c *cursor) skipBalanced() {
	depth := 0
	for ; c.i < len(c.toks); c.i++ {
		switch c.toks[c.i].text {
		case "(", "[", "{":
			if c.toks[c.i].kind == tPunct {
				depth++
			}
		case ")", "]", "}":
			if c.toks[c.i].kind == tPunct {
				depth--
				if depth == 0 {
					c.i++
					return
				}
			}
		}
	}
}

// list reads item ',' item ... up to close, allowing a trailing comma.
func (c *cursor) list(closer string, item func() error) error {
	for !c.is(0, closer) {
		if err := item(); err != nil {
			return err
		}
		if !c.is(0, ",") {
			break
		}
		c.i++
	}
	return c.expect(closer)
}
