package e2e

import (
	"encoding/json"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

const (
	orchardBluray = "The.Quiet.Orchard.2021.1080p.BluRay.x264-FKW"
	orchardCMRG   = "The.Quiet.Orchard.2021.1080p.WEB-DL.H264-CMRG"
	orchard1987   = "The.Quiet.Orchard.1987.1080p.WEB-DL.H264-FKW"
	orchardLQ     = "The.Quiet.Orchard.2021.1080p.WEB-DL.H264-LQ"
	orchardTiny   = "The.Quiet.Orchard.2021.1080p.WEB-DL.H264-TINY"
	orchardBloat  = "The.Quiet.Orchard.2021.1080p.WEB-DL.H264-BLOAT"
	keepersE10    = "The.Lantern.Keepers.S02E10.1080p.WEB-DL.H264-NTb"
	keepersPack   = "The.Lantern.Keepers.S02.1080p.WEB-DL.H264-FKW"
)

type liftKey struct {
	Code     string `json:"code"`
	TargetID string `json:"targetId,omitempty"`
}

type manualRelease struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Indexers []string `json:"indexers"`
	Blocked  *string  `json:"blocked"`
}

type manualEdition struct {
	Edition string `json:"edition"`
	Words   string `json:"words"`
}

// manualPick is a plan row's pick or one of its choices.
type manualPick struct {
	Release manualRelease  `json:"release"`
	Cells   []string       `json:"cells"`
	Lifts   []liftKey      `json:"lifts"`
	Edition *manualEdition `json:"edition"`
}

type manualLine struct {
	IndexerID string `json:"indexerId"`
	Name      string `json:"name"`
	Sentence  string `json:"sentence"`
	Done      bool   `json:"done"`
	Automatic bool   `json:"automatic"`
}

type manualSheet struct {
	ID        string       `json:"id"`
	Searching bool         `json:"searching"`
	Line      string       `json:"line"`
	Indexers  []manualLine `json:"indexers"`
	Plan      struct {
		Headline string `json:"headline"`
		Rows     []struct {
			Pick    *manualPick `json:"pick"`
			Outcome string      `json:"outcome"`
			Missing string      `json:"missing"`
		} `json:"rows"`
	} `json:"plan"`
}

func (v manualSheet) line(t *testing.T, name string) manualLine {
	t.Helper()
	i := slices.IndexFunc(v.Indexers, func(l manualLine) bool { return l.Name == name })
	if i < 0 {
		t.Fatalf("the Manual search has no line for %s: %+v", name, v.Indexers)
	}
	return v.Indexers[i]
}

// picks is the plan's rows' picks.
func (v manualSheet) picks() []manualPick {
	var out []manualPick
	for _, r := range v.Plan.Rows {
		if r.Pick != nil {
			out = append(out, *r.Pick)
		}
	}
	return out
}

// manualOffer is one Release as listManualSearchReleases lists it.
type manualOffer struct {
	Release manualRelease `json:"release"`
	Grab    struct {
		Grabbable bool           `json:"grabbable"`
		Cells     []string       `json:"cells"`
		Lifts     []liftKey      `json:"lifts"`
		Edition   *manualEdition `json:"edition"`
	} `json:"grab"`
}

func (o manualOffer) pick() manualPick {
	return manualPick{Release: o.Release, Cells: o.Grab.Cells, Lifts: o.Grab.Lifts, Edition: o.Grab.Edition}
}

type manualGrab struct {
	grabView
	Source       string   `json:"source"`
	GrabbedBy    *string  `json:"grabbedBy"`
	ManualSearch *string  `json:"manualSearchId"`
	Overrides    []string `json:"overrides"`
	ReleaseTitle string   `json:"releaseTitle"`
	Import       *struct {
		Placed  int `json:"placed"`
		Waiting int `json:"waiting"`
	} `json:"import"`
	Actions struct {
		Blocklistable bool `json:"blocklistable"`
	} `json:"actions"`
}

type blockRow struct {
	InfoHash string `json:"infoHash"`
	Title    string `json:"title"`
	Cause    string `json:"cause"`
	Work     *struct {
		Title string `json:"title"`
	} `json:"work"`
}

