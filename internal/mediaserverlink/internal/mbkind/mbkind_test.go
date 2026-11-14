package mbkind_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/emby"
	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/jellyfin"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/mediaserverlink"
	"github.com/jatassi/slipstream/internal/mediaserverlink/internal/mbkind"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	wire "github.com/jatassi/slipstream/internal/providers/jellyfin"
	"github.com/jatassi/slipstream/internal/storage"
)

// product is one kind and the fake that plays its server.
type product struct {
	kind    link.Kind
	flavor  wire.Flavor
	spec    func(jellyfin.Server) fake.Spec
	reports string
	scans   string
}

var products = []product{
	{mbkind.Jellyfin, wire.Jellyfin, jellyfin.Playing, "jellyfin.report", "jellyfin.scan"},
	{mbkind.Emby, wire.Emby, emby.Playing, "emby.report", "emby.scan"},
}

const (
	serverID = "ddc0b5b7f2824180aae9f2f245dd00ca"
	apiKey   = "mbkind-key"
)

// world is one fake server with a Movies library scanned with
// Metropolis, and a Fresh library never scanned with content.
type world struct {
	t       *testing.T
	p       product
	f       faketest.Served
	checked link.Checked
	admin   *wire.Client
	adminID string
	guestID string
}

func each(t *testing.T, test func(t *testing.T, w *world)) {
	for _, p := range products {
		t.Run(string(p.kind.About().Name), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) { test(t, newWorld(t, p)) })
		})
	}
}

