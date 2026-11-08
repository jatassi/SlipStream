package trackers_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/indexers/indexerstest"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/secret/secrettest"
	"github.com/jatassi/slipstream/internal/storage"
	tc "github.com/jatassi/slipstream/internal/torrent/clients"
	"github.com/jatassi/slipstream/internal/trackers"
)

// board is who holds Downloading as a routing test sets it: the Nodes
// assigned it, the Nodes that only cover it, and the Node a Work's Grabs
// go to by its library root.
type board struct {
	assigned, covering []role.NodeID
	byRoot             role.NodeID
}

func (b *board) Holding(role.Name) []role.NodeID { return b.assigned }
func (b *board) Holds(_ role.Name, n role.NodeID) bool {
	return slices.Contains(b.assigned, n) || slices.Contains(b.covering, n)
}
func (*board) WrittenOff(role.Name, role.NodeID) bool              { return false }
func (b *board) Route(role.Name, role.NodeID) (role.NodeID, error) { return b.byRoot, nil }

// routing is a library where Heron's Indexer searches from seedbox, which
// holds Downloading as den does, and Grabs go to attic by their root.
// Transmission is an external client on den.
type routing struct {
	*library
	seedbox, den, attic role.NodeID
	heron               indexers.ID
	transmission        downloadclients.ClientID
}

