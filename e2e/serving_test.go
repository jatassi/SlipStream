package e2e

import (
	"bufio"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
	"github.com/jatassi/slipstream/internal/identity"
)

// TestAServingWorkerFrontsItsController gives seedbox Serving: its address
// answers with home's API and carries home's live stream. With home
// stopped, seedbox answers the API 503 controller_unreachable and its
// status page says what it holds, never what anyone asked for.
func TestAServingWorkerFrontsItsController(t *testing.T) {
	t.Parallel()
	run := worldtest.Launch(t, "home")
	a := api(t, run)
	box := a.Node("seedbox")
	a.SetRoles("home", box.NodeID, "searching", "downloading", "serving")

	var system struct {
		DataDir string `json:"dataDir"`
	}
	r := a.Admin("seedbox", http.MethodGet, "/api/v1/system", "")
	if err := json.Unmarshal([]byte(r.Body), &system); err != nil || r.Status != http.StatusOK || system.DataDir != a.Node("home").DataDir {
		t.Fatalf("getSystem at seedbox's address = %d %s, want home's at %s", r.Status, r.Body, a.Node("home").DataDir)
	}

	session := a.SignIn("seedbox")
	req, err := http.NewRequestWithContext(worldtest.Context(t), http.MethodGet, box.URL+"/api/v1/events", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Cookie", identity.CookieName+"="+session)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	stream := bufio.NewReader(res.Body)
	for line := ""; line != "event: live.resync\n"; {
		if line, err = stream.ReadString('\n'); err != nil || res.StatusCode != http.StatusOK {
			t.Fatalf("the live stream at seedbox's address = %d, read %q, %v; want live.resync", res.StatusCode, line, err)
		}
	}

	act(t, run, "home", world.StopNode)
	worldtest.Expect(t, "getSystem at seedbox's address with home stopped", a.Admin("seedbox", http.MethodGet, "/api/v1/system", ""),
		worldtest.Reply{Status: http.StatusServiceUnavailable, Body: `{"status":503,"code":"controller_unreachable","title":"The Controller can't be reached"}` + "\n"})
	var card struct {
		Controller string   `json:"controller"`
		Since      string   `json:"since"`
		Holding    []string `json:"holding"`
	}
	worldtest.Until(t, "seedbox notes home can't be reached", func() (bool, error) {
		r := a.Get("seedbox", "/api/v1/serving")
		return r.Status == http.StatusOK && json.Unmarshal([]byte(r.Body), &card) == nil && card.Since != "", nil
	})
	if card.Controller != "home" || strings.Join(card.Holding, ",") != "Searching,Downloading" {
		t.Fatalf("getServing with home stopped = %+v", card)
	}
}
