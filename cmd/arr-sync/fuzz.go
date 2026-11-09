package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

// fuzzSeeds writes each source's titles, one per line, as seeds for the
// title fuzz test. Only titles are taken: the sources' expected answers
// follow other parsers' rules. It returns the titles per written file.
func fuzzSeeds(t *tree, p pins, cos map[string]checkout) (map[string]int, error) {
	t.own(p.Fuzz.Out)
	counts := map[string]int{}
	var record strings.Builder
	record.WriteString("# Fuzz seeds\n\nThe title fuzz test's seeds are the release names in the test data of the projects below, without their expected results. Each file is one project's titles, one per line, sorted.\n")
	for _, src := range p.Fuzz.Sources {
		co := cos[src.From]
		titles, err := readTitles(co, src)
		if err != nil {
			return nil, err
		}
		titles = slices.DeleteFunc(titles, func(s string) bool { return strings.TrimSpace(s) == "" || strings.ContainsAny(s, "\r\n") })
		slices.Sort(titles)
		titles = slices.Compact(titles)
		file := path.Join(p.Fuzz.Out, src.From+".txt")
		t.add(file, []byte(strings.Join(titles, "\n")+"\n"))
		counts[file] = len(titles)
		fmt.Fprintf(&record, "\n## %s\n\nSource: %s\nCommit: %s\nLicense: `%s` in the source repository\nFiles: %s\n", co.Name, repoURL(co.Repo), co.Commit, co.License, strings.Join(src.Files, ", "))
	}
	t.add(path.Join(p.Fuzz.Out, "VENDORED.md"), []byte(record.String()))
	return counts, nil
}

func readTitles(co checkout, src fuzzSource) ([]string, error) {
	read, ok := titleReaders[src.Reader]
	if !ok {
		return nil, fmt.Errorf("%s: fuzz reader %q isn't one of yaml-keys, python-cases or json-titles", pinsFile, src.Reader)
	}
	var titles []string
	for _, pattern := range src.Files {
		files, err := filepath.Glob(filepath.Join(co.dir, filepath.FromSlash(pattern)))
		if err != nil || len(files) == 0 {
			return nil, fmt.Errorf("%s %s matches no file", co.Name, pattern)
		}
		for _, f := range files {
			rel, err := filepath.Rel(co.dir, f)
			if err != nil {
				return nil, fmt.Errorf("fuzz source %s: %w", f, err)
			}
			b, err := co.read(filepath.ToSlash(rel))
			if err != nil {
				return nil, err
			}
			got, err := read(b)
			if err != nil {
				return nil, fmt.Errorf("%s %s: %w", co.Name, rel, err)
			}
			titles = append(titles, got...)
		}
	}
	return titles, nil
}

var titleReaders = map[string]func([]byte) ([]string, error){
	"yaml-keys":    yamlKeys,
	"python-cases": pythonCases,
	"json-titles":  jsonTitles,
}

// yamlKeys reads guessit's test files, whose top-level keys are the names
// under test. A leading - or + marks an expected failure or success.
func yamlKeys(b []byte) ([]string, error) {
	var titles []string
	dec := yaml.NewDecoder(bytes.NewReader(b))
	for {
		var doc yaml.Node
		if err := dec.Decode(&doc); errors.Is(err, io.EOF) {
			return titles, nil
		} else if err != nil {
			return nil, fmt.Errorf("read YAML: %w", err)
		}
		if len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
			continue
		}
		m := doc.Content[0].Content
		for i := 0; i < len(m); i += 2 {
			if k := m[i].Value; m[i].Kind == yaml.ScalarNode && k != "__default__" {
				titles = append(titles, strings.TrimLeft(k, "-+"))
			}
		}
	}
}

var pythonCase = regexp.MustCompile(`(?m)^\s*\(\s*(r?)(?:"((?:[^"\\]|\\.)*)"|'((?:[^'\\]|\\.)*)')\s*,`)

// pythonCases reads the first string of each ("name", {...}) tuple in
// PTT's parametrized tests.
func pythonCases(b []byte) ([]string, error) {
	var titles []string
	for _, m := range pythonCase.FindAllSubmatch(b, -1) {
		s := string(m[2]) + string(m[3])
		if len(m[1]) == 0 {
			var err error
			if s, err = pythonUnescape(s); err != nil {
				return nil, err
			}
		}
		titles = append(titles, s)
	}
	return titles, nil
}

func pythonUnescape(s string) (string, error) {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i+1 == len(s) {
			b.WriteByte(s[i])
			continue
		}
		i++
		width := map[byte]int{'x': 2, 'u': 4, 'U': 8}[s[i]]
		switch {
		case width > 0 && i+width < len(s):
			r, err := hexRune(s[i+1 : i+1+width])
			if err != nil {
				return "", fmt.Errorf("read the Python string %q: %w", s, err)
			}
			b.WriteRune(r)
			i += width
		case strings.IndexByte(`\\'"`, s[i]) >= 0:
			b.WriteByte(s[i])
		case s[i] == 'n', s[i] == 't', s[i] == 'r':
			b.WriteByte(' ')
		default:
			b.WriteByte('\\')
			b.WriteByte(s[i])
		}
	}
	return b.String(), nil
}

func jsonTitles(b []byte) ([]string, error) {
	var rows []struct {
		Title string `json:"title"`
	}
	if err := json.Unmarshal(b, &rows); err != nil {
		return nil, fmt.Errorf("read JSON: %w", err)
	}
	titles := make([]string, len(rows))
	for i, r := range rows {
		titles[i] = r.Title
	}
	return titles, nil
}

func repoURL(repo string) string {
	return strings.TrimSuffix(repo, ".git")
}
