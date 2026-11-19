//go:build scale

package scale_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	_ "modernc.org/sqlite"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
	mspeech "github.com/jatassi/slipstream/internal/media/speech"
	"github.com/jatassi/slipstream/internal/media/speech/speechtest"
	"github.com/jatassi/slipstream/internal/node"
)

// The subtitles scale simulation: 4,000 films and 1,000 series of 46
// episodes on eight Storage Nodes, a Subtitles Node and the Controller.
// English joins every Profile, the map and search drains run until
// SubDL's 50 downloads are spent, the Controller restarts, and the drains
// run again. It runs only on demand:
//
//	go test -tags scale -run TestSubtitlesScale ./fakes/world/sim/scale/ -v -timeout 6h
var (
	subFilms    = flag.Int("scale.subtitles.films", 4000, "films, one file each")
	subSeries   = flag.Int("scale.subtitles.series", 1000, "series, each of 2 seasons of 23 episodes")
	subHours    = flag.Int("scale.subtitles.hours", 24, "fake hours each drain runs, before and after the restart")
	mapBuild    = flag.Duration("scale.subtitles.map", 10*time.Second, "fake time each Speech Map takes to build")
	subtitledBy = flag.Int("scale.subtitles.listed", 120, "films SubDL lists a subtitle for, which fail their timing, so SubDL spends its 50 downloads")
)

const (
	storageNodes    = 8
	episodesASeason = 23
	seasons         = 2
	// The design's bounds (Scale and bounded reads).
	marksBudget = 50 * time.Millisecond
	gainBudget  = 250 * time.Millisecond
)

func TestSubtitlesScale(t *testing.T) {
	wall := startWallClock(t)
	synctest.Test(t, func(t *testing.T) {
		s := subtitlesWorld(t)
		var w *sim.World
		launched := wall.took(func() {
			w = sim.Launch(t, s, sim.WithWalk(), sim.Edit(func(_ string, e *node.Env) { e.Speech = slowMapper{*mapBuild} }))
		})
		r := subRun{t: t, w: w, wall: wall}
		r.lib = r.open()
		t.Logf("launched %d Nodes with %d films and %d series (%d files) in %s (wall)", len(s.Nodes), *subFilms, *subSeries, r.fileCount(), launched.Round(time.Millisecond))
		wallStart, fakeStart := wall.now(), time.Now()
		mark := func(what string) {
			t.Logf("%s: %s wall, %s fake since launch", what, wall.now().Sub(wallStart).Round(time.Second), time.Since(fakeStart).Round(time.Second))
		}
		r.awaitLibrary(*subFilms + *subSeries)
		mark("the library lists every Work")
		r.put(http.MethodPut, "/api/v1/subtitles/providers/subdl/key", `{"key":"scale"}`)
		r.slots(1)

		gains := map[string]time.Duration{}
		for _, profile := range []string{"Movies", "TV"} {
			gains[profile] = r.wantEnglish(profile)
		}
		reconciled := r.awaitWants()
		mark("every file wants English")
		mapsAt1 := r.drain(time.Hour)
		r.slots(4)
		mapsAt4 := r.drain(time.Hour)
		mark("two hours of drains")
		out, queue := r.drainUntilSubDLOut(time.Duration(*subHours-2) * time.Hour)
		mark("the first drain")
		before := r.searchRuns()
		r.report(scaleReport{
			gains: gains, reconciled: reconciled, mapsAt1: mapsAt1, mapsAt4: mapsAt4,
			runs: before, subdlOut: out, queue: queue, marks: r.libraryMarks(),
		})

		w.Node("home").Stop()
		w.Node("home").Start()
		settled := r.awaitConverged()
		mark("the restart settled")
		t.Logf("after the restart, intents and leases settled in %s fake, %s wall", settled.fake.Round(time.Second), settled.wall.Round(time.Millisecond))
		r.drain(time.Duration(*subHours) * time.Hour)
		mark("the second drain")
		after := r.searchRuns()
		for i, run := range slices.Backward(after[:len(after)-len(before)]) {
			t.Logf("search run %d after the restart: %s, %s", i+1, run.Result, run.Summary)
		}
	})
}

