package reachability

import (
	"cmp"
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/netmodel"
	"github.com/jatassi/slipstream/internal/nodes"
	netv1 "github.com/jatassi/slipstream/internal/pb/slipstream/network/v1"
	"github.com/jatassi/slipstream/internal/reachability/internal/db"
	"github.com/jatassi/slipstream/internal/reachability/judge"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// Page is Remote access as the Admin sees it.
type Page struct {
	Routes      []Route
	Status      judge.Status
	LastRun     time.Time
	Testers     []role.NodeID
	Suggestions []Suggestion
	Nodes       []NodeLine
	Words       Words
	Zone        *time.Location
}

func (p Page) NothingWanted() bool { return len(p.Routes) == 0 }

// Route is one wanted Exposure and its last result.
type Route struct {
	Exposure Exposure
	Subject  string
	Public   string
	// Judged is false until a check of this version stored a result.
	Judged   bool
	Result   judge.Judged
	JudgedAt time.Time
	// Away is the origin offline: the result is from before it left.
	Away     bool
	Origin   role.NodeID
	TestedBy role.NodeID
}

// NodeLine is a Node's class, or that nothing on it is checked.
type NodeLine struct {
	Node    role.NodeID
	Name    string
	Watched bool
	Read    bool
	Class   netmodel.Class4
}

// Suggestion is something SlipStream could set up for use away from home.
type Suggestion struct {
	Kind    judge.Kind
	Server  link.ServerID
	Subject string
	Public  string
	// AddressPrivate is the UI and portal with no Address outsiders can
	// use: the Admin sets one first.
	AddressPrivate bool
}

// RemoteAccess reads the page.
func (r *Reachability) RemoteAccess(ctx context.Context, _ identity.Admin) (Page, error) {
	named, err := r.cfg.Roster.Names(ctx)
	if err != nil {
		return Page{}, err
	}
	var in pageRows
	err = r.cfg.Library.Read(ctx, func(ctx context.Context, rd storage.Reader[storage.Library]) error {
		in, err = r.pageRowsIn(ctx, rd)
		return err
	})
	if err != nil {
		return Page{}, err
	}
	p := Page{Words: r.words(named), Zone: r.cfg.Zone, Routes: r.routes(in.wanted, in.origins, in.results, in.addr)}
	var judged []judge.Judged
	for _, rt := range p.Routes {
		if rt.Judged {
			judged = append(judged, rt.Result)
		}
		if rt.JudgedAt.After(p.LastRun) {
			p.LastRun = rt.JudgedAt
		}
		if !rt.TestedBy.IsZero() && !slices.Contains(p.Testers, rt.TestedBy) {
			p.Testers = append(p.Testers, rt.TestedBy)
		}
	}
	p.Status = judge.Count(judged)
	if len(in.wanted) > 0 {
		p.Nodes = r.nodeLines(named, in.origins, in.wanted, in.stored)
	}
	p.Suggestions = r.suggestions(ctx, in.wanted, in.dismissed, in.recs, in.addr)
	return p, nil
}

// pageRows is everything the page reads of the library, in one Read.
type pageRows struct {
	addr      identity.Address
	dismissed []db.ReachabilityDismissed
	recs      []mediaservers.Record
	stored    map[role.NodeID]stored
	results   map[judge.ExposureID]result
	origins   origins
	wanted    []Exposure
}

func (r *Reachability) pageRowsIn(ctx context.Context, rd storage.Reader[storage.Library]) (pageRows, error) {
	q := db.New(rd)
	var (
		in  pageRows
		err error
	)
	if in.wanted, err = wantedIn(ctx, rd); err != nil {
		return in, err
	}
	if in.origins, err = originsIn(ctx, rd, r.cfg.Roster.Self()); err != nil {
		return in, err
	}
	if in.addr, err = identity.AddressIn(ctx, rd); err != nil {
		return in, err
	}
	if in.dismissed, err = q.ListDismissals(ctx); err != nil {
		return in, err
	}
	if in.recs, err = mediaservers.ServersIn(ctx, rd); err != nil {
		return in, err
	}
	if in.results, err = resultsIn(ctx, q); err != nil {
		return in, err
	}
	in.stored, err = storedIn(ctx, q)
	return in, err
}

func resultsIn(ctx context.Context, q *db.Queries) (map[judge.ExposureID]result, error) {
	rows, err := q.ListResults(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[judge.ExposureID]result, len(rows))
	for _, row := range rows {
		if out[row.ExposureID], err = resultOf(row); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (r *Reachability) routes(wanted []Exposure, o origins, results map[judge.ExposureID]result, addr identity.Address) []Route {
	out := make([]Route, 0, len(wanted))
	for _, e := range wanted {
		origin, _ := o.of(e)
		rt := Route{Exposure: e, Subject: o.subject(e), Origin: origin}
		if e.Public != nil {
			rt.Public = e.Public.String()
		} else {
			rt.Public = addr.String()
		}
		if res, ok := results[e.ID]; ok {
			rt.Judged, rt.Result, rt.JudgedAt, rt.TestedBy = res.read, res.judged, res.judgedAt, res.testedBy
			rt.Away = !r.cfg.Roster.InStep(origin)
		}
		out = append(out, rt)
	}
	return out
}

func (r *Reachability) nodeLines(named map[role.NodeID]nodes.Named, o origins, wanted []Exposure, stored map[role.NodeID]stored) []NodeLine {
	var out []NodeLine
	for id, n := range named {
		if n.Removed {
			continue
		}
		line := NodeLine{Node: id, Name: n.Name, Watched: r.watches(o, wanted, id)}
		if s, ok := stored[id]; ok && s.read && s.facts.Watched {
			line.Read = true
			line.Class, _ = r.cfg.Rules.Classify(s.facts.Observed())
		}
		out = append(out, line)
	}
	slices.SortFunc(out, func(a, b NodeLine) int {
		return cmp.Or(-cmp.Compare(btoi(a.Node == o.self), btoi(b.Node == o.self)), cmp.Compare(a.Name, b.Name))
	})
	return out
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

// mediaSuggestion is what a Media Server's facts suggest: whether it is
// meant for use away from home, and the public address it publishes.
type mediaSuggestion struct {
	ok     bool
	public string
}

// suggestions are the UI and portal when its Address is public, and each
// Media Server that has remote access on or publishes a public address.
// Each lookup is kept for timings.reachability.suggest.
func (r *Reachability) suggestions(ctx context.Context, wanted []Exposure, dismissed []db.ReachabilityDismissed, recs []mediaservers.Record, addr identity.Address) []Suggestion {
	skip := map[link.ServerID]bool{}
	ui := false
	for _, e := range wanted {
		skip[e.Server] = true
		ui = ui || e.Kind == judge.UIPortal
	}
	for _, d := range dismissed {
		skip[d.ServerID] = true
		ui = ui || d.Kind == KindName(judge.UIPortal)
	}
	var out []Suggestion
	if !ui {
		out = append(out, Suggestion{Kind: judge.UIPortal, Subject: uiSubject, Public: addr.String(), AddressPrivate: !r.addressPublic(ctx, addr)})
	}
	for _, rec := range recs {
		if skip[rec.ID] {
			continue
		}
		if s := r.mediaSuggestion(ctx, rec); s.ok {
			out = append(out, Suggestion{Kind: judge.MediaServer, Server: rec.ID, Subject: rec.Name, Public: s.public})
		}
	}
	return out
}

// addressPublic is whether the Controller resolves Address to a Global
// IPv4 address: it is decided by resolution, never by the name.
func (r *Reachability) addressPublic(ctx context.Context, addr identity.Address) bool {
	if addr.IsZero() {
		return false
	}
	now := r.now()
	if v, ok := r.public.get(addr.String(), now, r.cfg.Timings.Suggest); ok {
		return v
	}
	a, err := r.resolve(ctx, r.cfg.Roster.Self(), addr.URL().Hostname())
	if err != nil {
		return false
	}
	public := slices.ContainsFunc(a.A, r.cfg.Rules.Global)
	r.public.put(addr.String(), public, now)
	return public
}

func (r *Reachability) mediaSuggestion(ctx context.Context, rec mediaservers.Record) mediaSuggestion {
	now := r.now()
	if v, ok := r.exposed.get(rec.ID, now, r.cfg.Timings.Suggest); ok {
		return v
	}
	x, err := r.cfg.MediaServers.Exposure(ctx, rec.ID)
	if err != nil {
		return mediaSuggestion{}
	}
	doc := serverDocFrom(x, rec)
	s := mediaSuggestion{public: prefill(doc.Facts)}
	s.ok = x.Exposed != nil && x.Exposed.Remote == link.Yes
	for _, c := range doc.Facts.Published {
		s.ok = s.ok || (!c.Local && !c.Relay && r.cfg.Rules.Global(c.Addr))
	}
	r.exposed.put(rec.ID, s, now)
	return s
}

// prefill is the public address a suggestion offers: the first https
// custom URL that isn't a plex.direct name, else the first custom URL.
func prefill(f judge.ServerFacts) string {
	for _, c := range f.Custom {
		if strings.HasPrefix(c, "https://") && !strings.Contains(c, ".plex.direct") {
			return strings.TrimSuffix(strings.TrimSuffix(c, ":443"), "/")
		}
	}
	if len(f.Custom) > 0 {
		return f.Custom[0]
	}
	return ""
}

// NodeNetwork is one Node's Network section.
type NodeNetwork struct {
	Node  role.NodeID
	Name  string
	Line  NodeLine
	Facts netmodel.NetworkFacts
	// Now is the facts read on opening, with nothing sent outside the
	// machine; false is the last stored pass, for a Node that is away.
	Now bool
}

// NodeNetwork reads node's Network section. A Node in step answers with
// its interfaces, listeners and firewall read now and nothing leaving the
// machine.
func (r *Reachability) NodeNetwork(ctx context.Context, _ identity.Admin, node role.NodeID) (NodeNetwork, error) {
	named, err := r.cfg.Roster.Names(ctx)
	if err != nil {
		return NodeNetwork{}, err
	}
	if n, ok := named[node]; !ok || n.Removed {
		return NodeNetwork{}, ErrNodeNotFound
	}
	var (
		s      stored
		o      origins
		wanted []Exposure
	)
	err = r.cfg.Library.Read(ctx, func(ctx context.Context, rd storage.Reader[storage.Library]) error {
		var err error
		if wanted, err = wantedIn(ctx, rd); err != nil {
			return err
		}
		if o, err = originsIn(ctx, rd, r.cfg.Roster.Self()); err != nil {
			return err
		}
		row, err := db.New(rd).GetNode(ctx, node)
		if errors.Is(err, storage.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		s, err = storedOf(row)
		return err
	})
	if err != nil {
		return NodeNetwork{}, err
	}
	line := r.nodeLines(map[role.NodeID]nodes.Named{node: named[node]}, o, wanted, map[role.NodeID]stored{node: s})[0]
	out := NodeNetwork{Node: node, Name: named[node].Name, Line: line, Facts: s.facts}
	if !r.cfg.Roster.InStep(node) {
		return out, nil
	}
	resp, err := r.cfg.Network(node).Read(ctx, &netv1.ReadRequest{Ports: []uint32{uint32(r.cfg.WebPort())}, LocalOnly: true})
	if err != nil {
		return out, nil //nolint:nilerr // a Node that can't answer now shows its last stored pass
	}
	f, err := netmodel.ParseFacts(node, resp.GetFacts(), r.cfg.Rules)
	if err != nil {
		return NodeNetwork{}, err
	}
	out.Facts, out.Now = f, true
	return out, nil
}
