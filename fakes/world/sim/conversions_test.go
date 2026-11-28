package sim_test

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/sim/simtest"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
	"github.com/jatassi/slipstream/internal/converting"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storing"
)

// brickSource is Brick Lane Echo's file in the subtitles Scenario's
// movies folder, the stem its runs are cued by.
const brickSource = "movies/Brick Lane Echo (2019) {tmdb-9000202}/Brick.Lane.Echo.2019.1080p.WEB-DL.H264-FKW.mkv"

// madeTiers are the qualities a Target made at each height accepts, at
// any size.
var madeTiers = map[int]string{
	720: `[["webdl-720p","webrip-720p","hdtv-720p","bluray-720p"]],"cutoff":"webdl-720p",` +
		`"sizes":[{"quality":"webdl-720p","min":0,"max":null},{"quality":"webrip-720p","min":0,"max":null},{"quality":"hdtv-720p","min":0,"max":null},{"quality":"bluray-720p","min":0,"max":null}]`,
	480: `[["webrip-480p","dvd","sdtv"]],"cutoff":"webrip-480p"`,
}

type convWorld struct {
	t    *testing.T
	w    *sim.World
	conv *simtest.Conversions
	work string
}

type convCell struct {
	TargetName   string `json:"targetName"`
	State        string `json:"state"`
	Stage        string `json:"stage"`
	ConversionID string `json:"conversionId"`
	File         *struct {
		Path  string `json:"path"`
		Probe struct {
			Height int `json:"height"`
		} `json:"probe"`
	} `json:"file"`
}

// convRow is Brick Lane Echo's 720p Conversion in the Conversions tab.
type convRow struct {
	ID     string `json:"id"`
	Stage  string `json:"stage"`
	Target struct {
		Name string `json:"name"`
	} `json:"target"`
	Node *struct {
		Name string `json:"name"`
	} `json:"node"`
	Wait *struct {
		Code string `json:"code"`
	} `json:"wait"`
	Failure *struct {
		Code  string   `json:"code"`
		Class string   `json:"class"`
		Nodes []string `json:"nodes"`
	} `json:"failure"`
	Attempts []convAttempt `json:"attempts"`
}

type convAttempt struct {
	Node    string `json:"node"`
	Outcome string `json:"outcome"`
	Code    string `json:"code"`
}

func (r convRow) on() string {
	if r.Node == nil {
		return ""
	}
	return r.Node.Name
}

