package node

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/buildinfo"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/node/surface"
	"github.com/jatassi/slipstream/internal/system"
	"github.com/jatassi/slipstream/web"
)

// startHTTP binds http.listen and serves every operation on it. Its stop
// closes the hub first, ending every stream, so Shutdown never waits out the
// stop deadline on one.
func (n *Node) startHTTP(_ context.Context, cfg Config, l *Listeners, s surface.Services, gate *identity.Gate) error {
	ln, err := n.env.Net.Listen("tcp", cfg.HTTP.Listen.String())
	if err != nil {
		return fmt.Errorf("listen for HTTP on %s: %w", cfg.HTTP.Listen, err)
	}
	l.HTTP = boundTCP(ln)
	s.System = system.Facts{
		DataDir:   n.env.DataDir.Path(),
		PID:       os.Getpid(),
		Seed:      cfg.Seed,
		StartedAt: n.started,
		Listeners: system.Listeners{HTTP: l.HTTP, Torrent: l.Torrent, Link: l.Link, Control: l.Control},
	}
	if stamp, ok := buildinfo.Current(); ok {
		s.System.Build = &stamp
	}
	log := n.rootLog.With("subsystem", "http")
	r := api.NewRouter(api.Services{
		Node:           s.Node,
		Gate:           gate,
		Web:            web.Dist,
		TrustedOrigins: cfg.HTTP.TrustedOrigins,
		TrustedProxies: cfg.HTTP.TrustedProxies,
		Log:            log,
	})
	surface.Mount(r, s)
	srv := api.NewServer(r.Handler(), cfg.Timings.HTTP, log)
	n.stack.Go("http", func(context.Context) error {
		if err := srv.Serve(ln); !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP: %w", err)
		}
		return nil
	})
	n.stack.OnStop("http-shutdown", func(ctx context.Context) error {
		s.Hub.Close()
		return srv.Shutdown(ctx)
	})
	return nil
}
