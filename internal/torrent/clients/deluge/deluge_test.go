package deluge_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	fakedeluge "github.com/jatassi/slipstream/fakes/deluge"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/torrent/clients"
	"github.com/jatassi/slipstream/internal/torrent/clients/deluge"
)

const (
	node     = "Media server"
	size     = 1000
	password = "orchard"
)

var (
	slipstream, _ = clients.ParseLabel(clients.DefaultLabel, clients.Deluge)
	local         = clients.Daemon{Host: "127.0.0.1", Port: 58846}
	other         = clients.Daemon{Host: "10.0.0.5", Port: 58846}
)

type release struct {
	name     string
	hash     bencode.InfoHash
	metainfo []byte
}

// rig is one Node's fake Deluge and sessions with it.
type rig struct {
	t        *testing.T
	served   faketest.Served
	client   fakedeluge.Client
	releases []release
	wire     *wire
}

// wire counts the requests sessions sent, and the ones a web UI never
// answered.
type wire struct {
	mu    sync.Mutex
	inner http.RoundTripper
	sent  []string
}

func (w *wire) RoundTrip(req *http.Request) (*http.Response, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err //nolint:wrapcheck // the session wraps it
	}
	var call struct {
		Method string `json:"method"`
	}
	_ = json.Unmarshal(body, &call)
	w.mu.Lock()
	w.sent = append(w.sent, call.Method)
	w.mu.Unlock()
	req.Body = io.NopCloser(bytes.NewReader(body))
	return w.inner.RoundTrip(req) //nolint:wrapcheck // the session wraps it
}

func (w *wire) take() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := w.sent
	w.sent = nil
	return out
}

// daemon is a daemon at d with its own folder under t's temp dir.
func daemon(t *testing.T, d clients.Daemon, folder string) fakedeluge.Daemon {
	return fakedeluge.Daemon{Host: d.Host, Port: d.Port, Dir: t.TempDir(), Folder: folder, Rate: 100}
}

func newRig(t *testing.T, c fakedeluge.Client) *rig {
	t.Helper()
	c.Node, c.Version = node, "2.1"
	if c.Password == "" {
		c.Password = password
	}
	if c.Daemons == nil {
		c.Daemons = []fakedeluge.Daemon{daemon(t, local, "/downloads")}
	}
	for i := range c.Daemons {
		if c.Daemons[i].Dir == "" {
			c.Daemons[i].Dir = t.TempDir()
		}
		if c.Daemons[i].Rate == 0 {
			c.Daemons[i].Rate = 100
		}
	}
	catalog := faketest.Catalog(t, size, "orchard", "lantern")
	r := &rig{t: t, client: c}
	for _, rel := range catalog.Releases {
		r.releases = append(r.releases, release{name: rel.Name, hash: rel.InfoHash, metainfo: rel.Metainfo})
	}
	r.served = faketest.InMemory(t, fakedeluge.Playing(c), scenario.Scenario{Seed: "deluge"}, faketest.WithReleases(catalog), faketest.As(node))
	r.wire = &wire{inner: r.served.Client.Transport}
	return r
}

// open is a session with password to daemon, nil for the only one.
func (r *rig) open(password string, daemon *clients.Daemon) clients.Client {
	r.t.Helper()
	a, err := clients.ParseAddress(r.served.Address.URL(""), clients.Deluge)
	if err != nil {
		r.t.Fatal(err)
	}
	return deluge.Open(clients.Session{
		HTTP: &http.Client{Transport: r.wire}, Address: a, Secrets: clients.Secrets{Password: password}, Daemon: daemon,
		Call: 30 * time.Second, Log: slog.New(slog.NewTextHandler(r.t.Output(), &slog.HandlerOptions{Level: slog.LevelDebug})),
	})
}

func (r *rig) session() clients.Client { return r.open(password, nil) }

func (r *rig) add(c clients.Client, rel release, folder string) {
	r.t.Helper()
	if err := c.Add(r.t.Context(), clients.Add{Metainfo: rel.metainfo, Hash: rel.hash, Folder: folder, Label: slipstream}); err != nil {
		r.t.Fatalf("Add(%s): %v", rel.name, err)
	}
}

func (r *rig) poll(c clients.Client, rel release) (clients.Status, bool) {
	r.t.Helper()
	got, err := c.Poll(r.t.Context(), []bencode.InfoHash{rel.hash})
	if err != nil {
		r.t.Fatalf("Poll: %v", err)
	}
	s, ok := got[rel.hash]
	return s, ok
}

