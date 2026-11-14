package plexkind_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fake/faketest"
	fakeplex "github.com/jatassi/slipstream/fakes/plex"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/mediaserverlink/internal/plexkind"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/storage"
)

const (
	machine = "fake-plex-den"
	token   = "server-token"
)

// den is fake-plex on Windows paths with sections Films (1), Shows (2)
// and Fresh (4), the first two scanned with the files placed before the
// first call, and a Plex kind connection to it.
type den struct {
	t     *testing.T
	f     faketest.Served
	media string
	root  link.ServerPath
	conn  link.Conn
}

func newDen(t *testing.T, seed ...string) *den {
	t.Helper()
	d := &den{t: t, media: t.TempDir(), root: mustPath(t, `C:\Media`)}
	for _, rel := range seed {
		d.place(rel)
	}
	for _, dir := range []string{"Films", "Shows", "Fresh"} {
		if err := os.MkdirAll(filepath.Join(d.media, dir), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	srv := fakeplex.Server{
		Name: "Den", Version: "1.43.5.11029", Identifier: machine, Tokens: []string{token}, Owner: "jackson", Root: d.root.String(),
		Sections: []fakeplex.Section{
			{Key: "1", Title: "Films", Folders: []string{d.at("Films").String()}, Scanned: true},
			{Key: "2", Title: "Shows", Type: "show", Folders: []string{d.at("Shows").String()}, Scanned: true},
			{Key: "4", Title: "Fresh", Folders: []string{d.at("Fresh").String()}},
		},
	}
	d.f = faketest.InMemory(t, fakeplex.Playing(srv), scenario.Scenario{Seed: "plex"}, faketest.WithMedia(d.media), faketest.As("Den"))
	d.conn = d.dial(token)
	return d
}

func (d *den) dial(tok string) link.Conn {
	return plexkind.Kind.Dial(link.Dialed{
		Server:  link.ServerRef{ID: link.ServerID{UUID: storage.NewUUID()}, Kind: "plex", Identity: machine},
		Address: d.f.Address.URL(""), Credential: link.Credential{Token: tok}, DeviceID: "slipstream-test",
		HTTP: d.f.Client, Timings: link.DefaultTimings(),
	})
}

// at is rel under the server's root, as Plex names it.
func (d *den) at(rel string) link.ServerPath { return d.root.Join(mustRel(d.t, rel)) }

func (d *den) local(rel string) string { return filepath.Join(d.media, filepath.FromSlash(rel)) }

func (d *den) place(rel string) {
	d.t.Helper()
	if err := os.MkdirAll(filepath.Dir(d.local(rel)), 0o700); err != nil {
		d.t.Fatal(err)
	}
	if err := os.WriteFile(d.local(rel), []byte(rel), 0o600); err != nil {
		d.t.Fatal(err)
	}
}

func (d *den) move(from, to string) {
	d.t.Helper()
	if err := os.MkdirAll(filepath.Dir(d.local(to)), 0o700); err != nil {
		d.t.Fatal(err)
	}
	if err := os.Rename(d.local(from), d.local(to)); err != nil {
		d.t.Fatal(err)
	}
}

func (d *den) remove(rel string) {
	d.t.Helper()
	if err := os.Remove(d.local(rel)); err != nil {
		d.t.Fatal(err)
	}
}

// change is a change in Films.
func change(kind link.ChangeKind, to, from link.ServerPath) link.JobChange {
	return link.JobChange{ID: link.ChangeID{UUID: storage.NewUUID()}, Kind: kind, Library: "1", Path: to, From: from}
}

// drive advances j until it is done, waiting out each wait, and answers
// the steps it went through and the last answer, with every outcome and
// learned entry gathered.
func (d *den) drive(j link.Job) ([]link.Step, link.Advanced) {
	d.t.Helper()
	var steps []link.Step
	var all link.Advanced
	for range 1000 {
		a, err := d.conn.Advance(d.t.Context(), j)
		if err != nil {
			d.t.Fatalf("Advance %T at %s: %v", j.Work, j.Step, err)
		}
		all.Outcomes = append(all.Outcomes, a.Outcomes...)
		all.Learned = append(all.Learned, a.Learned...)
		if a.Job.Step != j.Step {
			steps = append(steps, a.Job.Step)
		}
		if a.Done {
			all.Job, all.Done = a.Job, true
			return steps, all
		}
		time.Sleep(time.Until(a.Wait))
		j = a.Job
	}
	d.t.Fatalf("%T never finished", j.Work)
	return nil, link.Advanced{}
}

// refreshes are the refreshes the fake took, as lines.
func (d *den) refreshes() []string {
	d.t.Helper()
	var out []string
	for _, r := range d.f.Records(d.t) {
		if r.Kind != fakeplex.Refreshes.Kind {
			continue
		}
		var x fakeplex.Refresh
		if err := json.Unmarshal(r.Data, &x); err != nil {
			d.t.Fatal(err)
		}
		switch {
		case x.Item != "":
			out = append(out, "item "+x.Item)
		case x.Path != "":
			out = append(out, "folder "+x.Path)
		default:
			out = append(out, "section "+x.Section)
		}
	}
	return out
}

// item is the item at rel, read through a Lookup Job.
func (d *den) item(rel string) (link.ItemKey, bool) {
	d.t.Helper()
	_, a := d.drive(link.Job{Work: link.Lookup{Paths: []link.ServerPath{d.at(rel)}}, Step: link.StepNone})
	if len(a.Learned) != 1 {
		d.t.Fatalf("Lookup of %s learned %v", rel, a.Learned)
	}
	return a.Learned[0].Item, !a.Learned[0].Gone
}

// watched scrobbles the item at rel and tags it with a collection, as the
// recorded run seeded its items. SlipStream itself writes neither.
func (d *den) watched(rel string) {
	d.t.Helper()
	k, ok := d.item(rel)
	if !ok {
		d.t.Fatalf("no item at %s", rel)
	}
	d.send(http.MethodGet, "/:/scrobble", url.Values{"key": {k.Item}, "identifier": {"com.plexapp.plugins.library"}})
	d.send(http.MethodPut, "/library/metadata/"+k.Item, url.Values{"collection[0].tag.tag": {"Restored"}})
}

func (d *den) send(method, path string, q url.Values) {
	d.t.Helper()
	req, err := http.NewRequestWithContext(d.t.Context(), method, d.f.Address.URL(path)+"?"+q.Encode(), http.NoBody)
	if err != nil {
		d.t.Fatal(err)
	}
	req.Header.Set("X-Plex-Token", token)
	resp, err := d.f.Client.Do(req)
	if err != nil {
		d.t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		d.t.Fatalf("%s %s: status %d", method, path, resp.StatusCode)
	}
}

func outcomes(a link.Advanced) []string {
	out := make([]string, len(a.Outcomes))
	for i, o := range a.Outcomes {
		out[i] = fmt.Sprintf("%s %v", o.Result, o.Lost)
	}
	return out
}

func check[T any](t *testing.T, what string, got, want T) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s =\n%v\nwant\n%v", what, got, want)
	}
}

