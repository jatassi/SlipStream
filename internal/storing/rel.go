package storing

import (
	"errors"
	"fmt"
	"strings"
)

// Rel is a root-relative, '/'-separated path that can't leave its root:
// no leading '/', no '\', no drive, and no empty, "." or ".." segment. It
// is the only relative path that crosses into storing.
type Rel struct{ s string }

var ErrBadRel = errors.New("not a root-relative path")

func ParseRel(s string) (Rel, error) {
	if s == "" {
		return Rel{}, fmt.Errorf("%w: empty", ErrBadRel)
	}
	if strings.ContainsRune(s, '\\') {
		return Rel{}, fmt.Errorf("%w: %q has a backslash", ErrBadRel, s)
	}
	if drive(s) {
		return Rel{}, fmt.Errorf("%w: %q names a drive", ErrBadRel, s)
	}
	for seg := range strings.SplitSeq(s, "/") {
		if seg == "" || seg == "." || seg == ".." {
			return Rel{}, fmt.Errorf("%w: %q has an empty, . or .. segment", ErrBadRel, s)
		}
	}
	return Rel{s}, nil
}

func (r Rel) String() string { return r.s }

func (r Rel) segments() []string { return strings.Split(r.s, "/") }

// split is rel's folder, the zero Rel for the root, and its name.
func (r Rel) split() (folder Rel, name string) {
	i := strings.LastIndexByte(r.s, '/')
	if i < 0 {
		return Rel{}, r.s
	}
	return Rel{r.s[:i]}, r.s[i+1:]
}

// join is name in the folder r, the zero Rel for the root.
func (r Rel) join(name string) Rel {
	if r.s == "" {
		return Rel{name}
	}
	return Rel{r.s + "/" + name}
}

func drive(s string) bool {
	return len(s) >= 2 && s[1] == ':' && ('a' <= s[0]|0x20 && s[0]|0x20 <= 'z')
}