func (r *rig) status(c clients.Client, rel release) clients.Status {
	r.t.Helper()
	s, ok := r.poll(c, rel)
	if !ok {
		r.t.Fatalf("Deluge doesn't list %s", rel.name)
	}
	return s
}

func (r *rig) cue(name string, args map[string]any) {
	r.t.Helper()
	args["node"] = node
	r.served.Cue(r.t, "deluge-"+name, args)
}

func (r *rig) act(c clients.Client, a clients.Act, rel release) {
	r.t.Helper()
	if err := c.Act(r.t.Context(), a, []bencode.InfoHash{rel.hash}); err != nil {
		r.t.Fatalf("Act(%d): %v", a, err)
	}
}

func problem(t *testing.T, err error, want clients.Cause) clients.Problem {
	t.Helper()
	p, ok := errors.AsType[*clients.Problem](err)
	if !ok || p.Cause != want {
		t.Fatalf("err = %v, want a Problem with cause %s", err, want)
	}
	return *p
}

func TestDelugeSignsInAndConnectsToItsOnlyDaemon(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newRig(t, fakedeluge.Client{})
		c := r.session()
		tested, err := c.Test(t.Context(), slipstream)
		if err != nil {
			t.Fatal(err)
		}
		if tested.Version.String() != "2.1.1" || !slices.Equal(tested.Daemons, []clients.Daemon{local}) || len(tested.Warnings) != 0 || tested.MovesAfterFinish {
			t.Fatalf("Test = %+v, want 2.1.1 with its one daemon and no warnings", tested)
		}
		if v, err := c.Version(t.Context()); err != nil || v != tested.Version {
			t.Fatalf("Version = %s, %v, want %s", v, err, tested.Version)
		}
		r.add(c, r.releases[0], "")
		r.status(c, r.releases[0])
		if err := c.Close(t.Context()); err != nil {
			t.Fatal(err)
		}
		if _, err := c.Poll(t.Context(), nil); err != nil {
			t.Fatalf("a Poll after Close signs in again: %v", err)
		}
	})
}

func TestDelugeConnectsToTheChosenDaemon(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newRig(t, fakedeluge.Client{Daemons: []fakedeluge.Daemon{daemon(t, local, "/downloads"), daemon(t, other, "/srv/torrents")}})
		tested, err := r.session().Test(t.Context(), slipstream)
		if p := problem(t, err, clients.DaemonNeeded); p.Cause.State() != clients.NeedsAttention || !slices.Equal(p.Cause.Fields(), []clients.Field{clients.FieldDaemon}) {
			t.Fatalf("two daemons and none chosen: %+v, want it to need attention on the daemon field", p)
		}
		if !slices.Equal(tested.Daemons, []clients.Daemon{local, other}) {
			t.Fatalf("a Test that needs a daemon listed %+v, want both to choose from", tested.Daemons)
		}

		chosen := r.open(password, &other)
		if _, err := chosen.Test(t.Context(), slipstream); err != nil {
			t.Fatal(err)
		}
		if folder, err := chosen.DefaultFolder(t.Context(), slipstream); err != nil || folder != "/srv/torrents" {
			t.Fatalf("DefaultFolder on the chosen daemon = %q, %v, want its /srv/torrents", folder, err)
		}
		r.add(chosen, r.releases[0], "")
		if got := r.status(chosen, r.releases[0]); got.Folder != "/srv/torrents" {
			t.Fatalf("an add to the chosen daemon: %+v, want it in that daemon's folder", got)
		}

		elsewhere := clients.Daemon{Host: "10.0.0.9", Port: 58846}
		tested, err = r.open(password, &elsewhere).Test(t.Context(), slipstream)
		problem(t, err, clients.WrongDaemon)
		if len(tested.Daemons) != 2 {
			t.Fatalf("a Test of a daemon the web UI doesn't list: %+v, want the two it lists", tested.Daemons)
		}
	})
}