func TestFortyEpisodesOfASeasonAreOneFolderRefresh(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := newDen(t, "Shows/Show/Season 01/Show - S01E01.mkv")
		p := newPlan(t)
		p.due.Mappings[0].Folder = d.at("Shows")
		for e := 2; e <= 41; e++ {
			rel := fmt.Sprintf("Show/Season 01/Show - S01E%02d.mkv", e)
			d.place("Shows/" + rel)
			p.row("e", p.shows, link.ChangeAdded, "", rel)
		}
		jobs := plexkind.Kind.Plan(p.due)
		if len(jobs) != 1 {
			t.Fatalf("Plan = %d jobs, want one Refresh", len(jobs))
		}
		steps, a := d.drive(jobs[0])
		check(t, "steps", steps, []link.Step{link.StepScanned})
		check(t, "refreshes", d.refreshes(), []string{`folder C:\Media\Shows\Show\Season 01`})
		if len(a.Outcomes) != 40 || slices.ContainsFunc(a.Outcomes, func(o link.Outcome) bool { return o.Result != link.ResultAccepted }) {
			t.Errorf("outcomes = %v, want 40 accepted", a.Outcomes)
		}
		if _, ok := d.item("Shows/Show/Season 01/Show - S01E41.mkv"); !ok {
			t.Error("the last episode has no item after the refresh")
		}
	})
}

