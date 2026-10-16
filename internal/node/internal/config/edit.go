package config

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/jatassi/slipstream/internal/atomicfile"
)

// Change sets Key to Text in config.yaml, or removes Key when Text is nil.
type Change struct {
	Key  string
	Text *string
}

// ErrNotRegular refuses a config.yaml that a rename can't replace in place:
// a symlink, or a file bind-mounted on its own.
var ErrNotRegular = errors.New("config.yaml is not a regular file, so SlipStream can't rewrite it; edit it by hand")

// Edit applies changes to dir/config.yaml's node tree, so comments and key
// order survive. It creates the file when missing, creates the mappings a
// key needs, and removes a mapping a removal leaves empty. It writes only
// if check accepts the edited file, read through fsys: dir as it is on disk
// with config.yaml replaced by the edit. On any error the file is unchanged.
func Edit(dir string, changes []Change, check func(fsys fs.FS) error) error {
	path := filepath.Join(dir, string(SourceYAML))
	info, err := os.Lstat(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
	case err != nil:
		return fmt.Errorf("read %s: %w", path, err)
	case !info.Mode().IsRegular():
		return ErrNotRegular
	}
	old, err := readOptional(os.DirFS(dir), string(SourceYAML))
	if err != nil {
		return err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(old, &doc); err != nil {
		return fmt.Errorf("%s: %w", SourceYAML, err)
	}
	if len(doc.Content) == 0 {
		doc = yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{{Kind: yaml.MappingNode}}}
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return fmt.Errorf("%s: the top level is not a mapping", SourceYAML)
	}
	for _, c := range changes {
		path := strings.Split(c.Key, ".")
		if c.Text == nil {
			unset(root, path)
		} else {
			set(root, path, *c.Text)
		}
	}
	var b bytes.Buffer
	enc := yaml.NewEncoder(&b)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return fmt.Errorf("encode %s: %w", SourceYAML, err)
	}
	if err := check(overlay{dir: os.DirFS(dir), yaml: b.Bytes()}); err != nil {
		return err
	}
	err = atomicfile.Write(path, b.Bytes())
	if errors.Is(err, syscall.EBUSY) {
		return ErrNotRegular
	}
	return err
}

// find is the index of key's value in mapping m, or -1.
func find(m *yaml.Node, key string) int {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return i + 1
		}
	}
	return -1
}

func set(m *yaml.Node, path []string, text string) {
	i := find(m, path[0])
	if i < 0 {
		m.Content = append(m.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: path[0]}, &yaml.Node{Kind: yaml.MappingNode})
		i = len(m.Content) - 1
	}
	if len(path) > 1 {
		if m.Content[i].Kind != yaml.MappingNode {
			m.Content[i] = &yaml.Node{Kind: yaml.MappingNode}
		}
		set(m.Content[i], path[1:], text)
		return
	}
	v := m.Content[i]
	if v.Kind != yaml.ScalarNode {
		v = &yaml.Node{Kind: yaml.ScalarNode, LineComment: v.LineComment}
		m.Content[i] = v
	}
	v.Value, v.Tag = text, ""
}

// unset removes path from m and reports whether m is left empty.
func unset(m *yaml.Node, path []string) bool {
	i := find(m, path[0])
	if i < 0 {
		return len(m.Content) == 0
	}
	if len(path) == 1 || (m.Content[i].Kind == yaml.MappingNode && unset(m.Content[i], path[1:])) {
		m.Content = append(m.Content[:i-1], m.Content[i+1:]...)
	}
	return len(m.Content) == 0
}

// overlay is a directory with config.yaml replaced.
type overlay struct {
	dir  fs.FS
	yaml []byte
}

func (o overlay) Open(name string) (fs.File, error) {
	if name != string(SourceYAML) {
		return o.dir.Open(name)
	}
	return memFile{bytes.NewReader(o.yaml)}, nil
}

// memFile is the edited config.yaml, its own FileInfo.
type memFile struct{ *bytes.Reader }

func (f memFile) Stat() (fs.FileInfo, error) { return f, nil }
func (memFile) Close() error                 { return nil }
func (memFile) Name() string                 { return string(SourceYAML) }
func (memFile) Mode() fs.FileMode            { return 0o600 }
func (memFile) ModTime() time.Time           { return time.Time{} }
func (memFile) IsDir() bool                  { return false }
func (memFile) Sys() any                     { return nil }
