//go:build scale

package scale_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/sim/simtest"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
	"github.com/jatassi/slipstream/internal/conversions"
	"github.com/jatassi/slipstream/internal/converting"
	nodespkg "github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/storage"
)

var (
	convMovies = flag.Int("scale.conversions.movies", 5000, "1080p movies on home, each converted to 720p")
	convNodes  = flag.Int("scale.conversions.nodes", 10, "Workers holding Converting")
)

// released is the day every movie is out before, as Test.Today puts the
// sim's year 2000 in 2026.
const released = "2026-01-01"

// placedStages are the stages a Conversion holds its Node in.
const placedStages = `('copying', 'ready', 'running', 'paused', 'returning')`

// TestConversionsScale is the scale simulation of S44 (Conversion), run on
// demand and never in CI:
//
//	go test -tags scale -run TestConversionsScale ./fakes/world/sim/scale/ -v -timeout 2h
//
// It puts 5,000 1080p movies on home, a Target made at 720p from each in
// the movie Profile, and ten Workers holding Converting whose runs hold
// until the test ends. It reports the drain, each placement Write, the
// wall time of an idle pass over the queue, a Worker's lapse and the
// Conversions tab's first page, and asserts only that no Worker holds more
// than one source past its Conversions at once and that the lapse puts
// back every Conversion it held.
func TestConversionsScale(t *testing.T) {
	wall := startWallClock(t)
	synctest.Test(t, func(t *testing.T) {
		s, stems := conversionsWorld(t)
		conv := simtest.NewConversions()
		for _, stem := range stems {
			conv.Cue(stem, simtest.Hold)
		}
		var w *sim.World
		launched := wall.took(func() { w = sim.Launch(t, s, sim.WithWalk(), conv.Option()) })
		r := convRun{t: t, w: w, api: w.API(), wall: wall}
		t.Logf("launched home and %d Workers with %d movies in %s (wall)", *convNodes, *convMovies, launched.Round(time.Millisecond))
		for i := 1; i <= *convNodes; i++ {
			if err := os.MkdirAll(w.Node(worker(i)).DataDir().Resolve(converting.DefaultSettings().Scratch), 0o700); err != nil {
				t.Fatal(err)
			}
		}
		r.await("the scans record every movie", time.Duration(*convMovies)*time.Second, func() bool {
			return r.count(`SELECT count(*) FROM files_file`) >= *convMovies
		})

		since, wallSince := r.derive720(), wall.now()
		drained := r.timed(since, wallSince, func() bool { return r.count(`SELECT count(*) FROM conversions_conversion`) >= *convMovies })
		t.Logf("drain: %d Conversions queued in %s (fake), %s (wall), %d nudges a Write",
			*convMovies, drained.fake.Round(time.Second), drained.wall.Round(time.Millisecond), conversions.DefaultTimings().DrainBatch)

		placed := r.timed(since, wallSince, func() bool {
			return r.count(`SELECT count(*) FROM conversions_conversion WHERE stage = 'running'`) == *convNodes
		})
		t.Logf("placement: a run on every Worker %s (fake), %s (wall) after the nudges", placed.fake.Round(time.Second), placed.wall.Round(time.Millisecond))
		r.placements(since)
		r.boundedPerNode()

		pass := conversions.DefaultTimings().Pass
		const passes = 10
		idle := wall.took(func() { time.Sleep(passes * pass) })
		t.Logf("idle: %d passes of %s with %d queued took %s (wall), %s a pass",
			passes, pass, r.count(`SELECT count(*) FROM conversions_conversion WHERE stage = 'queued'`), idle.Round(time.Millisecond), (idle / passes).Round(time.Millisecond))

		r.lapse(worker(1))
		r.boundedPerNode()
		r.firstPage()
	})
}

