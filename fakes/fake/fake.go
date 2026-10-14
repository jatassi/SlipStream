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
	"regexp"
	"time"

	"github.com/jatassi/slipstream/fakes/events"
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
	// Online says whether a suite checks the fake against the real service.
	// The world refuses the zero value at launch.
	Online Online
	// Captures are the record kinds the fake keeps for tests and verify to
	// read; Env.Capture refuses any other.
	Captures []CaptureSpec
}

// CaptureSpec is a capture a fake declares. Name is its fakeworld reader
// verb, as in fakeworld announces list, and Kind the kind of its records.
type CaptureSpec struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
}

// Online is a fake's stance on running against the real service: a
// conformance suite, or a reason it has none.
type Online struct {
	suite  string
	reason string
}

// OnlineSuite names the suite under conformance/ that runs against both
// the fake and the real service.
func OnlineSuite(suite string) Online { return Online{suite: suite} }

func NoOnlineRun(reason string) Online { return Online{reason: reason} }

// Check refuses a Spec that declares no online stance.
func (s Spec) Check() error {
	if s.Online.suite == "" && s.Online.reason == "" {
		return fmt.Errorf("fake %s declares no Online stance; set fake.OnlineSuite(suite) or fake.NoOnlineRun(reason)", s.Name)
	}
	return nil
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
	// Fronts is where each origin reaches each target.
	Fronts Fronts
	// Files is the run dir a capture's files land under.
	Files string

	captures []CaptureSpec
	sink     Sink
}

// Sink takes the records a fake's captures write: the run's event log.
type Sink interface {
	Append(events.Record) (events.Record, error)
}

// NewEnv keys the fake's random streams under the Scenario's seed, the way
// a Node's are. Its captures write to sink, which is nil for a fake that
// runs outside any event log.
func NewEnv(s scenario.Scenario, spec Spec, log *slog.Logger, faults *fault.Plan, sink Sink) Env {
	return Env{
		Name:     spec.Name,
		Log:      log.With("fake", spec.Name),
		Streams:  node.NewStreams(s.Seed + "/fake/" + spec.Name),
		Faults:   faults,
		captures: spec.Captures,
		sink:     sink,
	}
}

// Front is where an origin reaches a target: the address of the world's
// front for target as Node dials it, "world" for the harness.
type Front struct {
	Target  fault.Target
	Node    string
	Address Address
}

type Fronts []Front

// Find is the address of target's front for origin.
func (fs Fronts) Find(target fault.Target, origin string) (Address, bool) {
	for _, f := range fs {
		if f.Target == target && f.Node == origin {
			return f.Address, true
		}
	}
	return Address{}, false
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
// closes every connection. The fault middleware runs before each handler,
// and a rule may name each route and the path values its pattern names.
func HTTP(env Env, routes Routes) Server {
	mux := http.NewServeMux()
	for name, rt := range routes {
		route := fault.Route{Name: name}
		for _, m := range wildcard.FindAllStringSubmatch(rt.Pattern, -1) {
			route.Params = append(route.Params, m[1])
		}
		mux.Handle(rt.Pattern, env.Faults.HTTP(fault.Target(env.Name), route, originOf, rt.Handler))
	}
	return httpServer{h: mux}
}

// wildcard is a path value in an http.ServeMux pattern.
var wildcard = regexp.MustCompile(`\{([^{}.]+)(?:\.\.\.)?\}`)

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
