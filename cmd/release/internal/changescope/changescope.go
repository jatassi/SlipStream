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
// that holds. readByChecks are the exceptions, a path or a folder ending in /:
// the parity-owners lint reads docs/parity-matrix.md, the tokens generated and
// contrast checks read DESIGN.md, and the vendor-synced lint reads the
// vendored anatomy skill.
var (
	inertRoots   = []string{"docs/", ".agents/", ".claude/", ".grok/"}
	readByChecks = []string{"docs/parity-matrix.md", "DESIGN.md", ".claude/skills/anatomy/"}
)

func inert(p string) bool {
	if slices.ContainsFunc(readByChecks, func(read string) bool {
		return p == read || strings.HasSuffix(read, "/") && strings.HasPrefix(p, read)
	}) {
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

// onlineRoots hold the fakes, the clients and the suites the online job
// checks against the real services.
var onlineRoots = []string{"fakes/tmdb/", "fakes/contenttypes/music/", "internal/providers/tmdb/", "conformance/"}

// Online reports whether a change touches, on either side, anything the
// online job checks.
func Online(changes []nativetouch.Change) bool { return touches(changes, onlineRoots) }

// Link reports whether a change touches, on either side, the link the
// link-measure job measures.
func Link(changes []nativetouch.Change) bool { return touches(changes, []string{"internal/link/"}) }

var packagingRoots = []string{
	"internal/install/", "internal/node/internal/desktop/", "cmd/slipstream-app/",
	"cmd/release/internal/pack/", "cmd/release/internal/smoke/", "cmd/release/package.go", "cmd/release/smoke.go",
	".config/dotnet-tools.json", "native/inputs.json", "LICENSE",
}

// Packaging reports whether a change touches, on either side, anything
// the packages are built from.
func Packaging(changes []nativetouch.Change) bool { return touches(changes, packagingRoots) }

// diagramRoots hold what the Diagram check reads: the harness and the
// modules it imports, the scripts, the vendored anatomy kit and skill, and
// the Vite server and packages it runs on. The diagram roots lint holds them
// to the check's imports.
var diagramRoots = []string{
	"web/src/diagram/", "web/src/routes/_app/nodes/", "web/src/nodes/", "web/src/live/",
	"web/src/ui/", "web/src/api/", "web/src/shell/",
	"web/src/format.ts", "web/src/styles.css", "web/src/theme.gen.css",
	"web/scripts/", "web/vendor/anatomy/", ".claude/skills/anatomy/",
	"web/vite.config.ts", "web/package.json", "web/bun.lock", "web/tsconfig.json",
}

// Diagram reports whether a change touches, on either side, anything the
// Diagram check reads.
func Diagram(changes []nativetouch.Change) bool { return touches(changes, diagramRoots) }

func touches(changes []nativetouch.Change, roots []string) bool {
	return slices.ContainsFunc(changes, func(c nativetouch.Change) bool {
		return slices.ContainsFunc(roots, func(root string) bool {
			return strings.HasPrefix(c.Old.Path, root) || strings.HasPrefix(c.New.Path, root)
		})
	})
}
