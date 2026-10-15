package install

import (
	"fmt"
	"path"
	"strings"
)

// linuxDataName is DataName as Linux spells a per-user folder.
var linuxDataName = strings.ToLower(DataName)

// Layout is the process facts a Kind's paths derive from, gathered once at
// the process boundary, so every OS's rules run on any host.
type Layout struct {
	GOOS string
	// Executable is the symlink-resolved path of the running binary.
	Executable string
	Environ    []string
}

// Getenv is the last value of name in l's environment, or "".
func (l Layout) Getenv(name string) string {
	var v string
	for _, kv := range l.Environ {
		if value, ok := strings.CutPrefix(kv, name+"="); ok {
			v = value
		}
	}
	return v
}

// under joins elems onto the absolute directory the variable name holds.
func (l Layout) under(name string, elems ...string) (string, error) {
	dir := l.Getenv(name)
	if !l.isAbs(dir) {
		return "", fmt.Errorf("%s is %q, not an absolute path", name, dir)
	}
	return l.join(append([]string{dir}, elems...)...), nil
}

// The path helpers follow l.GOOS rather than the host, accepting either
// separator on Windows.

func (l Layout) toSlash(p string) string {
	if l.GOOS == "windows" {
		return strings.ReplaceAll(p, `\`, "/")
	}
	return p
}

func (l Layout) fromSlash(p string) string {
	if l.GOOS == "windows" {
		return strings.ReplaceAll(p, "/", `\`)
	}
	return p
}

func (l Layout) join(elems ...string) string {
	for i, e := range elems {
		elems[i] = l.toSlash(e)
	}
	return l.fromSlash(path.Join(elems...))
}

func (l Layout) dir(p string) string { return l.fromSlash(path.Dir(l.toSlash(p))) }

func (l Layout) base(p string) string { return path.Base(l.toSlash(p)) }

func (l Layout) isAbs(p string) bool {
	if l.GOOS != "windows" {
		return strings.HasPrefix(p, "/")
	}
	p = l.toSlash(p)
	return strings.HasPrefix(p, "//") || (len(p) >= 3 && p[1] == ':' && p[2] == '/')
}
