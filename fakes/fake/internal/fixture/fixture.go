// Package fixture is the fake a new boundary's fake starts from: copy this
// package to fakes/<boundary>, rename it, and fill in the handler.
package fixture

import (
	"net/http"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/node"
)

var Fake = fake.Spec{
	Name: "fixture",
	TLS:  true,
	New: func(_ scenario.Scenario, env fake.Env) (fake.Server, error) {
		return fake.HTTP(env, fake.Routes{"whoami": {Pattern: "GET /whoami", Handler: whoami}}), nil
	},
	Endpoints: func(*node.EndpointsBlock, fake.Address) {},
}

func whoami(w http.ResponseWriter, _ *http.Request) {
	_, _ = w.Write([]byte("fixture"))
}