// launchConverting launches the subtitles Scenario with its roots on home
// and the Converting Role on workers alone, so each run reads a copy of
// its source, and waits for Brick Lane Echo's file to fill its Target. It
// makes each worker's scratch folder, which a real test run makes.
func launchConverting(t *testing.T, conv *simtest.Conversions, workers []string) convWorld {
	t.Helper()
	s, err := scenario.Load("subtitles")
	if err != nil {
		t.Fatal(err)
	}
	s.Nodes[0].Roles = []role.Name{role.Storage, role.Subtitles, role.Metadata, role.MediaServerLink, role.Notifying, role.Serving}
	for _, name := range workers {
		s.Nodes = append(s.Nodes, scenario.Node{Name: name, Setup: scenario.SetupJoin, Roles: []role.Name{role.Converting}})
	}
	c := convWorld{t: t, w: sim.Launch(t, s, sim.WithWalk(), conv.Option()), conv: conv}
	for _, name := range workers {
		if err := os.MkdirAll(c.w.Node(name).DataDir().Resolve(converting.DefaultSettings().Scratch), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	c.work = subWorld{t: t, w: c.w, at: "home"}.work(brick)
	c.await("its file fills its Target", func(cells []convCell) bool { return len(cells) == 1 && cells[0].File != nil })
	return c
}

func (c convWorld) call(method, route, body string) []byte {
	c.t.Helper()
	r := c.w.API().Admin("home", method, route, body)
	if r.Status/100 != 2 {
		c.t.Fatalf("%s %s = %d %s, want success", method, route, r.Status, r.Body)
	}
	return []byte(r.Body)
}

func decode[T any](t *testing.T, body []byte) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(body, &v); err != nil {
		t.Fatalf("%s: %v", body, err)
	}
	return v
}

func (c convWorld) cells() []convCell {
	c.t.Helper()
	return decode[struct {
		Cells []convCell `json:"cells"`
	}](c.t, c.call(http.MethodGet, "/api/v1/works/"+c.work+"/cells", "")).Cells
}

func (c convWorld) await(what string, ok func([]convCell) bool) []convCell {
	c.t.Helper()
	var cells []convCell
	for range simtest.Settled / time.Second {
		if cells = c.cells(); ok(cells) {
			return cells
		}
		time.Sleep(time.Second)
	}
	c.t.Fatalf("Brick Lane Echo's cells are %+v after %s, want %s", cells, simtest.Settled, what)
	return nil
}

// row is Brick Lane Echo's 720p Conversion in whichever band of the
// Conversions tab lists it, with no stage once it has finished.
func (c convWorld) row() convRow {
	c.t.Helper()
	l := decode[struct {
		Attention  []convRow `json:"attention"`
		Converting []convRow `json:"converting"`
		Next       []struct {
			Rows []json.RawMessage `json:"rows"`
		} `json:"next"`
	}](c.t, c.call(http.MethodGet, "/api/v1/conversions?work="+c.work, ""))
	rows := slices.Concat(l.Attention, l.Converting)
	for _, lane := range l.Next {
		for _, raw := range lane.Rows {
			rows = append(rows, decode[convRow](c.t, raw))
		}
	}
	for _, r := range rows {
		if r.Target.Name == "720p" {
			return r
		}
	}
	return convRow{}
}

func (c convWorld) awaitRow(what string, ok func(convRow) bool) convRow {
	c.t.Helper()
	var r convRow
	for range simtest.Settled / time.Second {
		if r = c.row(); ok(r) {
			return r
		}
		time.Sleep(time.Second)
	}
	c.t.Fatalf("Brick Lane Echo's 720p Conversion is %+v after %s, want %s", r, simtest.Settled, what)
	return r
}

// derive adds a Target made at each height from the movie Profile's first
// Target to that Profile, which converts each of its Works.
func (c convWorld) derive(heights ...int) {
	c.t.Helper()
	profiles := decode[struct {
		Profiles []map[string]json.RawMessage `json:"profiles"`
	}](c.t, c.call(http.MethodGet, "/api/v1/profiles", ""))
	i := slices.IndexFunc(profiles.Profiles, func(p map[string]json.RawMessage) bool {
		return string(p["contentType"]) == `"movie"` && string(p["default"]) == "true"
	})
	p := profiles.Profiles[i]
	entries := decode[[]map[string]any](c.t, p["targets"])
	from := entries[0]["targetId"]
	for _, h := range heights {
		made := decode[struct {
			ID string `json:"id"`
		}](c.t, c.call(http.MethodPost, "/api/v1/targets", `{"name":"`+strconv.Itoa(h)+`p","tiers":`+madeTiers[h]+`,"conversion":`+
			`{"codec":"h264","maxHeight":`+strconv.Itoa(h)+`,"kbps":`+strconv.Itoa(h*4)+`,"depth":8,"range":"sdr","keepDolbyVision":false,`+
			`"curve":"bt2390","audio":"copy","dropCommentary":false,"container":"mkv","runOnNodeId":null}}`))
		entries = append(entries, map[string]any{"targetId": made.ID, "sourcing": "derive", "fromTargetId": from})
	}
	body, err := json.Marshal(map[string]any{
		"name": p["name"], "contentType": p["contentType"], "targets": entries, "editions": p["editions"], "languages": p["languages"],
	})
	if err != nil {
		c.t.Fatal(err)
	}
	c.call(http.MethodPut, "/api/v1/profiles/"+decode[string](c.t, p["id"]), string(body))
}

// running waits until each made Target's Conversion runs.
func (c convWorld) running(n int) []convCell {
	c.t.Helper()
	return c.await(strconv.Itoa(n)+" Conversions run", func(cells []convCell) bool {
		return len(cells) == n+1 && !slices.ContainsFunc(cells[1:], func(x convCell) bool { return x.Stage != "running" })
	})
}

func (c convWorld) met() []convCell {
	c.t.Helper()
	return c.await("every made Target is met", func(cells []convCell) bool {
		return len(cells) > 1 && !slices.ContainsFunc(cells[1:], func(x convCell) bool { return x.State != "met" })
	})
}

type convTransfer struct {
	Purpose     string `json:"purpose"`
	JobName     string `json:"jobName"`
	Destination struct {
		Kind string `json:"kind"`
	} `json:"destination"`
}

// transfers is how many Conversion Transfers of Brick Lane Echo go to a
// destination of kind.
func (c convWorld) transfers(kind string) int {
	c.t.Helper()
	listed := decode[struct {
		Transfers []convTransfer `json:"transfers"`
	}](c.t, c.call(http.MethodGet, "/api/v1/transfers", ""))
	return len(slices.DeleteFunc(listed.Transfers, func(x convTransfer) bool {
		return x.Purpose != "conversion" || x.JobName != brick || x.Destination.Kind != kind
	}))
}

// recorded is how many History entries of kind are about Brick Lane Echo,
// or about none, for a kind that names no Work.
func (c convWorld) recorded(kind string) int {
	c.t.Helper()
	q := url.Values{"groups": {"conversions"}, "kinds": {kind}}
	if kind == "conversion_failed" {
		q.Set("work", c.work)
	}
	return len(decode[struct {
		Entries []json.RawMessage `json:"entries"`
	}](c.t, c.call(http.MethodGet, "/api/v1/history?"+q.Encode(), "")).Entries)
}

type convCause struct {
	Count int `json:"count"`
}

// causes is why no Node can convert some of the 720p Target's
// Conversions.
func (c convWorld) causes() []convCause {
	c.t.Helper()
	l := decode[struct {
		Targets []struct {
			Name       string `json:"name"`
			Conversion *struct {
				Causes []convCause `json:"causes"`
			} `json:"conversion"`
		} `json:"targets"`
	}](c.t, c.call(http.MethodGet, "/api/v1/targets", ""))
	for _, t := range l.Targets {
		if t.Name == "720p" {
			return t.Conversion.Causes
		}
	}
	c.t.Fatal("no Target is named 720p")
	return nil
}

func (c convWorld) awaitCauses(what string, ok func([]convCause) bool) {
	c.t.Helper()
	simtest.Eventually(c.t, what, func() bool { return ok(c.causes()) })
}

func (c convWorld) node(name string) worldtest.Row {
	c.t.Helper()
	rows := c.w.API().Nodes("home")
	return rows[slices.IndexFunc(rows, func(r worldtest.Row) bool { return r.Name == name })]
}

func (c convWorld) nodeID(name string) string { return c.node(name).ID }

// setConverting sets the Node's Conversions at once, null for the
// default, and its processor switch, keeping its hours.
func (c convWorld) setConverting(name string, atOnce *int, processor bool) {
	c.t.Helper()
	route := "/api/v1/nodes/" + c.nodeID(name) + "/converting"
	now := decode[map[string]json.RawMessage](c.t, c.call(http.MethodGet, route, ""))
	body, err := json.Marshal(map[string]any{"atOnce": atOnce, "hours": now["hours"], "processor": processor})
	if err != nil {
		c.t.Fatal(err)
	}
	c.call(http.MethodPut, route, string(body))
}

type convProof struct {
	ProvenAt *time.Time `json:"provenAt"`
	LostAt   *time.Time `json:"lostAt"`
}

func (c convWorld) proofs(name string) []convProof {
	c.t.Helper()
	return decode[struct {
		Proofs []convProof `json:"proofs"`
	}](c.t, c.call(http.MethodGet, "/api/v1/nodes/"+c.nodeID(name)+"/converting", "")).Proofs
}

// partition cuts home off from name, both ways, past the offline grace,
// and answers the heal.
func (c convWorld) partition(name string) (heal func()) {
	c.t.Helper()
	paths := []*fault.Plan{c.w.Path("home", name), c.w.Path(name, "home")}
	for _, p := range paths {
		simtest.Flip(c.t, p, fault.Stall{})
	}
	time.Sleep(nodes.OfflineGrace.Default)
	return func() {
		for _, p := range paths {
			p.Clear("")
		}
	}
}

func TestConversionsAcrossNodes(t *testing.T) {
	for _, tc := range []struct {
		name    string
		workers []string
		heights []int
		cues    []simtest.Outcome
		atOnce  int
		// off are the workers whose processor is off before the Targets
		// are made.
		off  []string
		then func(c convWorld)
	}{
		{name: "it lands and fills every Target", heights: []int{720, 480}, then: convWorld.fillsEvery},
		{name: "it copies the source once for two Targets", heights: []int{720, 480}, cues: []simtest.Outcome{simtest.Hold, simtest.Hold}, atOnce: 2, then: convWorld.copiesOnce},
		{name: "it cancels", cues: []simtest.Outcome{simtest.Hold}, then: convWorld.cancels},
		{
			name: "a run fault retries on its Node, then on another, then fails", workers: []string{"den", "attic"},
			cues: []simtest.Outcome{simtest.RunFault(converting.Encoder), simtest.RunFault(converting.Encoder), simtest.RunFault(converting.Encoder)},
			then: failsAfter(3, 2),
		},
		{
			name: "a run fault on the only Node fails after two attempts",
			cues: []simtest.Outcome{simtest.RunFault(converting.Encoder), simtest.RunFault(converting.Encoder)},
			then: failsAfter(2, 1),
		},
		{name: "a source fault fails at once and keeps the proof", cues: []simtest.Outcome{simtest.SourceFault(converting.Unreadable)}, then: convWorld.sourceFault},
		{name: "a disk filling mid-run puts it back with no failure", cues: []simtest.Outcome{simtest.DiskFull, simtest.Hold}, then: convWorld.diskFull},
		{
			name: "a restart keeps it and opens no cause, and a lapse puts it back and opens one",
			cues: []simtest.Outcome{simtest.Hold, simtest.Hold}, then: convWorld.restartsThenLapses,
		},
		{
			name: "taking the Role off its Node puts it back", workers: []string{"den", "attic"},
			cues: []simtest.Outcome{simtest.Hold, simtest.Hold}, then: convWorld.roleTakenOff,
		},
		{
			name: "no Node can convert records once for both 1080p Works, and again once the cause returns", off: []string{"den"},
			cues: []simtest.Outcome{simtest.SourceFault(converting.Unreadable)}, then: convWorld.noNodeCanConvert,
		},
		{
			name: "a report from a Node it left is refused", workers: []string{"den", "attic"}, off: []string{"attic"},
			cues: []simtest.Outcome{simtest.Hold, simtest.Hold}, then: convWorld.refusesTheLapsedReport,
		},
		{name: "a landing the Importer refuses ends its output Transfer", cues: []simtest.Outcome{simtest.Hold}, then: convWorld.refusedLanding},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				conv := simtest.NewConversions()
				conv.Cue(brickSource, tc.cues...)
				workers := tc.workers
				if workers == nil {
					workers = []string{"den"}
				}
				c := launchConverting(t, conv, workers)
				if tc.atOnce > 0 {
					c.setConverting("den", &tc.atOnce, true)
				}
				for _, name := range tc.off {
					c.setConverting(name, nil, false)
				}
				heights := tc.heights
				if heights == nil {
					heights = []int{720}
				}
				c.derive(heights...)
				tc.then(c)
			})
		})
	}
}