func TestElevenFoldersOfASectionAreOneSectionRefresh(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := newDen(t, "Shows/Show/Season 01/Show - S01E01.mkv")
		p := newPlan(t)
		p.due.Mappings[0].Folder = d.at("Shows")
		for s := 2; s <= 12; s++ {
			rel := fmt.Sprintf("Show/Season %02d/Show - S%02dE01.mkv", s, s)
			d.place("Shows/" + rel)
			p.row("s", p.shows, link.ChangeAdded, "", rel)
		}
		jobs := plexkind.Kind.Plan(p.due)
		if len(jobs) != 1 {
			t.Fatalf("Plan = %d jobs, want one Rescan", len(jobs))
		}
		_, a := d.drive(jobs[0])
		check(t, "refreshes", d.refreshes(), []string{"section 2"})
		if len(a.Outcomes) != 11 {
			t.Errorf("outcomes = %v, want 11 accepted", a.Outcomes)
		}
	})
}

func TestARenameInItsFolderKeepsWatchDataWithOneRefresh(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := newDen(t, "Films/Metropolis (1927)/Metropolis (1927).mkv")
		d.watched("Films/Metropolis (1927)/Metropolis (1927).mkv")
		d.move("Films/Metropolis (1927)/Metropolis (1927).mkv", "Films/Metropolis (1927)/Metropolis (1927) Remastered.mkv")
		ch := change(link.ChangeMoved, d.at("Films/Metropolis (1927)/Metropolis (1927) Remastered.mkv"), d.at("Films/Metropolis (1927)/Metropolis (1927).mkv"))
		steps, a := d.drive(link.Job{Work: link.Carry{Changes: []link.JobChange{ch}}, Step: link.StepNone})
		check(t, "steps", steps, []link.Step{link.StepSnapshotted, link.StepScanned, link.StepConfirmed, link.StepSourceScanned})
		check(t, "refreshes", d.refreshes(), []string{`folder C:\Media\Films\Metropolis (1927)`})
		check(t, "outcomes", outcomes(a), []string{"accepted []"})
	})
}

func TestAMoveRefreshesTheDestinationThenTheSourceAndKeepsWatchData(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := newDen(t, "Films/Metropolis (1927)/Metropolis (1927).mkv")
		d.watched("Films/Metropolis (1927)/Metropolis (1927).mkv")
		old, _ := d.item("Films/Metropolis (1927)/Metropolis (1927).mkv")
		d.move("Films/Metropolis (1927)/Metropolis (1927).mkv", "Films/Metropolis/Metropolis (1927).mkv")
		ch := change(link.ChangeMoved, d.at("Films/Metropolis/Metropolis (1927).mkv"), d.at("Films/Metropolis (1927)/Metropolis (1927).mkv"))
		_, a := d.drive(link.Job{Work: link.Carry{Changes: []link.JobChange{ch}}, Step: link.StepNone})
		check(t, "refreshes", d.refreshes(), []string{`folder C:\Media\Films\Metropolis`, `folder C:\Media\Films\Metropolis (1927)`})
		check(t, "outcomes", outcomes(a), []string{"accepted []"})
		moved, _ := a.Job.Work.(link.Carry)
		snap := moved.Changes[0].Snapshot
		if snap == nil || snap.Item != old || len(snap.Collections) != 1 || snap.Users[0].PlayCount != 1 {
			t.Errorf("snapshot = %+v, want item %v played once in one collection", snap, old)
		}
		if len(a.Learned) != 1 || a.Learned[0].Path.Key() != ch.Path.Key() {
			t.Errorf("learned = %v, want the item at the new path", a.Learned)
		}
	})
}

