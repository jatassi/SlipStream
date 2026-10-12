// Package nativetouch decides whether a change touches native code, so CI
// knows when to run the targets gated on native changes (spec #126,
// Native-touch rule). This file is the only description of the rule.
package nativetouch

import (
	"bufio"
	"bytes"
	"go/build/constraint"
	"path"
	"strings"
)

// Side is one side of a changed file. A zero Side means the file is absent
// on that side: the old side of an added file or the new side of a deleted one.
type Side struct {
	Path    string
	Content []byte
}

// Change is one changed file. A rename has both paths; both are checked.
type Change struct {
	Old, New Side
}

type Match struct {
	Clause string
	Path   string
}

type Clause struct {
	Name  string
	match func(Change) []string
}

var nativeRoots = []string{
	"internal/torrent/libtorrent/",
	"internal/node/internal/lock/",
	"internal/node/internal/control/",
	"native/",
	"cmd/release/",
}

var cFamilyExts = []string{".c", ".cc", ".cpp", ".h", ".hpp", ".m", ".mm"}

var Clauses = []Clause{
	{"native-root", eachSide(underNativeRoot)},
	{"c-family", eachSide(hasCFamilyExt)},
	{"go-os-arch-suffix", eachSide(hasOSArchSuffix)},
	{"go-os-arch-constraint", eachSide(hasOSArchConstraint)},
	{"go-directive", goDirectiveChanged},
}

func Matches(changes []Change) []Match {
	var matches []Match
	for _, c := range changes {
		for _, clause := range Clauses {
			for _, p := range clause.match(c) {
				matches = append(matches, Match{clause.Name, p})
			}
		}
	}
	return matches
}

func eachSide(pred func(Side) bool) func(Change) []string {
	return func(c Change) []string {
		var paths []string
		for _, s := range []Side{c.Old, c.New} {
			if s.Path != "" && pred(s) && (len(paths) == 0 || paths[0] != s.Path) {
				paths = append(paths, s.Path)
			}
		}
		return paths
	}
}

func underNativeRoot(s Side) bool {
	for _, root := range nativeRoots {
		if strings.HasPrefix(s.Path, root) {
			return true
		}
	}
	return false
}

func hasCFamilyExt(s Side) bool {
	ext := path.Ext(s.Path)
	for _, e := range cFamilyExts {
		if ext == e {
			return true
		}
	}
	return false
}

// hasOSArchSuffix follows go/build's goodOSArchFile: everything up to the
// first underscore is ignored, then a trailing _test is dropped, then the
// name ends in _GOOS, _GOARCH or _GOOS_GOARCH.
func hasOSArchSuffix(s Side) bool {
	name := path.Base(s.Path)
	if path.Ext(name) != ".go" {
		return false
	}
	name = strings.TrimSuffix(name, ".go")
	i := strings.Index(name, "_")
	if i < 0 {
		return false
	}
	parts := strings.Split(name[i:], "_")
	if parts[len(parts)-1] == "test" {
		parts = parts[:len(parts)-1]
	}
	n := len(parts)
	if n >= 2 && knownOS[parts[n-2]] && knownArch[parts[n-1]] {
		return true
	}
	return n >= 1 && (knownOS[parts[n-1]] || knownArch[parts[n-1]])
}

// hasOSArchConstraint reads the //go:build line in the file header, before
// the package clause, and reports whether any tag names an OS or architecture.
func hasOSArchConstraint(s Side) bool {
	if path.Ext(s.Path) != ".go" {
		return false
	}
	sc := bufio.NewScanner(bytes.NewReader(s.Content))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "package ") {
			return false
		}
		if !constraint.IsGoBuild(line) {
			continue
		}
		expr, err := constraint.Parse(line)
		if err != nil {
			return false
		}
		named := false
		expr.Eval(func(tag string) bool {
			named = named || tag == "unix" || knownOS[tag] || knownArch[tag]
			return true
		})
		return named
	}
	return false
}

type goDirectives struct {
	Go, Toolchain string
}

func goDirectiveChanged(c Change) []string {
	if c.Old.Path != "go.mod" && c.New.Path != "go.mod" {
		return nil
	}
	if parseGoDirectives(c.Old) == parseGoDirectives(c.New) {
		return nil
	}
	return []string{"go.mod"}
}

func parseGoDirectives(s Side) goDirectives {
	var d goDirectives
	if s.Path != "go.mod" {
		return d
	}
	for line := range strings.Lines(string(s.Content)) {
		line, _, _ = strings.Cut(line, "//")
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		switch fields[0] {
		case "go":
			d.Go = fields[1]
		case "toolchain":
			d.Toolchain = fields[1]
		}
	}
	return d
}
