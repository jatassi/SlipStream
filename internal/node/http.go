package node

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net"
	"net/http"
	"net/netip"
	"os"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/buildinfo"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/node/surface"
	"github.com/jatassi/slipstream/internal/serving"
	"github.com/jatassi/slipstream/internal/system"
	"github.com/jatassi/slipstream/web"
)

// startHTTP binds http.listen, or a later port, and serves every operation
// on it. Its stop ends every stream and log read first, so Shutdown never
// waits out the stop deadline on one.
func (n *Node) startHTTP(_ context.Context, cfg Config, l *Listeners, s surface.Services, gate *identity.Gate) error {
	ln, err := bindHTTP(n.env.Net.Listen, cfg.HTTP.Listen, cfg.HTTP.PortsToTry)
	if err != nil {
		return err
	}
	l.HTTP = boundTCP(ln)
	log := n.rootLog.With("subsystem", "http")
	if l.HTTP.Port() != cfg.HTTP.Listen.Port() && cfg.HTTP.Listen.Port() != 0 {
		log.Warn("web port taken, serving on a later one", "configured", cfg.HTTP.Listen.String(), "bound", l.HTTP.String())
	}
	facts := system.Facts{
		DataDir:        n.env.DataDir.Path(),
		LogsDir:        n.env.DataDir.LogsDir(),
		PID:            os.Getpid(),
		Seed:           cfg.Seed,
		StartedAt:      n.started,
		HTTPConfigured: cfg.HTTP.Listen,
		Listeners:      system.Listeners{HTTP: l.HTTP, Torrent: l.Torrent, Link: l.Link, Control: l.Control},
	}
	if stamp, ok := buildinfo.Current(); ok {
		facts.Build = &stamp
	}
	s.System = facts
	n.host.facts.Store(&facts)
	n.reportHost()
	r := api.NewRouter(api.Services{
		Node:           s.Node,
		Gate:           gate,
		Web:            web.Dist,
		TrustedOrigins: cfg.HTTP.TrustedOrigins,
		TrustedProxies: cfg.HTTP.TrustedProxies,
		Log:            log,
	})
	surface.Mount(r, s)
	handler := serving.Front(r.Handler(), n.servingWorker)
	srv := api.NewServer(handler, cfg.Timings.HTTP, log)
	served := api.NewServer(handler, cfg.Timings.HTTP, log)
	served.ConnContext = api.OverLink
	n.stack.Go("http", func(context.Context) error {
		if err := srv.Serve(ln); !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP: %w", err)
		}
		return nil
	})
	n.stack.Go("http-served", func(context.Context) error {
		if err := served.Serve(n.served); !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve Serve streams: %w", err)
		}
		return nil
	})
	n.stack.OnStop("http-shutdown", func(ctx context.Context) error {
		s.Hub.Close()
		r.EndStreams()
		n.logs.EndReads()
		if w := n.servingWorker(); w != nil {
			w.Close()
		}
		return errors.Join(srv.Shutdown(ctx), served.Shutdown(ctx))
	})
	return nil
}

// bindHTTP listens on addr's port, then each next port, tries ports in
// all, and keeps the first that binds. Any listen error moves on, since
// the error for a taken port differs by OS and network. Port 0 binds any
// free port in one try, and the ports end at 65535.
func bindHTTP(listen func(network, address string) (net.Listener, error), addr netip.AddrPort, tries int) (net.Listener, error) {
	port := addr.Port()
	for tried := 1; ; tried++ {
		ln, err := listen("tcp", netip.AddrPortFrom(addr.Addr(), port).String())
		if err == nil {
			return ln, nil
		}
		if tried >= tries || port == 0 || port == math.MaxUint16 {
			if port == addr.Port() {
				return nil, fmt.Errorf("listen for HTTP on %s: %w", addr, err)
			}
			return nil, fmt.Errorf("listen for HTTP on %s, ports %d to %d: %w", addr.Addr(), addr.Port(), port, err)
		}
		port++
	}
}
