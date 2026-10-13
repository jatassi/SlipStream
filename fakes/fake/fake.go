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
	Log     *slog.Logger
	Streams *node.Streams
	// Releases is the world's release catalog, nil in a fake's own tests
	// that hand it none.
	Releases *releases.Catalog
}

// NewEnv keys the fake's random streams under the Scenario's seed, the way
// a Node's are.
func NewEnv(s scenario.Scenario, name string, log *slog.Logger) Env {
	return Env{
		Log:     log.With("fake", name),
		Streams: node.NewStreams(s.Seed + "/fake/" + name),
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

// HTTP serves h until the context ends, then closes every connection.
func HTTP(h http.Handler) Server {
	return httpServer{h: h}
}

type httpServer struct {
	h http.Handler
}

func (s httpServer) Serve(ctx context.Context, l net.Listener) error {
	srv := &http.Server{
		Handler:           s.h,
		ReadHeaderTimeout: 10 * time.Second,
		BaseContext:       func(net.Listener) context.Context { return ctx },
	}
	stop := context.AfterFunc(ctx, func() { _ = srv.Close() })
	defer stop()
	if err := srv.Serve(l); !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve http: %w", err)
	}
	return nil
}
