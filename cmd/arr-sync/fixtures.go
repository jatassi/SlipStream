package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path"
	"slices"
	"strings"
)

// fixtureFiles writes <owner>/testdata/upstream/<upstream>/<Fixture>.json
// for each pinned test file. It returns the rows per written file and the
// title each row reads first.
func fixtureFiles(t *tree, p pins, cos map[string]checkout) (rows map[string]int, titles []string, err error) {
	rows = map[string]int{}
	for _, set := range p.Fixtures {
		co := cos[set.From]
		dir := path.Join(set.Owner, "testdata/upstream", set.From)
		t.own(dir)
		for _, name := range set.Files {
			written := path.Join(dir, stem(name)+".json")
			f, n, err := fixtureFile(t, co, path.Join(set.Dir, name), written)
			if err != nil {
				return nil, nil, err
			}
			rows[written] = n
			titles = append(titles, f.titles()...)
		}
	}
	slices.Sort(titles)
	return rows, slices.Compact(titles), nil
}

func fixtureFile(t *tree, co checkout, file, written string) (f fixture, rows int, err error) {
	s, err := co.source(file)
	if err != nil {
		return f, 0, err
	}
	if f, err = readFixture(s); err != nil {
		return f, 0, err
	}
	out, rows, err := fixtureJSON(co, file, f)
	if err != nil {
		return f, 0, err
	}
	t.add(written, out)
	return f, rows, nil
}

func (f fixture) titles() []string {
	var titles []string
	for _, m := range f.Methods {
		for _, c := range m.Cases {
			if title, ok := firstTitle(c.Args); ok {
				titles = append(titles, title)
			}
		}
	}
	return titles
}

// fixtureJSON writes one row per line, so a resync's diff reads row by row.
func fixtureJSON(co checkout, file string, f fixture) (out []byte, rows int, err error) {
	var b bytes.Buffer
	fmt.Fprintf(&b, "{\n  \"upstream\": %s,\n  \"commit\": %s,\n  \"file\": %s,\n  \"methods\": [", quoteJSON(co.Name), quoteJSON(co.Commit), quoteJSON(file))
	for i, m := range f.Methods {
		if i > 0 {
			b.WriteString(",")
		}
		head, err := marshal(struct {
			Name   string  `json:"name"`
			Params []param `json:"params"`
			Ignore string  `json:"ignore,omitempty"`
		}{m.Name, m.Params, m.Ignore})
		if err != nil {
			return nil, 0, err
		}
		fmt.Fprintf(&b, "\n    %s, \"cases\": [", strings.TrimSuffix(head, "}"))
		for j, c := range m.Cases {
			line, err := marshal(c)
			if err != nil {
				return nil, 0, err
			}
			if j > 0 {
				b.WriteString(",")
			}
			b.WriteString("\n      " + line)
			rows++
		}
		b.WriteString("\n    ]}")
	}
	b.WriteString("\n  ]\n}\n")
	return b.Bytes(), rows, nil
}

func marshal(v any) (string, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return "", fmt.Errorf("encode a fixture row: %w", err)
	}
	return strings.TrimSuffix(b.String(), "\n"), nil
}

func quoteJSON(s string) string {
	q, _ := marshal(s)
	return q
}

// firstTitle is a row's first argument when it is a name: a string, or a
// path made OS-agnostic.
func firstTitle(args []any) (string, bool) {
	if len(args) == 0 {
		return "", false
	}
	switch a := args[0].(type) {
	case string:
		return a, a != ""
	case map[string]any:
		if a["call"] == "AsOsAgnostic" {
			s, ok := a["on"].(string)
			return s, ok
		}
	}
	return "", false
}
