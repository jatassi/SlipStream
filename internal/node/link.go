package node

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"runtime"

	"github.com/jatassi/slipstream/internal/buildinfo"
	"github.com/jatassi/slipstream/internal/link"
)

type osNetwork struct{}

func (osNetwork) Listen(network, address string) (net.Listener, error) {
	return new(net.ListenConfig).Listen(context.Background(), network, address) //nolint:wrapcheck // the caller names the listener
}

func (osNetwork) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return new(net.Dialer).DialContext(ctx, network, address) //nolint:wrapcheck // the link names the peer
}

// startLink binds link.listen and serves the link on it. Every Node does,
// and an Unclaimed one refuses every handshake.
func (n *Node) startLink(cfg Config, streams *Streams, l *Listeners) error {
	ln, err := n.env.Net.Listen("tcp", cfg.Link.Listen.String())
	if err != nil {
		return fmt.Errorf("listen for the link on %s: %w", cfg.Link.Listen, err)
	}
	l.Link = boundTCP(ln)
	peers := claimPeers{n}
	n.lockstep = http.NewServeMux()
	n.info = nodeInfo(n.env.Version)
	n.link, err = link.New(link.Config{
		Key:      n.key,
		Node:     n.info,
		Timings:  cfg.Timings.Link,
		Settings: cfg.Link,
		Net:      n.env.Net,
		Rand:     streams.Stream("link"),
		Trust:    peers,
		Lockstep: n.env.Test.lockstep(n.lockstep),
		Pair:     peers.Pair,
		Log:      n.rootLog.With("subsystem", "link"),
	})
	if err != nil {
		_ = ln.Close()
		return err
	}
	n.stack.OnStop("link", func(context.Context) error { return n.link.Close() })
	n.stack.Go("link-serve", func(ctx context.Context) error { return n.link.Serve(ctx, ln) })
	return nil
}

// nodeInfo is this build as the link's Hello carries it.
func nodeInfo(version string) link.NodeInfo {
	return link.NodeInfo{Version: version, OS: runtime.GOOS, Arch: runtime.GOARCH, InstallKind: buildinfo.InstallKind()}
}

func boundTCP(ln net.Listener) netip.AddrPort {
	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		panic(fmt.Sprintf("node: a tcp listener is bound to %T", ln.Addr()))
	}
	return addr.AddrPort()
}
