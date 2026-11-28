// Package shellapi is what the app shell reads on every signed-in load:
// the counts beside its navigation.
package shellapi

import (
	"context"
	"net/http"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/imports"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/reconciling"
)

// Shell is what the counts read on a Controller. Each is false on any
// other Node.
type Shell interface {
	Grabs() (*grabs.Grabs, bool)
	Nodes() (*nodes.Service, bool)
	DownloadClients() (*downloadclients.Clients, bool)
	Health() (*health.Health, bool)
	Imports() (*imports.Imports, bool)
	Reconciling() (*reconciling.Service, bool)
}

type counts struct {
	Activity    int                       `json:"activity" minimum:"0" doc:"The Grabs neither imported nor failed"`
	Downloads   downloadclients.Aggregate `json:"downloads" doc:"Every download at once; torrents.live keeps it current"`
	Alerts      alerts                    `json:"alerts"`
	Pending     int                       `json:"pendingImports" minimum:"0" doc:"The Imports, of any source, with a file waiting for the Admin, ignored ones aside"`
	Reconciling api.Nullable[banner]      `json:"reconciling" doc:"What Reconciling after a restore waits on, for the shell's line; null outside it. reconciling.changed keeps it current"`
}

type banner struct {
	Rehome bool     `json:"rehome" doc:"Whether the old Controller's folders still need finding. The shell opens that step until it is saved"`
	Nodes  []string `json:"nodes" nullable:"false" doc:"The name of each Node whose torrents Reconciling waits on"`
	Roots  int      `json:"roots" minimum:"0" doc:"How many library roots Reconciling waits on to finish scanning"`
}

type alerts struct {
	Nodes       *string `json:"nodes" enum:"warn,fail" doc:"warn while a Node has been offline longer than a brief drop, or the Controller is doing a Role's work for an offline Node; fail while a Node has been offline or on another version longer than the offline grace; null otherwise"`
	Health      *string `json:"health" enum:"warn,fail" doc:"The worst level across every Health category, null while nothing is wrong"`
	Tasks       *string `json:"tasks" enum:"warn,fail" doc:"The worst level in the Scheduled tasks Health category, null while nothing is wrong"`
	Conversions *string `json:"conversions" enum:"warn,fail" doc:"fail while a Conversion failed and waits for an Admin, warn while no Node can convert one; null otherwise"`
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
		g, grabsOK := sh.Grabs()
		roster, nodesOK := sh.Nodes()
		clients, clientsOK := sh.DownloadClients()
		watch, healthOK := sh.Health()
		pending, importsOK := sh.Imports()
		keeper, reconcilingOK := sh.Reconciling()
		if !grabsOK || !nodesOK || !clientsOK || !healthOK || !importsOK || !reconcilingOK {
			panic("shellapi: getCounts ran on a Node that is no Controller")
		}
		n, err := g.InFlight(ctx, by)
		if err != nil {
			return nil, err
		}
		downloads, err := clients.Downloads(ctx)
		if err != nil {
			return nil, err
		}
		waiting, err := pending.Count(ctx)
		if err != nil {
			return nil, err
		}
		b, err := keeper.Banner(ctx)
		if err != nil {
			return nil, err
		}
		out := counts{Activity: n, Downloads: downloads, Alerts: alertsOf(roster.Board().Alert(), watch), Pending: waiting}
		if b != nil {
			out.Reconciling = api.Some(banner{Rehome: b.Rehome, Nodes: b.Nodes, Roots: b.Roots})
		}
		return &countsOutput{Body: out}, nil
	})
}