func TestDelugeSignsInAndConnectsAgain(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newRig(t, fakedeluge.Client{SessionTimeout: time.Minute})
		c := r.session()
		orchard, lantern := r.releases[0], r.releases[1]
		r.add(c, orchard, "/downloads/Movies")

		time.Sleep(2 * time.Minute)
		if got := r.status(c, orchard); got.Label != "slipstream" {
			t.Fatalf("a poll after the session expired: %+v", got)
		}
		r.cue("restart", map[string]any{})
		r.add(c, lantern, "")
		if got := r.status(c, lantern); got.Label != "slipstream" {
			t.Fatalf("an add after the web UI restarted: %+v, want it added and labeled", got)
		}

		r.cue("disconnect", map[string]any{})
		start := time.Now()
		r.status(c, orchard)
		if waited := time.Since(start); waited > 0 {
			t.Fatalf("a poll of a web UI that lost its daemon waited %s, want it connected again first", waited)
		}
	})
}

func TestDelugeReportsAWebUIConnectedToAnotherDaemon(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newRig(t, fakedeluge.Client{Daemons: []fakedeluge.Daemon{daemon(t, local, "/downloads"), daemon(t, other, "/srv/torrents")}})
		c := r.open(password, &local)
		orchard := r.releases[0]
		r.add(c, orchard, "")
		r.status(c, orchard)

		r.cue("switch", map[string]any{"host": other.Host, "port": other.Port})
		_, err := c.Poll(t.Context(), []bencode.InfoHash{orchard.hash})
		if p := problem(t, err, clients.WrongDaemon); p.Cause.State() != clients.NeedsAttention {
			t.Fatalf("a poll from a web UI on another daemon: %+v, want it to need attention", p)
		}
		if _, err := c.Test(t.Context(), slipstream); err != nil {
			t.Fatalf("a Test connects the chosen daemon again: %v", err)
		}
		r.status(c, orchard)
	})
}

func TestDelugeRefusesAWrongPasswordWithoutLockingOut(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newRig(t, fakedeluge.Client{})
		wrong := r.open("wrong", nil)
		for range 10 {
			_, err := wrong.Test(t.Context(), slipstream)
			if problem(t, err, clients.WrongCredentials).Cause.State() != clients.SignInRefused {
				t.Fatal("a wrong password isn't a refused sign-in")
			}
		}
		r.wire.take()
		_, err := wrong.Poll(t.Context(), nil)
		problem(t, err, clients.WrongCredentials)
		if got := r.wire.take(); !slices.Equal(got, []string{"auth.login"}) {
			t.Fatalf("a poll with a wrong password sent %v, want one sign-in", got)
		}
		if _, err := r.session().Test(t.Context(), slipstream); err != nil {
			t.Fatalf("the right password after ten wrong ones: %v", err)
		}
	})
}

func TestDelugeNeedsItsLabelPlugin(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := daemon(t, local, "/downloads")
		d.LabelOff = true
		r := newRig(t, fakedeluge.Client{Daemons: []fakedeluge.Daemon{d}})
		c := r.session()
		_, err := c.Test(t.Context(), slipstream)
		if p := problem(t, err, clients.LabelPluginOff); !slices.Equal(p.Cause.Fields(), []clients.Field{clients.FieldLabel}) {
			t.Fatalf("the Label plugin off: %+v, want the label field marked", p)
		}
		orchard := r.releases[0]
		problem(t, c.Add(t.Context(), clients.Add{Metainfo: orchard.metainfo, Hash: orchard.hash, Label: slipstream}), clients.LabelPluginOff)
		problem(t, c.Add(t.Context(), clients.Add{Metainfo: orchard.metainfo, Hash: orchard.hash, Label: slipstream}), clients.LabelPluginOff)
	})
}

func TestDelugeAddsWithTheLabelAndFolder(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newRig(t, fakedeluge.Client{})
		c := r.session()
		folder, err := c.DefaultFolder(t.Context(), slipstream)
		if err != nil || folder != "/downloads" {
			t.Fatalf("DefaultFolder = %q, %v, want /downloads", folder, err)
		}
		orchard, lantern := r.releases[0], r.releases[1]
		if err := c.Add(t.Context(), clients.Add{Metainfo: orchard.metainfo, Hash: orchard.hash, Folder: "/downloads/Movies", Label: slipstream, Paused: true}); err != nil {
			t.Fatal(err)
		}
		r.add(c, orchard, "/downloads/Elsewhere")
		got := r.status(c, orchard)
		if got.Name != "orchard.mkv" || got.Label != "slipstream" || got.Folder != "/downloads/Movies" || got.Size != size || got.Run != clients.Paused || got.Finished {
			t.Fatalf("after a paused Add: %+v, want orchard.mkv paused in /downloads/Movies labeled slipstream, the duplicate add ignored", got)
		}
		if _, ok := r.poll(c, lantern); ok {
			t.Fatal("Poll lists a torrent nobody added")
		}
		r.add(c, lantern, "")
		if got := r.status(c, lantern); got.Folder != "/downloads" || got.Run != clients.Started || got.Label != "slipstream" {
			t.Fatalf("a torrent added with no folder: %+v, want it started in the default folder", got)
		}
	})
}

