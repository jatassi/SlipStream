package imagecdn

import (
	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/internal/artwork"
	"github.com/jatassi/slipstream/fakes/scenario"
)

// WithStore is the fake serving store in place of the checked-in one.
func WithStore(store artwork.Store) fake.Spec {
	spec := Fake
	spec.New = func(_ scenario.Scenario, env fake.Env) (fake.Server, error) { return serve(store, env), nil }
	return spec
}