func newRouting(t *testing.T) *routing {
	t.Helper()
	r := &routing{
		library: newLibrary(t), seedbox: role.NodeID{UUID: storage.NewUUID()}, den: role.NodeID{UUID: storage.NewUUID()},
		attic: role.NodeID{UUID: storage.NewUUID()}, heron: indexers.ID{UUID: storage.NewUUID()},
	}
	for id, name := range map[role.NodeID]string{r.seedbox: "seedbox", r.den: "den", r.attic: "attic"} {
		r.exec(`INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, randomblob(32), ?, 0)`, id, name)
	}
	r.board.assigned, r.board.byRoot = []role.NodeID{r.seedbox, r.den}, r.attic
	sealer := secrettest.Sealer(t)
	address, err := tc.ParseAddress("http://192.168.1.20:9091", tc.Transmission)
	if err != nil {
		t.Fatal(err)
	}
	label, err := tc.ParseLabel(tc.DefaultLabel, tc.Transmission)
	if err != nil {
		t.Fatal(err)
	}
	err = r.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		if err := indexerstest.Torznab(ctx, tx, sealer, r.heron, "Heron", r.seedbox, "https://heron.test/api", "k"); err != nil {
			return err
		}
		r.transmission, err = downloadclients.Import(ctx, tx, sealer, downloadclients.Imported{
			Node: r.den, Kind: tc.Transmission, Name: "Transmission", Address: address, Label: label,
		}, storage.Now())
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// reached is a Tracker of privacy that Heron's Indexer reaches.
func (r *routing) reached(privacy indexers.Privacy) indexers.TrackerID {
	r.t.Helper()
	id := r.tracker("Heron "+string(privacy), privacy)
	r.exec(`INSERT INTO indexers_reach (indexer_id, tracker_id) VALUES (?, ?)`, r.heron, id)
	return id
}

func (r *routing) setRoute(tracker indexers.TrackerID, c downloadclients.Choice) error {
	_, err := r.x.SetRoute(r.t.Context(), identity.Admin{}, tracker, r.seedbox, c)
	return err
}

// route is where a Grab of tracker that Heron found goes, sent to sent.
func (r *routing) route(tracker indexers.TrackerID, sent *downloadclients.Choice) downloadclients.Target {
	r.t.Helper()
	router := downloadclients.Router{Roster: r.board, Routes: trackers.Routing{Roster: r.board}}
	var got downloadclients.Target
	err := r.lib.Read(r.t.Context(), func(ctx context.Context, rd storage.Reader[storage.Library]) error {
		var err error
		got, err = router.Route(ctx, rd, downloadclients.Ask{Root: r.attic, Indexer: r.heron, Tracker: tracker, Sent: sent})
		return err
	})
	if err != nil {
		r.t.Fatal(err)
	}
	return got
}

func sameTarget(a, b downloadclients.Target) bool {
	return a.Node == b.Node && (a.Client == nil) == (b.Client == nil) && (a.Client == nil || *a.Client == *b.Client)
}

// TestRouteFollowsTheTrackerRoutingTable is the spec's Tracker routing
// table: a private Tracker's Grab goes to its stored route, else to the
// built-in client of the Indexer's home while that home itself holds
// Downloading, and every other Grab goes by its Work's root. A send-to
// stands while its client's Node holds Downloading.
func TestRouteFollowsTheTrackerRoutingTable(t *testing.T) {
	for _, c := range []struct {
		name    string
		privacy indexers.Privacy
		// set stores routes and moves the roster, and answers where a
		// Grab is sent, nil for nowhere.
		set  func(t *testing.T, r *routing, tracker indexers.TrackerID) *downloadclients.Choice
		want func(r *routing) downloadclients.Target
	}{
		{
			name: "private with no route goes to the home's built-in client", privacy: indexers.Private,
			want: func(r *routing) downloadclients.Target { return downloadclients.Target{Node: r.seedbox} },
		},
		{
			name: "private with no route ignores where the home's New Grabs go", privacy: indexers.Private,
			set: func(_ *testing.T, r *routing, _ indexers.TrackerID) *downloadclients.Choice {
				r.exec(`INSERT INTO downloadclients_node (node_id, grabs_to) VALUES (?, ?)`, r.seedbox, r.transmission)
				return nil
			},
			want: func(r *routing) downloadclients.Target { return downloadclients.Target{Node: r.seedbox} },
		},
		{
			name: "private with no route goes by its root while the home only covers Downloading", privacy: indexers.Private,
			set: func(_ *testing.T, r *routing, _ indexers.TrackerID) *downloadclients.Choice {
				r.board.assigned, r.board.covering = []role.NodeID{r.den}, []role.NodeID{r.seedbox}
				return nil
			},
			want: func(r *routing) downloadclients.Target { return downloadclients.Target{Node: r.attic} },
		},
		{
			name: "private with no route goes by its root while the home doesn't download", privacy: indexers.Private,
			set: func(_ *testing.T, r *routing, _ indexers.TrackerID) *downloadclients.Choice {
				r.board.assigned = []role.NodeID{r.den}
				return nil
			},
			want: func(r *routing) downloadclients.Target { return downloadclients.Target{Node: r.attic} },
		},
		{
			name: "private with a stored route to a built-in client", privacy: indexers.Private,
			set: func(t *testing.T, r *routing, tracker indexers.TrackerID) *downloadclients.Choice {
				mustNot(t, r.setRoute(tracker, downloadclients.Choice{BuiltInOn: r.den}))
				return nil
			},
			want: func(r *routing) downloadclients.Target { return downloadclients.Target{Node: r.den} },
		},
		{
			name: "private with a stored route to an external client", privacy: indexers.Private,
			set: func(t *testing.T, r *routing, tracker indexers.TrackerID) *downloadclients.Choice {
				mustNot(t, r.setRoute(tracker, downloadclients.Choice{Client: &r.transmission}))
				return nil
			},
			want: func(r *routing) downloadclients.Target {
				return downloadclients.Target{Node: r.den, Client: &r.transmission}
			},
		},
		{
			name: "private with a stored route whose Node no longer holds Downloading goes to the default", privacy: indexers.Private,
			set: func(t *testing.T, r *routing, tracker indexers.TrackerID) *downloadclients.Choice {
				mustNot(t, r.setRoute(tracker, downloadclients.Choice{BuiltInOn: r.den}))
				r.board.assigned = []role.NodeID{r.seedbox}
				return nil
			},
			want: func(r *routing) downloadclients.Target { return downloadclients.Target{Node: r.seedbox} },
		},
		{
			name: "private with a stored route to a client turned off goes to the default", privacy: indexers.Private,
			set: func(t *testing.T, r *routing, tracker indexers.TrackerID) *downloadclients.Choice {
				mustNot(t, r.setRoute(tracker, downloadclients.Choice{Client: &r.transmission}))
				r.exec(`UPDATE downloadclients_client SET is_on = 0`)
				return nil
			},
			want: func(r *routing) downloadclients.Target { return downloadclients.Target{Node: r.seedbox} },
		},
		{
			name: "sent to a client while its Node holds Downloading, over the stored route", privacy: indexers.Private,
			set: func(t *testing.T, r *routing, tracker indexers.TrackerID) *downloadclients.Choice {
				mustNot(t, r.setRoute(tracker, downloadclients.Choice{Client: &r.transmission}))
				return &downloadclients.Choice{BuiltInOn: r.den}
			},
			want: func(r *routing) downloadclients.Target { return downloadclients.Target{Node: r.den} },
		},
		{
			name: "sent to a client whose Node no longer holds Downloading is routed again", privacy: indexers.Private,
			set: func(_ *testing.T, r *routing, _ indexers.TrackerID) *downloadclients.Choice {
				r.board.assigned = []role.NodeID{r.seedbox}
				return &downloadclients.Choice{BuiltInOn: r.den}
			},
			want: func(r *routing) downloadclients.Target { return downloadclients.Target{Node: r.seedbox} },
		},
		{
			name: "private whose Indexer was deleted goes by its root", privacy: indexers.Private,
			set: func(t *testing.T, r *routing, tracker indexers.TrackerID) *downloadclients.Choice {
				mustNot(t, r.setRoute(tracker, downloadclients.Choice{BuiltInOn: r.den}))
				r.exec(`DELETE FROM indexers_indexer WHERE id = ?`, r.heron)
				return nil
			},
			want: func(r *routing) downloadclients.Target { return downloadclients.Target{Node: r.attic} },
		},
		{name: "public goes by its root", privacy: indexers.Public, want: func(r *routing) downloadclients.Target { return downloadclients.Target{Node: r.attic} }},
		{name: "semi-private goes by its root", privacy: indexers.SemiPrivate, want: func(r *routing) downloadclients.Target { return downloadclients.Target{Node: r.attic} }},
		{name: "unknown goes by its root", privacy: indexers.Unknown, want: func(r *routing) downloadclients.Target { return downloadclients.Target{Node: r.attic} }},
		{
			name: "a stored route of an unknown Tracker waits for it to turn private", privacy: indexers.Unknown,
			set: func(t *testing.T, r *routing, tracker indexers.TrackerID) *downloadclients.Choice {
				mustNot(t, r.setRoute(tracker, downloadclients.Choice{BuiltInOn: r.den}))
				return nil
			},
			want: func(r *routing) downloadclients.Target { return downloadclients.Target{Node: r.attic} },
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			r := newRouting(t)
			tracker := r.reached(c.privacy)
			var sent *downloadclients.Choice
			if c.set != nil {
				sent = c.set(t, r, tracker)
			}
			if got, want := r.route(tracker, sent), c.want(r); !sameTarget(got, want) {
				t.Errorf("Route = %+v, want %+v", got, want)
			}
		})
	}
	t.Run("a Grab with no Tracker goes by its root", func(t *testing.T) {
		r := newRouting(t)
		if got := r.route(indexers.TrackerID{}, nil); !sameTarget(got, downloadclients.Target{Node: r.attic}) {
			t.Errorf("Route = %+v, want attic's built-in client", got)
		}
	})
}

