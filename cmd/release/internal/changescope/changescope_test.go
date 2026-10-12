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
				added(".impeccable/mocks/a.png"),
				added("brand/logo.svg"),
			},
			want: Docs,
		},
		{
			name:         "parity matrix is read by a lint",
			changes:      []nativetouch.Change{modified("docs/parity-matrix.md")},
			want:         PR,
			wantNotInert: []string{"docs/parity-matrix.md"},
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
			changes:      []nativetouch.Change{modified(".github/workflows/ci.yml")},
			want:         Native,
			wantNotInert: []string{".github/workflows/ci.yml"},
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

func TestUnmarshalText(t *testing.T) {
	for _, want := range []Scope{Docs, PR, Native} {
		var got Scope
		if err := got.UnmarshalText([]byte(want.String())); err != nil || got != want {
			t.Errorf("UnmarshalText(%s) = %s, %v", want, got, err)
		}
	}
	var s Scope
	if err := s.UnmarshalText([]byte("all")); err == nil {
		t.Error("UnmarshalText(all) succeeded")
	}
}
