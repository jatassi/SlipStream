package node

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/buildinfo"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/node/internal/control"
	"github.com/jatassi/slipstream/internal/node/internal/logs"
	"github.com/jatassi/slipstream/internal/node/surface"
	"github.com/jatassi/slipstream/internal/system"
	"github.com/jatassi/slipstream/internal/torrent/engine"
	"github.com/jatassi/slipstream/internal/torrent/libtorrent"
	"github.com/jatassi/slipstream/web"
)

// machine is the part of a Node a testing/synctest bubble cannot hold: a
// real Accept and libtorrent's alert poll in cgo never count as blocked, so
// fake time would stop, and lumberjack's mill goroutine never exits. The
// in-process Node test swaps in in-memory ones.
type machine struct {
	listen      control.ListenFunc
	openTorrent func(engine.Config) (engine.Engine, error)
	openLog     func(dir string, stderr io.Writer) (*logs.Log, error)
}

var host = machine{
	listen:      new(net.ListenConfig).Listen,
	openTorrent: func(c engine.Config) (engine.Engine, error) { return libtorrent.Open(c) },
	openLog:     logs.Open,
}

// startHTTP binds http.listen and serves every operation on it. Its stop
// closes the hub first, ending every stream, so Shutdown never waits out the
// stop deadline on one.
func (n *Node) startHTTP(ctx context.Context, cfg Config, l *Listeners, s surface.Services, gate *identity.Gate) error {
	ln, err := n.machine.listen(ctx, "tcp", cfg.HTTP.Listen.String())
	if err != nil {
		return fmt.Errorf("listen for HTTP on %s: %w", cfg.HTTP.Listen, err)
	}
	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		panic(fmt.Sprintf("node: a tcp listener is bound to %T", ln.Addr()))
	}
	l.HTTP = addr.AddrPort()
	s.System = system.Facts{
		DataDir:   n.env.DataDir.Path(),
		PID:       os.Getpid(),
		Seed:      cfg.Seed,
		StartedAt: n.started,
		Listeners: system.Listeners{HTTP: l.HTTP, Torrent: l.Torrent, Control: l.Control},
	}
	if stamp, ok := buildinfo.Current(); ok {
		s.System.Build = &stamp
	}
	log := n.rootLog.With("subsystem", "http")
	r := api.NewRouter(api.Services{
		Identity:       s.Identity,
		Gate:           gate,
		Web:            web.Dist,
		TrustedOrigins: cfg.HTTP.TrustedOrigins,
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
