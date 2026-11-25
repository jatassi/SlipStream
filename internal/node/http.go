package node

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net"
	"net/http"
	"net/netip"
	"os"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/buildinfo"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/node/internal/host"
	"github.com/jatassi/slipstream/internal/node/surface"
	"github.com/jatassi/slipstream/internal/serving"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/system"
	"github.com/jatassi/slipstream/web"
)

// startHTTP binds http.listen, or a later port, and serves every operation
// on it. Its stop ends every stream and log read first, so Shutdown never
// waits out the stop deadline on one.
func (n *Node) startHTTP(ctx context.Context, cfg Config, l *Listeners, s surface.Services, gate *identity.Gate) error {
	loopback6 := false
	if cfg.HTTP.Listen.Addr() == setting.ThisMachine {
		var err error
		if loopback6, err = hasLoopback6(ctx, n.env.Network.Interfaces); err != nil {
			return err
		}
	}
	lns, err := bindHTTP(n.env.Net.Listen, cfg.HTTP.Listen, cfg.HTTP.PortsToTry, loopback6)
	if err != nil {
		return err
	}
	l.HTTP = boundTCP(lns[0])
	own := webListener(lns)
	n.web.Store(&own)
	log := n.rootLog.With("subsystem", "http")
	for _, ln := range lns {
		logFamilies(log, ln)
	}
	if l.HTTP.Port() != cfg.HTTP.Listen.Port() && cfg.HTTP.Listen.Port() != 0 {
		log.Warn("the web port was taken, so the web server listens on the next free port", "configured", cfg.HTTP.Listen.String(), "bound", l.HTTP.String())
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
		Zone:           n.zone(),
		Reconciling:    surfaceNode{n}.refusal,
		Superseded:     n.superseded,
		Log:            log,
	})
	surface.Mount(r, s)
	handler := serving.Front(r.Handler(), n.servingWorker)
	srv := api.NewServer(handler, cfg.Timings.HTTP, log)
	srv.ConnContext = api.OnListener
	served := api.NewServer(handler, cfg.Timings.HTTP, log)
	served.ConnContext = api.OverLink
	for _, ln := range lns {
		n.stack.Go("http "+ln.Addr().String(), func(context.Context) error {
			if err := srv.Serve(ln); !errors.Is(err, http.ErrServerClosed) {
				return fmt.Errorf("serve HTTP on %s: %w", ln.Addr(), err)
			}
			return nil
		})
	}
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

// logFamilies logs the IP families ln accepts, which its address doesn't
// show for [::].
func logFamilies(log *slog.Logger, ln net.Listener) {
	tcp, ok := ln.(*net.TCPListener)
	if !ok { // a test host's in-memory listener has no socket
		return
	}
	f, err := host.FamiliesOf(tcp)
	if err != nil {
		log.Warn("couldn't read which IP families the web server accepts", "address", ln.Addr().String(), "error", err)
		return
	}
	log.Debug("the web server listens", "address", ln.Addr().String(), "ipv4", f.IPv4, "ipv6", f.IPv6)
}

// bindHTTP listens on addr's port, then each next port, tries ports in
// all, and keeps the first that binds. This machine only binds ::1 and
// 127.0.0.1 on one port free on both, or 127.0.0.1 alone on a host
// without loopback6, one with IPv6 turned off. Any listen error moves on, since
// the error for a taken port differs by OS and network. Port 0 binds any
// free port in one try, and the ports end at 65535.
func bindHTTP(listen func(network, address string) (net.Listener, error), addr netip.AddrPort, tries int, loopback6 bool) ([]net.Listener, error) {
	hosts := []netip.Addr{addr.Addr()}
	if addr.Addr() == setting.ThisMachine {
		hosts = []netip.Addr{netip.AddrFrom4([4]byte{127, 0, 0, 1})}
		if loopback6 {
			hosts = []netip.Addr{setting.ThisMachine, hosts[0]}
		}
	}
	port := addr.Port()
	for tried := 1; ; tried++ {
		lns, err := bindAll(listen, hosts, port)
		if err == nil {
			return lns, nil
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

// bindAll listens on port at every host, or on none. Port 0 binds the
// first host's free port, then the same port on the rest.
func bindAll(listen func(network, address string) (net.Listener, error), hosts []netip.Addr, port uint16) ([]net.Listener, error) {
	lns := make([]net.Listener, 0, len(hosts))
	for _, h := range hosts {
		ln, err := listen("tcp", netip.AddrPortFrom(h, port).String())
		if err != nil {
			for _, bound := range lns {
				_ = bound.Close()
			}
			return nil, err
		}
		lns = append(lns, ln)
		port = boundTCP(ln).Port()
	}
	return lns, nil
}
