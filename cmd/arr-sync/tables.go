package main

import (
	"fmt"
	"path"
	"strconv"
	"strings"
	"unicode"

	"github.com/dlclark/regexp2"
)

// patternTables writes each owner's internal/upstream package: one
// zz_<upstream>_<file>.go per upstream C# file, holding the declarations
// that owner takes from it, and zz_pattern.go with the types. It returns
// the number of patterns per owner.
func patternTables(t *tree, p pins, cos map[string]checkout) (map[string]int, error) {
	groups := map[tableKey][]regexDecl{}
	counts := map[string]int{}
	for _, pf := range p.Patterns {
		if err := groupPatterns(groups, counts, pf, cos[pf.From]); err != nil {
			return nil, err
		}
	}
	owners := map[string]bool{}
	for k, decls := range groups {
		owners[k.owner] = true
		co := cos[k.from]
		var b strings.Builder
		b.WriteString("package upstream\n\nvar (\n")
		for _, d := range decls {
			writeDecl(&b, p, k.owner, co.Name, k.file, d)
		}
		b.WriteString(")\n")
		out, err := goFile([]checkout{co}, []string{k.file}, b.String())
		if err != nil {
			return nil, err
		}
		t.add(path.Join(k.owner, "internal/upstream", "zz_"+k.from+"_"+snake(stem(k.file))+".go"), out)
	}
	for owner := range owners {
		t.own(path.Join(owner, "internal/upstream"))
		out, err := goFile(nil, nil, patternTypes)
		if err != nil {
			return nil, err
		}
		t.add(path.Join(owner, "internal/upstream/zz_pattern.go"), out)
	}
	return counts, nil
}

type tableKey struct{ owner, from, file string }

// readStrings reads the string literals in the initializer of field
// name: "name = new HashSet<string> { ".a", ".b" }", or a Dictionary's
// keys, whose values aren't strings.
func readStrings(s *source, name string) ([]string, error) {
	c := &cursor{source: s}
	for c.i < len(c.toks) && (!c.is(0, name) || !c.is(1, "=") && !c.is(1, "=>")) {
		c.i++
	}
	for c.i < len(c.toks) && !c.is(0, "{") {
		c.i++
	}
	if c.i >= len(c.toks) {
		return nil, s.errorf(0, "no field %s with an initializer", name)
	}
	start := c.i
	c.skipBalanced()
	var list []string
	for _, t := range c.toks[start:c.i] {
		if t.kind == tString {
			list = append(list, t.text)
		}
	}
	if len(list) == 0 {
		return nil, s.errorf(c.toks[start].line, "field %s lists no strings", name)
	}
	return list, nil
}

func groupPatterns(groups map[tableKey][]regexDecl, counts map[string]int, pf patternFile, co checkout) error {
	s, err := co.source(pf.File)
	if err != nil {
		return err
	}
	decls, err := readRegexes(s, pf.Skip)
	if err != nil {
		return err
	}
	for _, name := range pf.Strings {
		list, err := readStrings(s, name)
		if err != nil {
			return err
		}
		decls = append(decls, regexDecl{name: name, list: list})
	}
	seen := map[string]bool{}
	for _, d := range decls {
		seen[d.name] = true
		owner := pf.Owner
		if o, ok := pf.Fields[d.name]; ok {
			owner = o
		}
		if err := compileAll(co, pf.File, d); err != nil {
			return err
		}
		k := tableKey{owner, pf.From, pf.File}
		groups[k] = append(groups[k], d)
		counts[owner] += len(d.items)
	}
	for name := range pf.Fields {
		if !seen[name] {
			return fmt.Errorf("%s moves %s %s %s, which the file no longer declares", pinsFile, co.Name, pf.File, name)
		}
	}
	return nil
}

func writeDecl(b *strings.Builder, p pins, owner, app, file string, d regexDecl) {
	key := app + "." + stem(file) + "." + d.name
	ident := goIdent(app, stem(file), d.name)
	if isCore(owner) {
		ident = coreIdent(p.CoreWords, ident)
	}
	if d.list != nil {
		quoted := make([]string, len(d.list))
		for i, v := range d.list {
			quoted[i] = strconv.Quote(v)
		}
		fmt.Fprintf(b, "\t%s = []string{%s}\n", ident, strings.Join(quoted, ", "))
		return
	}
	elem := "Pattern"
	if d.replace {
		elem = "Replace"
	}
	if !d.array {
		fmt.Fprintf(b, "\t%s = %s%s\n", ident, elem, item(d.replace, key, d.items[0]))
		return
	}
	fmt.Fprintf(b, "\t%s = []%s{\n", ident, elem)
	for i, it := range d.items {
		fmt.Fprintf(b, "\t\t%s,\n", item(d.replace, key+"["+strconv.Itoa(i)+"]", it))
	}
	b.WriteString("\t}\n")
}