func newWorld(t *testing.T, p product) *world {
	media := t.TempDir()
	for _, dir := range []string{"Movies", "Fresh"} {
		if err := os.MkdirAll(filepath.Join(media, dir), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	w := &world{t: t, p: p}
	w.place("Movies/Metropolis (1927)/Metropolis (1927).mkv", media)
	srv := jellyfin.Server{
		Name: "Den", Version: map[bool]string{true: "12.4.0", false: "4.10.1.0"}[p.flavor == wire.Jellyfin], ID: serverID, Keys: []string{apiKey},
		Users: []jellyfin.User{{Name: "jackson", Password: "admin-password", Admin: true}, {Name: "guest", Password: "guest-password"}},
		Root:  "/media",
		Libraries: []jellyfin.Library{
			{Name: "Movies", Type: "movies", Folders: []string{"/media/Movies"}, Scanned: true},
			{Name: "Fresh", Type: "movies", Folders: []string{"/media/Fresh"}},
		},
	}
	w.f = faketest.InMemory(t, p.spec(srv), scenario.Scenario{Seed: "mbkind"}, faketest.WithMedia(media), faketest.As("Den"))
	var err error
	if w.checked, err = w.conn(apiKey).Check(t.Context()); err != nil {
		t.Fatal(err)
	}
	base, err := url.Parse(w.f.Address.URL(""))
	if err != nil {
		t.Fatal(err)
	}
	w.admin = wire.New(base, p.flavor, apiKey, wire.App{Client: "test", Device: "test", DeviceID: "test", Version: "1"}, w.f.Client)
	users, err := w.admin.Users(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, u := range users {
		if u.Policy.IsAdministrator {
			w.adminID = u.ID
		} else {
			w.guestID = u.ID
		}
	}
	return w
}

func (w *world) dialed(key string) link.Dialed {
	return link.Dialed{
		Server:  link.ServerRef{ID: link.ServerID{UUID: storage.NewUUID()}, Kind: w.p.kind.About().Name, Identity: serverID},
		Address: w.f.Address.URL(""), Credential: link.Credential{APIKey: key}, DeviceID: "mbkind-device",
		MonitorDelay: w.checked.MonitorDelay, HTTP: w.f.Client, Timings: link.DefaultTimings(),
	}
}

func (w *world) conn(key string) link.Conn { return w.p.kind.Dial(w.dialed(key)) }

func (w *world) place(rel string, media ...string) {
	dir := w.media()
	if len(media) > 0 {
		dir = media[0]
	}
	file := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		w.t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(rel), 0o600); err != nil {
		w.t.Fatal(err)
	}
}

func (w *world) move(from, to string) {
	dst := filepath.Join(w.media(), filepath.FromSlash(to))
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		w.t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(w.media(), filepath.FromSlash(from)), dst); err != nil {
		w.t.Fatal(err)
	}
}

func (w *world) media() string { return w.f.Media }

func (w *world) library(name string) link.Library {
	i := slices.IndexFunc(w.checked.Libraries, func(l link.Library) bool { return l.Name == name })
	if i < 0 {
		w.t.Fatalf("no library %q in %v", name, w.checked.Libraries)
	}
	return w.checked.Libraries[i]
}

func (w *world) at(rel string) link.ServerPath { return path(w.t, "/media/"+rel) }

func (w *world) change(kind link.ChangeKind, from, to string) link.JobChange {
	ch := link.JobChange{ID: link.ChangeID{UUID: storage.NewUUID()}, Kind: kind, Library: w.library("Movies").Key, Path: w.at(to)}
	if from != "" {
		ch.From = w.at(from)
	}
	return ch
}

// drive runs j to its end through Direct as the Controller's sender does,
// waiting out each Wait, and answers every step's answer. between runs
// after each step.
func (w *world) drive(j link.Job, between ...func(link.Advanced)) []link.Advanced {
	w.t.Helper()
	port := mediaserverlink.Direct(w.p.kind, w.dialed(apiKey))
	var all []link.Advanced
	for range 500 {
		a, err := port.Advance(w.t.Context(), j)
		if err != nil {
			w.t.Fatalf("Advance %T at %s: %v", j.Work, j.Step, err)
		}
		all = append(all, a)
		for _, b := range between {
			b(a)
		}
		if a.Done {
			return all
		}
		time.Sleep(time.Until(a.Wait))
		j = a.Job
	}
	w.t.Fatalf("%T didn't end in 500 steps", j.Work)
	return nil
}

func job(w link.Work) link.Job { return link.Job{Work: w, Step: link.StepNone} }

func outcomes(as []link.Advanced) map[link.ChangeID]link.Outcome {
	out := map[link.ChangeID]link.Outcome{}
	for _, a := range as {
		for _, o := range a.Outcomes {
			out[o.Change] = o
		}
	}
	return out
}

// item is the item at rel as the kind's Lookup finds it.
func (w *world) item(rel string) (link.ItemKey, bool) {
	w.t.Helper()
	as := w.drive(job(link.Lookup{Paths: []link.ServerPath{w.at(rel)}}))
	e := as[0].Learned[0]
	return e.Item, !e.Gone
}

// eventually waits for the server to hold an item at rel, as a monitor
// delay passes.
func (w *world) eventually(rel string) link.ItemKey {
	w.t.Helper()
	for range 60 {
		if it, ok := w.item(rel); ok {
			return it
		}
		time.Sleep(5 * time.Second)
	}
	w.t.Fatalf("no item at %s", rel)
	return link.ItemKey{}
}

func (w *world) records(kind string) []json.RawMessage {
	var out []json.RawMessage
	for _, r := range w.f.Records(w.t) {
		if r.Kind == kind {
			out = append(out, r.Data)
		}
	}
	return out
}

func (w *world) reports() []string {
	var out []string
	for _, raw := range w.records(w.p.reports) {
		var r struct {
			Updates []struct{ Path, UpdateType string }
		}
		if err := json.Unmarshal(raw, &r); err != nil {
			w.t.Fatal(err)
		}
		for _, u := range r.Updates {
			out = append(out, u.UpdateType+" "+u.Path)
		}
	}
	return out
}

func TestCheckReadsIdentityMonitorDelayAndLibraries(t *testing.T) {
	each(t, func(t *testing.T, w *world) {
		c := w.checked
		if c.Identity.ID != serverID || c.Identity.Name != "Den" {
			t.Errorf("identity = %+v", c.Identity)
		}
		if c.Identity.Version.Compare(w.p.kind.About().Floor) < 0 {
			t.Errorf("version %s is under the floor %s", c.Identity.Version, w.p.kind.About().Floor)
		}
		want := map[wire.Flavor]time.Duration{wire.Jellyfin: time.Minute, wire.Emby: 90 * time.Second}[w.p.flavor]
		if c.MonitorDelay != want {
			t.Errorf("monitor delay = %v, want %v", c.MonitorDelay, want)
		}
		if f := w.library("Movies").Folders; len(f) != 1 || f[0].String() != "/media/Movies" {
			t.Errorf("Movies folders = %v", f)
		}
	})
}

func TestAWrongKeyIsRefusedAndAnotherServerIsNamed(t *testing.T) {
	each(t, func(t *testing.T, w *world) {
		_, err := w.conn("not-the-key").Check(t.Context())
		if p, ok := link.ServerProblem(err); !ok || p.Reason != link.ReasonRefused {
			t.Errorf("Check with a wrong key = %v, want refused", err)
		}
		d := w.dialed(apiKey)
		d.Server.Identity = "another"
		_, err = w.p.kind.Dial(d).Advance(t.Context(), job(link.Libraries{}))
		if p, ok := link.ServerProblem(err); !ok || p.Reason != link.ReasonOtherServer || p.Found != serverID {
			t.Errorf("Advance on another server = %v, want other_server naming %s", err, serverID)
		}
		if _, err := w.conn("").Reach(t.Context()); err != nil {
			t.Errorf("Reach with no credential: %v", err)
		}
	})
}

func TestSignInCreatesAKeyTheServerAccepts(t *testing.T) {
	each(t, func(t *testing.T, w *world) {
		cred, err := w.conn("").SignIn(t.Context(), "jackson", "admin-password")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.conn(cred.APIKey).Check(t.Context()); err != nil {
			t.Errorf("Check with the signed-in key: %v", err)
		}
		_, err = w.conn("").SignIn(t.Context(), "jackson", "wrong")
		if p, ok := link.ServerProblem(err); !ok || p.Reason != link.ReasonRefused {
			t.Errorf("SignIn with a wrong password = %v, want refused", err)
		}
	})
}

func TestRefreshReportsEveryPathOnceAndTheServerTakesThem(t *testing.T) {
	each(t, func(t *testing.T, w *world) {
		w.place("Movies/Heat (1995)/Heat (1995).mkv")
		w.place("Movies/Alien (1979)/Alien (1979).mkv")
		a, b := w.change(link.ChangeAdded, "", "Movies/Heat (1995)/Heat (1995).mkv"), w.change(link.ChangeAdded, "", "Movies/Alien (1979)/Alien (1979).mkv")
		got := outcomes(w.drive(job(link.Refresh{Changes: []link.JobChange{a, b}})))
		for _, ch := range []link.JobChange{a, b} {
			if got[ch.ID].Result != link.ResultAccepted {
				t.Errorf("%s: outcome %+v, want accepted", ch.Path, got[ch.ID])
			}
		}
		want := []string{"Created /media/Movies/Heat (1995)/Heat (1995).mkv", "Created /media/Movies/Alien (1979)/Alien (1979).mkv"}
		if r := w.reports(); !slices.Equal(r, want) {
			t.Errorf("reports = %q, want %q", r, want)
		}
		w.eventually("Movies/Heat (1995)/Heat (1995).mkv")
		w.eventually("Movies/Alien (1979)/Alien (1979).mkv")
	})
}

func TestJellyfinReportsAgainWhenAScanStartsInsideTheMonitorDelay(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t, products[0])
		w.place("Movies/Heat (1995)/Heat (1995).mkv")
		ch := w.change(link.ChangeAdded, "", "Movies/Heat (1995)/Heat (1995).mkv")
		scanned := false
		w.drive(job(link.Refresh{Changes: []link.JobChange{ch}}), func(a link.Advanced) {
			if a.Job.Step == link.StepReported && !scanned {
				scanned = true
				time.Sleep(10 * time.Second)
				w.f.Cue(t, "jellyfin-scan", struct{}{})
			}
		})
		want := "Created /media/Movies/Heat (1995)/Heat (1995).mkv"
		if r := w.reports(); !slices.Equal(r, []string{want, want}) {
			t.Errorf("reports = %q, want the path reported twice", r)
		}
	})
}

