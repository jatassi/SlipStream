package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/regex"
)

// trashSnapshot copies the pinned TRaSH formats verbatim, with TRaSH's
// license beside them, and returns how many it copied.
func trashSnapshot(t *tree, p pins, co checkout) (int, error) {
	return snapshot(t, p.Trash.Root, co, p.Trash.Snapshot, p.Trash.Out, "The Custom Formats below are TRaSH Guides' JSON at the commit above, copied verbatim. The formats snapshot test compiles and runs every one of them.")
}

// trashNaming copies the pinned TRaSH naming templates verbatim, with
// TRaSH's license beside them, and returns how many files it copied.
func trashNaming(t *tree, p pins, co checkout) (int, error) {
	return snapshot(t, p.Trash.Root, co, p.Trash.NamingSnapshot, p.Trash.NamingOut, "The naming templates below are TRaSH Guides' JSON at the commit above, copied verbatim. The naming tests parse every one of them.")
}

// snapshot copies the JSON files of dirs to out, keyed by their path
// under root, with the upstream's license and a VENDORED.md that says
// about.
func snapshot(t *tree, root string, co checkout, dirs []string, out, about string) (int, error) {
	t.own(out)
	n := 0
	for _, dir := range dirs {
		entries, err := os.ReadDir(filepath.Join(co.dir, filepath.FromSlash(dir)))
		if err != nil {
			return 0, fmt.Errorf("%s %s: %w", co.Name, dir, err)
		}
		for _, e := range entries {
			if e.IsDir() || path.Ext(e.Name()) != ".json" {
				continue
			}
			file := path.Join(dir, e.Name())
			b, err := co.read(file)
			if err != nil {
				return 0, err
			}
			t.add(path.Join(out, strings.TrimPrefix(file, root+"/")), b)
			n++
		}
	}
	license, err := co.read(co.License)
	if err != nil {
		return 0, err
	}
	t.add(path.Join(out, "LICENSE"), license)
	t.add(path.Join(out, "VENDORED.md"), vendored(co, about))
	return n, nil
}

type trashFormat struct {
	Name           string `json:"name"`
	Specifications []struct {
		Name           string          `json:"name"`
		Implementation string          `json:"implementation"`
		Negate         bool            `json:"negate"`
		Required       bool            `json:"required"`
		Fields         json.RawMessage `json:"fields"`
	} `json:"specifications"`
}

// trashReaders writes the formats the stream readers read titles with,
// condition by condition, and returns how many conditions it wrote.
func trashReaders(t *tree, p pins, co checkout) (int, error) {
	var b strings.Builder
	b.WriteString(trashTypes)
	b.WriteString("var (\n")
	n := 0
	for _, file := range p.Trash.Readers {
		src, err := co.read(file)
		if err != nil {
			return 0, err
		}
		var f trashFormat
		if err := json.Unmarshal(src, &f); err != nil {
			return 0, fmt.Errorf("%s %s: %w", co.Name, file, err)
		}
		key := strings.TrimPrefix(file, p.Trash.Root+"/")
		app, _, _ := strings.Cut(key, "/")
		ident := coreIdent(p.CoreWords, goIdent("Trash", app, stem(file)))
		fmt.Fprintf(&b, "\t%s = Format{%q, %q, []Condition{\n", ident, "TRaSH "+key, f.Name)
		for _, s := range f.Specifications {
			value, err := fieldValue(s.Fields)
			if err != nil {
				return 0, fmt.Errorf("%s %s %s: %w", co.Name, file, s.Name, err)
			}
			if s.Implementation == "ReleaseTitleSpecification" {
				if _, err := regex.Compile(value, regex.Options{IgnoreCase: true}); err != nil {
					return 0, fmt.Errorf("%s %s %s doesn't compile: %w", co.Name, file, s.Name, err)
				}
			}
			fmt.Fprintf(&b, "\t\t{%q, %q, %t, %t, %s},\n", s.Name, s.Implementation, s.Negate, s.Required, goString(value))
			n++
		}
		b.WriteString("\t}}\n")
	}
	b.WriteString(")\n")
	out, err := goFile([]checkout{co}, []string{p.Trash.Root}, b.String())
	if err != nil {
		return 0, err
	}
	t.add(path.Join(p.Trash.ReadersOwner, "internal/upstream/zz_trash.go"), out)
	return n, nil
}

func fieldValue(fields json.RawMessage) (string, error) {
	var f struct {
		Value json.RawMessage `json:"value"`
	}
	if err := json.Unmarshal(fields, &f); err != nil {
		return "", fmt.Errorf("read fields: %w", err)
	}
	var s string
	if json.Unmarshal(f.Value, &s) == nil {
		return s, nil
	}
	return string(f.Value), nil
}

// titlesFile is every fixture title that fits on one line, one per line,
// for the formats snapshot test, which reads no Content Type's testdata.
// It returns how many it left out.
func titlesFile(t *tree, p pins, titles []string) int {
	kept := slices.DeleteFunc(slices.Clone(titles), func(s string) bool { return strings.ContainsAny(s, "\r\n") })
	t.add(p.Trash.Titles, []byte(strings.Join(kept, "\n")+"\n"))
	return len(titles) - len(kept)
}

const trashTypes = `package upstream

// Format is one TRaSH Guides Custom Format, as the stream readers read it.
type Format struct {
	Key        string
	Name       string
	Conditions []Condition
}

// Condition is one of a format's specifications, as TRaSH writes it.
type Condition struct {
	Name     string
	Kind     string
	Negate   bool
	Required bool
	Value    string
}

`
