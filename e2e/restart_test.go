package e2e

import (
	"encoding/json"
	"net/http"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/testbin"
	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

type started struct {
	PID       int       `json:"pid"`
	StartedAt time.Time `json:"startedAt"`
}

func getSystem(t *testing.T, a *worldtest.API, name string) started {
	t.Helper()
	r := a.Admin(name, http.MethodGet, "/api/v1/system", "")
	var s started
	if err := json.Unmarshal([]byte(r.Body), &s); r.Status != http.StatusOK || err != nil {
		t.Fatalf("%s getSystem = %d %s", name, r.Status, r.Body)
	}
	return s
}

// linkUp waits for the Controller home to list the Node id's link up
// since a time other than since, and returns when that was.
func linkUp(t *testing.T, a *worldtest.API, id, since string) string {
	t.Helper()
	var now string
	worldtest.Until(t, "home lists "+id+"'s link up since a new time", func() (bool, error) {
		link := row(t, a.Nodes("home"), id).Link
		now = link.Since
		return link.State == "up" && now != since, nil
	})
	return now
}

// checkRestarted checks the Node name came back on its listeners, as the
// same process on Unix and as a successor on Windows, and that the world
// logged the restart.
func checkRestarted(t *testing.T, run *world.Run, was, now world.Node) {
	t.Helper()
	inPlace := runtime.GOOS != "windows"
	if now.State != world.NodeRunning || (now.PID == was.PID) != inPlace || now.Listeners["http"] != was.Listeners["http"] || now.Listeners["link"] != was.Listeners["link"] {
		t.Fatalf("%s restarted %s as PID %d on http %s and link %s after PID %d; want running, in place %t, on http %s and link %s",
			was.Name, now.State, now.PID, now.Listeners["http"], now.Listeners["link"], was.PID, inPlace, was.Listeners["http"], was.Listeners["link"])
	}
	q := events.Query{Kind: events.NodeRestarted, Node: was.Name, Match: map[string]string{"oldPid": strconv.Itoa(was.PID), "pid": strconv.Itoa(now.PID)}}
	if _, err := events.Wait(worldtest.Context(t), run.Manifest().RunDir, q); err != nil {
		t.Fatalf("events.jsonl has no %s record for %s from PID %d to %d: %v", events.NodeRestarted, was.Name, was.PID, now.PID, err)
	}
}

func doctor(t *testing.T, run *world.Run) {
	t.Helper()
	if r := world.Doctor(worldtest.Context(t), run.Manifest().RunDir, testbin.ModuleRoot(t)); !r.OK {
		t.Fatalf("doctor after the restart:\n%s", r)
	}
}

func TestRestartingTheControllerKeepsItsWorkerLinked(t *testing.T) {
	t.Parallel()
	run := worldtest.Launch(t, "home")
	a := api(t, run)
	home, box := a.Node("home"), a.Node("seedbox")
	linked := linkUp(t, a, box.NodeID, "")
	before := getSystem(t, a, "home")
	stream := worldtest.NewMessages(t, run.Manifest(), world.AdminIdentity)
	stream.Next("live.resync")

	restarted := act(t, run, "home", world.RestartNode)
	checkRestarted(t, run, home, restarted)
	a = api(t, run)
	after := getSystem(t, a, "home")
	if after.PID != restarted.PID || !after.StartedAt.After(before.StartedAt) {
		t.Fatalf("home's getSystem after the restart is PID %d started %s, want PID %d started after %s", after.PID, after.StartedAt, restarted.PID, before.StartedAt)
	}
	stream.Next("live.resync")
	linkUp(t, a, box.NodeID, linked)
	doctor(t, run)
}

func TestRestartingAWorkerFromItsHost(t *testing.T) {
	t.Parallel()
	run := worldtest.Launch(t, "home")
	a := api(t, run)
	box := a.Node("seedbox")
	linked := linkUp(t, a, box.NodeID, "")

	restarted := act(t, run, "seedbox", world.RestartNode)
	checkRestarted(t, run, box, restarted)
	a = api(t, run)
	worldtest.Expect(t, "seedbox GET /api/v1/status", a.Get("seedbox", "/api/v1/status"), worldtest.Reply{Status: 200, Body: `{"claim":"worker"}` + "\n"})
	linkUp(t, a, box.NodeID, linked)
	again := act(t, run, "seedbox", world.RestartNode)
	checkRestarted(t, run, restarted, again)
	doctor(t, run)
}