func (c convWorld) fillsEvery() {
	cells := c.met()
	for i, h := range []int{1080, 720, 480} {
		if f := cells[i].File; f == nil || f.Probe.Height != h {
			c.t.Errorf("%s is filled by %+v, want a %dp file", cells[i].TargetName, f, h)
		}
	}
	if cells[1].File.Path == cells[2].File.Path || cells[1].File.Path == cells[0].File.Path {
		c.t.Errorf("the made files sit at %s and %s beside %s, want a name of their own", cells[1].File.Path, cells[2].File.Path, cells[0].File.Path)
	}
}

func (c convWorld) copiesOnce() {
	c.running(2)
	if n := c.transfers("scratch"); n != 1 {
		c.t.Errorf("%d Transfers copy the source while both run, want 1", n)
	}
	c.conv.Release(brickSource)
	c.met()
	if n := c.conv.Ran(brickSource); n != 2 {
		c.t.Errorf("den ran %d Conversions of the source, want 2", n)
	}
}

func (c convWorld) cancels() {
	c.call(http.MethodPost, "/api/v1/conversions/"+c.running(1)[1].ConversionID+"/cancel", "")
	c.await("its made Target is missing", func(cells []convCell) bool { return cells[1].State == "missing" })
	time.Sleep(simtest.Settled)
	if cells := c.cells(); cells[1].State != "missing" || c.conv.Ran(brickSource) != 1 {
		c.t.Errorf("after the cancel the made Target is %s and den ran %d Conversions, want missing and 1", cells[1].State, c.conv.Ran(brickSource))
	}
}