func TestEmbyAcceptsAReportWithoutSettling(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t, products[1])
		ch := w.change(link.ChangeRemoved, "", "Movies/Metropolis (1927)/Metropolis (1927).mkv")
		if err := os.Remove(filepath.Join(w.media(), "Movies", "Metropolis (1927)", "Metropolis (1927).mkv")); err != nil {
			t.Fatal(err)
		}
		as := w.drive(job(link.Refresh{Changes: []link.JobChange{ch}}))
		if len(as) != 1 || outcomes(as)[ch.ID].Result != link.ResultAccepted {
			t.Errorf("steps = %+v, want one accepted step", as)
		}
		if _, ok := w.item("Movies/Metropolis (1927)/Metropolis (1927).mkv"); ok {
			t.Error("Emby still holds the deleted item")
		}
	})
}

// watched marks Metropolis played for both users, part way for the guest
// as a favorite, and puts it in the collection Silents.
func (w *world) watched() link.ItemKey {
	w.t.Helper()
	it, ok := w.item("Movies/Metropolis (1927)/Metropolis (1927).mkv")
	if !ok {
		w.t.Fatal("no Metropolis")
	}
	ctx := w.t.Context()
	if err := w.admin.MarkPlayed(ctx, w.adminID, it.Item); err != nil {
		w.t.Fatal(err)
	}
	if err := w.admin.SetUserData(ctx, w.guestID, it.Item, wire.UserData{PlaybackPositionTicks: int64(20 * time.Minute / 100), IsFavorite: true}); err != nil {
		w.t.Fatal(err)
	}
	if _, err := w.admin.CreateCollection(ctx, "Silents", []string{it.Item}); err != nil {
		w.t.Fatal(err)
	}
	return it
}