// manualWorld is the manual-search Scenario once its first scans added
// The Quiet Orchard, w's Work, and The Lantern Keepers, series.
type manualWorld struct {
	walked
	series string
}

func launchManual(t *testing.T) *manualWorld {
	t.Helper()
	run := worldtest.Launch(t, "manual-search")
	w := &manualWorld{walked: walked{run: run, m: run.Manifest(), a: api(t, run)}}
	var library struct {
		Rows []struct {
			ID    string `json:"id"`
			Label string `json:"label"`
		} `json:"rows"`
	}
	worldtest.Until(t, "the first scans add both Works", func() (bool, error) {
		decode(t, "listLibrary", w.a.Admin("home", http.MethodGet, "/api/v1/library", ""), &library)
		return len(library.Rows) == 2, nil
	})
	for _, r := range library.Rows {
		switch r.Label {
		case "The Quiet Orchard":
			w.work = r.ID
		case "The Lantern Keepers":
			w.series = r.ID
		}
	}
	return w
}

// open starts a Manual search of work within body and returns its id.
func (w *manualWorld) open(t *testing.T, work, body string) string {
	t.Helper()
	r := w.a.Admin("home", http.MethodPost, "/api/v1/works/"+work+"/manual-search", body)
	if r.Status != http.StatusAccepted {
		t.Fatalf("startManualSearch = %d %s, want 202", r.Status, r.Body)
	}
	var started struct {
		ManualSearchID string `json:"manualSearchId"`
	}
	if err := json.Unmarshal([]byte(r.Body), &started); err != nil {
		t.Fatal(err)
	}
	return started.ManualSearchID
}

func (w *manualWorld) sheet(t *testing.T, work string) manualSheet {
	t.Helper()
	return *decode(t, "getManualSearch", w.a.Admin("home", http.MethodGet, "/api/v1/works/"+work+"/manual-search", ""), &manualSheet{})
}

// settled reads work's Manual search until every Indexer has responded.
func (w *manualWorld) settled(t *testing.T, work string) manualSheet {
	t.Helper()
	var v manualSheet
	worldtest.Until(t, "every Indexer responds to the Manual search", func() (bool, error) {
		v = w.sheet(t, work)
		return !v.Searching, nil
	})
	return v
}

func (w *manualWorld) offers(t *testing.T, work string) []manualOffer {
	t.Helper()
	var list struct {
		Releases []manualOffer `json:"releases"`
	}
	decode(t, "listManualSearchReleases", w.a.Admin("home", http.MethodGet, "/api/v1/works/"+work+"/manual-search/releases", ""), &list)
	return list.Releases
}

func (w *manualWorld) offer(t *testing.T, work, title string) manualOffer {
	t.Helper()
	offers := w.offers(t, work)
	i := slices.IndexFunc(offers, func(o manualOffer) bool { return o.Release.Title == title })
	if i < 0 {
		t.Fatalf("the Manual search lists no %s", title)
	}
	return offers[i]
}

// grabManual grabs each pick from Manual search id, unblock among them
// with unblock, and returns their Grabs.
func (w *manualWorld) grabManual(t *testing.T, work, id string, unblock bool, picks ...manualPick) []string {
	t.Helper()
	var sent []map[string]any
	for _, p := range picks {
		pick := map[string]any{"releaseId": p.Release.ID, "cells": p.Cells, "lifts": append([]liftKey{}, p.Lifts...), "unblock": unblock}
		if p.Edition != nil {
			pick["edition"] = p.Edition.Edition
		}
		sent = append(sent, pick)
	}
	body, err := json.Marshal(map[string]any{"manualSearchId": id, "picks": sent})
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Results []struct {
			GrabID  *string         `json:"grabId"`
			Problem json.RawMessage `json:"problem"`
		} `json:"results"`
	}
	decode(t, "grabFromManualSearch", w.a.Admin("home", http.MethodPost, "/api/v1/works/"+work+"/manual-search/grabs", string(body)), &out)
	if len(out.Results) != len(picks) {
		t.Fatalf("grabFromManualSearch answered %d picks with %d results", len(picks), len(out.Results))
	}
	var grabs []string
	for i, p := range picks {
		r := out.Results[i]
		if r.GrabID == nil {
			t.Fatalf("grabFromManualSearch refused %s: %s", p.Release.Title, r.Problem)
		}
		grabs = append(grabs, *r.GrabID)
	}
	return grabs
}