// failsAfter fails the Conversion after its attempts, each an encoder
// fault, the first two on one Node, across Nodes.
func failsAfter(attempts, across int) func(c convWorld) {
	return func(c convWorld) {
		r := c.awaitRow("it fails", func(r convRow) bool { return r.Stage == "failed" })
		var on []string
		for _, a := range r.Attempts {
			if a.Outcome != "failed" || a.Code != converting.Encoder {
				c.t.Errorf("an attempt ended %s %s, want failed encoder", a.Outcome, a.Code)
			}
			on = append(on, a.Node)
		}
		distinct := slices.Compact(slices.Sorted(slices.Values(on)))
		if len(on) != attempts || on[0] != on[1] || len(distinct) != across {
			c.t.Errorf("it ran on %v, want %d attempts across %d Nodes, the first two on one", on, attempts, across)
		}
		if f := r.Failure; f == nil || f.Class != "run" || !slices.Equal(slices.Sorted(slices.Values(f.Nodes)), distinct) {
			c.t.Errorf("it failed with %+v, want a run fault on %v", f, distinct)
		}
		if n, ran := c.recorded("conversion_failed"), c.conv.Ran(brickSource); n != 1 || ran != attempts {
			c.t.Errorf("Conversion failed is recorded %d times after %d runs, want once after %d", n, ran, attempts)
		}
	}
}