// TestRoutesListEachHomeAndRefuseWhatCantRoute lists seedbox's route with
// its default and the Admin's pick, marks a pick whose Node no longer
// downloads as skipped, and refuses a route on a public Tracker, from a
// Node that is no Indexer's home, and to a Node that doesn't download.
func TestRoutesListEachHomeAndRefuseWhatCantRoute(t *testing.T) {
	r := newRouting(t)
	private, public := r.reached(indexers.Private), r.reached(indexers.Public)
	ctx := t.Context()

	routes, err := r.x.SetRoute(ctx, identity.Admin{}, private, r.seedbox, downloadclients.Choice{BuiltInOn: r.den})
	if err != nil {
		t.Fatal(err)
	}
	if len(routes) != 1 || routes[0].Home != r.seedbox || routes[0].Stored == nil || routes[0].Stored.BuiltInOn != r.den ||
		routes[0].Default == nil || routes[0].Default.Node != r.seedbox || routes[0].To == nil || routes[0].To.Node != r.den || routes[0].Skipped {
		t.Fatalf("after the save the routes are %+v, want seedbox's to den over its default", routes)
	}
	if r.rerouted != 1 || len(r.changed.trackers) != 1 || *r.changed.trackers[0] != private.String() {
		t.Fatalf("the save rerouted %d times and published %v, want once each for the Tracker", r.rerouted, r.changed.trackers)
	}

	r.board.assigned = []role.NodeID{r.seedbox}
	routes, err = r.x.Routes(ctx, identity.Admin{}, private)
	if err != nil {
		t.Fatal(err)
	}
	if len(routes) != 1 || !routes[0].Skipped || routes[0].To == nil || routes[0].To.Node != r.seedbox {
		t.Fatalf("once den no longer downloads the routes are %+v, want den skipped for seedbox", routes)
	}
	if routes, err = r.x.ClearRoute(ctx, identity.Admin{}, private, r.seedbox); err != nil || routes[0].Stored != nil {
		t.Fatalf("ClearRoute = %+v, %v; want seedbox's route gone", routes, err)
	}

	if _, err := r.x.SetRoute(ctx, identity.Admin{}, public, r.seedbox, downloadclients.Choice{BuiltInOn: r.seedbox}); !errors.As(err, new(*trackers.PublicError)) {
		t.Errorf("a route on a public Tracker answered %v, want *PublicError", err)
	}
	notHome := &trackers.NotAHomeError{}
	if _, err := r.x.SetRoute(ctx, identity.Admin{}, private, r.den, downloadclients.Choice{BuiltInOn: r.seedbox}); !errors.As(err, &notHome) || notHome.Node != "den" {
		t.Errorf("a route from den answered %v, want *NotAHomeError naming den", err)
	}
	if err := r.setRoute(private, downloadclients.Choice{BuiltInOn: r.attic}); !errors.Is(err, downloadclients.ErrNotDownloading) {
		t.Errorf("a route to attic, which doesn't download, answered %v, want ErrNotDownloading", err)
	}
}

