package sim

import (
	"context"
	"log/slog"
	"net/netip"
	"testing"

	"github.com/jatassi/slipstream/fakes/catalog"
	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/node"
)

// servers are the fakes a World serves on its fault network until it stops.
type servers struct {
	ctx    context.Context
	cancel context.CancelFunc
	done   []chan error
}

func startServers(t *testing.T) *servers {
	ctx, cancel := context.WithCancel(context.WithoutCancel(t.Context()))
	return &servers{ctx: ctx, cancel: cancel}
}

// catalog serves the Indexer catalog every Controller refreshes at start.
func (f *servers) catalog(t *testing.T, s scenario.Scenario, local *fault.Host, endpoints *node.EndpointsBlock) {
	t.Helper()
	server, err := catalog.Fake.New(s, fake.NewEnv(s, catalog.Fake, slog.New(slog.NewTextHandler(t.Output(), nil)), fault.NewPlan(nil, nil), nil))
	if err != nil {
		t.Fatal(err)
	}
	catalog.Fake.Endpoints(endpoints, f.serve(t, local, server))
}

func (f *servers) serve(t *testing.T, local *fault.Host, server fake.Server) fake.Address {
	t.Helper()
	l, err := local.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- server.Serve(f.ctx, l) }()
	f.done = append(f.done, done)
	return fake.Address{AddrPort: netip.MustParseAddrPort(l.Addr().String())}
}

func (f *servers) stop(t *testing.T) {
	f.cancel()
	for _, done := range f.done {
		if err := <-done; err != nil {
			t.Errorf("a fake: %v", err)
		}
	}
}
