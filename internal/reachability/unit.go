package reachability

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/netmodel"
	netv1 "github.com/jatassi/slipstream/internal/pb/slipstream/network/v1"
	"github.com/jatassi/slipstream/internal/reachability/internal/db"
	"github.com/jatassi/slipstream/internal/reachability/judge"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks"
)

// loaded is what one unit reads of the library before it reaches any Node.
type loaded struct {
	exposure Exposure
	row      db.ReachabilityExposure
	public   *url.URL
	origins  origins
	stored   map[role.NodeID]stored
	pins     map[role.NodeID]netip.Addr
}

var errNoAddress = tasks.SkipOwned("Skipped because SlipStream has no Address. Set Address to the https:// name outsiders use.")

func (r *Reachability) load(ctx context.Context, id judge.ExposureID) (loaded, bool, error) {
	var l loaded
	err := r.cfg.Library.Read(ctx, func(ctx context.Context, rd storage.Reader[storage.Library]) error {
		q := db.New(rd)
		var err error
		if l.row, err = q.GetExposure(ctx, id); err != nil {
			return err
		}
		l.exposure, l.public = exposureOf(l.row), exposureOf(l.row).Public
		if l.exposure.Kind == judge.UIPortal {
			if l.public, err = addressIn(ctx, rd); err != nil {
				return err
			}
		}
		if l.origins, err = originsIn(ctx, rd, r.cfg.Roster.Self()); err != nil {
			return err
		}
		if l.stored, err = storedIn(ctx, q); err != nil {
			return err
		}
		l.pins, err = pinsIn(ctx, q)
		return err
	})
	if errors.Is(err, storage.ErrNoRows) {
		return l, false, nil
	}
	return l, err == nil, err
}

func addressIn(ctx context.Context, rd storage.Reader[storage.Library]) (*url.URL, error) {
	addr, err := identity.AddressIn(ctx, rd)
	if err != nil {
		return nil, err
	}
	if addr.IsZero() {
		return nil, errNoAddress
	}
	return addr.URL(), nil
}