// subtitlesWorld is home, the Controller holding Metadata, eight Storage
// Nodes each with a Movies and a TV root, and subs holding Subtitles.
func subtitlesWorld(t *testing.T) scenario.Scenario {
	t.Helper()
	var b strings.Builder
	fmt.Fprintf(&b, `description: The scale simulation of Subtitles.
seed: subtitles-scale
admin: {email: ada@example.test, display_name: Ada Okafor, pin: "2468"}
nodes:
  - {name: home, setup: controller, roles: [metadata]}
  - {name: subs, setup: join, roles: [subtitles]}
`)
	for i := 1; i <= storageNodes; i++ {
		fmt.Fprintf(&b, "  - {name: store%d, setup: join, roles: [storage]}\n", i)
	}
	fmt.Fprintf(&b, `timings:
  tmdb: {interval: 1ms, burst: 100}
  tvdb: {interval: 1ms, burst: 100}
tmdb:
  generated: {count: %d, seed: subtitles-scale}
  movies:
`, *subFilms-*subtitledBy)
	for i := range *subtitledBy {
		fmt.Fprintf(&b, "    - {id: %d, imdb_id: tt%d, title: Listed Film %d, released: 2010-01-01, runtime: 2m}\n", listedID(i), listedID(i), i+1)
	}
	fmt.Fprintf(&b, `tvdb:
  generated: [{count: %d, seasons: %d, episodes: %d, seed: subtitles-scale}]
subtitle_providers:
  catalog:
`, *subSeries, seasons, episodesASeason)
	for i := range *subtitledBy {
		fmt.Fprintf(&b, "    - {id: listed-%d, provider: subdl, work: Listed Film %d, language: en, release: Listed.Film.%d.2010.1080p.WEB-DL-OTHER}\n", i+1, i+1, i+1)
	}
	b.WriteString("library:\n  storage_levels: {warn_below: 0B, error_below: 0B}\n  roots:\n")
	for i := 1; i <= storageNodes; i++ {
		fmt.Fprintf(&b, "    - {content_type: movie, folder: movies%d, node: store%d}\n    - {content_type: series, folder: tv%d, node: store%d}\n", i, i, i, i)
	}
	b.WriteString("  folders:\n")
	const video = `video: {width: 1920, height: 1080, seconds: 1, hdr: sdr}, audio: [{codec: aac, channels: 2, language: eng}]`
	films := make([][]string, storageNodes)
	for i := range *subFilms {
		id, name := listedID(i), fmt.Sprintf("Listed Film %d (2010)", i+1)
		if i >= *subtitledBy {
			id = scenario.GeneratedFirstID + int64(i-*subtitledBy)
			name = fmt.Sprintf("Film %d", i+1)
		}
		films[i%storageNodes] = append(films[i%storageNodes], fmt.Sprintf(`      - {path: "%s {tmdb-%d}/%s.mkv", %s}`, name, id, name, video))
	}
	shows := make([][]string, storageNodes)
	for i := range *subSeries {
		name := fmt.Sprintf("Show %d [tvdbid-%d]", i+1, scenario.TVDBGeneratedFirstID+int64(i))
		for season := 1; season <= seasons; season++ {
			for e := 1; e <= episodesASeason; e++ {
				shows[i%storageNodes] = append(shows[i%storageNodes], fmt.Sprintf(`      - {path: "%s/Season %02d/Show %d - S%02dE%02d.mkv", %s}`, name, season, i+1, season, e, video))
			}
		}
	}
	for i := range storageNodes {
		for _, f := range []struct {
			folder string
			files  []string
		}{{fmt.Sprintf("movies%d", i+1), films[i]}, {fmt.Sprintf("tv%d", i+1), shows[i]}} {
			if len(f.files) == 0 {
				continue
			}
			fmt.Fprintf(&b, "    - folder: %s\n      files:\n%s\n", f.folder, strings.Join(f.files, "\n"))
		}
	}
	s, err := scenario.Parse("subtitles-scale", []byte(b.String()))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func listedID(i int) int64 { return 9_500_001 + int64(i) }

// slowMapper builds every file's Speech Map from two minutes of speech in
// d of fake time, as a Node's ffmpeg and libfvad take real time.
type slowMapper struct{ d time.Duration }

var scaleSpeech = func() []speechtest.Run {
	var runs []speechtest.Run
	for at := time.Duration(0); at < 2*time.Minute; at += 5 * time.Second {
		runs = append(runs, speechtest.Run{Start: at, End: at + 2*time.Second})
	}
	return runs
}()

func (m slowMapper) Map(ctx context.Context, _ string, stream int) (mspeech.Map, error) {
	select {
	case <-time.After(m.d):
	case <-ctx.Done():
		return mspeech.Map{}, fmt.Errorf("build a Speech Map: %w", context.Cause(ctx))
	}
	return speechtest.Map(scaleSpeech, 2*time.Minute, stream)
}

type subRun struct {
	t    *testing.T
	w    *sim.World
	wall wallClock
	lib  *sql.DB
}

func (r subRun) call(method, path, body string) (status int, reply string) {
	got := r.w.API().Admin("home", method, path, body)
	return got.Status, got.Body
}

func (r subRun) put(method, path, body string) {
	r.t.Helper()
	if status, reply := r.call(method, path, body); status/100 != 2 {
		r.t.Fatalf("%s %s = %d %s", method, path, status, reply)
	}
}

func (r subRun) fileCount() int { return *subFilms + *subSeries*seasons*episodesASeason }

func (r subRun) awaitLibrary(works int) {
	r.t.Helper()
	for range 6 * 60 {
		_, body := r.call(http.MethodGet, "/api/v1/library", "")
		var lib struct {
			Rows []json.RawMessage `json:"rows"`
		}
		if json.Unmarshal([]byte(body), &lib) == nil && len(lib.Rows) >= works {
			return
		}
		time.Sleep(10 * time.Second)
	}
	r.t.Fatalf("the library never listed %d Works", works)
}

// slots sets every Storage Node's audio scans at once.
func (r subRun) slots(n int) {
	r.t.Helper()
	for _, row := range r.w.API().Nodes("home") {
		if strings.HasPrefix(row.Name, "store") {
			r.put(http.MethodPatch, "/api/v1/subtitles/nodes/"+row.ID+"/settings", fmt.Sprintf(`{"values":{"subtitles.mapSlots":%d}}`, n))
		}
	}
}

// wantEnglish adds English to the Profile and answers how long the PUT,
// whose change Write it waits on, took on the wall clock, from its
// request to its reply.
func (r subRun) wantEnglish(profile string) time.Duration {
	r.t.Helper()
	_, body := r.call(http.MethodGet, "/api/v1/profiles", "")
	var listed struct {
		Profiles []map[string]json.RawMessage `json:"profiles"`
	}
	if err := json.Unmarshal([]byte(body), &listed); err != nil {
		r.t.Fatal(err)
	}
	for _, p := range listed.Profiles {
		var name, id string
		_ = json.Unmarshal(p["name"], &name)
		_ = json.Unmarshal(p["id"], &id)
		if name != profile {
			continue
		}
		put := `{"name":` + string(p["name"]) + `,"contentType":` + string(p["contentType"]) + `,"targets":` + string(p["targets"]) +
			`,"editions":` + string(p["editions"]) + `,"languages":[{"language":"en","forced":false,"hearingImpaired":"allow","audio":"always"}]}`
		api := r.w.API()
		var got worldtest.Reply
		took := r.wall.took(func() {
			got = api.Admin("home", http.MethodPut, "/api/v1/profiles/"+id+"?confirm=0&confirmSubtitles=0", put)
		})
		if got.Status != http.StatusOK {
			r.t.Fatalf("PUT %s's Profile = %d %s", profile, got.Status, got.Body)
		}
		return took
	}
	r.t.Fatalf("no Profile is named %s", profile)
	return 0
}

// open opens the Controller's library.db, query only, beside the running
// Node.
func (r subRun) open() *sql.DB {
	r.t.Helper()
	path := filepath.Join(r.w.Node("home").DataDir().Path(), "library.db")
	d, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path)+"?_pragma=busy_timeout(5000)&_pragma=query_only(1)")
	if err != nil {
		r.t.Fatal(err)
	}
	r.t.Cleanup(func() { _ = d.Close() })
	return d
}