func (w *manualWorld) grabRecord(t *testing.T, id string) manualGrab {
	t.Helper()
	return *decode(t, "getGrab", w.a.Admin("home", http.MethodGet, "/api/v1/grabs/"+id, ""), &manualGrab{})
}

// finished reads Grab id until it is imported or failed.
func (w *manualWorld) finished(t *testing.T, id string) manualGrab {
	t.Helper()
	var g manualGrab
	worldtest.Until(t, "Grab "+id+" is imported or failed", func() (bool, error) {
		g = w.grabRecord(t, id)
		return g.Stage == "imported" || g.Stage == "failed", nil
	})
	return g
}

func (w *manualWorld) imported(t *testing.T, id string) manualGrab {
	t.Helper()
	g := w.finished(t, id)
	if g.Stage != "imported" {
		t.Fatalf("Grab %s of %s ended %s with %+v after %v, want imported", id, g.ReleaseTitle, g.Stage, g.Failure, g.stages())
	}
	return g
}

// rowImported waits for grab, the Grab of pick from Manual search id, to
// import, placing a file for each of pick's cells with none left waiting.
func (w *manualWorld) rowImported(t *testing.T, id, grab string, pick manualPick) {
	t.Helper()
	g := w.imported(t, grab)
	if g.ReleaseTitle != pick.Release.Title || g.Source != "manual" || g.ManualSearch == nil || *g.ManualSearch != id ||
		g.GrabbedBy == nil || *g.GrabbedBy != "the API key" || len(g.Overrides) != 0 {
		t.Fatalf("the Grab %s is %+v, want %s from Manual search %s through the API key with no overrides", grab, g, pick.Release.Title, id)
	}
	if g.Import == nil || g.Import.Placed != len(pick.Cells) || g.Import.Waiting != 0 {
		t.Fatalf("the Grab of %s imported %+v, want its %d cells placed and no file waiting", pick.Release.Title, g.Import, len(pick.Cells))
	}
}

func (w *manualWorld) blocklist(t *testing.T) []blockRow {
	t.Helper()
	var page struct {
		Entries []blockRow `json:"entries"`
		Total   int        `json:"total"`
	}
	decode(t, "listBlocklist", w.a.Admin("home", http.MethodGet, "/api/v1/blocklist", ""), &page)
	if page.Total != len(page.Entries) {
		t.Fatalf("listBlocklist lists %d of %d entries, want them all on one page", len(page.Entries), page.Total)
	}
	return page.Entries
}

func (w *manualWorld) item(t *testing.T, work, key string) string {
	t.Helper()
	tr := decode(t, "getWork", w.a.Admin("home", http.MethodGet, "/api/v1/works/"+work, ""), &tree{})
	i := slices.IndexFunc(tr.Items, func(n treeNode) bool { return n.Key == key })
	if i < 0 {
		t.Fatalf("Work %s has no Item %s", work, key)
	}
	return tr.Items[i].ID
}

func (w *manualWorld) cellStates(t *testing.T) map[string]string {
	t.Helper()
	var list struct {
		Cells []struct {
			TargetName string `json:"targetName"`
			State      string `json:"state"`
		} `json:"cells"`
		NeedsYou []json.RawMessage `json:"needsYou"`
	}
	decode(t, "getWorkCells", w.a.Admin("home", http.MethodGet, "/api/v1/works/"+w.work+"/cells", ""), &list)
	out := map[string]string{}
	for _, c := range list.Cells {
		out[c.TargetName] = c.State
	}
	return out
}

var respondedLine = regexp.MustCompile(`^Searching 4 Indexers\. [123] (has|have) responded\.$`)