func (c convWorld) sourceFault() {
	r := c.awaitRow("it fails", func(r convRow) bool { return r.Stage == "failed" })
	if len(r.Attempts) != 1 || r.Attempts[0].Code != converting.Unreadable || r.Failure == nil || r.Failure.Class != "source" {
		c.t.Errorf("it failed with %+v after %+v, want a source fault after one unreadable attempt", r.Failure, r.Attempts)
	}
	proofs := c.proofs("den")
	if len(proofs) == 0 || slices.ContainsFunc(proofs, func(p convProof) bool { return p.ProvenAt == nil || p.LostAt != nil }) {
		c.t.Errorf("den's proofs are %+v, want each still proven", proofs)
	}
	if n, ran := c.recorded("conversion_failed"), c.conv.Ran(brickSource); n != 1 || ran != 1 {
		c.t.Errorf("Conversion failed is recorded %d times after %d runs, want once after 1", n, ran)
	}
}

func (c convWorld) diskFull() {
	r := c.awaitRow("it runs again", func(r convRow) bool { return r.Stage == "running" && len(r.Attempts) == 2 })
	if a := r.Attempts[0]; a.Node != "den" || a.Outcome != "disk_full" || r.Failure != nil {
		c.t.Errorf("its first attempt ended %+v and it carries %+v, want disk_full on den and no failure", a, r.Failure)
	}
	c.conv.Release(brickSource)
	c.met()
	if n := c.recorded("conversion_failed"); n != 0 {
		c.t.Errorf("Conversion failed is recorded %d times, want none", n)
	}
}

func (c convWorld) restartsThenLapses() {
	c.running(1)
	den := c.w.Node("den")
	den.Stop()
	den.Start()
	r := c.awaitRow("it runs again on den", func(r convRow) bool { return r.Stage == "running" && len(r.Attempts) == 2 })
	if a := r.Attempts[0]; a.Node != "den" || a.Outcome != "put_back" || r.on() != "den" {
		c.t.Errorf("after den's restart its first attempt ended %+v and it runs on %s, want put back on den and running there", a, r.on())
	}
	if n := c.recorded("no_node_can_convert"); n != 0 {
		c.t.Errorf("No Node can convert is recorded %d times after den's restart, want none", n)
	}

	heal := c.partition("den")
	r = c.awaitRow("it waits for a Node", func(r convRow) bool { return r.Stage == "queued" })
	if a := r.Attempts[len(r.Attempts)-1]; a.Outcome != "put_back" || r.Wait == nil || r.Wait.Code != "no_node" {
		c.t.Errorf("with den lapsed its last attempt ended %+v and it waits %+v, want put back and waiting for a Node", a, r.Wait)
	}
	if n := c.recorded("no_node_can_convert"); n != 1 {
		c.t.Errorf("No Node can convert is recorded %d times with den lapsed, want once", n)
	}
	heal()
	c.met()
}

