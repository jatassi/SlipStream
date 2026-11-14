package link

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jatassi/slipstream/internal/storing"
)

// ServerPath is a path as a Media Server stores it, with that server's
// separator. It is the one path type a server sees: no other code joins or
// compares a server path.
//
// A path that starts with a drive letter or a backslash is Windows. It is
// kept with backslashes, and Within, Rel and Key fold case on it, because
// Windows file systems do.
type ServerPath struct {
	s       string
	windows bool
}

var ErrNotAbsolute = errors.New("not an absolute server path")

// ParsePath parses an absolute path or folder as a server stores it:
// "/media/movies", "D:\Movies", "D:/Movies" or "\\nas\media\movies".
// Repeated and trailing separators are dropped.
func ParsePath(s string) (ServerPath, error) {
	windows := drive(s) || strings.HasPrefix(s, `\`) || strings.Contains(s, `\`)
	if !windows {
		if !strings.HasPrefix(s, "/") {
			return ServerPath{}, fmt.Errorf("%w: %q", ErrNotAbsolute, s)
		}
		return ServerPath{s: "/" + strings.Join(segments(s, "/"), "/")}, nil
	}
	w := strings.ReplaceAll(s, "/", `\`)
	switch {
	case drive(w) && (len(w) == 2 || w[2] == '\\'):
		return ServerPath{s: join(w[:2], segments(w[2:], `\`)), windows: true}, nil
	case strings.HasPrefix(w, `\\`):
		segs := segments(w, `\`)
		if len(segs) < 2 {
			return ServerPath{}, fmt.Errorf("%w: %q names no share", ErrNotAbsolute, s)
		}
		return ServerPath{s: `\\` + strings.Join(segs, `\`), windows: true}, nil
	}
	return ServerPath{}, fmt.Errorf("%w: %q has no drive letter or share", ErrNotAbsolute, s)
}

func drive(s string) bool {
	return len(s) >= 2 && s[1] == ':' && 'a' <= s[0]|0x20 && s[0]|0x20 <= 'z'
}

func segments(s, sep string) []string {
	var out []string
	for seg := range strings.SplitSeq(s, sep) {
		if seg != "" {
			out = append(out, seg)
		}
	}
	return out
}

func join(volume string, segs []string) string {
	return volume + `\` + strings.Join(segs, `\`)
}

func (p ServerPath) IsZero() bool { return p.s == "" }

func (p ServerPath) Windows() bool { return p.windows }

func (p ServerPath) String() string { return p.s }

// parts is p's volume ("" for POSIX, "D:" or `\\nas\share`) and its
// segments under it.
func (p ServerPath) parts() (volume string, segs []string) {
	switch {
	case !p.windows:
		return "", segments(p.s, "/")
	case strings.HasPrefix(p.s, `\\`):
		all := segments(p.s, `\`)
		return `\\` + all[0] + `\` + all[1], all[2:]
	default:
		return p.s[:2], segments(p.s[2:], `\`)
	}
}

func (p ServerPath) from(volume string, segs []string) ServerPath {
	switch {
	case !p.windows:
		return ServerPath{s: "/" + strings.Join(segs, "/")}
	case strings.HasPrefix(volume, `\\`) && len(segs) == 0:
		return ServerPath{s: volume, windows: true}
	default:
		return ServerPath{s: join(volume, segs), windows: true}
	}
}

// Join is the root-relative path r under folder p, with p's separator.
func (p ServerPath) Join(r storing.Rel) ServerPath {
	volume, segs := p.parts()
	return p.from(volume, append(segs, strings.Split(r.String(), "/")...))
}

// Dir is p's folder. A volume's root is its own folder.
func (p ServerPath) Dir() ServerPath {
	volume, segs := p.parts()
	if len(segs) == 0 {
		return p
	}
	return p.from(volume, segs[:len(segs)-1])
}

// Key is p as the path index and every comparison see it: case folded on
// Windows.
func (p ServerPath) Key() string {
	if p.windows {
		return strings.ToLower(p.s)
	}
	return p.s
}

// Within is whether p is folder or sits under it, segment by segment.
func (p ServerPath) Within(folder ServerPath) bool {
	_, ok := p.under(folder)
	return ok
}

// Rel is p relative to folder, the inverse of Join. It is false when p
// isn't under folder, folder itself included.
func (p ServerPath) Rel(folder ServerPath) (storing.Rel, bool) {
	segs, ok := p.under(folder)
	if !ok || len(segs) == 0 {
		return storing.Rel{}, false
	}
	r, err := storing.ParseRel(strings.Join(segs, "/"))
	return r, err == nil
}

// under is p's segments below folder, false when p isn't folder or under
// it.
func (p ServerPath) under(folder ServerPath) ([]string, bool) {
	if p.windows != folder.windows {
		return nil, false
	}
	pv, ps := p.parts()
	fv, fs := folder.parts()
	if len(ps) < len(fs) || !p.same(pv, fv) {
		return nil, false
	}
	for i, seg := range fs {
		if !p.same(ps[i], seg) {
			return nil, false
		}
	}
	return ps[len(fs):], true
}

func (p ServerPath) same(a, b string) bool {
	if p.windows {
		return strings.EqualFold(a, b)
	}
	return a == b
}