func TestACarryWhoseNewItemNeverAppearsReportsWhatItLost(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := newDen(t, "Films/Metropolis (1927)/Metropolis (1927).mkv")
		d.watched("Films/Metropolis (1927)/Metropolis (1927).mkv")
		d.remove("Films/Metropolis (1927)/Metropolis (1927).mkv")
		ch := change(link.ChangeMoved, d.at("Films/Metropolis/Metropolis (1927).mkv"), d.at("Films/Metropolis (1927)/Metropolis (1927).mkv"))
		start := time.Now()
		_, a := d.drive(link.Job{Work: link.Carry{Changes: []link.JobChange{ch}}, Step: link.StepNone})
		if waited := time.Since(start); waited < link.DefaultTimings().MovedConfirm {
			t.Errorf("the carry ended after %s, before Timings.MovedConfirm", waited)
		}
		check(t, "outcomes", outcomes(a), []string{"accepted [watch_state collections]"})
	})
}

func TestACarryWithNoOldItemIsConverted(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := newDen(t, "Films/Metropolis (1927)/Metropolis (1927).mkv")
		d.place("Films/Nosferatu (1922)/Nosferatu (1922).mkv")
		ch := change(link.ChangeMoved, d.at("Films/Nosferatu (1922)/Nosferatu (1922).mkv"), d.at("Downloads/Nosferatu (1922).mkv"))
		_, a := d.drive(link.Job{Work: link.Carry{Changes: []link.JobChange{ch}}, Step: link.StepNone})
		check(t, "outcomes", outcomes(a), []string{"converted []"})
		if _, ok := d.item("Films/Nosferatu (1922)/Nosferatu (1922).mkv"); !ok {
			t.Error("the converted change's file has no item")
		}
	})
}

func TestARemovedSidecarRefreshesItsItem(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := newDen(t, "Films/Metropolis (1927)/Metropolis (1927).mkv", "Films/Metropolis (1927)/Metropolis (1927).en.srt")
		k, _ := d.item("Films/Metropolis (1927)/Metropolis (1927).mkv")
		d.remove("Films/Metropolis (1927)/Metropolis (1927).en.srt")
		ch := change(link.ChangeSidecarRemoved, d.at("Films/Metropolis (1927)/Metropolis (1927).en.srt"), link.ServerPath{})
		d.drive(link.Job{Work: link.Refresh{Changes: []link.JobChange{ch}}, Step: link.StepNone})
		check(t, "refreshes", d.refreshes(), []string{"item " + k.Item})
	})
}

func TestFreshScansTheSectionOnceAndWaitsForItsEnd(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := newDen(t, "Films/Metropolis (1927)/Metropolis (1927).mkv")
		d.place("Fresh/Nosferatu (1922)/Nosferatu (1922).mkv")
		_, a := d.drive(link.Job{Work: link.Fresh{Library: "4", Folder: d.at("Fresh")}, Step: link.StepNone})
		check(t, "refreshes", d.refreshes(), []string{"section 4"})
		if f := a.Job.Work.(link.Fresh); f.Requested.IsZero() {
			t.Errorf("the scan's end came back as %+v, want the request it answers", f)
		}
		d.drive(link.Job{Work: link.Fresh{Library: "4", Folder: d.at("Fresh")}, Step: link.StepNone})
		check(t, "refreshes after a second Fresh", d.refreshes(), []string{"section 4"})
	})
}

func TestFreshDecidesBeforeItRefreshes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := newDen(t, "Films/Metropolis (1927)/Metropolis (1927).mkv")
		d.place("Fresh/Nosferatu (1922)/Nosferatu (1922).mkv")
		a, err := d.conn.Advance(t.Context(), link.Job{Work: link.Fresh{Library: "4", Folder: d.at("Fresh")}, Step: link.StepNone})
		if err != nil {
			t.Fatal(err)
		}
		if f := a.Job.Work.(link.Fresh); a.Done || f.Requested.IsZero() || len(d.refreshes()) != 0 {
			t.Errorf("the first call = %+v with refreshes %v, want the decision and no refresh", a, d.refreshes())
		}
	})
}