func (c convWorld) roleTakenOff() {
	from := c.awaitRow("it runs", func(r convRow) bool { return r.Stage == "running" }).on()
	c.w.API().SetRoles("home", c.nodeID(from))
	r := c.awaitRow("it runs on another Node", func(r convRow) bool { return r.Stage == "running" && len(r.Attempts) == 2 })
	if a := r.Attempts[0]; a.Node != from || a.Outcome != "put_back" || r.on() == from {
		c.t.Errorf("its first attempt ended %+v and it runs on %s, want put back on %s and running elsewhere", a, r.on(), from)
	}
	c.conv.Release(brickSource)
	c.met()
}

func (c convWorld) noNodeCanConvert() {
	c.awaitCauses("one cause holds both 1080p Works' Conversions", func(cs []convCause) bool { return len(cs) == 1 && cs[0].Count == 2 })
	if n := c.recorded("no_node_can_convert"); n != 1 {
		c.t.Errorf("No Node can convert is recorded %d times for 2 Works, want once", n)
	}
	c.setConverting("den", nil, true)
	id := c.awaitRow("it fails", func(r convRow) bool { return r.Stage == "failed" }).ID
	c.awaitCauses("the cause closes", func(cs []convCause) bool { return len(cs) == 0 })
	c.setConverting("den", nil, false)
	c.call(http.MethodPost, "/api/v1/conversions/"+id+"/retry", "")
	c.awaitCauses("the cause opens again", func(cs []convCause) bool { return len(cs) == 1 && cs[0].Count == 1 })
	if n := c.recorded("no_node_can_convert"); n != 2 {
		c.t.Errorf("No Node can convert is recorded %d times once the cause returned, want twice", n)
	}
}

// refusesTheLapsedReport ends den's run while den is cut off, after its
// Conversion was put back and placed on attic. den's report of the
// ended run reaches home once den is back, and leaves the Conversion
// running on attic.
func (c convWorld) refusesTheLapsedReport() {
	c.awaitRow("it runs on den", func(r convRow) bool { return r.Stage == "running" && r.on() == "den" })
	heal := c.partition("den")
	c.awaitRow("it is put back", func(r convRow) bool { return r.Stage == "queued" })
	c.conv.Release(brickSource)
	c.setConverting("attic", nil, true)
	c.awaitRow("it runs on attic", func(r convRow) bool { return r.Stage == "running" && r.on() == "attic" })
	heal()
	simtest.Eventually(c.t, "den's link is up", func() bool { return c.node("den").Link.State == "up" })
	time.Sleep(simtest.Settled)
	if r := c.row(); r.Stage != "running" || r.on() != "attic" || len(r.Attempts) != 2 {
		c.t.Errorf("after den's report the Conversion is %s on %s after %+v, want still running on attic", r.Stage, r.on(), r.Attempts)
	}
}

type convRoot struct {
	ContentType string `json:"contentType"`
	Path        string `json:"path"`
}

// refusedLanding makes home's holding folder read-only, so the made file
// can't move there from where its Transfer staged it and its Import ends
// unplaced.
func (c convWorld) refusedLanding() {
	c.running(1)
	roots := decode[struct {
		Roots []convRoot `json:"roots"`
	}](c.t, c.call(http.MethodGet, "/api/v1/roots", "")).Roots
	movies := roots[slices.IndexFunc(roots, func(r convRoot) bool { return r.ContentType == "movie" })]
	c.w.Disk("home", filepath.Join(movies.Path, storing.HoldingDir), fault.ReadOnly{})
	c.conv.Release(brickSource)
	c.await("its Import ends unplaced", func(cells []convCell) bool { return cells[1].Stage == "" && cells[1].File == nil })
	simtest.Eventually(c.t, "its output Transfer ends", func() bool { return c.transfers("root") == 0 })
}
