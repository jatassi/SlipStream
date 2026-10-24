package world_test

import (
	"encoding/json"
	"net/http"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

func TestTCPPeersDivergenceTurnsUTPOff(t *testing.T) {
	t.Parallel()
	run := worldtest.Launch(t, "single-machine")
	m := run.Manifest()
	if !slices.Contains(m.Divergences, "tcp-peers") {
		t.Fatalf("divergences = %v, want tcp-peers", m.Divergences)
	}
	a := worldtest.NewAPI(worldtest.Context(t), t, http.DefaultClient, m)
	rows := a.Nodes("home")
	if len(rows) == 0 {
		t.Fatal("listNodes has no row")
	}
	for _, n := range rows {
		got := a.Admin("home", http.MethodGet, "/api/v1/download-clients/"+n.ID+"/settings", "")
		var s struct {
			UTP struct {
				Value bool `json:"value"`
			} `json:"utp"`
		}
		if err := json.Unmarshal([]byte(got.Body), &s); err != nil || got.Status != http.StatusOK || s.UTP.Value {
			t.Errorf("node %s's client settings = %d %s, want uTP off", n.ID, got.Status, got.Body)
		}
	}
}