// TestAMergeKeepsTheKeptTrackersRouteFromEachHome merges a private Tracker
// with routes into another with a route from the same home: the kept
// Tracker's route stays and the merged one's goes with it.
func TestAMergeKeepsTheKeptTrackersRouteFromEachHome(t *testing.T) {
	r := newRouting(t)
	kept, merged := r.reached(indexers.Private), r.tracker("Heron Mirror", indexers.Private)
	r.exec(`INSERT INTO indexers_reach (indexer_id, tracker_id) VALUES (?, ?)`, r.heron, merged)
	mustNot(t, r.setRoute(kept, downloadclients.Choice{BuiltInOn: r.den}))
	mustNot(t, r.setRoute(merged, downloadclients.Choice{Client: &r.transmission}))

	err := r.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return trackers.RepointTracker(ctx, tx, merged, kept)
	})
	mustNot(t, err)
	r.exec(`DELETE FROM indexers_tracker WHERE id = ?`, merged)
	routes, err := r.x.Routes(t.Context(), identity.Admin{}, kept)
	mustNot(t, err)
	if len(routes) != 1 || routes[0].Stored == nil || routes[0].Stored.Client != nil || routes[0].Stored.BuiltInOn != r.den {
		t.Fatalf("after the merge the kept Tracker's routes are %+v, want its own to den", routes)
	}
	var left int
	err = r.lib.Read(t.Context(), func(ctx context.Context, rd storage.Reader[storage.Library]) error {
		return rd.QueryRowContext(ctx, `SELECT count(*) FROM trackers_route`).Scan(&left)
	})
	if err != nil || left != 1 {
		t.Fatalf("after the merge %d routes are stored (%v), want only the kept one", left, err)
	}
}

// TestRemovingATrackerDeletesItsRoutes removes a private Tracker with a
// route once no Indexer reaches it, and no route is left.
func TestRemovingATrackerDeletesItsRoutes(t *testing.T) {
	r := newRouting(t)
	tracker := r.reached(indexers.Private)
	mustNot(t, r.setRoute(tracker, downloadclients.Choice{BuiltInOn: r.den}))
	r.exec(`DELETE FROM indexers_reach WHERE tracker_id = ?`, tracker)
	mustNot(t, r.x.Remove(t.Context(), identity.Admin{}, tracker))
	var left int
	err := r.lib.Read(t.Context(), func(ctx context.Context, rd storage.Reader[storage.Library]) error {
		return rd.QueryRowContext(ctx, `SELECT count(*) FROM trackers_route`).Scan(&left)
	})
	if err != nil || left != 0 {
		t.Fatalf("after the removal %d routes are stored (%v), want none", left, err)
	}
}

// TestRoutedToListsEachClientsPrivateTrackers puts a private Tracker on
// seedbox's built-in client by default and another on Transmission by the
// Admin's pick. A public Tracker goes by its Works' roots, so no client
// lists it.
func TestRoutedToListsEachClientsPrivateTrackers(t *testing.T) {
	r := newRouting(t)
	byDefault, picked := r.reached(indexers.Private), r.tracker("Heron Mirror", indexers.Private)
	r.exec(`INSERT INTO indexers_reach (indexer_id, tracker_id) VALUES (?, ?)`, r.heron, picked)
	r.reached(indexers.Public)
	mustNot(t, r.setRoute(picked, downloadclients.Choice{Client: &r.transmission}))

	here, err := r.x.RoutedTo(t.Context(), identity.Admin{})
	mustNot(t, err)
	if got := here.BuiltIn[r.seedbox]; len(got) != 1 || got[0].ID != byDefault || got[0].Stored {
		t.Errorf("seedbox's built-in client lists %+v, want only Heron private by default", got)
	}
	if got := here.External[r.transmission]; len(got) != 1 || got[0].ID != picked || !got[0].Stored {
		t.Errorf("Transmission lists %+v, want only Heron Mirror by the Admin's pick", got)
	}
	if len(here.BuiltIn) != 1 || len(here.External) != 1 {
		t.Errorf("RoutedTo = %+v, want only seedbox's built-in client and Transmission", here)
	}
}
