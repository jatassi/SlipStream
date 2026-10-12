package config

import (
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"reflect"
	"strings"

	"go.yaml.in/yaml/v3"
)

type Source string

const (
	SourceDefault Source = "default"
	SourceYAML    Source = "config.yaml"
	SourceDotEnv  Source = ".env"
	SourceEnv     Source = "env"
	// SourceRandom marks a value the Node drew because no layer set one.
	SourceRandom Source = "random"
)

const (
	envPrefix = "SLIPSTREAM_"
	// DataDirVar names the data dir, which no layer can set because the
	// layers live inside it.
	DataDirVar = envPrefix + "DATA_DIR"
)

type Key struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Default string `json:"default"`
	Source  Source `json:"source"`
}

// Problem is one reason a configuration cannot load. Line is 0 for the
// process environment and for a value no file set.
type Problem struct {
	Source Source
	Line   int
	Key    string
	Msg    string
}

func (p Problem) String() string {
	if p.Line > 0 {
		return fmt.Sprintf("%s:%d: %s: %s", p.Source, p.Line, p.Key, p.Msg)
	}
	return fmt.Sprintf("%s: %s: %s", p.Source, p.Key, p.Msg)
}

// Error lists every problem the layers have, so one start reports them all.
type Error struct {
	Problems []Problem
}

func (e *Error) Error() string {
	lines := make([]string, len(e.Problems))
	for i, p := range e.Problems {
		lines[i] = p.String()
	}
	return strings.Join(lines, "\n")
}

type origin struct {
	source Source
	line   int
}

type loader struct {
	leaves   []Leaf
	byPath   map[string]Leaf
	byEnv    map[string]Leaf
	branches map[string]bool
	origins  map[string]origin
	problems []Problem
}

// Load sets tree, a pointer to a struct that holds the defaults, from
// dir/config.yaml, then dir/.env, then environ, each layer overriding the
// one before it per key. Neither file has to exist.
func Load(tree any, dir string, environ []string) ([]Key, error) {
	l := &loader{
		leaves:   Walk(tree),
		byPath:   map[string]Leaf{},
		byEnv:    map[string]Leaf{},
		branches: map[string]bool{},
		origins:  map[string]origin{},
	}
	defaults := make([]string, len(l.leaves))
	for i, leaf := range l.leaves {
		defaults[i] = leaf.String()
		l.byPath[leaf.Path] = leaf
		l.byEnv[leaf.Env()] = leaf
		for p := leaf.Path; strings.Contains(p, "."); {
			p = p[:strings.LastIndex(p, ".")]
			l.branches[p] = true
		}
	}

	fsys := os.DirFS(dir)
	if err := l.loadYAML(fsys); err != nil {
		return nil, err
	}
	if err := l.loadDotEnv(fsys); err != nil {
		return nil, err
	}
	for _, v := range parseEnviron(environ) {
		l.applyVar(SourceEnv, v)
	}
	l.validate()
	if len(l.problems) > 0 {
		return nil, &Error{Problems: l.problems}
	}

	keys := make([]Key, len(l.leaves))
	for i, leaf := range l.leaves {
		o, ok := l.origins[leaf.Path]
		if !ok {
			o.source = SourceDefault
		}
		keys[i] = Key{Key: leaf.Path, Value: leaf.String(), Default: defaults[i], Source: o.source}
	}
	return keys, nil
}

func (l *loader) problem(src Source, line int, key, format string, args ...any) {
	l.problems = append(l.problems, Problem{Source: src, Line: line, Key: key, Msg: fmt.Sprintf(format, args...)})
}

// readOptional reads a layer file, which is empty when missing.
func readOptional(fsys fs.FS, name string) ([]byte, error) {
	data, err := fs.ReadFile(fsys, name)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", name, err)
	}
	return data, nil
}

func (l *loader) loadYAML(fsys fs.FS) error {
	data, err := readOptional(fsys, string(SourceYAML))
	if err != nil {
		return err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		l.problem(SourceYAML, 0, "(file)", "%v", err)
		return nil
	}
	if len(doc.Content) == 0 {
		return nil
	}
	l.yamlMapping(doc.Content[0], "")
	return nil
}