func stepsOf(as []link.Advanced) []link.Step {
	var out []link.Step
	for _, a := range as {
		if len(out) == 0 || out[len(out)-1] != a.Job.Step {
			out = append(out, a.Job.Step)
		}
	}
	return out
}

// silents are the ids of the items in the one collection.
func (w *world) silents() []string {
	w.t.Helper()
	page, err := w.admin.Items(w.t.Context(), wire.Query{User: w.adminID, Recursive: true, Types: []string{wire.TypeBoxSet}})
	if err != nil || len(page.Items) != 1 {
		w.t.Fatalf("collections = %+v, %v", page, err)
	}
	in, err := w.admin.Items(w.t.Context(), wire.Query{User: w.adminID, Parent: page.Items[0].ID})
	if err != nil {
		w.t.Fatal(err)
	}
	var out []string
	for _, i := range in.Items {
		out = append(out, i.ID)
	}
	return out
}

func TestCarryKeepsEveryUsersWatchDataAndCollections(t *testing.T) {
	each(t, func(t *testing.T, w *world) {
		w.watched()
		ch := w.change(link.ChangeMoved, "Movies/Metropolis (1927)/Metropolis (1927).mkv", "Movies/Metropolis (1927)/Metropolis (1927) - 1080p.mkv")
		w.move("Movies/Metropolis (1927)/Metropolis (1927).mkv", "Movies/Metropolis (1927)/Metropolis (1927) - 1080p.mkv")
		as := w.drive(job(link.Carry{Changes: []link.JobChange{ch}}))
		if steps, want := stepsOf(as), []link.Step{link.StepSnapshotted, link.StepOldRemoved, link.StepNewAdded, link.StepNewFound, link.StepRestored}; !slices.Equal(steps, want) {
			t.Errorf("steps = %v, want %v", steps, want)
		}
		if o := outcomes(as)[ch.ID]; o.Result != link.ResultAccepted || len(o.Lost) != 0 {
			t.Errorf("outcome = %+v, want accepted with nothing lost", o)
		}
		it := w.eventually("Movies/Metropolis (1927)/Metropolis (1927) - 1080p.mkv")
		ctx := t.Context()
		admin, err := w.admin.UserData(ctx, w.adminID, it.Item)
		if err != nil {
			t.Fatal(err)
		}
		guest, err := w.admin.UserData(ctx, w.guestID, it.Item)
		if err != nil {
			t.Fatal(err)
		}
		if !admin.Played || guest.Played || !guest.IsFavorite || guest.PlaybackPositionTicks != int64(20*time.Minute/100) {
			t.Errorf("new item's data: admin %+v, guest %+v", admin, guest)
		}
		if in := w.silents(); !slices.Contains(in, it.Item) {
			t.Errorf("Silents holds %v, want the new item %s", in, it.Item)
		}
	})
}

func TestACarryWithNoOldItemIsConverted(t *testing.T) {
	each(t, func(t *testing.T, w *world) {
		w.place("Movies/Up (2009)/Up (2009).mkv")
		ch := w.change(link.ChangeMoved, "Movies/Up/Up.mkv", "Movies/Up (2009)/Up (2009).mkv")
		o := outcomes(w.drive(job(link.Carry{Changes: []link.JobChange{ch}})))[ch.ID]
		if o.Result != link.ResultConverted || len(o.Lost) != 0 {
			t.Errorf("outcome = %+v, want converted with nothing lost", o)
		}
		if r := w.reports(); !slices.Equal(r, []string{"Created /media/Movies/Up (2009)/Up (2009).mkv"}) {
			t.Errorf("reports = %q, want only the new path Created", r)
		}
		w.eventually("Movies/Up (2009)/Up (2009).mkv")
	})
}