// TestManualSearchListsEachIndexersReleasesAsItAnswers opens Manual
// search on The Quiet Orchard. Heron answers 20 seconds late, so the
// first answers' Releases are readable while it is still searching, and
// each answer is an event on the stream. Tamarack, off for automatic
// search, is asked too.
func TestManualSearchListsEachIndexersReleasesAsItAnswers(t *testing.T) {
	t.Parallel()
	w := launchManual(t)
	stream := worldtest.NewMessages(t, w.m, world.AdminIdentity)
	id := w.open(t, w.work, "")
	changed := regexp.MustCompile(`^\{"searchId":"[0-9a-f-]{36}","workId":"` + w.work + `","manualSearchId":"` + id + `"\}$`)
	next := func() manualSheet {
		t.Helper()
		if data := stream.Next("grabs.search-changed"); !changed.MatchString(data) {
			t.Fatalf("grabs.search-changed carried %s, want a run of Manual search %s", data, id)
		}
		return w.sheet(t, w.work)
	}

	early := next()
	for !slices.ContainsFunc(early.Indexers, func(l manualLine) bool { return l.Done }) {
		early = next()
	}
	if heron := early.line(t, "Heron"); !early.Searching || heron.Done || heron.Sentence != "Searching" || !respondedLine.MatchString(early.Line) {
		t.Fatalf("after the first answer the Manual search reads %q, searching %t, Heron %+v; want Heron still searching", early.Line, early.Searching, heron)
	}
	if slices.ContainsFunc(w.offers(t, w.work), func(o manualOffer) bool { return o.Release.Title == orchardCMRG }) {
		t.Fatalf("the Manual search lists Heron's %s before Heron answered", orchardCMRG)
	}

	done := early
	for done.Searching {
		done = next()
	}
	if done.Line != "SlipStream searched 4 Indexers." {
		t.Fatalf("the finished Manual search reads %q", done.Line)
	}
	for name, want := range map[string]manualLine{
		"Fixture Site": {Sentence: "Found 13 Releases", Done: true, Automatic: true},
		"Heron":        {Sentence: "Found 2 Releases", Done: true, Automatic: true},
		"Tamarack":     {Sentence: "Found 1 Release", Done: true},
		"Cinder":       {Sentence: "Found 1 Release", Done: true, Automatic: true},
	} {
		got := done.line(t, name)
		want.IndexerID, want.Name = got.IndexerID, name
		if got != want {
			t.Errorf("%s's line is %+v, want %+v", name, got, want)
		}
	}
	cmrg := w.offer(t, w.work, orchardCMRG)
	if !slices.Equal(cmrg.Release.Indexers, []string{"Heron"}) || !cmrg.Grab.Grabbable {
		t.Fatalf("Heron's late Release is %+v, want it listed from Heron and grabbable", cmrg)
	}
	if copies := w.offer(t, w.work, release); !slices.Equal(copies.Release.Indexers, []string{"Fixture Site", "Heron"}) {
		t.Fatalf("%s is listed from %v, want once from Fixture Site and Heron", release, copies.Release.Indexers)
	}
}

// TestTheManualPlanLeadsWithWhatAutomaticSearchGrabs reads the plan's
// first row, Tamarack's Bluray, then turns automatic search on for
// Tamarack and searches automatically: the search grabs the same Release.
func TestTheManualPlanLeadsWithWhatAutomaticSearchGrabs(t *testing.T) {
	t.Parallel()
	w := launchManual(t)
	w.open(t, w.work, "")
	v := w.settled(t, w.work)
	if len(v.Plan.Rows) == 0 || v.Plan.Rows[0].Pick == nil {
		t.Fatalf("the plan has no first row to grab: %+v", v.Plan)
	}
	first := v.Plan.Rows[0].Pick.Release
	if first.Title != orchardBluray {
		t.Fatalf("the plan's first row grabs %s, want %s", first.Title, orchardBluray)
	}
	w.patchIndexer(t, v.line(t, "Tamarack").IndexerID, `{"automaticSearch":true}`)

	s := w.searched(t, w.search(t))
	if s.Outcome == nil || *s.Outcome != "grabbed" || s.GrabID == nil {
		t.Fatalf("the automatic search ended %+v, want a Grab", s)
	}
	if g := w.grabRecord(t, *s.GrabID); g.ReleaseTitle != first.Title || g.Source != "search" || g.GrabbedBy != nil {
		t.Fatalf("automatic search grabbed %s from %s by %v, want %s, the plan's first row", g.ReleaseTitle, g.Source, g.GrabbedBy, first.Title)
	}
}

