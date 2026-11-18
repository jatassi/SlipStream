package subdl

import (
	"net/http"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/node"
)

// Requests records every request fake-subdl-dl gets, by path alone: each
// is a divergence, since SlipStream never follows a keyed url.
var Requests = fake.CaptureSpec{Name: "subdl-dl-requests", Kind: "subdl.dl-request"}

// DL is the host SubDL's keyed url and unpack_files[].url fields name,
// dl.subdl.com in production. SlipStream downloads through the v2 route on
// the api host instead, so a request here is a bug.
var DL = fake.Spec{
	Name: "subdl-dl",
	TLS:  fake.FrontTLS,
	New:  newDL,
	// Endpoints points nothing at it: the Node has no endpoint for SubDL's
	// download host.
	Endpoints: func(*node.EndpointsBlock, fake.Address) {},
	Online:    fake.NoOnlineRun("SlipStream never calls SubDL's keyed download host, so there is nothing to check against"),
	Captures:  []fake.CaptureSpec{Requests},
}

type request struct {
	Path string `json:"path"`
}

func newDL(_ scenario.Scenario, env fake.Env) (fake.Server, error) {
	requests, err := env.Capture(Requests.Name)
	if err != nil {
		return nil, err
	}
	return fake.HTTP(env, fake.Routes{
		"any": {Pattern: "/", Handler: func(w http.ResponseWriter, r *http.Request) {
			requests.Record(r.Context(), request{Path: r.URL.Path})
			w.WriteHeader(http.StatusGone)
		}},
	}), nil
}
