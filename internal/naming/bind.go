package naming

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

// unstorable are the characters outside a token that a name can't hold on
// Windows. Parse already refuses "/" and "\".
const unstorable = `:*?<>|"`

// bind resolves t's refs against c for field f. It refuses an unknown
// token, a token whose scope f doesn't take, a modifier the token's kind
// doesn't take, and a literal holding a character a name can't hold, each
// as a *ParseError. It finds the episode blocks and leaves anchorEnd to
// the scheme.
func bind(t Template, c *catalog, f field) (bound, error) {
	b := bound{field: f, parts: make([]part, len(t.parts))}
	at := 0
	for i, p := range t.parts {
		switch p := p.(type) {
		case literal:
			if err := checkLiteral(string(p), at, f); err != nil {
				return bound{}, err
			}
			b.parts[i] = p
			at += len(p)
		case *ref:
			br, err := bindRef(p, c, f)
			if err != nil {
				return bound{}, err
			}
			b.parts[i] = br
			at += len(p.text)
		}
	}
	b.index()
	return b, nil
}

// index finds b's episode blocks and its shrinkable refs from its parts.
func (b *bound) index() {
	b.shrinkable = nil
	for i, p := range b.parts {
		if r, ok := p.(*boundRef); ok && r.entry.Scope == ItemScope && r.entry.Kind == KindTitle {
			b.shrinkable = append(b.shrinkable, i)
		}
	}
	b.block, b.alone = blockSpan(b.parts)
}

// nameOf is what a field's template names, as a sentence starts.
var nameOf = map[field]string{fileField: "the file name", folderField: "the folder name", levelField: "the folder name"}

func checkLiteral(s string, at int, f field) error {
	for i, r := range s {
		switch {
		case r == ':':
			return refuse(at+i, "", nameOf[f]+" can't hold ':'", "Remove it or use a token.")
		case strings.ContainsRune(unstorable, r):
			c := quote(string(r))
			if r == '"' {
				c = `'"'`
			}
			return refuse(at+i, "", "a template can't contain "+c+" because a name on Windows can't hold it", "Remove the "+c+".")
		case unicode.IsControl(r):
			return refuse(at+i, "", fmt.Sprintf("a template can't contain the control character U+%04X", r), "Remove it.")
		}
	}
	return nil
}

var fieldNames = map[field]string{folderField: "a Work folder name", levelField: "a level folder name"}

var fieldReasons = map[field]string{
	folderField: "it can differ between the Work's files",
	levelField:  "it can differ between the files in one level folder",
}

func bindRef(r *ref, c *catalog, f field) (*boundRef, error) {
	e := c.byKey[r.key]
	switch {
	case e == nil:
		return nil, refuse(r.at, r.text, r.text+" isn't a token", "Use a token from the token list, such as {Target}.")
	case !slices.Contains(fieldScopes[f], e.Scope) && e.Scope == FileScope:
		return nil, refuse(r.at, r.text, "folder names can't use "+r.text+", because it changes with each upgrade", "Remove it from the folder template.")
	case !slices.Contains(fieldScopes[f], e.Scope):
		return nil, refuse(r.at, r.text, r.text+" can't go in "+fieldNames[f]+" because "+fieldReasons[f], "Use it in the file template.")
	case r.tag != noTag && e.tag != noTag:
		return nil, refuse(r.at, r.text, r.text+" puts a tag inside a tag", "Remove the outer tag.")
	}
	m, err := modFor(r, e)
	if err != nil {
		return nil, err
	}
	br := &boundRef{ref: *r, entry: e}
	br.mod = m
	return br, nil
}

// modFor is r's modifier as e's kind reads it.
func modFor(r *ref, e *entry) (mod, error) {
	typed := ":" + modText(r.mod)
	refused := func(cause, fix string) error { return refuse(r.at, r.text, r.text+" "+cause, fix) }
	switch m := r.mod.(type) {
	case nil:
		return nil, nil
	case padding:
		if e.Kind != KindNumber {
			return nil, refused("pads a value that isn't a number", "Remove the "+quote(typed)+". Only a number can be padded.")
		}
		return m, nil
	case cut:
		if e.Kind != KindTitle && e.Kind != KindText {
			return nil, refused("cuts a value that isn't text", "Remove the "+quote(typed)+". Only a title or text can be cut.")
		}
		return m, nil
	case words:
		if e.Kind != KindList {
			return nil, refused("has a modifier this token doesn't take", "Remove the "+quote(typed)+".")
		}
		switch e.list {
		case langList, audioLangList:
			if l, ok := m.langs(); ok {
				return l, nil
			}
			return nil, refused("has "+quote(m.text)+", which isn't a list of language codes", `Join language codes with "+", as in :EN+DE.`)
		case pickList:
			if o, ok := m.one(); ok {
				return o, nil
			}
			return nil, refused("doesn't name exactly one format", "Name one format, as in {Custom Format:IMAX}.")
		case nameList:
		}
		if ff, ok := m.formats(); ok {
			return ff, nil
		}
		return nil, refused("has an empty format name", "Name a format between each pair of commas.")
	}
	panic(fmt.Sprintf("naming: Parse made a %T modifier", r.mod))
}

// modText is a modifier from Parse as typed after the ":".
func modText(m mod) string {
	switch m := m.(type) {
	case padding:
		return strings.Repeat("0", int(m))
	case cut:
		if m.fromEnd {
			return fmt.Sprintf("-%d", m.bytes)
		}
		return strconv.Itoa(m.bytes)
	case words:
		if m.drop {
			return "-" + m.text
		}
		return m.text
	}
	return ""
}