// TestGrabbingTheManualPlanImportsEachRow opens Manual search on E05 to
// E09 of The Lantern Keepers and grabs every row of the plan in one press.
// The season pack fills all five, and E10 too, with one Release, so it
// beats each episode's own and is the series' only row. Each Grab imports
// its row's cells and records the Manual search it came from.
func TestGrabbingTheManualPlanImportsEachRow(t *testing.T) {
	t.Parallel()
	w := launchManual(t)
	var items []string
	for _, key := range []string{"100125", "100126", "100127", "100128", "100129"} {
		items = append(items, `"`+w.item(t, w.series, key)+`"`)
	}
	id := w.open(t, w.series, `{"itemIds":[`+strings.Join(items, ",")+`]}`)
	v := w.settled(t, w.series)
	var rows []string
	for _, r := range v.Plan.Rows {
		if r.Pick != nil {
			rows = append(rows, r.Pick.Release.Title+": "+r.Outcome)
		} else {
			rows = append(rows, r.Missing)
		}
	}
	want := []string{keepersPack + ": It gets you 1080p for E05 to E10. E01 to E04 keep their files."}
	if v.Plan.Headline != "1 Grab gets you everything." || !slices.Equal(rows, want) {
		t.Fatalf("the plan reads %q with rows\n%s\nwant\n%s", v.Plan.Headline, strings.Join(rows, "\n"), strings.Join(want, "\n"))
	}
	picks := v.picks()
	for i, grab := range w.grabManual(t, w.series, id, false, picks...) {
		w.rowImported(t, id, grab, picks[i])
	}

	id = w.open(t, w.work, "")
	v = w.settled(t, w.work)
	rows = nil
	for _, r := range v.Plan.Rows {
		if r.Pick != nil {
			rows = append(rows, r.Pick.Release.Title+": "+r.Outcome)
		}
	}
	want = []string{
		orchardBluray + ": It gets you 1080p (standard) for The Quiet Orchard.",
		"The.Quiet.Orchard.2021.1080p.WEB-DL.H264-FKW: It gets you HD (standard) for The Quiet Orchard as an upgrade.",
	}
	if v.Plan.Headline != "2 Grabs get you everything." || !slices.Equal(rows, want) {
		t.Fatalf("the Orchard's plan reads %q with rows\n%s\nwant\n%s", v.Plan.Headline, strings.Join(rows, "\n"), strings.Join(want, "\n"))
	}
	picks = v.picks()
	for i, grab := range w.grabManual(t, w.work, id, false, picks...) {
		if g := w.imported(t, grab); g.ReleaseTitle != picks[i].Release.Title {
			t.Fatalf("the Orchard's Grab of row %d imported %s, want %s", i+1, g.ReleaseTitle, picks[i].Release.Title)
		}
	}
}