func storedIn(ctx context.Context, q *db.Queries) (map[role.NodeID]stored, error) {
	rows, err := q.ListNodes(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[role.NodeID]stored, len(rows))
	for _, row := range rows {
		if out[row.NodeID], err = storedOf(row); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func pinsIn(ctx context.Context, q *db.Queries) (map[role.NodeID]netip.Addr, error) {
	settings, err := q.ListNodeSettings(ctx)
	if err != nil {
		return nil, err
	}
	out := map[role.NodeID]netip.Addr{}
	for _, s := range settings {
		if s.StableIpv6 != nil {
			out[s.NodeID], _ = netip.ParseAddr(*s.StableIpv6)
		}
	}
	return out, nil
}

// runUnit judges one Exposure. A Fixable or Needs a relay verdict is a
// success: the unit fails only when it couldn't judge at all, or every
// broken route would also raise a Scheduled tasks warning.
func (r *Reachability) runUnit(ctx context.Context, u tasks.Unit, _ *tasks.Progress) error {
	id, err := exposureOfUnit(u.Key)
	if err != nil {
		return err
	}
	l, ok, err := r.load(ctx, id)
	if !ok {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, r.cfg.Timings.Unit)
	defer cancel()
	g, err := r.gather(ctx, l)
	if err != nil {
		return err
	}
	g.test(ctx, r)
	j := judge.Judge(g.f)
	if err := r.store(ctx, g, j); err != nil {
		return err
	}
	r.changed(ctx, id.UUID)
	return nil
}

// gathered is one unit's facts as it builds them.
type gathered struct {
	f      judge.Facts
	host   string
	scheme string
	kind   string
	ident  string
	fresh  map[role.NodeID]netmodel.NetworkFacts
	server *serverDoc
	r      *Reachability
}

func (r *Reachability) gather(ctx context.Context, l loaded) (*gathered, error) {
	origin, label := l.origins.of(l.exposure)
	e := judge.Exposure{ID: l.exposure.ID, Kind: l.exposure.Kind, Server: l.exposure.Server, Public: l.public}
	g := &gathered{
		host: l.public.Hostname(), scheme: l.public.Scheme, fresh: map[role.NodeID]netmodel.NetworkFacts{}, r: r,
		f: judge.Facts{
			Rules: r.cfg.Rules, Origin: judge.Origin{Node: origin, Label: label},
			Nodes: map[role.NodeID]netmodel.NetworkFacts{}, Paired: map[netip.Addr]role.NodeID{},
			Pins: l.pins, Churn: map[role.NodeID]bool{},
		},
	}
	if e.Kind == judge.UIPortal {
		e.Local = r.cfg.WebPort()
		g.f.Origin.Serving = r.cfg.Roster.Serving(role.Serving)
	} else {
		rec := l.origins.servers[e.Server]
		g.kind, g.ident = string(rec.Kind), rec.Identity
		doc, err := r.serverFacts(ctx, l.row, rec)
		if err != nil {
			return nil, err
		}
		g.server, g.f.Server = &doc, &doc.Facts
		e.Local = doc.Local
	}
	g.f.Exposure = e
	r.mu.Lock()
	for n, c := range r.churn {
		g.f.Churn[n] = c
	}
	r.mu.Unlock()
	if err := g.readNodes(ctx, l, origin); err != nil {
		return nil, err
	}
	return g, nil
}

// serverFacts is what the Media Server publishes now, or what it last did
// while its reaching Node is away.
func (r *Reachability) serverFacts(ctx context.Context, row db.ReachabilityExposure, rec mediaservers.Record) (serverDoc, error) {
	x, err := r.cfg.MediaServers.Exposure(ctx, rec.ID)
	if err != nil {
		return serverDoc{}, fmt.Errorf("read what %s publishes: %w", rec.Name, err)
	}
	if x.Exposed == nil {
		if doc, ok := serverDocOf(row); ok {
			return doc, nil
		}
	}
	return serverDocFrom(x, rec), nil
}

func serverDocFrom(x mediaservers.Exposure, rec mediaservers.Record) serverDoc {
	doc := serverDoc{V: factsVersion, Facts: judge.ServerFacts{Kind: x.Kind, Name: x.Name}}
	if u, err := url.Parse(rec.Address); err == nil {
		if p, err := strconv.ParseUint(u.Port(), 10, 16); err == nil {
			doc.Local = uint16(p)
		}
	}
	if x.Exposed != nil {
		if x.Exposed.Local.IsValid() {
			doc.Local = x.Exposed.Local.Port()
		}
		doc.Facts.Custom = x.Exposed.Custom
		if m := x.Exposed.Mapping; m != nil {
			doc.Facts.Mapping = &judge.PlexMapping{State: m.State, Error: m.Error, Pending: m.Pending, Public: m.Public}
		}
	}
	if x.Resource != nil {
		for _, c := range x.Resource.Connections {
			jc := judge.Connection{URL: c.URI.String(), Port: c.Port, Local: c.Local, Relay: c.Relay, Family: netmodel.IPv4}
			if a, err := netip.ParseAddr(c.Address); err == nil {
				jc.Addr = a.Unmap()
			}
			if c.IPv6 || jc.Addr.Is6() {
				jc.Family = netmodel.IPv6
			}
			doc.Facts.Published = append(doc.Facts.Published, jc)
		}
	}
	return doc
}

// readNodes reads the origin's and every in-step Node's facts and its
// lookup of the public host, all at once.
func (g *gathered) readNodes(ctx context.Context, l loaded, origin role.NodeID) error {
	r := g.r
	ports := []uint32{uint32(g.f.Exposure.PublicPort()), uint32(g.f.Exposure.Local)}
	var (
		mu   sync.Mutex
		wg   sync.WaitGroup
		errs []error
	)
	for _, c := range g.candidates(l) {
		g.f.Candidates = append(g.f.Candidates, c)
		if !c.InStep {
			if s, ok := l.stored[c.Node]; ok && s.read && c.Node != origin {
				g.f.Nodes[c.Node] = s.facts
			}
			continue
		}
		wg.Go(func() {
			fresh, dns, err := g.readNode(ctx, c, ports)
			if err != nil && c.Node != origin {
				return
			}
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, fmt.Errorf("read %s's network: %w", c.Name, err))
				return
			}
			g.fresh[c.Node] = fresh
			g.f.Nodes[c.Node], _ = r.cfg.Rules.Carry(l.stored[c.Node].carry, fresh)
			if dns != nil {
				g.f.DNS = append(g.f.DNS, *dns)
			}
		})
	}
	wg.Wait()
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	g.pair()
	return nil
}

// candidates is every live Node as a possible tester.
func (g *gathered) candidates(l loaded) []judge.Candidate {
	r := g.r
	relaying := r.cfg.Roster.Serving(role.Relaying)
	serves := g.f.Origin.Serving
	if g.f.Exposure.Kind == judge.MediaServer {
		rec := l.origins.servers[g.f.Exposure.Server]
		serves = []role.NodeID{rec.Reaching, rec.RunsOn}
	}
	out := make([]judge.Candidate, 0, len(l.origins.live))
	for id, name := range l.origins.live {
		c := judge.Candidate{Node: id, Name: name, InStep: r.cfg.Roster.InStep(id), Self: id == r.cfg.Roster.Self(), Relaying: contains(relaying, id), Serves: contains(serves, id)}
		if addr, _, ok := r.cfg.Roster.Remote(id); ok && !c.Self {
			c.Link = judge.LinkSeen{Addr: r.cfg.LinkSource(name, addr)}
		}
		out = append(out, c)
	}
	return out
}

