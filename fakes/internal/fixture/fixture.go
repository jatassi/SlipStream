// Package fixture is the fake a new boundary's fake starts from: copy this
// package to fakes/<boundary>, rename it, and fill in the handler.
package fixture

import (
	"context"
	"net/http"
	"sync"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/node"
)

var Fake = fake.Spec{
	Name: "fixture",
	TLS:  fake.FrontTLS,
	New: func(_ scenario.Scenario, env fake.Env) (fake.Server, error) {
		visits, err := env.Capture("visits")
		if err != nil {
			return nil, err
		}
		g := &greeting{text: "fixture"}
		if err := fake.OnCue(env, Greet.Name, g.set); err != nil {
			return nil, err
		}
		return fake.HTTP(env, fake.Routes{"whoami": {Pattern: "GET /whoami", Handler: func(w http.ResponseWriter, r *http.Request) {
			visits.Record(r.Context(), visit{Path: r.URL.Path}, fake.File{Name: "user-agent.txt", Data: []byte(r.UserAgent())})
			_, _ = w.Write([]byte(g.get()))
		}}}), nil
	},
	Endpoints: func(*node.EndpointsBlock, fake.Address) {},
	Online:    fake.NoOnlineRun("a test fixture serves no real service"),
	Captures:  []fake.CaptureSpec{{Name: "visits", Kind: "fixture.visit"}},
	Cues:      []fake.CueSpec{Greet},
}

// Greet changes what whoami answers from now on.
var Greet = fake.CueSpec{Name: "fixture-greet"}

type GreetArgs struct {
	Text string `json:"text"`
}

type greeting struct {
	mu   sync.Mutex
	text string
}

func (g *greeting) get() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.text
}

// set plays Greet and answers the greeting it replaced.
func (g *greeting) set(_ context.Context, a GreetArgs) (any, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	was := g.text
	g.text = a.Text
	return struct {
		Was string `json:"was"`
	}{was}, nil
}

type visit struct {
	Path string `json:"path"`
}