func (r subRun) count(query string, args ...any) int {
	r.t.Helper()
	var n int
	if err := r.lib.QueryRowContext(r.t.Context(), query, args...).Scan(&n); err != nil {
		r.t.Fatalf("%s: %v", query, err)
	}
	return n
}

type reconciled struct {
	wants int
	fake  time.Duration
	wall  time.Duration
}

// awaitWants waits until every file has its English want row.
func (r subRun) awaitWants() reconciled {
	r.t.Helper()
	start, wallStart := time.Now(), r.wall.now()
	var n int
	for range 6 * 60 * 60 {
		if n = r.count(`SELECT count(*) FROM subtitles_want WHERE language = 'en'`); n >= r.fileCount() {
			return reconciled{wants: n, fake: time.Since(start), wall: r.wall.now().Sub(wallStart)}
		}
		time.Sleep(time.Second)
	}
	r.t.Fatalf("the bulk pass wrote %d wants in 6 hours, want %d", n, r.fileCount())
	return reconciled{}
}

// drain lets the drains run for d of fake time and answers each Storage
// Node's Speech Maps built in it.
func (r subRun) drain(d time.Duration) map[string]int {
	r.t.Helper()
	before := r.maps()
	time.Sleep(d)
	synctest.Wait()
	out := map[string]int{}
	for name, n := range r.maps() {
		out[name] = n - before[name]
	}
	return out
}

