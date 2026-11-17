// Package capturescript is fake-capture-script: SlipStream's own stand-in
// for the program an Admin names on a Custom Script connection. The
// capture-script binary reports each run to this fake's front, which
// keeps it as a script capture: its arguments, every variable it got and
// the event file NOTIFICATION_EVENT_FILE named.
package capturescript

import (
	"encoding/json"
	"net/http"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/node"
)

// Script is the capture of every run the binary reported, in script.run
// records whose node is the origin of the report.
var Script = fake.CaptureSpec{Name: "script", Kind: "script.run"}

// Fake is the front capture-script reports to. Nothing in a Node's
// config points at it: the binary's --report flag names it.
var Fake = fake.Spec{
	Name:      "capture-script",
	New:       newServer,
	Endpoints: func(*node.EndpointsBlock, fake.Address) {},
	Online:    fake.NoOnlineRun("it is SlipStream's own stand-in for an Admin's script, and there is no real one to check against"),
	Captures:  []fake.CaptureSpec{Script},
}

// Run is one run of the binary as it saw it, with the exit it then made.
type Run struct {
	Argv  []string          `json:"argv"`
	Env   map[string]string `json:"env"`
	Event json.RawMessage   `json:"event"`
	Exit  int               `json:"exit"`
}

func newServer(_ scenario.Scenario, env fake.Env) (fake.Server, error) {
	runs, err := env.Capture(Script.Name)
	if err != nil {
		return nil, err
	}
	return fake.HTTP(env, fake.Routes{
		"runs": {Pattern: "POST /runs", Handler: func(w http.ResponseWriter, r *http.Request) {
			var run Run
			if err := json.NewDecoder(r.Body).Decode(&run); err != nil {
				http.Error(w, "The body isn't a JSON run.", http.StatusBadRequest)
				return
			}
			runs.Record(r.Context(), run, fake.File{Name: "event.json", Data: run.Event})
			w.WriteHeader(http.StatusNoContent)
		}},
	}), nil
}
