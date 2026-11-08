package sim

import (
	"log/slog"
	"testing"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/omdb"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/tmdb"
	"github.com/jatassi/slipstream/fakes/tvdb"
	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/contenttypes/movie"
	"github.com/jatassi/slipstream/internal/contenttypes/series"
	"github.com/jatassi/slipstream/internal/node"
)

// WithMetadata serves fake-tmdb, fake-omdb and fake-tvdb with the
// Scenario's catalogs and registers the movie and series types on every
// Node, for a World without the walk, such as a first setup's, whose
// Scenario has no library to seed.
func WithMetadata() Option {
	return func(o *options) {
		o.metadata = true
		o.edits = append(o.edits, func(_ string, e *node.Env) { e.Types = contenttype.NewSet(movie.Def, series.Def) })
	}
}

func (f *servers) metadata(t *testing.T, s scenario.Scenario, local *fault.Host, endpoints *node.EndpointsBlock) {
	t.Helper()
	log := slog.New(slog.NewTextHandler(t.Output(), nil))
	for _, spec := range []fake.Spec{tmdb.Fake, omdb.Fake, tvdb.Fake} {
		server, err := spec.New(s, fake.NewEnv(s, spec, log, fault.NewPlan(nil, nil), nil))
		if err != nil {
			t.Fatal(err)
		}
		spec.Endpoints(endpoints, f.serve(t, local, server))
	}
}
