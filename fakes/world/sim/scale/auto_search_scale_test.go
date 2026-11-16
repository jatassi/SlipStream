//go:build scale

// Package scale_test is the scale simulation of Automatic search, Backlog
// search and RSS sync: thousands of wanted Works over ten Nodes and fifty
// Indexers, run for a day of fake time. It reports and asserts nothing
// but that no Indexer went past its limit, so it runs only on demand:
//
//	go test -tags scale -run TestAutoSearchScale ./fakes/world/sim/scale/ -v -timeout 3h
package scale_test

import (
	"flag"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/sim/simtest"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

var (
	seriesCount = flag.Int("scale.series", 1000, "generated series, each of 4 seasons of 6 episodes, so 4 search groups")
	movieCount  = flag.Int("scale.movies", 4000, "generated movies, each one search group")
	hours       = flag.Int("scale.hours", 24, "hours of fake time to run, past Backlog search's first run at 8 hours")
)

const (
	nodes          = 10
	indexersOnEach = 5
	addsAtOnce     = 16
	// lateBy is past the search wait the test sets, 10 seconds, and under
	// the site's HTTP timeout, so the late Indexer's lines read late.
	lateBy     = 20 * time.Second
	searchWait = 10
	// The budgets the report prints against, from the design's open
	// question. Nothing asserts them.
	wantedBudget = 500 * time.Millisecond
)

// wallClock answers the real time inside a synctest bubble, whose
// time.Now is fake: a goroutine outside the bubble reads the clock for it.
type wallClock struct {
	ask    chan struct{}
	answer chan time.Time
}

func startWallClock(t *testing.T) wallClock {
	c := wallClock{ask: make(chan struct{}), answer: make(chan time.Time)}
	go func() {
		for range c.ask {
			c.answer <- time.Now()
		}
	}()
	t.Cleanup(func() { close(c.ask) })
	return c
}

func (c wallClock) now() time.Time {
	c.ask <- struct{}{}
	return <-c.answer
}

func (c wallClock) took(f func()) time.Duration {
	start := c.now()
	f()
	return c.now().Sub(start)
}

func TestAutoSearchScale(t *testing.T) {
	wall := startWallClock(t)
	synctest.Test(t, func(t *testing.T) {
		s := world(t)
		var w *sim.World
		launched := wall.took(func() { w = sim.Launch(t, s, sim.WithWalk()) })
		r := run{t: t, w: w, wall: wall}
		added := wall.took(r.addSeries)
		synctest.Wait()
		r.slowLateIndexer()
		t.Logf("launched %d Nodes with %d movies in %s, added %d series in %s (wall)", nodes, *movieCount, launched.Round(time.Millisecond), *seriesCount, added.Round(time.Millisecond))

		r.report(r.observe(time.Duration(*hours) * time.Hour))
	})
}

// world is the Scenario: home, the Controller, and nine Workers, every
// Node holding Searching with five Indexers homed on it. Every fifth
// Indexer allows 10 searches an hour and every fifth after it 100 a day;
// Tamarack-50 answers late. No site lists a Release, so every group stays
// wanted.
func world(t *testing.T) scenario.Scenario {
	t.Helper()
	var b strings.Builder
	fmt.Fprintf(&b, `description: The scale simulation of Automatic search.
seed: auto-search-scale
admin: {email: ada@example.test, display_name: Ada Okafor, pin: "2468"}
nodes:
  - {name: home, setup: controller, roles: [searching, downloading, storage, metadata]}
`)
	for i := 1; i < nodes; i++ {
		fmt.Fprintf(&b, "  - {name: worker%d, setup: join, roles: [searching]}\n", i)
	}
	fmt.Fprintf(&b, `timings:
  tmdb: {interval: 1ms, burst: 100}
  tvdb: {interval: 1ms, burst: 100}
tmdb:
  generated: {count: %d, seed: auto-search-scale}
tvdb:
  generated: [{count: %d, seasons: 4, episodes: 6, seed: auto-search-scale}]
library:
  storage_levels: {warn_below: 0B, error_below: 0B}
  roots: [{content_type: series, folder: tv}, {content_type: movie, folder: movies}]
  works: {content_type: movie, preset: all}
  indexers:
`, *movieCount, *seriesCount)
	for i := range nodes * indexersOnEach {
		fmt.Fprintf(&b, "    - %s\n", indexer(i))
	}
	s, err := scenario.Parse("auto-search-scale", []byte(b.String()))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func indexer(i int) string {
	home := "home"
	if n := i / indexersOnEach; n > 0 {
		home = fmt.Sprintf("worker%d", n)
	}
	site, name := "torznab", fmt.Sprintf("Fixture-%02d", i+1)
	switch {
	case i == nodes*indexersOnEach-1:
		site, name = "tamarack", fmt.Sprintf("Tamarack-%02d", i+1)
	case i%5 == 0:
		site, name = "heron", fmt.Sprintf("Heron-%02d", i+1)
	}
	out := fmt.Sprintf("{name: %s, kind: torznab, site: %s, home: %s", name, site, home)
	switch i % 5 {
	case 0:
		out += ", limit: {count: 10, per: hour}"
	case 1:
		out += ", limit: {count: 100, per: day}"
	}
	return out + "}"
}

type run struct {
	t    *testing.T
	w    *sim.World
	wall wallClock
}

func (r run) admin(method, path, body string) worldtest.Reply {
	r.t.Helper()
	reply := r.w.API().Admin("home", method, "/api/v1"+path, body)
	if reply.Status >= http.StatusBadRequest {
		r.t.Fatalf("%s %s = %d %s", method, path, reply.Status, reply.Body)
	}
	return reply
}

// addSeries adds every generated series, monitored, with no search on
// add, as an import of an existing library would.
func (r run) addSeries() {
	slots := make(chan struct{}, addsAtOnce)
	var wg sync.WaitGroup
	for i := range *seriesCount {
		slots <- struct{}{}
		wg.Go(func() {
			defer func() { <-slots }()
			r.admin(http.MethodPost, "/works", fmt.Sprintf(`{"contentType":"series","key":"%d","preset":"all"}`, scenario.TVDBGeneratedFirstID+int64(i)))
		})
	}
	wg.Wait()
}

func (r run) slowLateIndexer() {
	r.admin(http.MethodPatch, "/search-settings", fmt.Sprintf(`{"values":{"indexers.searchWait":%d}}`, searchWait))
	if _, err := r.w.Faults().Set(fault.Rule{ID: "late", Target: "fixture-site", Route: "tamarack.api", Fault: fault.Delay(lateBy)}); err != nil {
		r.t.Fatal(err)
	}
}

type status struct {
	Searching int `json:"searching"`
	Queued    int `json:"queued"`
}

type sample struct {
	at         time.Duration
	status     status
	wantedTook time.Duration
}

type observed struct {
	samples       []sample
	backlogTook   time.Duration
	backlogNote   string
	firstQueuedAt time.Duration
	drainedAt     time.Duration
}

// observe reads Wanted's first page every minute of fake time for span,
// timing each read on the wall clock, and runs Backlog search once at
// the start, timed on the wall clock too.
func (r run) observe(span time.Duration) observed {
	var o observed
	o.backlogTook = r.wall.took(func() { o.backlogNote = r.backlogRunNow() })
	start := time.Now()
	for elapsed := time.Duration(0); elapsed <= span; elapsed = time.Since(start) {
		var page worldtest.Reply
		took := r.wall.took(func() { page = r.admin(http.MethodGet, "/wanted", "") })
		st := simtest.Decode[struct {
			Status status `json:"status"`
		}](r.t, "listWanted", page).Status
		o.samples = append(o.samples, sample{at: elapsed, status: st, wantedTook: took})
		busy := st.Searching+st.Queued > 0
		if busy && o.firstQueuedAt == 0 {
			o.firstQueuedAt = elapsed
		}
		if !busy && o.firstQueuedAt != 0 && o.drainedAt == 0 {
			o.drainedAt = elapsed
		}
		time.Sleep(time.Minute)
		synctest.Wait()
	}
	return o
}

func (r run) backlogRunNow() string {
	id := simtest.RunNow(r.t, r.w.API(), "backlog-search")
	var note string
	simtest.Eventually(r.t, "Backlog search's run ends", func() bool {
		d := simtest.Decode[simtest.TaskDetail](r.t, "getTask", (r.admin(http.MethodGet, "/tasks/backlog-search", "")))
		for _, run := range d.Runs {
			if run.ID == id && run.Result != "" {
				note = run.Summary
				return true
			}
		}
		return false
	})
	return note
}

type indexerUse struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Searches struct {
		Count int    `json:"count"`
		Per   string `json:"per"`
	} `json:"searches"`
}

type query struct {
	Use        string    `json:"use"`
	ReservedAt time.Time `json:"reservedAt"`
}

func (r run) queries(id string) []query {
	var out []query
	for offset := 0; ; {
		page := simtest.Decode[struct {
			Queries    []query `json:"queries"`
			NextOffset *int    `json:"nextOffset"`
		}](r.t, "listIndexerQueries", (r.admin(http.MethodGet, fmt.Sprintf("/indexers/%s/queries?offset=%d", id, offset), "")))
		for _, q := range page.Queries {
			if q.Use == "search" || q.Use == "recent" {
				out = append(out, q)
			}
		}
		if page.NextOffset == nil {
			return out
		}
		offset = *page.NextOffset
	}
}

// mostIn is the most of qs inside any one window.
func mostIn(qs []query, window time.Duration) int {
	most := 0
	for _, q := range qs {
		n := 0
		for _, other := range qs {
			if d := other.ReservedAt.Sub(q.ReservedAt); d >= 0 && d < window {
				n++
			}
		}
		most = max(most, n)
	}
	return most
}

func (r run) report(o observed) {
	t := r.t
	var peak status
	var slowest, total time.Duration
	for _, s := range o.samples {
		if s.status.Searching > peak.Searching {
			peak.Searching = s.status.Searching
		}
		if s.status.Queued > peak.Queued {
			peak.Queued = s.status.Queued
		}
		slowest = max(slowest, s.wantedTook)
		total += s.wantedTook
	}
	last := o.samples[len(o.samples)-1]
	t.Logf("Backlog search Run now: %q in %s (wall)", o.backlogNote, o.backlogTook.Round(time.Millisecond))
	t.Logf("Wanted's first page: slowest %s, mean %s over %d reads (wall; budget %s)", slowest.Round(time.Millisecond), (total / time.Duration(len(o.samples))).Round(time.Millisecond), len(o.samples), wantedBudget)
	t.Logf("peak open runs %d, peak queued groups %d", peak.Searching, peak.Queued)
	switch {
	case o.firstQueuedAt == 0:
		t.Logf("nothing queued in %d hours", *hours)
	case o.drainedAt == 0:
		t.Logf("first queued at %s; not drained after %d hours: %d searching, %d queued", o.firstQueuedAt, *hours, last.status.Searching, last.status.Queued)
	default:
		t.Logf("first queued at %s, drained at %s (fake time)", o.firstQueuedAt, o.drainedAt)
	}
	ixs := simtest.Decode[struct {
		Indexers []indexerUse `json:"indexers"`
	}](t, "listIndexers", (r.admin(http.MethodGet, "/indexers", ""))).Indexers
	for _, ix := range ixs {
		qs := r.queries(ix.ID)
		hour, day := mostIn(qs, time.Hour), mostIn(qs, 24*time.Hour)
		if ix.Searches.Count == 0 {
			t.Logf("%s: %d queries, most %d in an hour, %d in a day, no limit", ix.Name, len(qs), hour, day)
			continue
		}
		t.Logf("%s: %d queries, most %d in an hour, %d in a day, limit %d per %s", ix.Name, len(qs), hour, day, ix.Searches.Count, ix.Searches.Per)
		if most := map[string]int{"hour": hour, "day": day}[ix.Searches.Per]; most > ix.Searches.Count {
			t.Errorf("%s took %d queries in a %s, past its limit of %d", ix.Name, most, ix.Searches.Per, ix.Searches.Count)
		}
	}
}
