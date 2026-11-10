package downloadclients_test

import (
	"context"
	"errors"
	"testing"

	dc "github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/secret/secrettest"
	"github.com/jatassi/slipstream/internal/storage"
)

// rootRoster routes each Work's Grabs to the Node of its root, which holds
// Downloading.
type rootRoster struct{}

func (rootRoster) Route(_ role.Name, home role.NodeID) (role.NodeID, error) { return home, nil }
func (rootRoster) Holds(role.Name, role.NodeID) bool                        { return true }
func (rootRoster) WrittenOff(role.Name, role.NodeID) bool                   { return false }

// noRoutes is Tracker routing with no private Tracker.
type noRoutes struct{}

func (noRoutes) Route(context.Context, storage.Reader[storage.Library], indexers.TrackerID, indexers.ID) (dc.Target, bool, error) {
	return dc.Target{}, false, nil
}

func route(t *testing.T, lib library, node role.NodeID) dc.Target {
	t.Helper()
	var got dc.Target
	err := lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		got, err = dc.Router{Roster: rootRoster{}, Routes: noRoutes{}}.Route(ctx, r, dc.Ask{Root: node, Tracker: indexers.TrackerID{UUID: storage.NewUUID()}})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func sendGrabsTo(t *testing.T, lib library, node role.NodeID, id dc.ClientID) {
	t.Helper()
	exec(t, lib, `INSERT INTO downloadclients_node (node_id, grabs_to) VALUES (?, ?)
		ON CONFLICT (node_id) DO UPDATE SET grabs_to = excluded.grabs_to`, node, id)
}

func TestRouteSendsANodesGrabsWhereItsNewGrabsGo(t *testing.T) {
	for _, c := range []struct {
		name string
		// set places the Node's clients and answers the one Route must
		// pick, nil for the built-in client.
		set func(t *testing.T, lib library, node role.NodeID) *dc.ClientID
	}{
		{"no setting", func(*testing.T, library, role.NodeID) *dc.ClientID { return nil }},
		{"settings but no New Grabs go to", func(t *testing.T, lib library, node role.NodeID) *dc.ClientID {
			exec(t, lib, "INSERT INTO downloadclients_node (node_id, download_limit) VALUES (?, 100)", node)
			return nil
		}},
		{"an external client of the Node's", func(t *testing.T, lib library, node role.NodeID) *dc.ClientID {
			id := mustImport(t, lib, secrettest.Sealer(t), transmission(t, node, "Transmission", "http://192.168.1.20:9091"))
			sendGrabsTo(t, lib, node, id)
			return &id
		}},
		{"an off client", func(t *testing.T, lib library, node role.NodeID) *dc.ClientID {
			id := mustImport(t, lib, secrettest.Sealer(t), transmission(t, node, "Transmission", "http://192.168.1.20:9091"))
			sendGrabsTo(t, lib, node, id)
			exec(t, lib, "UPDATE downloadclients_client SET is_on = 0 WHERE id = ?", id)
			return nil
		}},
		{"a deleted client", func(t *testing.T, lib library, node role.NodeID) *dc.ClientID {
			id := mustImport(t, lib, secrettest.Sealer(t), transmission(t, node, "Transmission", "http://192.168.1.20:9091"))
			sendGrabsTo(t, lib, node, id)
			exec(t, lib, "UPDATE downloadclients_client SET deleted_at = ?, secrets_sealed = NULL WHERE id = ?", storage.Now(), id)
			return nil
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			lib := open(t)
			node, other := addNode(t, lib, "Media server"), addNode(t, lib, "seedbox")
			want := c.set(t, lib, node)
			got := route(t, lib, node)
			if got.Node != node || (got.Client == nil) != (want == nil) || (want != nil && *got.Client != *want) {
				t.Errorf("Route = %+v, want Media server with client %v", got, want)
			}
			if got := route(t, lib, other); got.Node != other || got.Client != nil {
				t.Errorf("Route for seedbox = %+v, want its built-in client", got)
			}
		})
	}
}

// downloadingOn holds Downloading on the Nodes it names, and routes each
// Work's Grabs to the Node of its root.
type downloadingOn map[role.NodeID]bool

func (downloadingOn) Route(_ role.Name, home role.NodeID) (role.NodeID, error) { return home, nil }
func (d downloadingOn) Holds(_ role.Name, node role.NodeID) bool               { return d[node] }
func (downloadingOn) WrittenOff(role.Name, role.NodeID) bool                   { return false }

func TestAPinnedClientThatCantTakeTheGrabNeverFallsThrough(t *testing.T) {
	for _, c := range []struct {
		name string
		// sent places the client the Admin chose on den, made unusable.
		sent     func(t *testing.T, lib library, den role.NodeID) dc.Choice
		wantName string
	}{
		{"an off client", func(t *testing.T, lib library, den role.NodeID) dc.Choice {
			id := mustImport(t, lib, secrettest.Sealer(t), transmission(t, den, "Transmission", "http://192.168.1.20:9091"))
			exec(t, lib, "UPDATE downloadclients_client SET is_on = 0 WHERE id = ?", id)
			return dc.Choice{Client: &id}
		}, "Transmission on Den"},
		{"a deleted client", func(t *testing.T, lib library, den role.NodeID) dc.Choice {
			id := mustImport(t, lib, secrettest.Sealer(t), transmission(t, den, "Transmission", "http://192.168.1.20:9091"))
			exec(t, lib, "UPDATE downloadclients_client SET deleted_at = ?, secrets_sealed = NULL WHERE id = ?", storage.Now(), id)
			return dc.Choice{Client: &id}
		}, "Transmission on Den"},
		{"a built-in client on a Node that stopped downloading", func(_ *testing.T, _ library, den role.NodeID) dc.Choice {
			return dc.Choice{BuiltInOn: den}
		}, "Den's built-in client"},
	} {
		t.Run(c.name, func(t *testing.T) {
			lib := open(t)
			root, den := addNode(t, lib, "Media server"), addNode(t, lib, "Den")
			sent := c.sent(t, lib, den)
			router := dc.Router{Roster: downloadingOn{root: true}, Routes: noRoutes{}}
			ask := func(pinned bool) (dc.Target, error) {
				var got dc.Target
				err := lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
					var err error
					got, err = router.Route(ctx, r, dc.Ask{Root: root, Tracker: indexers.TrackerID{UUID: storage.NewUUID()}, Sent: &sent, Pinned: pinned})
					return err
				})
				return got, err
			}

			_, err := ask(true)
			p, ok := errors.AsType[*dc.PinnedError](err)
			if !ok || p.Name != c.wantName || !dc.Unusable(p.Cause) {
				t.Fatalf("pinned Route err = %v, want a *PinnedError naming %s with Usable's refusal", err, c.wantName)
			}
			got, err := ask(false)
			if err != nil || got.Node != root || got.Client != nil {
				t.Errorf("unpinned Route = %+v, %v, want the root's built-in client", got, err)
			}
		})
	}
}