// yamlMapping walks one mapping beside the tree, so an unknown or duplicate
// key anywhere is reported with its full path and line.
func (l *loader) yamlMapping(n *yaml.Node, at string) {
	if n.Kind != yaml.MappingNode {
		l.problem(SourceYAML, n.Line, cmp.Or(at, "(top level)"), "expected a mapping")
		return
	}
	seen := map[string]int{}
	for i := 0; i+1 < len(n.Content); i += 2 {
		k, v := n.Content[i], n.Content[i+1]
		path := k.Value
		if at != "" {
			path = at + "." + k.Value
		}
		if first, dup := seen[k.Value]; dup {
			l.problem(SourceYAML, k.Line, path, "duplicate key, first set on line %d", first)
			continue
		}
		seen[k.Value] = k.Line
		if leaf, ok := l.byPath[path]; ok {
			l.yamlLeaf(leaf, k, v)
			continue
		}
		if l.branches[path] {
			l.yamlMapping(v, path)
			continue
		}
		l.problem(SourceYAML, k.Line, path, "unknown key")
	}
}

func (l *loader) yamlLeaf(leaf Leaf, k, v *yaml.Node) {
	var err error
	switch {
	case leaf.isList() && v.Kind == yaml.SequenceNode:
		items := make([]string, len(v.Content))
		for i, item := range v.Content {
			if item.Kind != yaml.ScalarNode {
				err = fmt.Errorf("item %d is not a scalar", i+1)
				break
			}
			items[i] = item.Value
		}
		if err == nil {
			err = leaf.setList(items)
		}
	case leaf.isList():
		err = errors.New("expected a list")
	case v.Kind == yaml.ScalarNode:
		err = leaf.codec.parse(v.Value, leaf.value)
	default:
		err = errors.New("expected a single value")
	}
	if err != nil {
		l.problem(SourceYAML, k.Line, leaf.Path, "%v", err)
		return
	}
	l.origins[leaf.Path] = origin{SourceYAML, k.Line}
}

type envVar struct {
	name, value string
	line        int
}

func parseEnviron(environ []string) []envVar {
	var vars []envVar
	for _, kv := range environ {
		name, value, _ := strings.Cut(kv, "=")
		if strings.HasPrefix(name, envPrefix) {
			vars = append(vars, envVar{name: name, value: value})
		}
	}
	return vars
}

// loadDotEnv reads KEY=value lines with no shell expansion. Blank lines and
// lines starting with # are skipped.
func (l *loader) loadDotEnv(fsys fs.FS) error {
	data, err := readOptional(fsys, string(SourceDotEnv))
	if err != nil {
		return err
	}
	for i, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSuffix(line, "\r")
		if trimmed := strings.TrimSpace(line); trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		name, value, ok := strings.Cut(line, "=")
		switch {
		case !ok:
			l.problem(SourceDotEnv, i+1, line, "not a KEY=value line")
		case !strings.HasPrefix(name, envPrefix):
			l.problem(SourceDotEnv, i+1, name, "missing the %s prefix", envPrefix)
		default:
			l.applyVar(SourceDotEnv, envVar{name: name, value: value, line: i + 1})
		}
	}
	return nil
}

func (l *loader) applyVar(src Source, v envVar) {
	if v.name == DataDirVar {
		return
	}
	leaf, ok := l.byEnv[v.name]
	if !ok {
		l.problem(src, v.line, v.name, "unknown key")
		return
	}
	if err := leaf.setText(v.value); err != nil {
		l.problem(src, v.line, v.name, "%v", err)
		return
	}
	l.origins[leaf.Path] = origin{src, v.line}
}

type hostnamer interface{ Hostname() string }

// validate checks the rules every section shares, so no section writes its
// own: durations are positive and endpoints name a host.
func (l *loader) validate() {
	for _, leaf := range l.leaves {
		o, ok := l.origins[leaf.Path]
		if !ok {
			o.source = SourceDefault
		}
		elems := leaf.elems()
		if len(elems) == 0 && isEndpointList(leaf) {
			l.problem(o.source, o.line, leaf.Path, "endpoint list is empty")
		}
		for _, e := range elems {
			if h, ok := e.Interface().(hostnamer); ok && h.Hostname() == "" {
				l.problem(o.source, o.line, leaf.Path, "endpoint is empty")
			}
			if e.Type() == durationType && e.Int() <= 0 {
				l.problem(o.source, o.line, leaf.Path, "duration %s is not positive", leaf.String())
			}
		}
	}
}

func isEndpointList(leaf Leaf) bool {
	return leaf.isList() && leaf.value.Type().Elem().Implements(reflect.TypeFor[hostnamer]())
}