// drainUntilSubDLOut drains d an hour at a time and answers the fake
// UTC time SubDL first reads out of downloads, with NextQueued read then,
// before its reset at midnight. When SubDL never runs out, NextQueued is
// read at the end.
func (r subRun) drainUntilSubDLOut(d time.Duration) (string, queueRead) {
	r.t.Helper()
	for hour := time.Hour; hour <= d; hour += time.Hour {
		r.drain(time.Hour)
		if r.providerState("subdl") == "out" {
			at, queue := time.Now().UTC().Format("15:04"), r.nextQueued()
			r.drain(d - hour)
			return at, queue
		}
	}
	return "", r.nextQueued()
}

func (r subRun) maps() map[string]int {
	out := map[string]int{}
	for i := 1; i <= storageNodes; i++ {
		name := fmt.Sprintf("store%d", i)
		entries, _ := os.ReadDir(r.w.Node(name).DataDir().SpeechMapsDir())
		out[name] = len(entries)
	}
	return out
}

type searchRun struct {
	Summary string `json:"summary"`
	Result  string `json:"result"`
}

func (r subRun) searchRuns() []searchRun {
	r.t.Helper()
	_, body := r.call(http.MethodGet, "/api/v1/tasks/subtitle-search", "")
	var d struct {
		Runs []searchRun `json:"runs"`
	}
	if err := json.Unmarshal([]byte(body), &d); err != nil {
		r.t.Fatal(err)
	}
	return d.Runs
}

func (r subRun) providerState(id string) string {
	r.t.Helper()
	_, body := r.call(http.MethodGet, "/api/v1/subtitles/providers", "")
	var list struct {
		Providers []struct {
			ID    string `json:"id"`
			State struct {
				Kind string `json:"kind"`
			} `json:"state"`
		} `json:"providers"`
	}
	if err := json.Unmarshal([]byte(body), &list); err != nil {
		r.t.Fatal(err)
	}
	for _, p := range list.Providers {
		if p.ID == id {
			return p.State.Kind
		}
	}
	return ""
}

// ownerQuery is the named query of the subtitles owner's sqlc file, with each
// sqlc.arg(x) as ?.
func ownerQuery(t *testing.T, file, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "internal", "subtitles", "queries", file))
	if err != nil {
		t.Fatal(err)
	}
	_, rest, ok := strings.Cut(string(data), "-- name: "+name+" ")
	if !ok {
		t.Fatalf("%s has no query %s", file, name)
	}
	_, rest, _ = strings.Cut(rest, "\n")
	body, _, _ := strings.Cut(rest, "-- name: ")
	return regexp.MustCompile(`sqlc\.arg\(\w+\)`).ReplaceAllString(strings.TrimSpace(body), "?")
}

type queueRead struct {
	kind      string
	plan      []string
	returned  int
	visited   int
	waiting   int
	movieWait int
}

// nextQueued reads NextQueued's plan and runs it for episodes, the kind
// that stays ready while SubDL is out. The plan is a search of the
// partial index subtitles_want_queue, whose WHERE is the query's, with no
// sort: every index row the range visits is a row it returns. visited
// counts the rows of that index range up to the last one returned.
func (r subRun) nextQueued() queueRead {
	r.t.Helper()
	q := ownerQuery(r.t, "impact.sql", "NextQueued")
	q = strings.TrimSuffix(q, ";")
	const n = 50
	rows, err := r.lib.QueryContext(r.t.Context(), "EXPLAIN QUERY PLAN "+q, "episode", n)
	if err != nil {
		r.t.Fatal(err)
	}
	read := queueRead{kind: "episode"}
	for rows.Next() {
		var id, parent, unused int
		var detail string
		if err := rows.Scan(&id, &parent, &unused, &detail); err != nil {
			r.t.Fatal(err)
		}
		read.plan = append(read.plan, detail)
	}
	_ = rows.Close()
	read.returned = r.count(`SELECT count(*) FROM (`+q+`)`, "episode", n)
	read.visited = r.count(`SELECT count(*) FROM (SELECT 1 FROM subtitles_want INDEXED BY subtitles_want_queue WHERE kind = ? AND queued_at IS NOT NULL AND attempt_id IS NULL ORDER BY queue_rank, queue_order, file_id LIMIT ?)`, "episode", n)
	read.waiting = r.count(`SELECT count(*) FROM subtitles_want WHERE kind = 'episode' AND queued_at IS NOT NULL AND attempt_id IS NULL`)
	read.movieWait = r.count(`SELECT count(*) FROM subtitles_want WHERE kind = 'feature' AND queued_at IS NOT NULL AND attempt_id IS NULL`)
	return read
}

