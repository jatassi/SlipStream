package main

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

const dialectTypes = `package formats

type upstreamValue struct {
	Name  string
	Value int
}

type upstreamIsoLanguage struct {
	Args     []string
	Language string
}

var (
`

// dialectTable writes the enums the formats dialects map from: names and
// numbers verbatim, as unexported data in package formats.
func dialectTable(t *tree, p pins, cos map[string]checkout) error {
	w := dialectWriter{cos: cos}
	w.b.WriteString(dialectTypes)
	for _, e := range p.Dialects.Enums {
		err := w.values(coreIdent(p.CoreWords, e.From+e.Enum), e.From, e.File, func(s *source) ([]enumValue, error) {
			return readEnum(s, e.Enum)
		})
		if err != nil {
			return err
		}
	}
	for _, l := range p.Dialects.Languages {
		if err := w.values(l.From+"Language", l.From, l.File, readLanguages); err != nil {
			return err
		}
	}
	for _, l := range p.Dialects.Iso {
		if err := w.iso(l.From+"IsoLanguages", l.From, l.File); err != nil {
			return err
		}
	}
	w.b.WriteString(")\n")
	out, err := goFile(w.sources, nil, w.b.String())
	if err != nil {
		return err
	}
	t.add(p.Dialects.Out, out)
	return nil
}

type dialectWriter struct {
	b       strings.Builder
	cos     map[string]checkout
	sources []checkout
}

func (w *dialectWriter) source(from, file string) (*source, error) {
	co := w.cos[from]
	if !slices.ContainsFunc(w.sources, func(c checkout) bool { return c.key == from }) {
		w.sources = append(w.sources, co)
	}
	return co.source(file)
}

func (w *dialectWriter) values(ident, from, file string, read func(*source) ([]enumValue, error)) error {
	s, err := w.source(from, file)
	if err != nil {
		return err
	}
	values, err := read(s)
	if err != nil {
		return err
	}
	fmt.Fprintf(&w.b, "\t%s = []upstreamValue{\n", ident)
	for _, v := range values {
		fmt.Fprintf(&w.b, "\t\t{%q, %d},\n", v.name, v.value)
	}
	w.b.WriteString("\t}\n")
	return nil
}

func (w *dialectWriter) iso(ident, from, file string) error {
	s, err := w.source(from, file)
	if err != nil {
		return err
	}
	isos, err := readIsoLanguages(s)
	if err != nil {
		return err
	}
	fmt.Fprintf(&w.b, "\t%s = []upstreamIsoLanguage{\n", ident)
	for _, iso := range isos {
		quoted := make([]string, len(iso.args))
		for i, c := range iso.args {
			quoted[i] = strconv.Quote(c)
		}
		fmt.Fprintf(&w.b, "\t\t{[]string{%s}, %q},\n", strings.Join(quoted, ", "), strings.TrimPrefix(iso.language, "Language."))
	}
	w.b.WriteString("\t}\n")
	return nil
}