// TestAManualGrabPastASoftRefusalRecordsItsOverride grabs a Release
// past each soft refusal it has at every cell it fills: a name giving
// the film's title with another year, a score under the minimum, and a
// size under and over what each Target allows. It imports, and its Grab
// says what it was grabbed past.
func TestAManualGrabPastASoftRefusalRecordsItsOverride(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		title string
		cells int
		lifts []string
		want  []string
	}{
		{orchard1987, 2, []string{"wrong_title"}, []string{"It was grabbed through the API key although its name gives The Quiet Orchard (1987), which isn't this Work."}},
		{orchardLQ, 2, []string{"score_below_minimum", "score_below_minimum"}, []string{
			"It was grabbed through the API key although score -5 is below the minimum of 0 at 1080p.",
			"It was grabbed through the API key although score -5 is below the minimum of 0 at HD.",
		}},
		{orchardTiny, 2, []string{"size_below", "size_below"}, []string{
			"It was grabbed through the API key although it's smaller than 1080p allows.",
			"It was grabbed through the API key although it's smaller than HD allows.",
		}},
		{orchardBloat, 2, []string{"size_above", "size_above"}, []string{
			"It was grabbed through the API key although it's larger than 1080p allows.",
			"It was grabbed through the API key although it's larger than HD allows.",
		}},
	} {
		t.Run(c.title, func(t *testing.T) {
			t.Parallel()
			w := launchManual(t)
			id := w.open(t, w.work, "")
			w.settled(t, w.work)
			o := w.offer(t, w.work, c.title)
			var codes []string
			for _, l := range o.Grab.Lifts {
				codes = append(codes, l.Code)
			}
			if !o.Grab.Grabbable || len(o.Grab.Cells) != c.cells || !slices.Equal(codes, c.lifts) {
				t.Fatalf("%s is grabbable %t for %d cells past %q, want grabbable for %d past %q", c.title, o.Grab.Grabbable, len(o.Grab.Cells), codes, c.cells, c.lifts)
			}
			g := w.imported(t, w.grabManual(t, w.work, id, false, o.pick())[0])
			if !slices.Equal(g.Overrides, c.want) {
				t.Fatalf("the Grab's overrides are %q, want %q", g.Overrides, c.want)
			}
		})
	}
}

// TestCancelAlsoBlocklistingKeepsTheReleaseOutUntilRemovedAndGrabbed
// holds the swarm, grabs the plan's WEB-DL and cancels it while it
// downloads, also blocklisting it. Its cell reads missing, the Blocklist
// lists it, and a new Manual search shows it blocklisted. "Remove from
// Blocklist and grab" takes it off the Blocklist, and the new Grab
// imports once the swarm is let go.
func TestCancelAlsoBlocklistingKeepsTheReleaseOutUntilRemovedAndGrabbed(t *testing.T) {
	t.Parallel()
	w := launchManual(t)
	w.setFault(t, fault.Rule{ID: "swarm-hold", Target: "seeders", Origin: "home"}, "throttle=16KiB/s")
	id := w.open(t, w.work, "")
	w.settled(t, w.work)
	webdl := w.offer(t, w.work, release)
	grab := w.grabManual(t, w.work, id, false, webdl.pick())[0]
	var held manualGrab
	worldtest.Until(t, "the held Grab downloads with its info hash known", func() (bool, error) {
		held = w.grabRecord(t, grab)
		return held.Stage == "downloading" && held.Actions.Blocklistable, nil
	})
	if r := w.a.Admin("home", http.MethodPost, "/api/v1/grabs/"+grab+"/cancel", `{"blocklist":true}`); r.Status != http.StatusNoContent {
		t.Fatalf("cancelGrab = %d %s, want 204", r.Status, r.Body)
	}
	g := w.finished(t, grab)
	if g.Stage != "failed" || g.Failure == nil || g.Failure.Code != "canceled" {
		t.Fatalf("the canceled Grab ended %s with %+v, want failed canceled", g.Stage, g.Failure)
	}
	if got := w.cellStates(t); got["1080p"] != "missing" {
		t.Fatalf("after the cancel the cells are %v, want 1080p missing", got)
	}
	entries := w.blocklist(t)
	if len(entries) != 1 || entries[0].InfoHash != held.InfoHash || entries[0].Title != release || entries[0].Cause != "by_hand" ||
		entries[0].Work == nil || entries[0].Work.Title != "The Quiet Orchard" {
		t.Fatalf("the Blocklist lists %+v, want %s alone, blocklisted by hand for The Quiet Orchard", entries, release)
	}

	again := w.open(t, w.work, `{"again":true}`)
	w.settled(t, w.work)
	blocked := w.offer(t, w.work, release)
	if b := blocked.Release.Blocked; b == nil || !strings.HasPrefix(*b, "It's been on the Blocklist since ") ||
		!strings.HasSuffix(*b, " because it was blocklisted through the API key when its Grab was canceled.") {
		t.Fatalf("the new Manual search shows %s blocked %v, want blocklisted through the API key at the cancel", release, b)
	}

	if _, err := w.run.ClearFault(worldtest.Context(t), "swarm-hold"); err != nil {
		t.Fatal(err)
	}
	g = w.imported(t, w.grabManual(t, w.work, again, true, blocked.pick())[0])
	if want := []string{"It was removed from the Blocklist through the API key to grab it."}; !slices.Equal(g.Overrides, want) {
		t.Fatalf("the Grab's overrides are %q, want %q", g.Overrides, want)
	}
	if entries := w.blocklist(t); len(entries) != 0 {
		t.Fatalf("the Blocklist lists %+v after Remove from Blocklist and grab, want nothing", entries)
	}
}