func TestDelugeHealsALabelThatNeverArrived(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newRig(t, fakedeluge.Client{})
		c := r.session()
		orchard := r.releases[0]
		r.cue("late-add", map[string]any{"after": "30s"})
		r.add(c, orchard, "")
		time.Sleep(30 * time.Second)
		if got := r.status(c, orchard); got.Label != "" {
			t.Fatalf("a torrent Deluge listed after its label was sent: %+v, want no label yet", got)
		}
		r.add(c, orchard, "")
		if got := r.status(c, orchard); got.Label != "slipstream" {
			t.Fatalf("after the Add again: %+v, want the label healed", got)
		}
	})
}

func TestDelugeReportsDownloadingQueuedAndFailed(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := daemon(t, local, "/downloads")
		d.Slots = 1
		r := newRig(t, fakedeluge.Client{Daemons: []fakedeluge.Daemon{d}})
		c := r.session()
		orchard, lantern := r.releases[0], r.releases[1]
		r.add(c, orchard, "/downloads/Movies")
		r.add(c, lantern, "/downloads/Movies")

		time.Sleep(4 * time.Second)
		got := r.status(c, orchard)
		if got.Run != clients.Started || got.Activity != clients.Downloading || got.Done != 400 || got.Finished || got.Rates.Down != 100 || got.Rates.Peers != 1 {
			t.Fatalf("downloading: %+v", got)
		}
		if got := r.status(c, lantern); got.Activity != clients.Queued || got.QueuePosition != 2 || got.Finished {
			t.Fatalf("past the queue's one slot: %+v, want queued second", got)
		}
		r.cue("error", map[string]any{"release": lantern.name, "message": "No space left on device"})
		if got := r.status(c, lantern); got.LocalError != "No space left on device" || got.Run != clients.Paused || got.Finished {
			t.Fatalf("a local error: %+v, want it paused with Deluge's message and not finished", got)
		}
	})
}

func TestDelugeReportsFinishedUntilTheSeedRatioStopsIt(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := daemon(t, local, "/downloads")
		d.SeedRatio = 1
		r := newRig(t, fakedeluge.Client{Daemons: []fakedeluge.Daemon{d}})
		c := r.session()
		orchard := r.releases[0]
		start := time.Now()
		r.add(c, orchard, "/downloads/Movies")
		time.Sleep(10 * time.Second)
		got := r.status(c, orchard)
		if !got.Finished || got.Activity != clients.Seeding || got.Done != size || !got.FinishedAt.Equal(start.Add(10*time.Second)) || got.StoppedAtLimit || got.Trackers != nil {
			t.Fatalf("complete: %+v, want finished and seeding from 10s", got)
		}
		files, err := c.Files(t.Context(), orchard.hash)
		if err != nil || !slices.Equal(files, []clients.File{{Index: 0, Path: "orchard.mkv", Size: size, Done: size}}) {
			t.Fatalf("Files = %+v, %v", files, err)
		}

		r.cue("tracker", map[string]any{"release": orchard.name, "message": "Unregistered torrent", "error": true})
		if got := r.status(c, orchard); !got.Finished || !slices.Equal(got.Trackers, []clients.TrackerMessage{{Message: "Unregistered torrent", Error: true}}) {
			t.Fatalf("a tracker error: %+v, want it as an error on a finished torrent", got)
		}
		r.cue("tracker", map[string]any{"release": orchard.name, "message": "Slow down"})
		if got := r.status(c, orchard); !got.Finished || !slices.Equal(got.Trackers, []clients.TrackerMessage{{Message: "Slow down"}}) {
			t.Fatalf("a tracker warning: %+v, want it as a warning", got)
		}
		r.cue("tracker", map[string]any{"release": orchard.name, "message": ""})

		time.Sleep(10 * time.Second)
		got = r.status(c, orchard)
		if !got.StoppedAtLimit || got.Run != clients.Paused || !got.Finished || got.Counters.Uploaded != size || got.Counters.Seeding != 10*time.Second || got.Trackers != nil {
			t.Fatalf("at the seed ratio: %+v, want stopped at the limit and still finished", got)
		}
		r.act(c, clients.Resume, orchard)
		if got := r.status(c, orchard); got.Run != clients.Paused {
			t.Fatalf("resumed past its stop ratio: %+v, want Deluge to keep it paused", got)
		}
	})
}

