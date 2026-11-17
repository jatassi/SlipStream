package world_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/jatassi/slipstream/fakes/capturescript"
	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/notifysink"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
	"github.com/jatassi/slipstream/internal/notifications/providers"
)

// TestAScenariosNotifyConnectionsReachTheSinkAndTheScript launches a
// Controller with a Discord and a Custom Script connection, tests each,
// and finds the Discord request in the notify capture and the script's
// run in the script capture. Doctor finds the typed webhook is the sink.
func TestAScenariosNotifyConnectionsReachTheSinkAndTheScript(t *testing.T) {
	t.Parallel()
	s, err := scenario.Load("single-machine")
	if err != nil {
		t.Fatal(err)
	}
	s.Library = nil
	s.Notify = []providers.ID{"discord", "script"}
	run := worldtest.LaunchScenario(t, s)
	m := run.Manifest()
	a := worldtest.NewAPI(worldtest.Context(t), t, http.DefaultClient, m)
	var list struct {
		Connections []struct {
			ID       string `json:"id"`
			Provider string `json:"provider"`
		} `json:"connections"`
	}
	if r := a.Admin("home", http.MethodGet, "/api/v1/notifications/connections", ""); r.Status != http.StatusOK || json.Unmarshal([]byte(r.Body), &list) != nil || len(list.Connections) != 2 {
		t.Fatalf("listNotificationConnections = %d %s, want the Scenario's two", r.Status, r.Body)
	}
	for _, c := range list.Connections {
		if r := a.Admin("home", http.MethodPost, "/api/v1/notifications/connections/"+c.ID+"/test", "{}"); r.Status != http.StatusOK {
			t.Errorf("a Test of the %s connection = %d %s, want 200", c.Provider, r.Status, r.Body)
		}
	}
	ctx := worldtest.Context(t)
	if _, err := events.Wait(ctx, m.RunDir, events.Query{Kind: notifysink.Notify.Kind, Node: "home", Match: map[string]string{"profile": "discord", "status": "200"}}); err != nil {
		t.Errorf("no Discord request from home reached the sink: %v", err)
	}
	if _, err := events.Wait(ctx, m.RunDir, events.Query{Kind: capturescript.Script.Kind, Node: "home", Match: map[string]string{"exit": "0"}}); err != nil {
		t.Errorf("capture-script reported no run from home: %v", err)
	}
	if c := world.Doctor(ctx, m.RunDir, ".").Checks[4]; !c.OK {
		t.Errorf("doctor's %q failed with %v, want the Discord webhook to be the sink's front", c.Name, c.Failures)
	}
}
