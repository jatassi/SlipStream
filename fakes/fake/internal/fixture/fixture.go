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
		visits, err := env.Capture("visits")
		if err != nil {
			return nil, err
		}
		return fake.HTTP(env, fake.Routes{"whoami": {Pattern: "GET /whoami", Handler: func(w http.ResponseWriter, r *http.Request) {
			visits.Record(r.Context(), visit{Path: r.URL.Path}, fake.File{Name: "user-agent.txt", Data: []byte(r.UserAgent())})
			_, _ = w.Write([]byte("fixture"))
		}}}), nil
	},
	Endpoints: func(*node.EndpointsBlock, fake.Address) {},
	Online:    fake.NoOnlineRun("a test fixture serves no real service"),
	Captures:  []fake.CaptureSpec{{Name: "visits", Kind: "fixture.visit"}},
}

type visit struct {
	Path string `json:"path"`
}
