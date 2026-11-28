package sim_test

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"slices"
	"strconv"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/sim/simtest"
	"github.com/jatassi/slipstream/internal/conversions"
	"github.com/jatassi/slipstream/internal/converting"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
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

// launchConverting launches the subtitles Scenario with its roots on home
// and the Converting Role on den alone, so each run reads a copy of its
// source, and waits for Brick Lane Echo's file to fill its Target. It
// makes den's scratch folder, which a real test run makes.
func launchConverting(t *testing.T, conv *simtest.Conversions) convWorld {
	t.Helper()
	s, err := scenario.Load("subtitles")
	if err != nil {
		t.Fatal(err)
	}
	s.Nodes[0].Roles = []role.Name{role.Storage, role.Subtitles, role.Metadata, role.MediaServerLink, role.Notifying, role.Serving}
	s.Nodes = append(s.Nodes, scenario.Node{Name: "den", Setup: scenario.SetupJoin, Roles: []role.Name{role.Converting}})
	c := convWorld{t: t, w: sim.Launch(t, s, sim.WithWalk(), conv.Option()), conv: conv}
	if err := os.MkdirAll(c.w.Node("den").DataDir().Resolve(converting.DefaultSettings().Scratch), 0o700); err != nil {
		t.Fatal(err)
	}
	c.work = subWorld{t: t, w: c.w, at: "home"}.work(brick)
	c.await("its file fills its Target", func(cells []convCell) bool { return len(cells) == 1 && cells[0].File != nil })
	return c
}

func (c convWorld) cells() []convCell {
	c.t.Helper()
	return simtest.Decode[struct {
		Cells []convCell `json:"cells"`
	}](c.t, "getWorkCells", c.w.API().Admin("home", http.MethodGet, "/api/v1/works/"+c.work+"/cells", "")).Cells
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

func (c convWorld) write(query string, args ...any) {
	c.t.Helper()
	err := c.w.Node("home").Half().Library.Write(c.t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, query, args...)
		return err
	})
	if err != nil {
		c.t.Fatal(err)
	}
}

// derive adds a Target made at each height from the movie Profile's first
// Target, below the API, which refuses Conversion settings in this build,
// and nudges Brick Lane Echo's Conversions.
func (c convWorld) derive(heights ...int) {
	c.t.Helper()
	for _, h := range heights {
		name := strconv.Itoa(h) + "p"
		r := c.w.API().Admin("home", http.MethodPost, "/api/v1/targets", `{"name":"`+name+`","tiers":`+madeTiers[h]+`}`)
		var made struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal([]byte(r.Body), &made); r.Status != http.StatusCreated || err != nil {
			c.t.Fatalf("createTarget = %d %s, want 201", r.Status, r.Body)
		}
		id, err := storage.ParseUUID(made.ID)
		if err != nil {
			c.t.Fatal(err)
		}
		c.write(`INSERT INTO targets_conversion (target_id, video_codec, max_height, bitrate_kbps, bit_depth, range, keep_dv, curve, audio, drop_commentary, container)
			VALUES (?, 'h264', ?, ?, 8, 'sdr', 0, 'bt2390', 'copy', 0, 'mkv')`, id, h, h*4)
		c.write(`INSERT INTO targets_profile_target (profile_id, position, target_id, sourcing, from_target_id)
			SELECT p.id, (SELECT max(position) + 1 FROM targets_profile_target WHERE profile_id = p.id), ?, 'derive',
				(SELECT target_id FROM targets_profile_target WHERE profile_id = p.id AND position = 1)
			FROM targets_profile p WHERE p.content_type = 'movie' AND p.is_default = 1`, id)
	}
	now := storage.FromTime(time.Now())
	c.write(`INSERT INTO conversions_nudge (work_id, why, since, created_at) SELECT id, 'profile', ?, ? FROM works_work WHERE label = ?`, now, now, brick)
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

// sourceCopies is how many Transfers copy a source to a scratch folder.
func (c convWorld) sourceCopies() int {
	c.t.Helper()
	listed := simtest.Decode[struct {
		Transfers []struct {
			Purpose     string `json:"purpose"`
			Destination struct {
				Kind string `json:"kind"`
			} `json:"destination"`
		} `json:"transfers"`
	}](c.t, "listTransfers", c.w.API().Admin("home", http.MethodGet, "/api/v1/transfers", ""))
	n := 0
	for _, x := range listed.Transfers {
		if x.Purpose == "conversion" && x.Destination.Kind == "scratch" {
			n++
		}
	}
	return n
}

func TestAMadeFileLandsInTheLibrary(t *testing.T) {
	for _, tc := range []struct {
		name    string
		heights []int
		held    int
		atOnce  int
		then    func(c convWorld)
	}{
		{name: "it lands and fills every Target", heights: []int{720, 480}, then: convWorld.fillsEvery},
		{name: "it copies the source once for two Targets", heights: []int{720, 480}, held: 2, atOnce: 2, then: convWorld.copiesOnce},
		{name: "it cancels", heights: []int{720}, held: 1, then: convWorld.cancels},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				conv := simtest.NewConversions()
				for range tc.held {
					conv.Cue(brickSource, simtest.Hold)
				}
				c := launchConverting(t, conv)
				if tc.atOnce > 0 {
					c.setAtOnce(tc.atOnce)
				}
				c.derive(tc.heights...)
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
	if n := c.sourceCopies(); n != 1 {
		c.t.Errorf("%d Transfers copy the source while both run, want 1", n)
	}
	c.conv.Release(brickSource)
	c.met()
	if n := c.conv.Ran(brickSource); n != 2 {
		c.t.Errorf("den ran %d Conversions of the source, want 2", n)
	}
}

func (c convWorld) cancels() {
	id, err := storage.ParseUUID(c.running(1)[1].ConversionID)
	if err != nil {
		c.t.Fatal(err)
	}
	if _, err := c.w.Node("home").Half().Conversions.Cancel(c.t.Context(), conversions.ID{UUID: id}); err != nil {
		c.t.Fatal(err)
	}
	c.await("its made Target is missing", func(cells []convCell) bool { return cells[1].State == "missing" })
	time.Sleep(simtest.Settled)
	if cells := c.cells(); cells[1].State != "missing" || c.conv.Ran(brickSource) != 1 {
		c.t.Errorf("after the cancel the made Target is %s and den ran %d Conversions, want missing and 1", cells[1].State, c.conv.Ran(brickSource))
	}
}

// setAtOnce lets den run n Conversions at once, below the API, which
// refuses it in this build.
func (c convWorld) setAtOnce(n int) {
	c.t.Helper()
	var den role.NodeID
	if err := c.w.Node("home").Half().Library.QueryRowContext(c.t.Context(), `SELECT id FROM nodes_node WHERE name = 'den'`).Scan(&den); err != nil {
		c.t.Fatal(err)
	}
	if _, err := c.w.Node("home").Half().Conversions.SetNodeSettings(c.t.Context(), den, conversions.NodeSettings{AtOnce: &n, Processor: true}); err != nil {
		c.t.Fatal(err)
	}
}