func TestFreshScansOnceThenTheServerTakesReports(t *testing.T) {
	each(t, func(t *testing.T, w *world) {
		fresh := w.library("Fresh")
		w.place("Fresh/Heat (1995)/Heat (1995).mkv")
		var decided int
		w.drive(job(link.Fresh{Library: fresh.Key, Folder: fresh.Folders[0]}), func(a link.Advanced) {
			if f, _ := a.Job.Work.(link.Fresh); !a.Done && !f.Requested.IsZero() && a.Wait.IsZero() {
				decided++
			}
		})
		if decided != 1 {
			t.Errorf("the Job came back decided with nothing sent %d times, want once before the scan", decided)
		}
		if n := len(w.records(w.p.scans)); n != 1 {
			t.Errorf("%d full scans, want 1", n)
		}
		again := w.drive(job(link.Fresh{Library: fresh.Key, Folder: fresh.Folders[0]}))
		if len(again) != 1 || !again[0].Done {
			t.Errorf("a second Fresh = %+v, want done at once from the item under the folder", again)
		}
		if n := len(w.records(w.p.scans)); n != 1 {
			t.Errorf("%d full scans after the second Fresh, want still 1", n)
		}
	})
}

func TestIndexPagesEveryFileOfALibrary(t *testing.T) {
	each(t, func(t *testing.T, w *world) {
		w.place("Movies/Heat (1995)/Heat (1995).mkv")
		w.place("Movies/Alien (1979)/Alien (1979).mkv")
		w.drive(job(link.Rescan{Libraries: []string{w.library("Movies").Key}}))
		w.eventually("Movies/Alien (1979)/Alien (1979).mkv")
		w.eventually("Movies/Heat (1995)/Heat (1995).mkv")
		d := w.dialed(apiKey)
		d.Timings.IndexPage = 2
		port := mediaserverlink.Direct(w.p.kind, d)
		j := job(link.Index{Library: w.library("Movies").Key})
		var paths []string
		for pages := 1; ; pages++ {
			a, err := port.Advance(t.Context(), j)
			if err != nil {
				t.Fatal(err)
			}
			for _, e := range a.Learned {
				paths = append(paths, e.Path.String())
			}
			if a.Done {
				if pages != 2 {
					t.Errorf("%d pages, want 2", pages)
				}
				break
			}
			j = a.Job
		}
		slices.Sort(paths)
		want := []string{"/media/Movies/Alien (1979)/Alien (1979).mkv", "/media/Movies/Heat (1995)/Heat (1995).mkv", "/media/Movies/Metropolis (1927)/Metropolis (1927).mkv"}
		if !slices.Equal(paths, want) {
			t.Errorf("indexed %q, want %q", paths, want)
		}
	})
}

// watched is every look a watch of the server passed on.
type watched struct {
	mu    sync.Mutex
	looks []link.Seen
}

func (w *world) watch(every time.Duration) *watched {
	ctx, cancel := context.WithCancel(w.t.Context())
	out := &watched{}
	done := make(chan error)
	go func() {
		done <- w.conn(apiKey).Watch(ctx, every, func(s link.Seen) {
			out.mu.Lock()
			defer out.mu.Unlock()
			out.looks = append(out.looks, s)
		})
	}()
	w.t.Cleanup(func() {
		cancel()
		if err := <-done; err != nil {
			w.t.Errorf("Watch = %v, want nil once its context ends", err)
		}
	})
	synctest.Wait()
	return out
}

// last is the last look of the kind asked for.
func (w *watched) last(polled bool) (link.Seen, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, s := range slices.Backward(w.looks) {
		if s.Polled == polled && len(s.Items) == 0 {
			return s, true
		}
	}
	return link.Seen{}, false
}

const metropolis = "Movies/Metropolis (1927)/Metropolis (1927).mkv"

type played struct{ Session, Item string }

// play starts Metropolis for jackson on Living room with d, a transcode
// for a reason and on hardware, and has the player report its playback.
func (w *world) play(d link.Decision) played {
	w.t.Helper()
	var p played
	raw, err := json.Marshal(w.f.Cue(w.t, string(w.p.kind.About().Name)+"-play", map[string]any{
		"path": w.at(metropolis).String(), "user": "jackson", "player": "Living room",
		"decision": d, "reasons": reasonsFor(d), "hardware": d == link.DecisionTranscode,
	}))
	if err == nil {
		err = json.Unmarshal(raw, &p)
	}
	if err != nil {
		w.t.Fatal(err)
	}
	synctest.Wait()
	return p
}