// conversionsWorld is home, the Controller holding Storage with one movie
// root, and the Workers holding Converting, with each movie's stem, the
// name its runs are cued by.
func conversionsWorld(t *testing.T) (scenario.Scenario, []string) {
	t.Helper()
	var b strings.Builder
	fmt.Fprintf(&b, `description: The scale simulation of Conversion.
seed: conversions-scale
admin: {email: ada@example.test, display_name: Ada Okafor, pin: "2468"}
nodes:
  - {name: home, setup: controller, roles: [storage, metadata]}
`)
	for i := 1; i <= *convNodes; i++ {
		fmt.Fprintf(&b, "  - {name: %s, setup: join, roles: [converting]}\n", worker(i))
	}
	fmt.Fprintf(&b, `timings:
  tmdb: {interval: 1ms, burst: 100}
tmdb:
  generated: {count: %d, seed: conversions-scale}
library:
  storage_levels: {warn_below: 0B, error_below: 0B}
  roots: [{content_type: movie, folder: movies, node: home}]
`, 2**convMovies)
	s, err := scenario.Parse("conversions-scale", []byte(b.String()))
	if err != nil {
		t.Fatal(err)
	}
	folder := scenario.LibraryFolder{Folder: "movies"}
	var stems []string
	for i := 0; len(stems) < *convMovies; i++ {
		m := s.TMDB.Generated.Movie(i)
		if m.Released >= released {
			continue
		}
		name := fmt.Sprintf("%s (%s)", m.Title, m.Dates[0].Date[:4])
		path := fmt.Sprintf("%s {tmdb-%d}/%s.1080p.WEB-DL.H264-FKW.mkv", name, m.ID, name)
		folder.Files = append(folder.Files, scenario.MediaFile{Path: path, Video: scenario.MediaVideo{Width: 1920, Height: 1080, Seconds: 6, HDR: "sdr"}})
		stems = append(stems, "movies/"+path)
	}
	s.Library.Folders = append(s.Library.Folders, folder)
	return s, stems
}

func worker(i int) string { return fmt.Sprintf("conv%d", i) }

type convRun struct {
	t    *testing.T
	w    *sim.World
	api  *worldtest.API
	wall wallClock
}

func (r convRun) count(query string, args ...any) int {
	r.t.Helper()
	var n int
	if err := r.w.Node("home").Half().Library.QueryRowContext(r.t.Context(), query, args...).Scan(&n); err != nil {
		r.t.Fatalf("%s: %v", query, err)
	}
	return n
}

func (r convRun) write(query string, args ...any) {
	r.t.Helper()
	err := r.w.Node("home").Half().Library.Write(r.t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, query, args...)
		return err
	})
	if err != nil {
		r.t.Fatal(err)
	}
}

// await waits in fake time, up to limit, until ok, failing with what.
func (r convRun) await(what string, limit time.Duration, ok func() bool) {
	r.t.Helper()
	for range limit / time.Second {
		if ok() {
			return
		}
		time.Sleep(time.Second)
	}
	r.t.Fatalf("not within %s: %s", limit, what)
}

type took struct{ fake, wall time.Duration }

// timed waits up to an hour of fake time until ok, polling each second,
// and answers how long since fake and wall it took.
func (r convRun) timed(fake, wall time.Time, ok func() bool) took {
	r.t.Helper()
	r.await("the wait ends", time.Hour, ok)
	return took{fake: time.Since(fake), wall: r.wall.now().Sub(wall)}
}

// derive720 adds a Target made at 720p from the movie Profile's first
// Target, below the API, which refuses Conversion settings in this build,
// nudges every movie in one Write and answers when.
func (r convRun) derive720() time.Time {
	r.t.Helper()
	reply := r.api.Admin("home", http.MethodPost, "/api/v1/targets", `{"name":"720p","tiers":[["webdl-720p","webrip-720p","hdtv-720p","bluray-720p"]],"cutoff":"webdl-720p",`+
		`"sizes":[{"quality":"webdl-720p","min":0,"max":null},{"quality":"webrip-720p","min":0,"max":null},{"quality":"hdtv-720p","min":0,"max":null},{"quality":"bluray-720p","min":0,"max":null}]}`)
	var made struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(reply.Body), &made); reply.Status != http.StatusCreated || err != nil {
		r.t.Fatalf("createTarget = %d %s, want 201", reply.Status, reply.Body)
	}
	id, err := storage.ParseUUID(made.ID)
	if err != nil {
		r.t.Fatal(err)
	}
	r.write(`INSERT INTO targets_conversion (target_id, video_codec, max_height, bitrate_kbps, bit_depth, range, keep_dv, curve, audio, drop_commentary, container)
		VALUES (?, 'h264', 720, 2880, 8, 'sdr', 0, 'bt2390', 'copy', 0, 'mkv')`, id)
	r.write(`INSERT INTO targets_profile_target (profile_id, position, target_id, sourcing, from_target_id)
		SELECT p.id, (SELECT max(position) + 1 FROM targets_profile_target WHERE profile_id = p.id), ?, 'derive',
			(SELECT target_id FROM targets_profile_target WHERE profile_id = p.id AND position = 1)
		FROM targets_profile p WHERE p.content_type = 'movie' AND p.is_default = 1`, id)
	now := time.Now()
	at := storage.FromTime(now)
	r.write(`INSERT INTO conversions_nudge (work_id, why, since, created_at) SELECT id, 'profile', ?, ? FROM works_work`, at, at)
	return now
}

