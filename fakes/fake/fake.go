// Package fake is the shape every fake under fakes/ follows. Each fake
// package exports one var Fake Spec, and the world's fake list names it.
package fake

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"time"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/internal/releases"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/node"
)

type Spec struct {
	Name string
	// TLS asks the world to terminate TLS with a leaf from the run CA before
	// Serve sees a connection.
	TLS bool
	// ContentType is the id of the Content Type whose metadata source the
	// fake serves, empty for a core boundary. The world starts such a fake
	// only for a binary that registers the type.
	ContentType string
	// New builds the fake from the whole Scenario. It does no IO.
	New func(scenario.Scenario, Env) (Server, error)
	// Endpoints points every Endpoints entry the fake's boundary owns at
	// the fake. It is the only place a fake touches Node config.
	Endpoints func(*node.EndpointsBlock, Address)
}

// Server serves a boundary's real protocol on l until ctx ends.
type Server interface {
	Serve(ctx context.Context, l net.Listener) error
}

type Env struct {
	// Name is the fake's Spec name, the fault Target its routes play.
	Name    string
	Log     *slog.Logger
	Streams *node.Streams
	// Releases is the world's release catalog, nil in a fake's own tests
	// that hand it none.
	Releases *releases.Catalog
	// Faults is the Plan whose HTTP rules play on the fake's routes.
	Faults *fault.Plan
}

// NewEnv keys the fake's random streams under the Scenario's seed, the way
// a Node's are.
func NewEnv(s scenario.Scenario, name string, log *slog.Logger, faults *fault.Plan) Env {
	return Env{
		Name:    name,
		Log:     log.With("fake", name),
		Streams: node.NewStreams(s.Seed + "/fake/" + name),
		Faults:  faults,
	}
}

// Address is where a fake listens, as a Node dials it.
type Address struct {
	AddrPort netip.AddrPort
	TLS      bool
}

func (a Address) URL(path string) string {
	scheme := "http"
	if a.TLS {
		scheme = "https"
	}
	return scheme + "://" + a.AddrPort.String() + path
}

// Route is one route an HTTP fake serves. Its name in Routes is what a
// fault rule's Route names.
type Route struct {
	// Pattern is an http.ServeMux pattern.
	Pattern string
	Handler http.HandlerFunc
}

type Routes map[string]Route

// HTTP serves routes behind env.Faults until the context ends, then
// closes every connection. The fault middleware runs before each handler.
func HTTP(env Env, routes Routes) Server {
	mux := http.NewServeMux()
	for name, rt := range routes {
		mux.Handle(rt.Pattern, env.Faults.HTTP(fault.Target(env.Name), name, originOf, rt.Handler))
	}
	return httpServer{h: mux}
}

type originKey struct{}

// OriginOf is the Node whose front a request arrived through, "world" for
// the world's own front, and "" outside a world.
func OriginOf(ctx context.Context) string {
	origin, _ := ctx.Value(originKey{}).(string)
	return origin
}

func originOf(r *http.Request) string { return OriginOf(r.Context()) }

type httpServer struct {
	h http.Handler
}

func (s httpServer) Serve(ctx context.Context, l net.Listener) error {
	srv := &http.Server{
		Handler:           s.h,
		ReadHeaderTimeout: 10 * time.Second,
		BaseContext:       func(net.Listener) context.Context { return ctx },
		ConnContext: func(ctx context.Context, c net.Conn) context.Context {
			front, _ := fault.FrontOf(c)
			return context.WithValue(ctx, originKey{}, front.Origin)
		},
	}
	stop := context.AfterFunc(ctx, func() { _ = srv.Close() })
	defer stop()
	if err := srv.Serve(l); !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve http: %w", err)
	}
	return nil
}
