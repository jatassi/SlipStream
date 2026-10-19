// Package shellapi is what the app shell reads on every signed-in load:
// the counts beside its navigation.
package shellapi

import (
	"context"
	"net/http"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/nodes"
)

// Shell is what the counts read on a Controller. Each is false on any
// other Node.
type Shell interface {
	Grabs() (*grabs.Grabs, bool)
	Nodes() (*nodes.Service, bool)
	Health() (*health.Health, bool)
}

type counts struct {
	Activity int    `json:"activity" minimum:"0" doc:"The Grabs neither imported nor failed"`
	Alerts   alerts `json:"alerts"`
}

type alerts struct {
	Nodes  *string `json:"nodes" enum:"warn,fail" doc:"warn while a Node is away past the blip or the Controller covers a Role for one, fail while a Node is out of step past the offline grace, null otherwise"`
	Health *string `json:"health" enum:"warn,fail" doc:"The worst level across every Health category, null while nothing is wrong"`
	Tasks  *string `json:"tasks" enum:"warn,fail" doc:"The worst level in the Scheduled tasks Health category, null while nothing is wrong"`
}

func alertsOf(a nodes.Alert, h *health.Health) alerts {
	out := alerts{Health: level(h.Worst()), Tasks: level(h.Worst(health.ScheduledTasksID))}
	if a != nodes.NoAlert {
		out.Nodes = new(a.String())
	}
	return out
}

func level(l health.Level) *string {
	if l == health.None {
		return nil
	}
	return new(l.String())
}

type countsOutput struct{ Body counts }

func Routes(r *api.Router, sh Shell) {
	o := r.Owner("shell", api.Admin)

	api.Register(o, api.Admin, api.Op{
		ID:      "getCounts",
		Method:  http.MethodGet,
		Path:    "/counts",
		Summary: "Get the counts the shell shows beside its navigation",
		Status:  http.StatusOK,
	}, func(ctx context.Context, by identity.Admin, _ *struct{}) (*countsOutput, error) {
		g, ok := sh.Grabs()
		roster, alsoOK := sh.Nodes()
		watch, healthOK := sh.Health()
		if !ok || !alsoOK || !healthOK {
			panic("shellapi: getCounts ran on a Node that is no Controller")
		}
		n, err := g.Moving(ctx, by)
		if err != nil {
			return nil, err
		}
		return &countsOutput{Body: counts{Activity: n, Alerts: alertsOf(roster.Board().Alert(), watch)}}, nil
	})
}
