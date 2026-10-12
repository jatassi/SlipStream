// Package changescope decides how much CI runs for a change.
package changescope

import (
	"fmt"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/nativetouch"
)

// Scope is ordered by how much CI runs for it.
type Scope int

const (
	Docs Scope = iota
	PR
	Native
)

var names = []string{"docs", "pr", "native"}

func (s Scope) String() string { return names[s] }

func (s Scope) MarshalText() ([]byte, error) { return []byte(s.String()), nil }

func (s *Scope) UnmarshalText(text []byte) error {
	i := slices.Index(names, string(text))
	if i < 0 {
		return fmt.Errorf("scope %q is not one of %s", text, strings.Join(names, ", "))
	}
	*s = Scope(i)
	return nil
}

func Of(changes []nativetouch.Change) Scope {
	switch {
	case len(nativetouch.Matches(changes)) > 0:
		return Native
	case len(changes) > 0 && len(NotInert(changes)) == 0:
		return Docs
	default:
		return PR
	}
}

// inertRoots name paths no CI check reads, and a path belongs here only while
// that holds. docs/parity-matrix.md is the exception: the parity-owners lint
// reads it.
var inertRoots = []string{"docs/", ".agents/", ".claude/", ".grok/", ".impeccable/", "brand/"}

func inert(p string) bool {
	if p == "docs/parity-matrix.md" {
		return false
	}
	if !strings.Contains(p, "/") && strings.HasSuffix(p, ".md") {
		return true
	}
	for _, root := range inertRoots {
		if strings.HasPrefix(p, root) {
			return true
		}
	}
	return false
}

// NotInert lists the changed paths, on either side, that some CI check reads.
func NotInert(changes []nativetouch.Change) []string {
	var paths []string
	for _, c := range changes {
		for _, s := range []nativetouch.Side{c.Old, c.New} {
			if s.Path != "" && !inert(s.Path) && !slices.Contains(paths, s.Path) {
				paths = append(paths, s.Path)
			}
		}
	}
	return paths
}