// placements logs each placement Write since, one per fake instant its
// attempts started at, with the Conversions it placed.
func (r convRun) placements(since time.Time) {
	r.t.Helper()
	rows, err := r.w.Node("home").Half().Library.QueryContext(r.t.Context(),
		`SELECT started_at, count(*) FROM conversions_attempt WHERE started_at >= ? GROUP BY started_at ORDER BY started_at`, storage.FromTime(since))
	if err != nil {
		r.t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	var writes, most, all int
	var each []string
	for rows.Next() {
		var at storage.Time
		var n int
		if err := rows.Scan(&at, &n); err != nil {
			r.t.Fatal(err)
		}
		writes, most, all = writes+1, max(most, n), all+n
		each = append(each, fmt.Sprintf("%d at +%s", n, at.Time().Sub(since).Round(time.Second)))
	}
	if err := rows.Err(); err != nil {
		r.t.Fatal(err)
	}
	r.t.Logf("placement Writes: %d placing %d Conversions, at most %d in one: %s", writes, all, most, strings.Join(each, ", "))
}

// boundedPerNode fails when a Worker holds more Conversions than its
// Conversions at once and one source copied ahead.
func (r convRun) boundedPerNode() {
	r.t.Helper()
	bound := conversions.DefaultAtOnce + 1
	if n := r.count(`SELECT coalesce(max(held), 0) FROM (SELECT count(*) AS held FROM conversions_conversion WHERE stage IN ` + placedStages + ` GROUP BY node_id)`); n > bound {
		r.t.Errorf("a Worker holds %d Conversions, want at most %d", n, bound)
	}
}

// lapse stops name and waits past the offline grace until nothing is
// placed on it and every other Worker runs again, then logs how long that
// took after the grace and the Writes that put its Conversions back.
func (r convRun) lapse(name string) {
	r.t.Helper()
	var id nodespkg.ID
	if err := r.w.Node("home").Half().Library.QueryRowContext(r.t.Context(), `SELECT id FROM nodes_node WHERE name = ?`, name).Scan(&id); err != nil {
		r.t.Fatal(err)
	}
	held := r.count(`SELECT count(*) FROM conversions_conversion WHERE node_id = ? AND stage IN `+placedStages, id)
	r.w.Node(name).Stop()
	stopped, wallStopped := time.Now(), r.wall.now()
	time.Sleep(nodespkg.OfflineGrace.Default)
	back := r.timed(stopped, wallStopped, func() bool {
		return r.count(`SELECT count(*) FROM conversions_conversion WHERE node_id = ? AND stage IN `+placedStages, id) == 0 &&
			r.count(`SELECT count(*) FROM conversions_conversion WHERE stage = 'running'`) == *convNodes-1
	})
	writes := r.count(`SELECT count(DISTINCT ended_at) FROM conversions_attempt WHERE node_id = ? AND outcome = 'put_back'`, id)
	putBack := r.count(`SELECT count(*) FROM conversions_attempt WHERE node_id = ? AND outcome = 'put_back'`, id)
	r.t.Logf("lapse: %s held %d Conversions, %d put back in %d Writes; every other Worker runs %s (fake) past the grace, %s (wall) since the stop",
		name, held, putBack, writes, (back.fake - nodespkg.OfflineGrace.Default).Round(time.Second), back.wall.Round(time.Millisecond))
	if putBack != held {
		r.t.Errorf("the lapse put back %d of %s's %d Conversions, want every one", putBack, name, held)
	}
}

// firstPage times the Conversions tab's first page through the API, and
// logs its counts.
func (r convRun) firstPage() {
	r.t.Helper()
	var reply worldtest.Reply
	took := r.wall.took(func() { reply = r.api.Admin("home", http.MethodGet, "/api/v1/conversions", "") })
	if reply.Status != http.StatusOK {
		r.t.Fatalf("listConversions = %d %s, want 200", reply.Status, reply.Body)
	}
	var page struct {
		Counts json.RawMessage `json:"counts"`
	}
	if err := json.Unmarshal([]byte(reply.Body), &page); err != nil {
		r.t.Fatal(err)
	}
	r.t.Logf("first page: %d bytes in %s (wall), counts %s", len(reply.Body), took.Round(time.Millisecond), page.Counts)
}
