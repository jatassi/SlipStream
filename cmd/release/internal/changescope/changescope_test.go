package changescope

import (
	"reflect"
	"testing"

	"github.com/jatassi/slipstream/cmd/release/internal/nativetouch"
)

func added(path string) nativetouch.Change {
	return nativetouch.Change{New: nativetouch.Side{Path: path}}
}

func modified(path string) nativetouch.Change {
	return nativetouch.Change{Old: nativetouch.Side{Path: path}, New: nativetouch.Side{Path: path}}
}

func TestOf(t *testing.T) {
	tests := []struct {
		name         string
		changes      []nativetouch.Change
		want         Scope
		wantNotInert []string
	}{
		{name: "no changes", want: PR},
		{
			name: "docs and agent files",
			changes: []nativetouch.Change{
				modified("docs/release.md"),
				modified("README.md"),
				added(".agents/skills/x/SKILL.md"),
				added(".claude/rules/x.md"),
				added(".grok/skills/x.md"),
			},
			want: Docs,
		},
		{
			name:         "design sources are read by checks",
			changes:      []nativetouch.Change{modified("DESIGN.md"), modified(".impeccable/design.json"), added("brand/logo.svg")},
			want:         PR,
			wantNotInert: []string{"DESIGN.md", ".impeccable/design.json", "brand/logo.svg"},
		},
		{
			name:         "parity matrix is read by a lint",
			changes:      []nativetouch.Change{modified("docs/parity-matrix.md")},
			want:         PR,
			wantNotInert: []string{"docs/parity-matrix.md"},
		},
		{
			name:         "vendored skill is read by a lint",
			changes:      []nativetouch.Change{modified(".claude/skills/anatomy/kit/tube.mjs"), modified(".claude/skills/pr/SKILL.md")},
			want:         PR,
			wantNotInert: []string{".claude/skills/anatomy/kit/tube.mjs"},
		},
		{
			name:         "rename out of docs",
			changes:      []nativetouch.Change{{Old: nativetouch.Side{Path: "docs/notes.go"}, New: nativetouch.Side{Path: "internal/notes.go"}}},
			want:         PR,
			wantNotInert: []string{"internal/notes.go"},
		},
		{
			name:         "nested readme",
			changes:      []nativetouch.Change{modified("README.md"), modified("web/README.md")},
			want:         PR,
			wantNotInert: []string{"web/README.md"},
		},
		{
			name:         "root non-markdown file",
			changes:      []nativetouch.Change{modified("skills-lock.json")},
			want:         PR,
			wantNotInert: []string{"skills-lock.json"},
		},
		{
			name:         "workflow",
			changes:      []nativetouch.Change{modified(".github/workflows/pr.yml")},
			want:         Native,
			wantNotInert: []string{".github/workflows/pr.yml"},
		},
		{
			name:         "docs with a native change",
			changes:      []nativetouch.Change{modified("docs/release.md"), modified("native/recipe/boost.cmake")},
			want:         Native,
			wantNotInert: []string{"native/recipe/boost.cmake"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Of(tt.changes); got != tt.want {
				t.Errorf("Of = %s, want %s", got, tt.want)
			}
			if got := NotInert(tt.changes); !reflect.DeepEqual(got, tt.wantNotInert) {
				t.Errorf("NotInert = %v, want %v", got, tt.wantNotInert)
			}
		})
	}
}

func TestPackaging(t *testing.T) {
	for _, c := range []struct {
		name    string
		changes []nativetouch.Change
		want    bool
	}{
		{"elsewhere", []nativetouch.Change{modified("internal/works/works.go"), modified("cmd/release/plan.go")}, false},
		{"an install kind", []nativetouch.Change{modified("internal/install/kind.go")}, true},
		{"the tray", []nativetouch.Change{added("internal/node/internal/desktop/tray_windows.go")}, true},
		{"the trampoline", []nativetouch.Change{modified("cmd/slipstream-app/main.go")}, true},
		{"a template", []nativetouch.Change{modified("cmd/release/internal/pack/templates/SlipStream.wxs")}, true},
		{"a smoke", []nativetouch.Change{modified("cmd/release/internal/smoke/linux.go")}, true},
		{"the package command", []nativetouch.Change{modified("cmd/release/package.go")}, true},
		{"the WiX pin", []nativetouch.Change{modified(".config/dotnet-tools.json")}, true},
		{"a tool pin", []nativetouch.Change{modified("native/inputs.json")}, true},
	} {
		if got := Packaging(c.changes); got != c.want {
			t.Errorf("%s: Packaging = %t, want %t", c.name, got, c.want)
		}
	}
}

func TestDiagram(t *testing.T) {
	for _, c := range []struct {
		name    string
		changes []nativetouch.Change
		want    bool
	}{
		{"elsewhere", []nativetouch.Change{modified("web/src/routes/_app/settings.tsx"), modified("internal/works/works.go")}, false},
		{"a part", []nativetouch.Change{modified("web/src/diagram/parts/dish.tsx")}, true},
		{"the harness page", []nativetouch.Change{modified("web/src/diagram/harness/index.html")}, true},
		{"the Nodes landing", []nativetouch.Change{modified("web/src/routes/_app/nodes/-landing/view.ts")}, true},
		{"the check", []nativetouch.Change{modified("web/scripts/diagram-check.ts")}, true},
		{"the anatomy skill", []nativetouch.Change{modified(".claude/skills/anatomy/scripts/line-ends.mjs")}, true},
		{"the lockfile", []nativetouch.Change{modified("web/bun.lock")}, true},
		{"a package patch", []nativetouch.Change{modified("web/patches/msw@3.0.3.patch")}, true},
		{"a rename out of the Diagram", []nativetouch.Change{{Old: nativetouch.Side{Path: "web/src/diagram/x.ts"}, New: nativetouch.Side{Path: "web/src/x.ts"}}}, true},
	} {
		if got := Diagram(c.changes); got != c.want {
			t.Errorf("%s: Diagram = %t, want %t", c.name, got, c.want)
		}
	}
}
