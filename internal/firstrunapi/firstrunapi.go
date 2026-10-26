// Package firstrunapi serves the Setup code check and first setup's
// steps.
package firstrunapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/firstrun"
	"github.com/jatassi/slipstream/internal/identity"
)

type Guide interface {
	// FirstRun is first setup, false unless the Node is a Controller.
	FirstRun() (*firstrun.FirstRun, bool)
}

func open(g Guide) *firstrun.FirstRun {
	f, ok := g.FirstRun()
	if !ok {
		panic("firstrunapi: an operation ran on a Node that is no Controller")
	}
	return f
}

var (
	stepUnknown      = api.NewCode(http.StatusNotFound, "step_unknown", "No first setup step has that id")
	firstRunFinished = api.NewCode(http.StatusConflict, "first_run_finished", "First setup is already finished")
)

func answer(err error) error {
	switch {
	case errors.Is(err, firstrun.ErrStepUnknown):
		return stepUnknown
	case errors.Is(err, firstrun.ErrFinished):
		return firstRunFinished
	}
	return err
}

// stepID is firstrun.IDs as an enum, so the web's union of step ids follows
// the server's list.
type stepID struct{}

func (stepID) Values() []string {
	ids := firstrun.IDs()
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = string(id)
	}
	return out
}

type firstRun struct {
	Open  bool        `json:"open" doc:"False once an Admin finished first setup, which never reopens"`
	Steps []stepState `json:"steps" nullable:"false" doc:"The shown steps, in order"`
}

type stepState struct {
	ID    api.Enum[stepID] `json:"id"`
	State string           `json:"state" enum:"todo,done,skipped"`
}

type firstRunOutput struct{ Body firstRun }

func firstRunOf(v firstrun.View) *firstRunOutput {
	out := firstRun{Open: v.Open, Steps: make([]stepState, len(v.Steps))}
	for i, s := range v.Steps {
		out.Steps[i] = stepState{ID: api.NewEnum[stepID](string(s.ID)), State: string(s.State)}
	}
	return &firstRunOutput{Body: out}
}

type firstRunStepInput struct {
	ID   string `path:"id"`
	Body struct {
		Outcome string `json:"outcome" enum:"done,skipped" doc:"done for Continue, skipped for Skip"`
	}
}

func Routes(r *api.Router, g Guide) {
	o := r.Owner("firstrun", api.Setup)

	api.Register(o, api.Setup, api.Op{
		ID:      "checkSetupCode",
		Method:  http.MethodPost,
		Path:    "/setup/code",
		Summary: "Check the Setup code without changing anything",
		Status:  http.StatusNoContent,
	}, func(context.Context, identity.Claim, *struct{}) (*struct{}, error) { return nil, nil })

	api.Register(o, api.Admin, api.Op{
		ID:      "getFirstRun",
		Method:  http.MethodGet,
		Path:    "/first-run",
		Summary: "Get first setup's steps",
		Status:  http.StatusOK,
	}, func(ctx context.Context, _ identity.Admin, _ *struct{}) (*firstRunOutput, error) {
		v, err := open(g).View(ctx)
		if err != nil {
			return nil, err
		}
		return firstRunOf(v), nil
	})

	api.Register(o, api.Admin, api.Op{
		ID:      "setFirstRunStep",
		Method:  http.MethodPut,
		Path:    "/first-run/steps/{id}",
		Summary: "Mark a first setup step done or skipped",
		Status:  http.StatusOK,
		Errors:  []*api.Code{stepUnknown, firstRunFinished},
	}, func(ctx context.Context, _ identity.Admin, in *firstRunStepInput) (*firstRunOutput, error) {
		v, err := open(g).Set(ctx, firstrun.StepID(in.ID), firstrun.Outcome(in.Body.Outcome))
		if err != nil {
			return nil, answer(err)
		}
		return firstRunOf(v), nil
	})

	api.Register(o, api.Admin, api.Op{
		ID:      "finishFirstRun",
		Method:  http.MethodPost,
		Path:    "/first-run/finish",
		Summary: "Finish first setup for good",
		Status:  http.StatusNoContent,
	}, func(ctx context.Context, _ identity.Admin, _ *struct{}) (*struct{}, error) {
		return nil, open(g).Finish(ctx)
	})
}
