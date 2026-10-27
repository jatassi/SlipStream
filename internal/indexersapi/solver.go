package indexersapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
)

var challengeSolverUnreachable = api.NewCode(http.StatusUnprocessableEntity, "challenge_solver_unreachable", "The Node can't reach the Challenge solver")

type challengeSolver struct {
	Address *string `json:"address" doc:"The address of the service that clears browser challenges for the Node; null when it has none"`
	Timeout int64   `json:"timeout" minimum:"10" maximum:"180" doc:"Seconds the solver may take to clear a page"`
}

type challengeSolverOutput struct{ Body challengeSolver }

type solverChange struct {
	Address string `json:"address" maxLength:"2048" doc:"An http or https address of a service that speaks FlareSolverr's API; empty removes the solver"`
	Timeout *int64 `json:"timeout,omitempty" doc:"Seconds from 10 to 180; null or left out is 60"`
}

type challengeSolverInput struct {
	ID   string `path:"id" format:"uuid"`
	Body solverChange
}

func solverOf(s indexers.Solver) *challengeSolverOutput {
	out := challengeSolver{Timeout: int64(s.Timeout.Seconds())}
	if s.Address != "" {
		out.Address = &s.Address
	}
	return &challengeSolverOutput{Body: out}
}

func nodeOf(raw string) role.NodeID {
	id, err := storage.ParseUUID(raw)
	if err != nil {
		return role.NodeID{}
	}
	return role.NodeID{UUID: id}
}

func (h handlers) getChallengeSolver(ctx context.Context, by identity.Admin, in *idInput) (*challengeSolverOutput, error) {
	xs := open(h.registry)
	s, err := xs.Solver(ctx, by, nodeOf(in.ID))
	if err != nil {
		return nil, answer(xs, err)
	}
	return solverOf(s), nil
}

func (h handlers) setChallengeSolver(ctx context.Context, by identity.Admin, in *challengeSolverInput) (*challengeSolverOutput, error) {
	xs := open(h.registry)
	s, err := xs.SetSolver(ctx, by, nodeOf(in.ID), in.Body.Address, in.Body.Timeout)
	var unreachable *indexers.SolverUnreachableError
	var outOfRange *setting.RangeError
	switch {
	case errors.As(err, &unreachable):
		return nil, challengeSolverUnreachable.Because(unreachable.Sentence())
	case errors.As(err, &outOfRange):
		return nil, api.Invalid.On("timeout", outOfRange.Sentence)
	case err != nil:
		return nil, answer(xs, err)
	}
	return solverOf(s), nil
}
