// Package contenttype is the contract every Content Type implements and
// the registration a main hands to node.Main.
//
// # Adding a Content Type
//
// Copy fakes/contenttypes/music, the test-only music type, into
// internal/contenttypes/<id> and rename it. A Content Type is one package
// with one Define call:
//
//	var Def = contenttype.Define("music", schemaFiles, DefaultEndpoints, New)
//
// and one value implementing Type, built by New from Deps and the loaded
// Endpoints section. Its own tables live in its own migration set, named
// by its id, with foreign keys onto works' ids; its metadata never sits in
// a core row. Register it by adding Def to both mains:
//
//	os.Exit(node.Main(os.Args, music.Def))
//
// # Adding a capability
//
// A later spec defines its facet interface in the core package that calls
// it and embeds it in Type. Every registered type, music and
// contenttypetest.Fixture included, then fails to compile until it
// implements the facet. There are no optional interfaces: a type lacking a
// capability implements the facet with the owner's documented empty
// answer.
package contenttype

import (
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"regexp"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/providers/tmdb"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// Type is a Content Type: the union of every facet the core asks of one,
// each defined by the package that consumes it.
type Type interface {
	works.Catalog
	naming.Facet
	indexers.Facet
	targets.Facet
}

type Deps struct {
	// HTTP dials through the Node's network, trusting the OS roots.
	HTTP *http.Client
	// UserAgent is sent on every request to a metadata source.
	UserAgent string
	// TMDB is the one TMDB client the Node builds from endpoints.tmdb; a
	// type that doesn't read TMDB ignores it.
	TMDB *tmdb.Client
	Log  *slog.Logger
}

type Registration interface {
	ID() works.TypeID
	// Schema is the type's library set, false when it keeps no tables.
	Schema() (storage.Schema[storage.Library], bool)
	// Endpoints is a fresh pointer to the type's section holding its
	// defaults, nil when it reaches no host.
	Endpoints() any
	// Open builds the Type from deps and the section Endpoints returned,
	// once loaded. It does no IO.
	Open(deps Deps, endpoints any) Type
}

type Def[E any] struct {
	id               works.TypeID
	schema           *storage.Schema[storage.Library]
	defaultEndpoints func() E
	open             func(Deps, E) Type
}

var idShape = regexp.MustCompile(`^[a-z][a-z0-9]{1,15}$`)

// Define registers a Content Type. id is fixed forever: it is the type's
// migration set, its config key and the API's contentType. migrations
// holds schema/NNNNN_slug.sql, nil when the type keeps no tables;
// defaultEndpoints is nil when it reaches no host. Define panics on a bad
// id.
func Define[E any](id string, migrations fs.FS, defaultEndpoints func() E, open func(Deps, E) Type) Def[E] {
	if !idShape.MatchString(id) {
		panic(fmt.Sprintf("contenttype: id %q is not %s", id, idShape))
	}
	if id == "goose" || id == "sqlite" {
		panic(fmt.Sprintf("contenttype: id %s names a set storage reserves", id))
	}
	d := Def[E]{id: works.TypeID(id), defaultEndpoints: defaultEndpoints, open: open}
	if migrations != nil {
		s := storage.NewSchema[storage.Library](id, migrations)
		d.schema = &s
	}
	return d
}

func (d Def[E]) ID() works.TypeID { return d.id }

func (d Def[E]) Schema() (storage.Schema[storage.Library], bool) {
	if d.schema == nil {
		return storage.Schema[storage.Library]{}, false
	}
	return *d.schema, true
}

func (d Def[E]) Endpoints() any {
	if d.defaultEndpoints == nil {
		return nil
	}
	e := d.defaultEndpoints()
	return &e
}

func (d Def[E]) Open(deps Deps, endpoints any) Type {
	var e E
	if endpoints != nil {
		loaded, ok := endpoints.(*E)
		if !ok {
			panic(fmt.Sprintf("contenttype: %s opened with endpoints %T, not the *%T its Endpoints returns", d.id, endpoints, e))
		}
		e = *loaded
	}
	return d.open(deps, e)
}

// Set is the Content Types a main registers, in id order.
type Set struct {
	defs []Registration
}

// NewSet panics when two registrations share an id.
func NewSet(defs ...Registration) Set {
	sorted := slices.SortedFunc(slices.Values(defs), func(a, b Registration) int { return strings.Compare(string(a.ID()), string(b.ID())) })
	for i := 1; i < len(sorted); i++ {
		if sorted[i].ID() == sorted[i-1].ID() {
			panic(fmt.Sprintf("contenttype: %s is registered twice", sorted[i].ID()))
		}
	}
	return Set{defs: sorted}
}

func (s Set) All() []Registration { return slices.Clone(s.defs) }

func (s Set) IDs() []works.TypeID {
	ids := make([]works.TypeID, len(s.defs))
	for i, d := range s.defs {
		ids[i] = d.ID()
	}
	return ids
}

func (s Set) Schemas() []storage.Schema[storage.Library] {
	var out []storage.Schema[storage.Library]
	for _, d := range s.defs {
		if schema, ok := d.Schema(); ok {
			out = append(out, schema)
		}
	}
	return out
}

func Project[F any](types map[works.TypeID]Type, pick func(Type) F) map[works.TypeID]F {
	out := make(map[works.TypeID]F, len(types))
	for id, t := range types {
		out[id] = pick(t)
	}
	return out
}