func TestDelugeReportsAMoveAsNotFinishedUntilItsFilesAreThere(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := daemon(t, local, "/downloads")
		d.MoveTo, d.MoveFor = "/downloads/done", 5*time.Second
		r := newRig(t, fakedeluge.Client{Daemons: []fakedeluge.Daemon{d}})
		c := r.session()
		tested, err := c.Test(t.Context(), slipstream)
		if err != nil || !tested.MovesAfterFinish {
			t.Fatalf("Test = %+v, %v, want MovesAfterFinish", tested, err)
		}
		orchard := r.releases[0]
		r.add(c, orchard, "")
		time.Sleep(12 * time.Second)
		if got := r.status(c, orchard); got.Activity != clients.Moving || got.Finished || got.Done != size || got.Folder != "/downloads" {
			t.Fatalf("moving its files: %+v, want moving and not finished", got)
		}
		time.Sleep(5 * time.Second)
		if got := r.status(c, orchard); got.Activity != clients.Seeding || !got.Finished || got.Folder != "/downloads/done" {
			t.Fatalf("after the move: %+v, want finished in /downloads/done", got)
		}
		if _, err := os.Stat(filepath.Join(d.Dir, "done", "orchard.mkv")); err != nil {
			t.Fatalf("after the move, its file: %v", err)
		}
	})
}

func TestDelugeWarnsOfALabelsMove(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := daemon(t, local, "/downloads")
		d.Labels = map[string]string{"slipstream": "/downloads/done"}
		r := newRig(t, fakedeluge.Client{Daemons: []fakedeluge.Daemon{d}})
		tested, err := r.session().Test(t.Context(), slipstream)
		if err != nil || !tested.MovesAfterFinish {
			t.Fatalf("Test = %+v, %v, want MovesAfterFinish from the label's move", tested, err)
		}
		c := r.session()
		orchard := r.releases[0]
		r.add(c, orchard, "")
		time.Sleep(15 * time.Second)
		if got := r.status(c, orchard); !got.Finished || got.Folder != "/downloads/done" {
			t.Fatalf("labeled with a move: %+v, want it finished in the label's folder", got)
		}
	})
}

func TestDelugeWarnsAndRemovesAtItsSeedRatio(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := daemon(t, local, "/downloads")
		d.SeedRatio, d.RemoveAtRatio = 1, true
		r := newRig(t, fakedeluge.Client{Daemons: []fakedeluge.Daemon{d}})
		c := r.session()
		tested, err := c.Test(t.Context(), slipstream)
		if err != nil || !slices.Equal(tested.Warnings, []clients.Warning{clients.RemovesAtLimit}) {
			t.Fatalf("Test = %+v, %v, want the RemovesAtLimit warning", tested, err)
		}
		orchard := r.releases[0]
		r.add(c, orchard, "/downloads/Movies")
		time.Sleep(25 * time.Second)
		if s, ok := r.poll(c, orchard); ok {
			t.Fatalf("at the seed ratio Deluge still lists %+v, want it removed", s)
		}
		if _, err := os.Stat(filepath.Join(d.Dir, "Movies", "orchard.mkv")); err != nil {
			t.Fatalf("removed at the ratio, its file: %v", err)
		}
	})
}

func TestDelugePausesResumesAndRechecks(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := daemon(t, local, "/downloads")
		d.Check = 2 * time.Second
		r := newRig(t, fakedeluge.Client{Daemons: []fakedeluge.Daemon{d}})
		c := r.session()
		orchard := r.releases[0]
		r.add(c, orchard, "")
		if got := r.status(c, orchard); got.Activity != clients.Checking || got.Run != clients.Started {
			t.Fatalf("checking after its add: %+v", got)
		}
		r.act(c, clients.Pause, orchard)
		if got := r.status(c, orchard); got.Run != clients.Paused || got.Activity != clients.Downloading {
			t.Fatalf("paused while checking: %+v, want it paused, as Deluge reads it", got)
		}
		time.Sleep(5 * time.Second)
		if got := r.status(c, orchard); got.Run != clients.Paused || got.Activity != clients.Downloading || got.Done != 0 {
			t.Fatalf("paused: %+v, want paused with nothing done", got)
		}
		for _, a := range []clients.Act{clients.Resume, clients.Recheck, clients.Announce} {
			r.act(c, a, orchard)
		}
		if got := r.status(c, orchard); got.Activity != clients.Checking || got.Run != clients.Started {
			t.Fatalf("rechecking after a resume: %+v", got)
		}
		time.Sleep(5 * time.Second)
		if got := r.status(c, orchard); got.Run != clients.Started || got.Activity != clients.Downloading || got.Done != 300 {
			t.Fatalf("resumed: %+v, want 300 bytes after the 2s check", got)
		}
	})
}

