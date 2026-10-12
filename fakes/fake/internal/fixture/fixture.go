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
	New: func(_ scenario.Scenario, _ fake.Env) (fake.Server, error) {
		return fake.HTTP(handler()), nil
	},
	Endpoints: func(*node.EndpointsBlock, fake.Address) {},
}

func handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /whoami", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("fixture"))
	})
	return mux
}