// A retry of the call that sent the refresh, its answer lost, finds the
// section's scannedAt past Started and ends with no second refresh, on
// Plex's clock however far the caller's runs ahead.
func TestFreshRetriedAfterItsRefreshStartsNoSecond(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := newDen(t, "Films/Metropolis (1927)/Metropolis (1927).mkv")
		d.place("Fresh/Nosferatu (1922)/Nosferatu (1922).mkv")
		decided, err := d.conn.Advance(t.Context(), link.Job{Work: link.Fresh{Library: "4", Folder: d.at("Fresh")}, Step: link.StepNone})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := d.conn.Advance(t.Context(), decided.Job); err != nil {
			t.Fatal(err)
		}
		time.Sleep(time.Minute)
		retry := decided.Job
		f := retry.Work.(link.Fresh)
		f.Requested = f.Requested.Add(time.Hour)
		retry.Work = f
		a, err := d.conn.Advance(t.Context(), retry)
		if err != nil {
			t.Fatal(err)
		}
		if !a.Done {
			t.Errorf("the retry = %+v, want the scan's end", a)
		}
		check(t, "refreshes", d.refreshes(), []string{"section 4"})
	})
}

func TestReachNeedsNoTokenAndCheckProvesIt(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := newDen(t, "Films/Metropolis (1927)/Metropolis (1927).mkv")
		id, err := d.dial("").Reach(t.Context())
		if err != nil || id.ID != machine || id.Version.String() != "1.43.5.11029" {
			t.Errorf("Reach with no token = %+v, %v", id, err)
		}
		if _, err := d.dial("not-the-token").Check(t.Context()); err == nil {
			t.Error("Check with another token passed")
		} else if p, ok := link.ServerProblem(err); !ok || p.Reason != link.ReasonRefused {
			t.Errorf("Check with another token = %v, want refused", err)
		}
		c, err := d.conn.Check(t.Context())
		if err != nil || c.Identity.Name != "Den" || len(c.Libraries) != 3 || c.Libraries[1].Folders[0].String() != `C:\Media\Shows` {
			t.Errorf("Check = %+v, %v", c, err)
		}
		other := plexkind.Kind.Dial(link.Dialed{
			Server: link.ServerRef{Identity: "another-plex"}, Address: d.f.Address.URL(""), HTTP: d.f.Client, Timings: link.DefaultTimings(),
		})
		if _, err := other.Reach(t.Context()); err == nil {
			t.Error("Reach of a saved server answered by another passed")
		} else if p, _ := link.ServerProblem(err); p.Reason != link.ReasonOtherServer || p.Found != machine {
			t.Errorf("Reach of another server = %v, want other_server found %s", err, machine)
		}
	})
}

func TestIndexPagesThroughEveryFile(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := newDen(t, "Shows/Show/Season 01/Show - S01E01.mkv", "Shows/Show/Season 01/Show - S01E02.mkv", "Shows/Show/Season 01/Show - S01E03.mkv")
		tm := link.DefaultTimings()
		tm.IndexPage = 2
		d.conn = plexkind.Kind.Dial(link.Dialed{
			Server: link.ServerRef{Identity: machine}, Address: d.f.Address.URL(""), Credential: link.Credential{Token: token},
			HTTP: d.f.Client, Timings: tm,
		})
		var paths, cursors []string
		w := link.Index{Library: "2"}
		for {
			a, err := d.conn.Advance(t.Context(), link.Job{Work: w, Step: link.StepNone})
			if err != nil {
				t.Fatal(err)
			}
			for _, e := range a.Learned {
				paths = append(paths, e.Path.String())
			}
			w, _ = a.Job.Work.(link.Index)
			cursors = append(cursors, w.Cursor)
			if a.Done != (w.Cursor == "") {
				t.Fatalf("a page with the next cursor %q answered done %t", w.Cursor, a.Done)
			}
			if a.Done {
				break
			}
		}
		check(t, "cursors", cursors, []string{"1:0", "1:2", ""})
		check(t, "paths", paths, []string{
			`C:\Media\Shows\Show\Season 01\Show - S01E01.mkv`, `C:\Media\Shows\Show\Season 01\Show - S01E02.mkv`, `C:\Media\Shows\Show\Season 01\Show - S01E03.mkv`,
		})
	})
}
