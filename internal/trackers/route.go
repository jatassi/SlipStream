package trackers

import (
	"cmp"
	"context"
	"errors"
	"slices"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/trackers/internal/db"
)

// RouteRoster is what Tracker routing reads of the Nodes.
type RouteRoster interface {
	// Holding is every Node assigned r, not only covering it.
	Holding(r role.Name) []role.NodeID
	// Holds reports whether node is assigned r, in step or not, or covers
	// it.
	Holds(r role.Name, node role.NodeID) bool
	WrittenOff(r role.Name, node role.NodeID) bool
}

// Routing is Tracker routing, which downloadclients.Router asks before it
// routes a Grab by its Work's library root. Only a private Tracker routes:
// a torrent's private flag is unknown before its fetch, so a semi-private
// or unknown Tracker's Grabs go by the root as a public one's do.
type Routing struct{ Roster RouteRoster }

// Route is the client a Grab of tracker that Indexer indexer found goes
// to, false when it goes by its Work's library root. The home is the
// Indexer's stored home, never a Node that covers for it, so a Grab whose
// Indexer was deleted goes by its root.
func (g Routing) Route(ctx context.Context, r storage.Reader[file], tracker indexers.TrackerID, indexer indexers.ID) (downloadclients.Target, bool, error) {
	if tracker.IsZero() {
		return downloadclients.Target{}, false, nil
	}
	p, err := indexers.PrivacyOf(ctx, r, tracker)
	if err != nil || p != indexers.Private {
		return downloadclients.Target{}, false, err
	}
	idx, err := indexers.Get(ctx, r, indexer)
	if errors.Is(err, indexers.ErrNoIndexer) {
		return downloadclients.Target{}, false, nil
	}
	if err != nil {
		return downloadclients.Target{}, false, err
	}
	stored, err := storedRoutes(ctx, r, tracker)
	if err != nil {
		return downloadclients.Target{}, false, err
	}
	way, err := g.way(ctx, r, idx.Home, stored[idx.Home])
	if err != nil || way.To == nil {
		return downloadclients.Target{}, false, err
	}
	return *way.To, true, nil
}

// Route is where a private Tracker's Grabs go when an Indexer on Home
// found them.
type Route struct {
	Home role.NodeID
	// To is where they go now: Stored while its client may take them,
	// else Default. It is nil while they go by the Work's library root.
	To *downloadclients.Target
	// Default is Home's built-in client while Home itself holds
	// Downloading, nil while it doesn't.
	Default *downloadclients.Target
	// Stored is the client the Admin picked, nil for none.
	Stored *downloadclients.Choice
	// Skipped is a Stored client that can't take a Grab now: an off or
	// deleted client, or one whose Node doesn't hold Downloading.
	Skipped bool
}

// way is where Grabs from home go, with stored the Admin's pick for it.
func (g Routing) way(ctx context.Context, r storage.Reader[file], home role.NodeID, stored *downloadclients.Choice) (Route, error) {
	out := Route{Home: home, Stored: stored}
	if slices.Contains(g.Roster.Holding(role.Downloading), home) {
		out.Default = &downloadclients.Target{Node: home}
		out.To = out.Default
	}
	if stored == nil {
		return out, nil
	}
	to, err := downloadclients.Usable(ctx, r, *stored, g.downloading)
	if downloadclients.Unusable(err) {
		out.Skipped = true
		return out, nil
	}
	if err != nil {
		return Route{}, err
	}
	out.To = &to
	return out, nil
}

func (g Routing) downloading(node role.NodeID) bool {
	return g.Roster.Holds(role.Downloading, node) && !g.Roster.WrittenOff(role.Downloading, node)
}

func storedRoutes(ctx context.Context, r storage.Reader[file], tracker indexers.TrackerID) (map[role.NodeID]*downloadclients.Choice, error) {
	rows, err := db.New(r).ListRoutesOf(ctx, tracker)
	if err != nil {
		return nil, err
	}
	out := make(map[role.NodeID]*downloadclients.Choice, len(rows))
	for _, row := range rows {
		out[row.HomeNodeID] = choiceOf(row)
	}
	return out, nil
}

func choiceOf(row db.TrackersRoute) *downloadclients.Choice {
	if row.ClientID != nil {
		return &downloadclients.Choice{Client: row.ClientID}
	}
	return &downloadclients.Choice{BuiltInOn: *row.BuiltInOn}
}

// ErrNoNode refuses a route from a Node no Node has the id of.
var ErrNoNode = errors.New("no Node has that id")

// NotAHomeError refuses a route from Node, whose Indexers don't reach
// Tracker.
type NotAHomeError struct{ Tracker, Node string }

func (e *NotAHomeError) Error() string { return "no Indexer on the Node reaches the Tracker" }

// Routes is where private Tracker id's Grabs go from the home of each of
// its Indexers, by the home's name. A Tracker that isn't private has
// none. It fails with ErrNoTracker.
func (t *Trackers) Routes(ctx context.Context, _ identity.Admin, id indexers.TrackerID) ([]Route, error) {
	var out []Route
	err := t.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		out, err = t.routesIn(ctx, r, id)
		return err
	})
	return out, err
}

