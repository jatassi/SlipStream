// Package simtest is the simulation tier's shared test helpers: the home
// Scenario, waits in fake time, and what an Admin reads of a world's roots,
// scheduled tasks, Health and Grabs.
package simtest

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
	"github.com/jatassi/slipstream/internal/backoff"
	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/install"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/tasks"
)

func Home(t *testing.T) scenario.Scenario {
	t.Helper()
	s, err := scenario.Load("home")
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func Decode[T any](t *testing.T, what string, r worldtest.Reply) T {
	t.Helper()
	var v T
	if r.Status != http.StatusOK {
		t.Fatalf("%s = %d %s, want 200", what, r.Status, r.Body)
	}
	if err := json.Unmarshal([]byte(r.Body), &v); err != nil {
		t.Fatalf("%s = %s: %v", what, r.Body, err)
	}
	return v
}

// Settled is longer than any wait the walk takes in fake time.
const Settled = 10 * time.Minute

// Eventually waits in fake time until ok, failing with what.
func Eventually(t *testing.T, what string, ok func() bool) {
	t.Helper()
	for range Settled / time.Second {
		if ok() {
			return
		}
		time.Sleep(time.Second)
	}
	t.Fatalf("not within %s: %s", Settled, what)
}

type RootRow struct {
	ID       string `json:"id"`
	NodeID   string `json:"nodeId"`
	Path     string `json:"path"`
	State    string `json:"state"`
	VolumeID string `json:"volumeId"`
}

func RootAt(t *testing.T, a *worldtest.API, path string) RootRow {
	t.Helper()
	reply := a.Admin("home", http.MethodGet, "/api/v1/roots", "")
	var list struct{ Roots []RootRow }
	if err := json.Unmarshal([]byte(reply.Body), &list); reply.Status != http.StatusOK || err != nil {
		t.Fatalf("listRoots = %d %s, %v", reply.Status, reply.Body, err)
	}
	for _, r := range list.Roots {
		if r.Path == path {
			return r
		}
	}
	t.Fatalf("listRoots has no root at %s: %s", path, reply.Body)
	return RootRow{}
}

type TaskDetail struct {
	Task struct {
		State struct {
			Kind string `json:"kind"`
		} `json:"state"`
	} `json:"task"`
	Units []struct {
		Label       string `json:"label"`
		DueOnReturn bool   `json:"dueOnReturn"`
	} `json:"units"`
	Runs []struct {
		ID        int64     `json:"id"`
		Trigger   string    `json:"trigger"`
		Result    string    `json:"result"`
		Summary   string    `json:"summary"`
		StartedAt time.Time `json:"startedAt"`
		Units     []struct {
			Label   string  `json:"label"`
			Node    string  `json:"node"`
			Outcome string  `json:"outcome"`
			Cause   *string `json:"cause"`
		} `json:"units"`
	} `json:"runs"`
}

func (d TaskDetail) RunLines() []string {
	out := []string{}
	for _, r := range d.Runs {
		out = append(out, r.Trigger+"|"+r.Result+"|"+r.Summary)
	}
	return out
}

func (d TaskDetail) UnitLines() []string {
	out := []string{}
	for _, u := range d.Runs[0].Units {
		cause := ""
		if u.Cause != nil {
			cause = *u.Cause
		}
		out = append(out, strings.Join([]string{u.Label, u.Node, u.Outcome, cause}, "|"))
	}
	return out
}

func (d TaskDetail) DueOnReturn() []string {
	out := []string{}
	for _, u := range d.Units {
		if u.DueOnReturn {
			out = append(out, u.Label)
		}
	}
	return out
}

func GetTask(t *testing.T, a *worldtest.API, id tasks.TaskID) TaskDetail {
	t.Helper()
	return Decode[TaskDetail](t, "getTask", a.Admin("home", http.MethodGet, "/api/v1/tasks/"+string(id), ""))
}

func RunNow(t *testing.T, a *worldtest.API, id tasks.TaskID) int64 {
	t.Helper()
	r := a.Admin("home", http.MethodPost, "/api/v1/tasks/"+string(id)+"/runs", "")
	var started struct {
		RunID int64 `json:"runId"`
	}
	if err := json.Unmarshal([]byte(r.Body), &started); r.Status != http.StatusAccepted || err != nil {
		t.Fatalf("runTask = %d %s, want 202", r.Status, r.Body)
	}
	return started.RunID
}

type HealthItem struct {
	Key       string
	Level     string
	Subject   string
	Text      string
	Since     time.Time
	ChangedAt time.Time
	Unsure    bool
	Action    *struct{ Kind, Href, Label string }
}

type HealthReport struct {
	Categories []struct {
		ID     string
		Unread string
		Items  []HealthItem
	}
}

func (r HealthReport) Items(t *testing.T, id string) map[string]HealthItem {
	t.Helper()
	for _, c := range r.Categories {
		if c.ID == id {
			if c.Unread != "" {
				t.Fatalf("Health couldn't read %s: %s", id, c.Unread)
			}
			out := map[string]HealthItem{}
			for _, it := range c.Items {
				out[it.Key] = it
			}
			return out
		}
	}
	t.Fatalf("the Health report has no category %s", id)
	return nil
}

func HealthItems(t *testing.T, a *worldtest.API, id string) map[string]HealthItem {
	t.Helper()
	return Decode[HealthReport](t, "getSystemHealth", a.Admin("home", http.MethodGet, "/api/v1/system/health", "")).Items(t, id)
}

type HealthChecked struct {
	Report    HealthReport
	Unreached []struct{ Key, Words string }
}

// CheckStorage is Check now on the Storage item key, or on the whole
// category when key is "".
func CheckStorage(t *testing.T, a *worldtest.API, key string) HealthChecked {
	t.Helper()
	path := "/api/v1/system/health/storage/check"
	if key != "" {
		path = "/api/v1/system/health/storage/items/" + key + "/check"
	}
	return Decode[HealthChecked](t, "Check now on storage "+key, a.Admin("home", http.MethodPost, path, ""))
}

func Partition(t *testing.T, w *sim.World) time.Time {
	t.Helper()
	Flip(t, w.Path("home", "seedbox"), fault.Stall{})
	link := node.Defaults(contenttype.Set{}, install.Dev).Timings.Link
	time.Sleep(link.Keepalive + link.WriteTimeout)
	synctest.Wait()
	return time.Now()
}

func Heal(w *sim.World, down time.Time) {
	healed := down.Add(time.Minute)
	time.Sleep(time.Until(healed))
	w.Path("home", "seedbox").Clear("")
	time.Sleep(time.Until(RedialAt("home/seedbox", down, healed)))
	synctest.Wait()
}

// RedialAt is when the Keep loop of the Node seeded seed lands its first
// dial at or after heal, its session to the Controller having ended at
// down: it waits a backoff drawn from the Node's link stream before each
// attempt, and each attempt before heal blocks on the stalled path for the
// whole handshake.
func RedialAt(seed string, down, heal time.Time) time.Time {
	timings := node.Defaults(contenttype.Set{}, install.Dev).Timings.Link
	r := node.NewStreams(seed).Stream("link")
	at := down
	for n := 0; ; n++ {
		at = at.Add(backoff.Delay(timings.RedialMin, timings.RedialMax, n, r))
		if !at.Before(heal) {
			return at
		}
		at = at.Add(timings.Handshake)
	}
}

// Flip plays f on every connection of path from now until it is cleared.
func Flip(t *testing.T, path *fault.Plan, f fault.Fault) {
	t.Helper()
	if _, err := path.Set(fault.Rule{Fault: f}); err != nil {
		t.Fatal(err)
	}
}

func OnSeedbox(edit func(e *node.Env)) sim.Option {
	return sim.Edit(func(name string, e *node.Env) {
		if name == "seedbox" {
			edit(e)
		}
	})
}

var walked = []string{"fetching", "downloading", "probing", "transferring", "importing", "imported"}

type GrabView struct {
	ID          string `json:"id"`
	NodeID      string `json:"nodeId"`
	FetchNodeID string `json:"fetchNodeId"`
	InfoHash    string `json:"infoHash"`
	Stage       string `json:"stage"`
	Failure     *struct {
		Stage   string `json:"stage"`
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"failure"`
	History []struct {
		Stage string `json:"stage"`
	} `json:"history"`
	Progress *struct {
		Total int64 `json:"total"`
	} `json:"progress"`
}

func (g GrabView) Stages() []string {
	var out []string
	for _, e := range g.History {
		out = append(out, e.Stage)
	}
	return out
}

// AddOrchard adds the Scenario's movie monitored and returns its Work.
func AddOrchard(t *testing.T, w *sim.World) string {
	t.Helper()
	added := Decode[struct {
		Work struct {
			ID string `json:"id"`
		} `json:"work"`
	}](t, "addWork", w.API().Admin("home", http.MethodPost, "/api/v1/works", `{"contentType":"movie","key":"9000001","preset":"all"}`))
	return added.Work.ID
}

// Search presses Search on work and returns its one run.
func Search(t *testing.T, w *sim.World, work string) string {
	t.Helper()
	reply := w.API().Admin("home", http.MethodPost, "/api/v1/works/"+work+"/search", "")
	var started struct {
		SearchIDs []string `json:"searchIds"`
	}
	if err := json.Unmarshal([]byte(reply.Body), &started); err != nil || reply.Status != http.StatusAccepted || len(started.SearchIDs) != 1 {
		t.Fatalf("searchWork = %d %s, want 202 with one search", reply.Status, reply.Body)
	}
	return started.SearchIDs[0]
}

func GrabsListed(t *testing.T, w *sim.World) []GrabView {
	t.Helper()
	return Decode[struct {
		Grabs []GrabView `json:"grabs"`
	}](t, "listGrabs", w.API().Admin("home", http.MethodGet, "/api/v1/grabs", "")).Grabs
}

// Await waits in fake time until the world's one Grab is done, and
// returns it.
func Await(t *testing.T, w *sim.World) GrabView {
	t.Helper()
	for range Settled / time.Second {
		if gs := GrabsListed(t, w); len(gs) == 1 && (gs[0].Stage == "imported" || gs[0].Stage == "failed") {
			return gs[0]
		}
		time.Sleep(time.Second)
	}
	t.Fatalf("no Grab finished within %s: %+v", Settled, GrabsListed(t, w))
	return GrabView{}
}

func ExpectImported(t *testing.T, g GrabView) {
	t.Helper()
	if g.Stage != "imported" || !slices.Equal(g.Stages(), walked) {
		t.Fatalf("the Grab ended %s with %+v after %v, want imported after %v", g.Stage, g.Failure, g.Stages(), walked)
	}
}
