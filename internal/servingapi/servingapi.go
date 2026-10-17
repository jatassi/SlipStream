// Package servingapi is what a Worker says about itself to anyone who
// reaches it: the Worker page's facts, and the status card a Serving
// Worker shows while its Controller can't be reached. It never names what
// anyone requested.
package servingapi

import (
	"context"
	"net/http"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/role"
)

// Facts is a Worker as it sees itself. Since is zero while its Controller
// can be reached.
type Facts struct {
	Name        string
	Fingerprint string
	Controller  string
	Since       time.Time
	Holding     []role.Name
	Seeding     int
}

// Worker is the running Node as getServing reads it.
type Worker interface {
	// Worker is this Node's Facts, false unless it is a Worker.
	Worker() (Facts, bool)
}

var notAWorker = api.NewCode(http.StatusConflict, "not_a_worker", "This Node is not a Worker")

type worker struct {
	Name        string     `json:"name" doc:"This Worker's name"`
	Fingerprint string     `json:"fingerprint" doc:"This Worker's key fingerprint, as its Controller lists it"`
	Controller  string     `json:"controller" doc:"Its Controller's name"`
	Since       *time.Time `json:"since,omitempty" doc:"Since when this Worker can't reach its Controller; absent while it can"`
	Holding     []string   `json:"holding" nullable:"false" doc:"The labels of the Roles it holds beside Serving, in table order"`
	Seeding     int        `json:"seeding" doc:"How many torrents it seeds"`
}

type workerOutput struct {
	Body worker
}

func Routes(r *api.Router, w Worker) {
	o := r.Owner("serving", api.Public)

	api.Register(o, api.Public, api.Op{
		ID:      "getServing",
		Method:  http.MethodGet,
		Path:    "/serving",
		Summary: "Report this Worker's name, key and Controller, and what it holds and seeds; a Serving Worker answers it itself",
		Status:  http.StatusOK,
		Errors:  []*api.Code{notAWorker},
	}, func(context.Context, identity.Anyone, *struct{}) (*workerOutput, error) {
		f, ok := w.Worker()
		if !ok {
			return nil, notAWorker
		}
		out := worker{Name: f.Name, Fingerprint: f.Fingerprint, Controller: f.Controller, Holding: []string{}, Seeding: f.Seeding}
		if !f.Since.IsZero() {
			out.Since = &f.Since
		}
		for _, r := range f.Holding {
			out.Holding = append(out.Holding, r.Label())
		}
		return &workerOutput{Body: out}, nil
	})
}