func TestDelugeRemovesWithAndWithoutData(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newRig(t, fakedeluge.Client{})
		c := r.session()
		orchard, lantern := r.releases[0], r.releases[1]
		r.add(c, orchard, "/downloads/Movies")
		r.add(c, lantern, "/downloads/Movies")
		time.Sleep(time.Minute)
		r.status(c, orchard)
		if err := c.Remove(t.Context(), orchard.hash, true); err != nil {
			t.Fatal(err)
		}
		if err := c.Remove(t.Context(), lantern.hash, false); err != nil {
			t.Fatal(err)
		}
		if err := c.Remove(t.Context(), lantern.hash, false); err != nil {
			t.Fatalf("removing a torrent Deluge no longer holds: %v", err)
		}
		got, err := c.Poll(t.Context(), []bencode.InfoHash{orchard.hash, lantern.hash})
		if err != nil || len(got) != 0 {
			t.Fatalf("Poll after the removes = %v, %v, want neither listed", got, err)
		}
		movies := filepath.Join(r.client.Daemons[0].Dir, "Movies")
		if _, err := os.Stat(filepath.Join(movies, "orchard.mkv")); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("removed with data, its file: %v", err)
		}
		if _, err := os.Stat(filepath.Join(movies, "lantern.mkv")); err != nil {
			t.Errorf("removed without data, its file: %v", err)
		}
	})
}

func TestDelugesOwnRemovesAndDroppedAddsShowInPolls(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newRig(t, fakedeluge.Client{})
		c := r.session()
		orchard := r.releases[0]
		r.add(c, orchard, "")
		r.cue("remove", map[string]any{"release": orchard.name})
		if _, ok := r.poll(c, orchard); ok {
			t.Fatal("Poll lists a torrent Deluge removed")
		}
		r.cue("drop-add", map[string]any{})
		r.add(c, orchard, "")
		if _, ok := r.poll(c, orchard); ok {
			t.Fatal("Poll lists a torrent Deluge dropped")
		}
		r.add(c, orchard, "")
		r.cue("empty", map[string]any{})
		if got, err := c.Poll(t.Context(), []bencode.InfoHash{orchard.hash}); err != nil || len(got) != 0 {
			t.Fatalf("Poll of an emptied Deluge = %v, %v", got, err)
		}
	})
}

func TestDelugeSpeaksOverLoopback(t *testing.T) {
	d := fakedeluge.Client{Node: node, Version: "2.1", Password: password, Daemons: []fakedeluge.Daemon{daemon(t, local, "/downloads")}}
	catalog := faketest.Catalog(t, size, "orchard")
	f := faketest.Loopback(t, fakedeluge.Playing(d), scenario.Scenario{Seed: "deluge"}, faketest.WithReleases(catalog), faketest.As(node))
	a, err := clients.ParseAddress(f.Address.URL(""), clients.Deluge)
	if err != nil {
		t.Fatal(err)
	}
	c := deluge.Open(clients.Session{HTTP: f.Client, Address: a, Secrets: clients.Secrets{Password: password}, Call: 10 * time.Second, Log: slog.New(slog.DiscardHandler)})
	rel := catalog.Releases[0]
	if err := c.Add(t.Context(), clients.Add{Metainfo: rel.Metainfo, Hash: rel.InfoHash, Label: slipstream}); err != nil {
		t.Fatal(err)
	}
	got, err := c.Poll(t.Context(), []bencode.InfoHash{rel.InfoHash})
	if err != nil || got[rel.InfoHash].Name != "orchard.mkv" || got[rel.InfoHash].Label != "slipstream" {
		t.Fatalf("Poll = %+v, %v", got, err)
	}
}
