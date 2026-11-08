package trackersapi_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/indexers/indexerstest"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/secret/secrettest"
	"github.com/jatassi/slipstream/internal/storage"
)

// TestATrackersRoutesListSetAndClearFromEachHome routes seedpool, which
// Heron's Indexer on seedbox reaches, to den's built-in client and back.
// A route from den, which homes no Indexer of it, and one for public Nyaa
// are refused.
func TestATrackersRoutesListSetAndClearFromEachHome(t *testing.T) {
	s := serve(t)
	seedbox, den := role.NodeID{UUID: storage.NewUUID()}, role.NodeID{UUID: storage.NewUUID()}
	for id, name := range map[role.NodeID]string{seedbox: "seedbox", den: "den"} {
		s.exec(t, `INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, randomblob(32), ?, 0)`, id, name)
	}
	s.holding.nodes = []role.NodeID{seedbox, den}
	heron := indexers.ID{UUID: storage.NewUUID()}
	err := s.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return indexerstest.Torznab(ctx, tx, secrettest.Sealer(t), heron, "Heron", seedbox, "https://heron.test/api", "k")
	})
	if err != nil {
		t.Fatal(err)
	}
	seedpool, nyaa := s.tracker(t, "seedpool", indexers.Private), s.tracker(t, "Nyaa", indexers.Public)
	for _, id := range []string{seedpool, nyaa} {
		u, err := storage.ParseUUID(id)
		if err != nil {
			t.Fatal(err)
		}
		s.exec(t, `INSERT INTO indexers_reach (indexer_id, tracker_id) VALUES (?, ?)`, heron, u)
	}
	routes := "/api/v1/trackers/" + seedpool + "/routes"
	// The replies mask seedbox as id1 and den as id2.
	onSeedbox := `{"homeNodeId":"id1","to":{"nodeId":"id1"},"default":{"nodeId":"id1"},"stored":null,"skipped":false}`
	toDen := `{"homeNodeId":"id1","to":{"nodeId":"id2"},"default":{"nodeId":"id1"},"stored":{"builtInOn":"id2"},"skipped":false}`

	s.owner.Expect(t, http.MethodGet, routes, "", http.StatusOK, `{"routes":[`+onSeedbox+`]}`)
	s.owner.Expect(t, http.MethodPut, routes+"/"+seedbox.String(), `{"builtInOn":"`+den.String()+`"}`, http.StatusOK, `{"routes":[`+toDen+`]}`)
	s.owner.Expect(t, http.MethodGet, routes, "", http.StatusOK, `{"routes":[`+toDen+`]}`)
	s.owner.Expect(t, http.MethodPut, routes+"/"+den.String(), `{"builtInOn":"`+den.String()+`"}`, http.StatusConflict,
		`{"status":409,"code":"route_not_a_home","title":"No Indexer on that Node reaches this Tracker","detail":"No Indexer on den reaches seedpool. Pick the home Node of one of its Indexers."}`)
	s.owner.Expect(t, http.MethodPut, routes+"/"+storage.NewUUID().String(), `{"builtInOn":"`+den.String()+`"}`, http.StatusNotFound,
		`{"status":404,"code":"node_not_found","title":"No Node has this id"}`)
	s.owner.Expect(t, http.MethodPut, routes+"/"+seedbox.String(), `{"clientId":"`+storage.NewUUID().String()+`"}`, http.StatusNotFound,
		`{"status":404,"code":"download_client_not_found","title":"No Download Client has this id"}`)
	s.owner.Expect(t, http.MethodPut, "/api/v1/trackers/"+nyaa+"/routes/"+seedbox.String(), `{"builtInOn":"`+den.String()+`"}`, http.StatusConflict,
		`{"status":409,"code":"tracker_public","title":"This Tracker is public","detail":"Nyaa is public. Only a private Tracker's Grabs can go to a client you pick."}`)
	s.owner.Expect(t, http.MethodGet, "/api/v1/trackers/"+nyaa+"/routes", "", http.StatusOK, `{"routes":[]}`)
	s.owner.Expect(t, http.MethodDelete, routes+"/"+seedbox.String(), "", http.StatusOK, `{"routes":[`+onSeedbox+`]}`)
}