// readNode is c's network now and its lookup of the public host, which
// is nil when the lookup itself failed.
func (g *gathered) readNode(ctx context.Context, c judge.Candidate, ports []uint32) (netmodel.NetworkFacts, *judge.DNSAnswer, error) {
	r := g.r
	resp, err := r.cfg.Network(c.Node).Read(ctx, &netv1.ReadRequest{Ports: ports, MaxAgeMs: r.maxAge(c.Node).Milliseconds()})
	if err != nil {
		return netmodel.NetworkFacts{}, nil, err
	}
	fresh, err := netmodel.ParseFacts(c.Node, resp.GetFacts(), r.cfg.Rules, r.now())
	if err != nil {
		return netmodel.NetworkFacts{}, nil, err
	}
	dns, err := r.resolve(ctx, c.Node, g.host)
	if err != nil {
		return fresh, nil, nil //nolint:nilerr // a failed lookup leaves the Node's facts standing
	}
	return fresh, &dns, nil
}

// pair maps each global address a Node holds or is seen from to that Node.
func (g *gathered) pair() {
	rules := g.r.cfg.Rules
	for id, nf := range g.f.Nodes {
		for _, i := range nf.Interfaces {
			for _, a := range i.Addrs {
				if rules.Global(a.Prefix.Addr()) {
					g.f.Paired[a.Prefix.Addr()] = id
				}
			}
		}
		if p := rules.Public4(nf.Observed()); p.IsValid() {
			g.f.Paired[p] = id
		}
	}
}

func contains(ids []role.NodeID, id role.NodeID) bool {
	for _, x := range ids {
		if x == id && !id.IsZero() {
			return true
		}
	}
	return false
}

// maxAge is how old node's last pass may be: the run's shared freshness,
// cut to the time since its last "Run again" or confirmed change.
func (r *Reachability) maxAge(node role.NodeID) time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()
	age := r.cfg.Timings.ReadFresh
	if at, ok := r.freshAfter[node]; ok {
		age = min(age, r.now().Sub(at))
	}
	return max(age, 0)
}

// store writes the result, each Node's facts with Carry run again on the
// row as it stands now, and the Media Server's facts, in one Write.
func (r *Reachability) store(ctx context.Context, g *gathered, j judge.Judged) error {
	now := r.now()
	return r.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		q := db.New(tx)
		if _, err := q.GetExposure(ctx, j.Exposure); errors.Is(err, storage.ErrNoRows) {
			return nil
		} else if err != nil {
			return err
		}
		if err := storeResult(ctx, q, j, now); err != nil {
			return err
		}
		for id, fresh := range g.fresh {
			if err := r.storeNode(ctx, q, id, fresh); err != nil {
				return err
			}
		}
		if g.server == nil {
			return nil
		}
		doc, err := encode(g.server)
		if err != nil {
			return err
		}
		return q.SetServerFacts(ctx, db.SetServerFactsParams{ID: j.Exposure, Facts: &doc, ObservedAt: storage.FromTime(now)})
	})
}

func storeResult(ctx context.Context, q *db.Queries, j judge.Judged, now time.Time) error {
	var prev time.Time
	switch row, err := q.GetResult(ctx, j.Exposure); {
	case err == nil:
		prev = row.FailingAt.Time()
	case !errors.Is(err, storage.ErrNoRows):
		return err
	}
	var testedBy role.NodeID
	for _, fv := range j.Families {
		if fv.Tested && !fv.Tester.IsZero() {
			testedBy = fv.Tester
			break
		}
	}
	p, err := resultRow(j, now, nextFailingAt(prev, j, now), testedBy)
	if err != nil {
		return err
	}
	return q.UpsertResult(ctx, p)
}

func (r *Reachability) storeNode(ctx context.Context, q *db.Queries, id role.NodeID, fresh netmodel.NetworkFacts) error {
	var s stored
	switch row, err := q.GetNode(ctx, id); {
	case err == nil:
		if s, err = storedOf(row); err != nil {
			return err
		}
	case !errors.Is(err, storage.ErrNoRows):
		return err
	}
	carried, next := r.cfg.Rules.Carry(s.carry, fresh)
	row, err := nodeRow(id, carried, next)
	if err != nil {
		return err
	}
	return q.UpsertNode(ctx, row)
}
