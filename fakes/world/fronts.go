package world

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/netip"
	"slices"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/internal/ca"
	"github.com/jatassi/slipstream/fakes/internal/front"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/tracker"
	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/node"
)

// bindFronts binds every front before any fake is built, since the
// release catalog and every address a Node sees name one: each fake as
// each Node and the world reach it, each seeded release's seeder as each
// Node reaches it, and each Node's link as every peer reaches it.
func (r *Run) bindFronts(ctx context.Context, undo *unwind, specs []fake.Spec) error {
	specs = slices.DeleteFunc(slices.Clone(specs), func(spec fake.Spec) bool {
		return spec.ContentType != "" && !slices.Contains(r.probe.ContentTypes, spec.ContentType)
	})
	planned := planFronts(specs, r.nodeNames(), r.scenario.Releases)
	r.faults = fault.NewPlan(planned, r.fired)
	r.fronts = front.NewSet(r.faults, r.appendRecord, r.log)
	undo.push(r.fronts.Close)
	for _, p := range planned {
		f, err := r.fronts.Listen(ctx, p)
		if err != nil {
			return err
		}
		r.listeners = append(r.listeners, f)
		r.table = append(r.table, fake.Front{Target: p.Target, Node: p.Origin, Address: fake.Address{AddrPort: f.Addr(), TLS: p.Layers&fault.TLS != 0}})
	}
	for _, spec := range specs {
		at, _ := r.table.Find(fault.Target(spec.Name), events.World)
		r.fakes = append(r.fakes, &fakeRun{spec: spec, at: at, inbox: front.NewInbox(at.AddrPort)})
	}
	return nil
}

// planFronts is every front a world of the fakes specs and the Nodes
// nodes binds for releases.
func planFronts(specs []fake.Spec, nodes []string, releases []scenario.Release) []fault.Front {
	var planned []fault.Front
	for _, spec := range specs {
		layers := fault.Conn | fault.HTTP
		if spec.TLS == fake.FrontTLS {
			layers |= fault.TLS
		}
		for _, origin := range append(slices.Clone(nodes), events.World) {
			planned = append(planned, fault.Front{Target: fault.Target(spec.Name), Origin: origin, Layers: layers})
		}
	}
	for _, rel := range releases {
		if !rel.Seeded {
			continue
		}
		for _, origin := range nodes {
			planned = append(planned, fault.Front{Target: tracker.SeedersTarget(rel.Name), Origin: origin, Layers: fault.Conn})
		}
	}
	for _, name := range nodes {
		planned = append(planned, fault.Front{Target: LinkTarget(name), Layers: fault.Conn})
	}
	return planned
}

// setFaults sets the Scenario's fault rules once every fake has declared
// its routes and before any Node starts, each with its fault.set record.
func (r *Run) setFaults(ctx context.Context, _ *unwind) error {
	for _, rule := range r.scenario.Faults {
		if _, err := r.SetFault(ctx, rule); err != nil {
			return fmt.Errorf("scenario %s fault %s: %w", r.scenario.Name, rule.ID, err)
		}
	}
	return nil
}

func (r *Run) nodeNames() []string {
	names := make([]string, len(r.nodes))
	for i, n := range r.nodes {
		names[i] = n.spec.Name
	}
	return names
}

// serveFronts has each of f's fronts push its connections into f's
// inbox, behind TLS from the run CA when f's Spec asks for it.
func (r *Run) serveFronts(f *fakeRun) error {
	var leaves func(fault.Front) *tls.Config
	if f.spec.TLS == fake.FrontTLS {
		var err error
		if leaves, err = r.leaves(f.spec.Name); err != nil {
			return fmt.Errorf("fake %s: %w", f.spec.Name, err)
		}
	}
	for _, l := range r.listeners {
		if l.Fault().Target != fault.Target(f.spec.Name) {
			continue
		}
		var cfg *tls.Config
		if leaves != nil {
			cfg = leaves(l.Fault())
		}
		l.Serve(front.Served(f.inbox, cfg))
	}
	return nil
}

// leaves are the certificates a fake's fronts present: a leaf from the run
// CA, and the bad leaf each TLS failure plays.
func (r *Run) leaves(name string) (func(fault.Front) *tls.Config, error) {
	names := leafNames(name)
	good, err := r.leaf(name)
	if err != nil {
		return nil, err
	}
	expired, err := r.ca.ExpiredLeaf(names...)
	if err != nil {
		return nil, err
	}
	wrongName, err := r.ca.Leaf("wrong-name.fake.localhost")
	if err != nil {
		return nil, err
	}
	stranger, err := ca.New(r.id + " untrusted")
	if err != nil {
		return nil, err
	}
	untrusted, err := stranger.Leaf(names...)
	if err != nil {
		return nil, err
	}
	bad := map[fault.TLSFailure]tls.Certificate{fault.Untrusted: untrusted, fault.Expired: expired, fault.WrongName: wrongName}
	return func(f fault.Front) *tls.Config { return front.TLS(r.faults, f, good, bad) }, nil
}

func leafNames(name string) []string { return []string{"127.0.0.1", name + ".fake.localhost"} }

// leaf is the run-CA certificate of the fake named name.
func (r *Run) leaf(name string) (tls.Certificate, error) { return r.ca.Leaf(leafNames(name)...) }

// LinkTarget is the fault target of the link front of the Node named
// name: one front every peer dials, since each dials the one address the
// Node advertises.
func LinkTarget(name string) fault.Target { return fault.Target("link/" + name) }

// serveLinkFronts forwards each Node's link front to the link listener of
// the Node as it runs now.
func (r *Run) serveLinkFronts() {
	for _, n := range r.nodes {
		at := r.linkFront(n.spec.Name)
		at.Serve(front.Forward(func() (netip.AddrPort, error) {
			ready, err := n.life.Live()
			return ready.Listeners.Link, err
		}))
	}
}

func (r *Run) linkFront(name string) *front.Front {
	i := slices.IndexFunc(r.listeners, func(l *front.Front) bool { return l.Fault().Target == LinkTarget(name) })
	return r.listeners[i]
}

// serveSeederFronts forwards each seeder front to its release's seeder.
func (r *Run) serveSeederFronts() {
	for _, l := range r.listeners {
		for _, rel := range r.releases.Releases {
			if rel.Seeded && l.Fault().Target == tracker.SeedersTarget(rel.Name) {
				l.Serve(front.Forward(func() (netip.AddrPort, error) { return r.seeders.Addr(rel.Name) }))
			}
		}
	}
}

// endpoints is the Endpoints block of the Node named name: every fake's
// entries at that Node's fronts.
func (r *Run) endpoints(name string) (node.EndpointsBlock, error) {
	listed := map[string]bool{}
	for _, k := range r.probe.Keys {
		listed[k.Key] = true
	}
	e := node.Defaults(contenttype.Set{}).Endpoints
	for _, f := range r.fakes {
		at, _ := r.table.Find(fault.Target(f.spec.Name), name)
		f.spec.Endpoints(&e, at)
		for _, p := range e.ContentTypes.Paths() {
			if !listed[p] {
				return node.EndpointsBlock{}, fmt.Errorf("fake %s sets %s, which this binary does not list", f.spec.Name, p)
			}
		}
	}
	return e, nil
}
