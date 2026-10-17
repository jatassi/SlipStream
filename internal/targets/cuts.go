package targets

import (
	"maps"
	"slices"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/works"
)

// ListedCut is a cut an Edition list can name, and the Content Type that
// adds it, empty for a built-in cut.
type ListedCut struct {
	release.Cut
	Type works.TypeID
}

// ListCuts is every built-in cut, then each Content Type's own, by type.
func (t *Targets) ListCuts(identity.Admin) []ListedCut {
	var out []ListedCut
	for _, c := range release.BuiltIn() {
		out = append(out, ListedCut{Cut: c})
	}
	for _, id := range slices.Sorted(maps.Keys(t.cfg.Types)) {
		for _, c := range t.cfg.Types[id].Cuts() {
			out = append(out, ListedCut{Cut: c, Type: id})
		}
	}
	return out
}
