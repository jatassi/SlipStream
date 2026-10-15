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

func TestOnline(t *testing.T) {
	for _, c := range []struct {
		name    string
		changes []nativetouch.Change
		want    bool
	}{
		{"no changes", nil, false},
		{"elsewhere", []nativetouch.Change{modified("internal/works/works.go"), modified("fakes/site/site.go")}, false},
		{"fake-tmdb", []nativetouch.Change{modified("fakes/tmdb/tmdb.go")}, true},
		{"the music client", []nativetouch.Change{modified("fakes/contenttypes/music/internal/musicbrainz/musicbrainz.go")}, true},
		{"the TMDB client", []nativetouch.Change{added("internal/providers/tmdb/tv.go")}, true},
		{"a suite", []nativetouch.Change{modified("conformance/tmdb/shape.json")}, true},
		{"a rename out of a suite", []nativetouch.Change{{Old: nativetouch.Side{Path: "conformance/tmdb/x.go"}, New: nativetouch.Side{Path: "internal/x.go"}}}, true},
	} {
		if got := Online(c.changes); got != c.want {
			t.Errorf("%s: Online = %t, want %t", c.name, got, c.want)
		}
	}
}