// TestAReleaseFaultBlocklistsTheReleaseAndFailsItsGrab grabs S02E10 of
// The Lantern Keepers from a Manual search of that episode. The tracker
// reports its torrent unregistered while it downloads, twice a minute
// apart, so the Grab fails release_fault and the Blocklist lists the
// Release until it is removed.
func TestAReleaseFaultBlocklistsTheReleaseAndFailsItsGrab(t *testing.T) {
	t.Parallel()
	w := launchManual(t)
	if r := w.a.Admin("home", http.MethodPatch, "/api/v1/download-clients/thresholds", `{"unregisteredGap":60}`); r.Status != http.StatusOK {
		t.Fatalf("setDownloadClientThresholds = %d %s, want 200", r.Status, r.Body)
	}
	scope := `{"itemIds":["` + w.item(t, w.series, "100130") + `"]}`
	id := w.open(t, w.series, scope)
	picks := w.settled(t, w.series).picks()
	if len(picks) != 1 || picks[0].Release.Title != keepersE10 {
		t.Fatalf("the plan for S02E10 grabs %+v, want %s alone", picks, keepersE10)
	}
	g := w.finished(t, w.grabManual(t, w.series, id, false, picks[0])[0])
	if g.Stage != "failed" || g.Failure == nil || g.Failure.Code != "release_fault" || !slices.Contains(g.stages(), "downloading") {
		t.Fatalf("the Grab ended %s with %+v after %v, want failed release_fault after downloading", g.Stage, g.Failure, g.stages())
	}
	entries := w.blocklist(t)
	if len(entries) != 1 || entries[0].InfoHash != g.InfoHash || entries[0].Title != keepersE10 || entries[0].Cause != "unregistered" ||
		entries[0].Work == nil || entries[0].Work.Title != "The Lantern Keepers" {
		t.Fatalf("the Blocklist lists %+v, want %s alone, unregistered, for The Lantern Keepers", entries, keepersE10)
	}

	again := w.open(t, w.series, `{"again":true,`+scope[1:])
	w.settled(t, w.series)
	o := w.offer(t, w.series, keepersE10)
	if o.Release.Blocked == nil || !strings.HasPrefix(*o.Release.Blocked, "It's been on the Blocklist since ") {
		t.Fatalf("the new Manual search shows %s blocked %v, want since when and why", keepersE10, o.Release.Blocked)
	}
	worldtest.Expect(t, "grabFromManualSearch without unblock", w.a.Admin("home", http.MethodPost, "/api/v1/works/"+w.series+"/manual-search/grabs",
		`{"manualSearchId":"`+again+`","picks":[{"releaseId":"`+o.Release.ID+`","cells":["`+strings.Join(o.Grab.Cells, `","`)+`"],"lifts":[]}]}`), worldtest.Reply{Status: 200,
		Body: `{"results":[{"grabId":null,"problem":{"code":"pick_changed","detail":"This Release is on the Blocklist now. Remove it from the Blocklist to grab it."}}]}` + "\n"})
	if r := w.a.Admin("home", http.MethodPost, "/api/v1/blocklist/remove", `{"infoHashes":["`+g.InfoHash+`"]}`); r.Status != http.StatusNoContent {
		t.Fatalf("removeFromBlocklist = %d %s, want 204", r.Status, r.Body)
	}
	if entries := w.blocklist(t); len(entries) != 0 {
		t.Fatalf("the Blocklist lists %+v after the remove, want nothing", entries)
	}
}