// item is one Pattern or Replace literal without its type name, which
// gofmt -s drops inside a slice.
func item(replace bool, key string, it regexItem) string {
	opts := "None"
	if len(it.options) > 0 {
		opts = strings.Join(it.options, " | ")
	}
	pattern := fmt.Sprintf("{%s, %s, %s}", strconv.Quote(key), goString(it.text), opts)
	if replace {
		return fmt.Sprintf("{Pattern%s, %s}", pattern, goString(it.with))
	}
	return pattern
}

// compileAll compiles every pattern of d on regexp2, the engine SlipStream
// runs them on, with the options .NET and regexp2 share.
func compileAll(co checkout, file string, d regexDecl) error {
	for i, it := range d.items {
		var opts regexp2.RegexOptions
		for _, o := range it.options {
			if v := netOptions[o]; v < 0x80 || v == 0x100 {
				opts |= regexp2.RegexOptions(v)
			}
		}
		if _, err := regexp2.Compile(it.text, opts); err != nil {
			return fmt.Errorf("%s %s %s[%d] doesn't compile: %w", co.Name, file, d.name, i, err)
		}
	}
	return nil
}

// goString is s as a Go literal, raw when it can be, so a pattern reads as
// upstream wrote it.
func goString(s string) string {
	raw := !strings.Contains(s, "`") && !strings.ContainsFunc(s, func(r rune) bool {
		return r != ' ' && (unicode.IsControl(r) || unicode.IsSpace(r) || r == unicode.ReplacementChar || !unicode.IsPrint(r))
	})
	if raw {
		return "`" + s + "`"
	}
	return strconv.Quote(s)
}

func stem(file string) string {
	return strings.TrimSuffix(path.Base(file), path.Ext(file))
}

func snake(s string) string {
	var b strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) && i > 0 && !unicode.IsUpper(rune(s[i-1])) {
			b.WriteByte('_')
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

func goIdent(parts ...string) string {
	var b strings.Builder
	for _, part := range parts {
		for _, word := range strings.FieldsFunc(part, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }) {
			b.WriteString(strings.ToUpper(word[:1]) + word[1:])
		}
	}
	return b.String()
}

// coreIdent renames each word of ident that words lists, as the
// content-type-confined lint splits words.
func coreIdent(words map[string]string, ident string) string {
	var b strings.Builder
	for _, w := range identWords(ident) {
		for from, to := range words {
			if strings.EqualFold(w, from) {
				w = to
			}
		}
		b.WriteString(w)
	}
	return b.String()
}

func identWords(name string) []string {
	var words []string
	runes := []rune(name)
	start := 0
	for i := 1; i < len(runes); i++ {
		lowerBefore := unicode.IsLower(runes[i-1]) || unicode.IsDigit(runes[i-1])
		acronymEnds := unicode.IsUpper(runes[i-1]) && i+1 < len(runes) && unicode.IsLower(runes[i+1])
		if unicode.IsUpper(runes[i]) && (lowerBefore || acronymEnds) {
			words = append(words, string(runes[start:i]))
			start = i
		}
	}
	return append(words, string(runes[start:]))
}

const patternTypes = `// Package upstream holds Sonarr's and Radarr's regular expressions at the
// commits cmd/arr-sync/pins.json names, as data. The hand-ported readers
// name them by identifier, and none holds a pattern of its own.
package upstream

// Pattern is one upstream regular expression: its upstream name, its text
// and its .NET options.
type Pattern struct {
	Key     string
	Text    string
	Options Options
}

// Replace is one upstream RegexReplace: a pattern and its replacement.
type Replace struct {
	Pattern
	With string
}

// Options are .NET's RegexOptions, by their .NET values.
type Options uint32

const (
	None                    Options = 0
	IgnoreCase              Options = 1
	Multiline               Options = 2
	ExplicitCapture         Options = 4
	Compiled                Options = 8
	Singleline              Options = 16
	IgnorePatternWhitespace Options = 32
	RightToLeft             Options = 64
	ECMAScript              Options = 256
	CultureInvariant        Options = 512
	NonBacktracking         Options = 1024
)
`
