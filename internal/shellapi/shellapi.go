// Package shellapi is what the app shell reads on every signed-in load:
// the counts beside its navigation.
package shellapi

import (
	"context"
	"net/http"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
)

type Pipeline interface {
	// Grabs is the grab pipeline, false unless the Node is a Controller.
	Grabs() (*grabs.Grabs, bool)
}

type counts struct {
	Activity int `json:"activity" minimum:"0" doc:"The Grabs neither imported nor failed"`
}

type countsOutput struct{ Body counts }

func Routes(r *api.Router, p Pipeline) {
	o := r.Owner("shell", api.Admin)

	api.Register(o, api.Admin, api.Op{
		ID:      "getCounts",
		Method:  http.MethodGet,
		Path:    "/counts",
		Summary: "Get the counts the shell shows beside its navigation",
		Status:  http.StatusOK,
	}, func(ctx context.Context, by identity.Admin, _ *struct{}) (*countsOutput, error) {
		g, ok := p.Grabs()
		if !ok {
			panic("shellapi: getCounts ran on a Node that is no Controller")
		}
		n, err := g.Moving(ctx, by)
		if err != nil {
			return nil, err
		}
		return &countsOutput{Body: counts{Activity: n}}, nil
	})
}
