package surface_test

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/api/apitest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/joincode"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/node/surface"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/pairing"
	"github.com/jatassi/slipstream/internal/system"
)

// worker is a Worker's Node: a pairing record and no library.db.
type worker struct{}

func (worker) Claim(context.Context) (system.Claim, error) { return system.Worker, nil }
func (worker) Identity() (*identity.Service, bool)         { return nil, false }

func (worker) ClaimController(context.Context, func(*identity.Service) error) error {
	panic("a Worker's closed gate admitted claimController")
}

func (worker) Nodes() (*nodes.Service, bool) { return nil, false }

func (worker) Join(context.Context, joincode.Code) (pairing.Joined, error) {
	panic("a Worker's closed gate admitted claimWorker")
}

func TestTheAccessWalkPassesOnAWorker(t *testing.T) {
	log := slog.New(slog.DiscardHandler)
	gate, _, err := identity.NewGate(t.Context(), worker{}.Claim, log)
	if err != nil {
		t.Fatal(err)
	}
	r := api.NewRouter(api.Services{Node: worker{}, Gate: gate, Web: fstest.MapFS{}, Log: log})
	hub := live.NewHub(surface.Events)
	t.Cleanup(hub.Close)
	surface.Mount(r, surface.Services{Node: worker{}, Hub: hub})
	if breaches := apitest.CheckWorkerAccess(r.Handler(), surface.Document()); len(breaches) != 0 {
		t.Fatalf("breaches:\n%s", strings.Join(breaches, "\n"))
	}
}