func reasonsFor(d link.Decision) []string {
	return map[link.Decision][]string{link.DecisionTranscode: {"VideoCodecNotSupported"}}[d]
}

// TestWatchSeesEachDecision plays Metropolis once per decision: the
// server's socket pushes the session at once, the next poll lists it with
// its decision, reasons and hardware, and a poll after the stop lists
// nothing. Emby shows a transcode only once the player reports its
// playback.
func TestWatchSeesEachDecision(t *testing.T) {
	for _, d := range link.Decisions() {
		t.Run(string(d), func(t *testing.T) {
			each(t, func(t *testing.T, w *world) { watchSees(t, w, d) })
		})
	}
}

func watchSees(t *testing.T, w *world, d link.Decision) {
	const every = 10 * time.Second
	looks := w.watch(every)
	p := w.play(d)
	if s, ok := looks.last(false); !ok || len(s.Sessions) != 1 {
		t.Errorf("the socket pushed %+v, want Metropolis playing", s)
	}
	if w.p.flavor == wire.Emby && d != link.DecisionDirectPlay {
		if err := w.admin.ReportPlaying(t.Context(), wire.Playing{ItemID: p.Item, PlaySessionID: p.Session, PlayMethod: wire.Transcode}); err != nil {
			t.Fatal(err)
		}
	}
	time.Sleep(every)
	synctest.Wait()
	s, _ := looks.last(true)
	want := fmt.Sprintf("%s plays %s for jackson on Living room, %s %q hardware %v", p.Session, w.at(metropolis), d, reasonsFor(d), d == link.DecisionTranscode)
	if got := sessionsOf(s); !slices.Equal(got, []string{want}) || s.Transcoding != (d == link.DecisionTranscode) {
		t.Errorf("the poll saw %q transcoding %v, want %q transcoding %v", got, s.Transcoding, want, d == link.DecisionTranscode)
	}
	w.f.Cue(t, string(w.p.kind.About().Name)+"-stop", map[string]string{"session": p.Session})
	time.Sleep(every)
	synctest.Wait()
	if s, _ := looks.last(true); len(s.Sessions) != 0 || s.Problem != nil {
		t.Errorf("the poll after the stop saw %+v, want no session", s)
	}
}

func sessionsOf(s link.Seen) []string {
	var out []string
	for _, x := range s.Sessions {
		out = append(out, fmt.Sprintf("%s plays %s for %s on %s, %s %q hardware %v", x.Key, x.Path, x.User, x.Player, x.Decision, x.Reasons, x.Hardware))
	}
	return out
}

func TestAPreparedSnapshotsPlayCountIsLostOnlyOnEmby(t *testing.T) {
	each(t, func(t *testing.T, w *world) {
		old, ok := w.item("Movies/Metropolis (1927)/Metropolis (1927).mkv")
		if !ok {
			t.Fatal("no Metropolis")
		}
		last := time.Date(2026, 11, 1, 20, 0, 0, 0, time.UTC)
		ch := w.change(link.ChangeMoved, "Movies/Metropolis (1927)/Metropolis (1927).mkv", "Movies/Metropolis/Metropolis.mkv")
		ch.Snapshot = &link.Snapshot{Item: old, Users: []link.UserData{{User: w.adminID, Played: true, PlayCount: 3, LastPlayed: last}}}
		w.move("Movies/Metropolis (1927)/Metropolis (1927).mkv", "Movies/Metropolis/Metropolis.mkv")
		j := job(link.Carry{Changes: []link.JobChange{ch}})
		j.Step, j.At = link.StepSnapshotted, time.Now()
		o := outcomes(w.drive(j))[ch.ID]
		var want []link.Loss
		if w.p.flavor == wire.Emby {
			want = []link.Loss{link.LostPlayCount, link.LostLastPlayed}
		}
		if o.Result != link.ResultAccepted || !slices.Equal(o.Lost, want) {
			t.Errorf("outcome = %+v, want accepted, lost %v", o, want)
		}
		it := w.eventually("Movies/Metropolis/Metropolis.mkv")
		d, err := w.admin.UserData(t.Context(), w.adminID, it.Item)
		if err != nil {
			t.Fatal(err)
		}
		if !d.Played {
			t.Errorf("the new item isn't played: %+v", d)
		}
	})
}
