package storing

import (
	"errors"
	"fmt"
	"strings"
)

// PathStyle is a Node's path dialect, a value so Windows rules are tested
// on a Linux runner and the reverse.
type PathStyle uint8

const (
	Unix PathStyle = iota + 1
	Windows
)

var ErrNotAbsolute = errors.New("not an absolute path")

func (s PathStyle) sep() byte {
	if s == Windows {
		return '\\'
	}
	return '/'
}

// parse parts p into its root, such as "/", `C:\`, `\\server\share\` or
// `\\?\C:\`, and the names under it, with "." and ".." resolved lexically.
func (s PathStyle) parse(p string) (root string, names []string, err error) {
	rest := p
	switch s {
	case Unix:
		if !strings.HasPrefix(p, "/") {
			return "", nil, fmt.Errorf("%w: %q", ErrNotAbsolute, p)
		}
		root = "/"
	case Windows:
		root, rest, err = windowsRoot(strings.ReplaceAll(p, "/", `\`))
		if err != nil {
			return "", nil, fmt.Errorf("%w: %q", err, p)
		}
	}
	for name := range strings.SplitSeq(rest, string(s.sep())) {
		switch name {
		case "", ".":
		case "..":
			names = names[:max(len(names)-1, 0)]
		default:
			names = append(names, name)
		}
	}
	return root, names, nil
}

// windowsRoot splits p, its separators already '\', into its root and the
// rest: a drive `C:\`, a share `\\server\share\`, or either behind `\\?\`
// or `\\.\`.
func windowsRoot(p string) (root, rest string, err error) {
	var prefix string
	switch {
	case len(p) >= 4 && (p[:4] == `\\?\` || p[:4] == `\\.\`):
		prefix, p = p[:4], p[4:]
		if len(p) < 4 || !strings.EqualFold(p[:4], `UNC\`) {
			return driveRoot(prefix, p)
		}
		prefix, p = prefix+`UNC\`, p[4:]
	case strings.HasPrefix(p, `\\`):
		prefix, p = `\\`, p[2:]
	default:
		return driveRoot("", p)
	}
	server, after, _ := strings.Cut(p, `\`)
	share, rest, _ := strings.Cut(after, `\`)
	if server == "" || share == "" {
		return "", "", ErrNotAbsolute
	}
	return prefix + server + `\` + share + `\`, rest, nil
}

func driveRoot(prefix, p string) (root, rest string, err error) {
	if !drive(p) || len(p) < 3 || p[2] != '\\' {
		return "", "", ErrNotAbsolute
	}
	return prefix + strings.ToUpper(p[:1]) + `:\`, p[3:], nil
}

func (s PathStyle) build(root string, names []string) string {
	return root + strings.Join(names, string(s.sep()))
}

// Clean makes p this style's canonical absolute form. Windows takes / or \,
// upper-cases the drive and keeps `\\server\share` and `\\?\` prefixes. A
// path that isn't absolute is ErrNotAbsolute.
func (s PathStyle) Clean(p string) (string, error) {
	root, names, err := s.parse(p)
	if err != nil {
		return "", err
	}
	return s.build(root, names), nil
}

// Segments are a clean path's breadcrumbs, root first: "/", `C:\` or
// `\\server\share`.
func (s PathStyle) Segments(p string) []Segment {
	root, names, err := s.parse(p)
	if err != nil {
		return nil
	}
	rootName := root
	if strings.Count(root, `\`) > 2 && !strings.HasSuffix(root, `:\`) {
		rootName = strings.TrimSuffix(root, `\`)
	}
	segs := []Segment{{Name: rootName, Path: root}}
	for i, name := range names {
		segs = append(segs, Segment{Name: name, Path: s.build(root, names[:i+1])})
	}
	return segs
}

// Parent is a clean path's parent, false at its root.
func (s PathStyle) Parent(p string) (string, bool) {
	root, names, err := s.parse(p)
	if err != nil || len(names) == 0 {
		return "", false
	}
	return s.build(root, names[:len(names)-1]), true
}

// Join is name in the clean folder dir.
func (s PathStyle) Join(dir, name string) string {
	if dir[len(dir)-1] == s.sep() {
		return dir + name
	}
	return dir + string(s.sep()) + name
}

// Under is rel under the clean folder root.
func (s PathStyle) Under(root string, rel Rel) string {
	p := root
	for _, name := range rel.segments() {
		p = s.Join(p, name)
	}
	return p
}

// Split parts typed text into the clean folder to list, the prefix of the
// last name, and the separator typed before it. ok is false for text that
// names no folder yet: relative, a bare drive "C:" or a server without a
// share.
func (s PathStyle) Split(typed string) (dir, prefix string, sep byte, ok bool) {
	seps := "/"
	if s == Windows {
		seps = `/\`
	}
	i := strings.LastIndexAny(typed, seps)
	if i < 0 {
		return "", "", 0, false
	}
	dir, err := s.Clean(typed[:i+1])
	if err != nil {
		return "", "", 0, false
	}
	return dir, typed[i+1:], typed[i], true
}
