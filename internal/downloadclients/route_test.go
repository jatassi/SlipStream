package downloadclients_test

import (
	"context"
	"testing"

	dc "github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/secret/secrettest"
	"github.com/jatassi/slipstream/internal/storage"
)

func route(t *testing.T, lib library, node role.NodeID) dc.Target {
	t.Helper()
	var got dc.Target
	err := lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		got, err = dc.Route(ctx, r, dc.Ask{Node: node, Tracker: indexers.TrackerID{UUID: storage.NewUUID()}})
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
