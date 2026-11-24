package api

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/netip"

	"github.com/jatassi/slipstream/internal/role"
)

// Listener is the kind of listener a request came in on.
type Listener uint8

const (
	// Bound is the Node's own HTTP listener.
	Bound Listener = iota + 1
	// Served is a Serve stream a Serving Worker opened over the link.
	Served
)

// Arrival is where one request came in: the listener, the Serving Node
// whose Serve stream carried it (zero for Bound), the TCP peer, the client
// after http.trusted_proxies, and Host.
type Arrival struct {
	Listener Listener
	Via      role.NodeID
	Peer     netip.Addr
	Source   netip.Addr
	Host     string
}

type arrivalKey struct{}

// ArrivalOf is the arrival of the request ctx serves, which the access
// check stores before the handler runs.
func ArrivalOf(ctx context.Context) Arrival {
	a, _ := ctx.Value(arrivalKey{}).(Arrival)
	return a
}

type boundKey struct{}

// OnListener is http.Server's ConnContext for the Node's own listener.
func OnListener(ctx context.Context, _ net.Conn) context.Context {
	return context.WithValue(ctx, boundKey{}, true)
}

type servedKey struct{}

// OverLink is http.Server's ConnContext for the connections a Serving
// Worker opens over the link: their requests' client is the address the
// Worker's Forwarded header names, over plain HTTP. c names the Worker
// whose Serve stream it is.
func OverLink(ctx context.Context, c net.Conn) context.Context {
	served, ok := c.(interface{ ServedFor() role.NodeID })
	if !ok {
		panic(fmt.Sprintf("api: a Serve stream's connection %T doesn't name its Node", c))
	}
	return context.WithValue(ctx, servedKey{}, served.ServedFor())
}

// arrivalOf is req's arrival with source as its client.
func arrivalOf(req *http.Request, source netip.Addr) Arrival {
	peer, _ := netip.ParseAddrPort(req.RemoteAddr)
	a := Arrival{Peer: peer.Addr().Unmap(), Source: source, Host: req.Host}
	if via, ok := req.Context().Value(servedKey{}).(role.NodeID); ok {
		a.Listener, a.Via = Served, via
	} else if req.Context().Value(boundKey{}) != nil {
		a.Listener = Bound
	}
	return a
}