type marksRead struct {
	rows int
	took time.Duration
}

// libraryMarks times ListLibraryMarks on the wall clock, the best of
// three, as the Library filter reads it.
func (r subRun) libraryMarks() marksRead {
	r.t.Helper()
	q := ownerQuery(r.t, "impact.sql", "ListLibraryMarks")
	best := marksRead{took: time.Hour}
	for range 3 {
		var n int
		took := r.wall.took(func() {
			rows, err := r.lib.QueryContext(r.t.Context(), q)
			if err != nil {
				r.t.Fatal(err)
			}
			for rows.Next() {
				n++
			}
			_ = rows.Close()
		})
		if took < best.took {
			best = marksRead{rows: n, took: took}
		}
	}
	return best
}

type converged struct {
	fake, wall      time.Duration
	intents, leases int
}

// awaitConverged waits after the restart until no intent row and no
// lease is left.
func (r subRun) awaitConverged() converged {
	r.t.Helper()
	start, wallStart := time.Now(), r.wall.now()
	var c converged
	for range 60 * 60 {
		c.intents = r.count(`SELECT count(*) FROM subtitles_write`)
		c.leases = r.count(`SELECT count(*) FROM subtitles_want WHERE attempt_id IS NOT NULL`)
		if c.intents == 0 && c.leases == 0 {
			c.fake, c.wall = time.Since(start), r.wall.now().Sub(wallStart)
			return c
		}
		time.Sleep(time.Second)
	}
	r.t.Fatalf("an hour after the restart %d intents and %d leases are left", c.intents, c.leases)
	return c
}

type scaleReport struct {
	gains            map[string]time.Duration
	reconciled       reconciled
	mapsAt1, mapsAt4 map[string]int
	runs             []searchRun
	subdlOut         string
	queue            queueRead
	marks            marksRead
}

// report logs and checks what the first drain left, before the restart,
// so a run stopped later still reports it.
func (r subRun) report(rep scaleReport) {
	t := r.t
	t.Helper()
	t.Logf("language gain PUT (change Write and reply): Movies %s, TV %s (wall, budget %s)", rep.gains["Movies"].Round(time.Millisecond), rep.gains["TV"].Round(time.Millisecond), gainBudget)
	t.Logf("bulk pass wrote %d wants in %s fake, %s wall", rep.reconciled.wants, rep.reconciled.fake.Round(time.Second), rep.reconciled.wall.Round(time.Millisecond))
	t.Logf("maps built in the first fake hour at 1 slot: %v", rep.mapsAt1)
	t.Logf("maps built in the next fake hour at 4 slots: %v", rep.mapsAt4)
	for i, run := range slices.Backward(rep.runs) {
		t.Logf("search run %d before the restart: %s, %s", i+1, run.Result, run.Summary)
	}
	t.Logf("SubDL first read out of downloads at %q (fake UTC), when %d film wants waited in the queue", rep.subdlOut, rep.queue.movieWait)
	t.Logf("NextQueued(%s, 50): returned %d, index range visited %d, %d waiting; plan %q", rep.queue.kind, rep.queue.returned, rep.queue.visited, rep.queue.waiting, rep.queue.plan)
	t.Logf("ListLibraryMarks: %d Works in %s (wall, budget %s)", rep.marks.rows, rep.marks.took.Round(time.Microsecond), marksBudget)

	if rep.subdlOut == "" {
		t.Errorf("SubDL never read out of downloads in the first drain")
	}
	if len(rep.queue.plan) != 1 || !strings.Contains(rep.queue.plan[0], "USING INDEX subtitles_want_queue (kind=?)") {
		t.Errorf("NextQueued's plan is %q, want one search of subtitles_want_queue with no sort", rep.queue.plan)
	}
	if rep.queue.visited >= 2*max(rep.queue.returned, 1) {
		t.Errorf("NextQueued visited %d rows for %d returned, want under twice", rep.queue.visited, rep.queue.returned)
	}
	if rep.marks.took >= marksBudget {
		t.Errorf("ListLibraryMarks took %s, want under %s", rep.marks.took, marksBudget)
	}
	for profile, took := range rep.gains {
		if took >= gainBudget {
			t.Errorf("adding English to %s took %s, want under %s", profile, took, gainBudget)
		}
	}
}