// SetRoute sends private Tracker id's Grabs from Indexers on home to the
// client c names. It fails with ErrNoTracker, *PublicError for a public
// Tracker, ErrNoNode, *NotAHomeError, and downloadclients.Usable's errors.
func (t *Trackers) SetRoute(ctx context.Context, _ identity.Admin, id indexers.TrackerID, home role.NodeID, c downloadclients.Choice) ([]Route, error) {
	return t.routeWrite(ctx, id, home, func(ctx context.Context, tx *storage.Tx[file]) error {
		if _, err := downloadclients.Usable(ctx, tx, c, t.cfg.Routing.downloading); err != nil {
			return err
		}
		p := db.SetRouteParams{TrackerID: id, HomeNodeID: home, ClientID: c.Client}
		if c.Client == nil {
			p.BuiltInOn = &c.BuiltInOn
		}
		return db.New(tx).SetRoute(ctx, p)
	})
}

// ClearRoute deletes the route the Admin picked for Tracker id's Grabs
// from Indexers on home, so they go to the default. It fails like
// SetRoute.
func (t *Trackers) ClearRoute(ctx context.Context, _ identity.Admin, id indexers.TrackerID, home role.NodeID) ([]Route, error) {
	return t.routeWrite(ctx, id, home, func(ctx context.Context, tx *storage.Tx[file]) error {
		return db.New(tx).ClearRoute(ctx, db.ClearRouteParams{TrackerID: id, HomeNodeID: home})
	})
}

func (t *Trackers) routeWrite(ctx context.Context, id indexers.TrackerID, home role.NodeID, write func(context.Context, *storage.Tx[file]) error) ([]Route, error) {
	var out []Route
	err := t.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		s, err := summaryIn(ctx, tx, id)
		if err != nil {
			return err
		}
		if s.Privacy == indexers.Public {
			return &PublicError{Name: s.Name}
		}
		homes, err := homesOf(ctx, tx, s.Indexers)
		if err != nil {
			return err
		}
		if !slices.Contains(homes, home) {
			names, err := nodeNames(ctx, tx)
			if err != nil {
				return err
			}
			name, ok := names[home]
			if !ok {
				return ErrNoNode
			}
			return &NotAHomeError{Tracker: s.Name, Node: name}
		}
		if err := write(ctx, tx); err != nil {
			return err
		}
		out, err = t.routesIn(ctx, tx, id)
		return err
	})
	if err != nil {
		return nil, err
	}
	t.publish(ctx, &id)
	t.cfg.Rerouted()
	return out, nil
}

func (t *Trackers) routesIn(ctx context.Context, r storage.Reader[file], id indexers.TrackerID) ([]Route, error) {
	s, err := summaryIn(ctx, r, id)
	if err != nil || s.Privacy != indexers.Private {
		return []Route{}, err
	}
	homes, err := homesOf(ctx, r, s.Indexers)
	if err != nil {
		return nil, err
	}
	stored, err := storedRoutes(ctx, r, id)
	if err != nil {
		return nil, err
	}
	names, err := nodeNames(ctx, r)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(homes, func(a, b role.NodeID) int { return cmp.Compare(names[a], names[b]) })
	out := make([]Route, len(homes))
	for i, home := range homes {
		if out[i], err = t.cfg.Routing.way(ctx, r, home, stored[home]); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// homesOf is the distinct home Nodes of ids.
func homesOf(ctx context.Context, r storage.Reader[file], ids []indexers.ID) ([]role.NodeID, error) {
	var out []role.NodeID
	for _, id := range ids {
		idx, err := indexers.Get(ctx, r, id)
		if err != nil {
			return nil, err
		}
		if !slices.Contains(out, idx.Home) {
			out = append(out, idx.Home)
		}
	}
	return out, nil
}

// Routed is a private Tracker whose Grabs go to a client from some home,
// and whether that client is the Admin's pick for one.
type Routed struct {
	ID     indexers.TrackerID
	Name   string
	Stored bool
}

// RoutedHere is the private Trackers whose Grabs go to each client now,
// by name: a Node's built-in client by the Node, an external one by id.
type RoutedHere struct {
	BuiltIn  map[role.NodeID][]Routed
	External map[downloadclients.ClientID][]Routed
}

// RoutedTo is where every private Tracker's Grabs go now.
func (t *Trackers) RoutedTo(ctx context.Context, _ identity.Admin) (RoutedHere, error) {
	out := RoutedHere{BuiltIn: map[role.NodeID][]Routed{}, External: map[downloadclients.ClientID][]Routed{}}
	err := t.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		all, err := summariesIn(ctx, r)
		if err != nil {
			return err
		}
		for _, s := range all {
			routes, err := t.routesIn(ctx, r, s.ID)
			if err != nil {
				return err
			}
			for _, route := range routes {
				stored := route.Stored != nil && !route.Skipped
				switch to := route.To; {
				case to == nil:
				case to.Client != nil:
					out.External[*to.Client] = routedAs(out.External[*to.Client], s, stored)
				default:
					out.BuiltIn[to.Node] = routedAs(out.BuiltIn[to.Node], s, stored)
				}
			}
		}
		return nil
	})
	return out, err
}

// routedAs adds s to list once, stored when any of its routes there is.
func routedAs(list []Routed, s Summary, stored bool) []Routed {
	if i := slices.IndexFunc(list, func(r Routed) bool { return r.ID == s.ID }); i >= 0 {
		list[i].Stored = list[i].Stored || stored
		return list
	}
	return append(list, Routed{ID: s.ID, Name: s.Name, Stored: stored})
}
