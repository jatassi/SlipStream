// Package shellapi is what the app shell reads on every signed-in load:
// the counts beside its navigation.
package shellapi

import (
	"context"
	"net/http"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/nodes"
)

// Shell is what the counts read on a Controller. Each is false on any
// other Node.
type Shell interface {
	Grabs() (*grabs.Grabs, bool)
	Nodes() (*nodes.Service, bool)
	DownloadClients() (*downloadclients.Clients, bool)
}

type counts struct {
	Activity  int                       `json:"activity" minimum:"0" doc:"The Grabs neither imported nor failed"`
	Downloads downloadclients.Aggregate `json:"downloads" doc:"Every download at once; torrents.live keeps it current"`
	Alerts    alerts                    `json:"alerts"`
}

type alerts struct {
	Nodes *string `json:"nodes" enum:"warn,fail" doc:"warn while a Node is away past the blip or the Controller covers a Role for one, fail while a Node is out of step past the offline grace, null otherwise"`
}

func alertsOf(a nodes.Alert) alerts {
	if a == nodes.NoAlert {
		return alerts{}
	}
	return alerts{Nodes: new(a.String())}
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
		if !grabsOK || !nodesOK || !clientsOK {
			panic("shellapi: getCounts ran on a Node that is no Controller")
		}
		n, err := g.Moving(ctx, by)
		if err != nil {
			return nil, err
		}
		downloads, err := clients.Downloads(ctx)
		if err != nil {
			return nil, err
		}
		return &countsOutput{Body: counts{Activity: n, Downloads: downloads, Alerts: alertsOf(roster.Board().Alert())}}, nil
	})
}
