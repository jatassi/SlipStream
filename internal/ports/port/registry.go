package port

import (
	"cmp"
	"fmt"
	"maps"
	"slices"

	"github.com/jatassi/slipstream/internal/searching"
)

// Registry is every port the binary carries, plus the retired and moved
// records. internal/ports/all builds the one Registry. The Controller
// reads it as a Catalog and the Node runs it through Driver. Both halves
// are in the same binary and Nodes run in lockstep (ADR 0003), so they
// never disagree.
type Registry struct {
	byID    map[ID]Port
	retired map[ID]Retired
	moved   []Moved
	// hidden maps a catalog id to the port that replaces it.
	hidden map[string]ID
}

// NewRegistry panics on a repeated id, a Replaces or Aliases id two ports
// claim, or a retired or moved id a live port still has.
func NewRegistry(ports []Port, retired []Retired, moved []Moved) Registry {
	r := Registry{byID: map[ID]Port{}, retired: map[ID]Retired{}, moved: slices.Clone(moved), hidden: map[string]ID{}}
	claimed := map[string]ID{}
	claim := func(key string, by ID) {
		if other, ok := claimed[key]; ok {
			panic(fmt.Sprintf("port: %q and %q both claim %q", other, by, key))
		}
		claimed[key] = by
	}
	for _, p := range ports {
		id := p.entry.ID
		claim("port:"+string(id), id)
		r.byID[id] = p
		for _, c := range slices.Concat(p.entry.Replaces, p.entry.Aliases) {
			claim("catalog:"+c, id)
		}
		for _, c := range p.entry.Replaces {
			r.hidden[c] = id
		}
	}
	for _, x := range retired {
		claim("port:"+string(x.ID), x.ID)
		r.retired[x.ID] = x
	}
	for _, m := range moved {
		claim("port:"+string(m.ID), m.ID)
	}
	return r
}

// Catalog is the Controller's view: Entries only, no Impl.
type Catalog interface {
	// Entries are the live ports, by name.
	Entries() []Entry
	Lookup(id ID) (Entry, bool)
	Retired(id ID) (Retired, bool)
	// Hiding is the port that replaces catalog id, if any.
	Hiding(catalogID string) (ID, bool)
	Moved() []Moved
}

func (r Registry) Entries() []Entry {
	out := make([]Entry, 0, len(r.byID))
	for _, id := range slices.Sorted(maps.Keys(r.byID)) {
		out = append(out, r.byID[id].entry)
	}
	slices.SortStableFunc(out, func(a, b Entry) int { return cmp.Compare(a.Name, b.Name) })
	return out
}

func (r Registry) Lookup(id ID) (Entry, bool) {
	p, ok := r.byID[id]
	return p.entry, ok
}

func (r Registry) Retired(id ID) (Retired, bool) {
	x, ok := r.retired[id]
	return x, ok
}

func (r Registry) Hiding(catalogID string) (ID, bool) {
	id, ok := r.hidden[catalogID]
	return id, ok
}

func (r Registry) Moved() []Moved { return slices.Clone(r.moved) }

// Driver is the native kind's searching.Driver over r. internal/node
// passes it as searching.Config.Native.
func Driver(r Registry) searching.Driver { return driver{r: r} }
