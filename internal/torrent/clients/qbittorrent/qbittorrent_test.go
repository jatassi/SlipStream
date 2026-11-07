package qbittorrent_test

import (
	"crypto/rand"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fake/faketest"
	fakeqbittorrent "github.com/jatassi/slipstream/fakes/qbittorrent"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/torrent/clients"
	"github.com/jatassi/slipstream/internal/torrent/clients/qbittorrent"
)

const (
	node     = "Media server"
	size     = 1000
	username = "jackson"
	password = "orchard"
	apiKey   = "qbt_0123456789abcdefghijklmnopqrstuv"
)

type release struct {
	name     string
	hash     bencode.InfoHash
	metainfo []byte
}

var slipstream, _ = clients.ParseLabel(clients.DefaultLabel, clients.QBittorrent)

// rig is one Node's fake qBittorrent and sessions with it.
type rig struct {
	t        *testing.T
	served   faketest.Served
	client   fakeqbittorrent.Client
	releases []release
	wire     *wire
}

// wire is every request a session sent and the status it got.
type wire struct {
	mu    sync.Mutex
	inner http.RoundTripper
	sent  []exchange
}

type exchange struct {
	call   string
	status int
	// hashes is how many hashes a torrents/info asked for.
	hashes int
}

func (w *wire) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Header.Get("Origin") != "" || req.Header.Get("Referer") != "" {
		return nil, errors.New("the session sent an Origin or Referer header")
	}
	res, err := w.inner.RoundTrip(req)
	if err != nil {
		return nil, err //nolint:wrapcheck // the session wraps it
	}
	e := exchange{call: strings.TrimPrefix(req.URL.Path, "/api/v2/"), status: res.StatusCode}
	if e.call == "torrents/info" {
		e.hashes = len(strings.Split(req.URL.Query().Get("hashes"), "|"))
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.sent = append(w.sent, e)
	return res, nil
}

func (w *wire) take() []exchange {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := w.sent
	w.sent = nil
	return out
}

func newRig(t *testing.T, c fakeqbittorrent.Client) *rig {
	t.Helper()
	c.Node, c.Dir, c.Folder = node, t.TempDir(), "/downloads"
	if c.Rate == 0 {
		c.Rate = 100
	}
	if c.Username == "" && c.APIKey == "" {
		c.Username, c.Password = username, password
	}
	catalog := faketest.Catalog(t, size, "orchard", "lantern")
	r := &rig{t: t, client: c}
	for _, rel := range catalog.Releases {
		r.releases = append(r.releases, release{name: rel.Name, hash: rel.InfoHash, metainfo: rel.Metainfo})
	}
	r.served = faketest.InMemory(t, fakeqbittorrent.Playing(c), scenario.Scenario{Seed: "qbittorrent"},
		faketest.WithReleases(catalog), faketest.As(node))
	r.wire = &wire{inner: r.served.Client.Transport}
	return r
}

// session is the parts of a Session a test sets.
type session struct {
	address            string
	username, password string
	apiKey             string
	trackerEvery       time.Duration
}

// open is a session, at the fake's own address when s names none.
func (r *rig) open(s session) clients.Client {
	r.t.Helper()
	if s.address == "" {
		s.address = r.served.Address.URL("")
	}
	a, err := clients.ParseAddress(s.address, clients.QBittorrent)
	if err != nil {
		r.t.Fatal(err)
	}
	return qbittorrent.Open(clients.Session{
		HTTP: &http.Client{Transport: r.wire}, Address: a, Username: s.username,
		Secrets: clients.Secrets{Password: s.password, APIKey: s.apiKey}, TrackerEvery: s.trackerEvery,
		Call: 30 * time.Second, Log: slog.New(slog.NewTextHandler(r.t.Output(), &slog.HandlerOptions{Level: slog.LevelDebug})),
	})
}

func (r *rig) session() clients.Client {
	return r.open(session{username: r.client.Username, password: r.client.Password, apiKey: r.client.APIKey})
}

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
		r.t.Fatalf("qBittorrent doesn't list %s", rel.name)
	}
	return s
}

func (r *rig) cue(name string, args map[string]any) {
	r.t.Helper()
	args["node"] = node
	r.served.Cue(r.t, "qbittorrent-"+name, args)
}

func (r *rig) act(c clients.Client, a clients.Act, rel release) {
	r.t.Helper()
	if err := c.Act(r.t.Context(), a, []bencode.InfoHash{rel.hash}); err != nil {
		r.t.Fatalf("Act(%d): %v", a, err)
	}
}

// inEachVersion runs test against a fake qBittorrent at each version it
// plays, over an in-memory network under synctest.
func inEachVersion(t *testing.T, test func(t *testing.T, version string)) {
	for _, v := range fakeqbittorrent.Versions {
		t.Run(v, func(t *testing.T) { synctest.Test(t, func(t *testing.T) { test(t, v) }) })
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

// only is the exchanges of the named calls.
func only(got []exchange, names ...string) []exchange {
	return slices.DeleteFunc(got, func(e exchange) bool { return !slices.Contains(names, e.call) })
}

func calls(got []exchange) []exchange {
	out := slices.Clone(got)
	for i := range out {
		out[i].hashes = 0
	}
	return out
}

// signIn is the exchanges of a session's first call: the call refused
// for want of a session, auth/login in the version's generation, the call
// again and app/webapiVersion.
func signIn(version string) []exchange {
	login := map[bool]int{false: http.StatusOK, true: http.StatusNoContent}[version == "5.2"]
	return []exchange{
		{call: "app/version", status: http.StatusForbidden}, {call: "auth/login", status: login},
		{call: "app/version", status: http.StatusOK}, {call: "app/webapiVersion", status: http.StatusOK},
	}
}

func TestQBittorrentSignsInInItsVersionsGeneration(t *testing.T) {
	wantVersion := map[string]string{"4.6": "4.6.7", "5.1": "5.1.4", "5.2": "5.2.3"}
	inEachVersion(t, func(t *testing.T, version string) {
		r := newRig(t, fakeqbittorrent.Client{Version: version})
		c := r.session()
		tested, err := c.Test(t.Context(), slipstream)
		if err != nil {
			t.Fatal(err)
		}
		if tested.Version.String() != wantVersion[version] || len(tested.Warnings) != 0 {
			t.Fatalf("Test = %+v, want version %s and no warnings", tested, wantVersion[version])
		}
		want := append(signIn(version), exchange{call: "app/preferences", status: http.StatusOK})
		if got := r.wire.take(); !slices.Equal(got, want) {
			t.Fatalf("a first Test sent %+v, want %+v", got, want)
		}
		r.poll(c, r.releases[0])
		if got := calls(r.wire.take()); !slices.Equal(got, []exchange{{call: "torrents/info", status: http.StatusOK}}) {
			t.Fatalf("a Poll on the session sent %+v, want one request", got)
		}
		if v, err := c.Version(t.Context()); err != nil || v != tested.Version {
			t.Fatalf("Version = %s, %v, want %s", v, err, tested.Version)
		}
		if err := c.Close(t.Context()); err != nil {
			t.Fatal(err)
		}
		if got := r.wire.take(); !slices.Equal(got, []exchange{{call: "auth/logout", status: http.StatusOK}}) {
			t.Fatalf("Close sent %+v, want auth/logout", got)
		}
	})
}

func TestQBittorrentSignsInAgainAndResendsWhenTheSessionEnds(t *testing.T) {
	inEachVersion(t, func(t *testing.T, version string) {
		r := newRig(t, fakeqbittorrent.Client{Version: version, SessionTimeout: time.Minute})
		c := r.session()
		orchard, lantern := r.releases[0], r.releases[1]
		if _, err := c.Version(t.Context()); err != nil {
			t.Fatal(err)
		}
		r.wire.take()
		time.Sleep(2 * time.Minute)
		r.add(c, orchard, "/downloads/Movies")
		login := signIn(version)[1]
		want := []exchange{{call: "torrents/add", status: http.StatusForbidden}, login, {call: "torrents/add", status: http.StatusOK}}
		if got := r.wire.take(); !slices.Equal(got, want) {
			t.Fatalf("an add after the session expired sent %+v, want %+v", got, want)
		}
		if got := r.status(c, orchard); got.Folder != "/downloads/Movies" || got.Label != "slipstream" {
			t.Fatalf("the add sent again: %+v, want it in /downloads/Movies labeled slipstream", got)
		}

		r.cue("restart", map[string]any{})
		r.wire.take()
		r.add(c, lantern, "")
		if got := only(r.wire.take(), "torrents/add", "auth/login"); !slices.Equal(got, want[1:]) {
			t.Fatalf("an add after a restart sent %+v, want %+v", got, want)
		}
		r.status(c, lantern)
	})
}

func TestQBittorrentSignsInWithAnAPIKey(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newRig(t, fakeqbittorrent.Client{Version: "5.2", Username: username, Password: password, APIKey: apiKey, BanAfter: 1})
		keyed := r.open(session{apiKey: apiKey})
		if _, err := keyed.Test(t.Context(), slipstream); err != nil {
			t.Fatal(err)
		}
		r.add(keyed, r.releases[0], "")
		r.status(keyed, r.releases[0])
		if got := r.wire.take(); slices.ContainsFunc(got, func(e exchange) bool { return e.status != http.StatusOK }) {
			t.Fatalf("a session with an API key sent %+v, want every request let in without signing in", got)
		}

		_, err := r.open(session{apiKey: "qbt_wrong"}).Test(t.Context(), slipstream)
		if problem(t, err, clients.WrongAPIKey).Cause.State() != clients.SignInRefused {
			t.Fatal("a wrong API key isn't a refused sign-in")
		}
		if got := r.wire.take(); len(got) != 1 {
			t.Fatalf("a wrong API key sent %+v, want one request", got)
		}
		if _, err := r.open(session{username: username, password: password}).Test(t.Context(), slipstream); err != nil {
			t.Fatalf("a wrong API key counted toward the ban: %v", err)
		}
	})
}

func TestQBittorrentRefusesAWrongPasswordOnceAndReportsTheBan(t *testing.T) {
	inEachVersion(t, func(t *testing.T, version string) {
		r := newRig(t, fakeqbittorrent.Client{Version: version, BanAfter: 2})
		wrong := r.open(session{username: username, password: "wrong"})
		_, err := wrong.Test(t.Context(), slipstream)
		if problem(t, err, clients.WrongCredentials).Cause.State() != clients.SignInRefused {
			t.Fatal("a wrong password isn't a refused sign-in")
		}
		if got := r.wire.take(); len(got) != 2 || got[1].call != "auth/login" {
			t.Fatalf("a wrong password sent %+v, want one sign-in", got)
		}
		_, err = wrong.Poll(t.Context(), nil)
		problem(t, err, clients.WrongCredentials)

		_, err = r.session().Test(t.Context(), slipstream)
		if problem(t, err, clients.Banned).Cause.State() != clients.SignInRefused {
			t.Fatal("the ban isn't a refused sign-in")
		}
		r.cue("restart", map[string]any{})
		if _, err := r.session().Test(t.Context(), slipstream); err != nil {
			t.Fatalf("the right password after a restart: %v", err)
		}
	})
}

func TestQBittorrentsHostCheckNeedsAttention(t *testing.T) {
	inEachVersion(t, func(t *testing.T, version string) {
		r := newRig(t, fakeqbittorrent.Client{Version: version, Domains: []string{"qbittorrent.lan"}})
		_, err := r.open(session{address: "http://qbittorrent.test:8080", username: username, password: password}).Test(t.Context(), slipstream)
		if p := problem(t, err, clients.HostRefused); p.Host != "qbittorrent.test" || p.Cause.State() != clients.NeedsAttention {
			t.Fatalf("the host check: %+v, want qbittorrent.test refused, needing attention", p)
		}
		if got := r.wire.take(); len(got) != 1 {
			t.Fatalf("the host check sent %+v, want no sign-in", got)
		}
		if _, err := r.open(session{address: "http://qbittorrent.lan:8080", username: username, password: password}).Test(t.Context(), slipstream); err != nil {
			t.Fatalf("a host in the server domains: %v", err)
		}
	})
}

func TestQBittorrentLetsInAnAddressThatNeedsNoSignIn(t *testing.T) {
	inEachVersion(t, func(t *testing.T, version string) {
		r := newRig(t, fakeqbittorrent.Client{Version: version, Username: username})
		c := r.open(session{})
		r.add(c, r.releases[0], "")
		r.status(c, r.releases[0])
		if got := r.wire.take(); slices.ContainsFunc(got, func(e exchange) bool { return e.call == "auth/login" }) {
			t.Fatalf("a session the client lets in sent %+v, want no sign-in", got)
		}
	})
}

func TestQBittorrentAddsWithTheCategoryAndFolder(t *testing.T) {
	inEachVersion(t, func(t *testing.T, version string) {
		r := newRig(t, fakeqbittorrent.Client{Version: version})
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
		if got.Name != "orchard.mkv" || got.Label != "slipstream" || got.Folder != "/downloads/Movies" || got.Size != size || got.Run != clients.Paused {
			t.Fatalf("after a paused Add: %+v, want orchard.mkv paused in category slipstream in /downloads/Movies, the duplicate add ignored", got)
		}
		if _, ok := r.poll(c, lantern); ok {
			t.Fatal("Poll lists a torrent nobody added")
		}
		r.add(c, lantern, "")
		if got := r.status(c, lantern); got.Folder != "/downloads" || got.Run != clients.Started {
			t.Fatalf("a torrent added with no folder: %+v, want it started in the default folder", got)
		}
		if folder, err := c.DefaultFolder(t.Context(), slipstream); err != nil || folder != "/downloads" {
			t.Fatalf("DefaultFolder once the add made the category = %q, %v, want /downloads", folder, err)
		}

		saved := newRig(t, fakeqbittorrent.Client{Version: version, Categories: map[string]string{"slipstream": "/downloads/slipstream"}})
		if folder, err := saved.session().DefaultFolder(t.Context(), slipstream); err != nil || folder != "/downloads/slipstream" {
			t.Fatalf("DefaultFolder of a category with a save path = %q, %v, want /downloads/slipstream", folder, err)
		}
	})
}

func TestQBittorrentReportsDownloadingQueuedAndFailed(t *testing.T) {
	inEachVersion(t, func(t *testing.T, version string) {
		r := newRig(t, fakeqbittorrent.Client{Version: version, Slots: 1})
		c := r.session()
		orchard, lantern := r.releases[0], r.releases[1]
		r.add(c, orchard, "/downloads/Movies")
		r.add(c, lantern, "/downloads/Movies")

		time.Sleep(4 * time.Second)
		got := r.status(c, orchard)
		if got.Run != clients.Started || got.Activity != clients.Downloading || got.Done != 400 || got.Finished || got.Rates.Down != 100 {
			t.Fatalf("downloading: %+v", got)
		}
		if got := r.status(c, lantern); got.Activity != clients.Queued || got.QueuePosition != 2 || got.Finished {
			t.Fatalf("past the queue's one slot: %+v, want queued second", got)
		}
		r.cue("error", map[string]any{"release": lantern.name, "message": "No space left on device"})
		if got := r.status(c, lantern); got.LocalError != "Errored" || got.Run != clients.Paused || got.Finished || got.StoppedAtLimit {
			t.Fatalf("a local error: %+v, want it stopped with qBittorrent's words and not finished", got)
		}
	})
}

func TestQBittorrentReportsFinishedUntilTheSeedRatioStopsIt(t *testing.T) {
	inEachVersion(t, func(t *testing.T, version string) {
		r := newRig(t, fakeqbittorrent.Client{Version: version, SeedRatio: 1})
		c := r.open(session{username: username, password: password, trackerEvery: 5 * time.Second})
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
		got = r.status(c, orchard)
		if !got.Finished || !slices.Equal(got.Trackers, []clients.TrackerMessage{{Message: "Unregistered torrent", Error: true}}) || got.LocalError != "" {
			t.Fatalf("a tracker error: %+v, want it as an error on a finished torrent", got)
		}
		r.cue("tracker", map[string]any{"release": orchard.name, "message": "Torrent not registered", "error": true})
		if got := r.status(c, orchard); got.Trackers[0].Message != "Unregistered torrent" {
			t.Fatalf("trackers read again before TrackerEvery: %+v", got.Trackers)
		}
		time.Sleep(5 * time.Second)
		if got := r.status(c, orchard); got.Trackers[0].Message != "Torrent not registered" {
			t.Fatalf("trackers after TrackerEvery: %+v, want the new message", got.Trackers)
		}
		r.cue("tracker", map[string]any{"release": orchard.name, "message": ""})

		time.Sleep(10 * time.Second)
		got = r.status(c, orchard)
		if !got.StoppedAtLimit || got.Run != clients.Paused || !got.Finished || got.Counters.Uploaded != size || got.Counters.Seeding != 10*time.Second {
			t.Fatalf("at the seed ratio: %+v, want stopped at the limit and still finished", got)
		}
	})
}

func TestQBittorrentReportsAMoveAsNotFinished(t *testing.T) {
	inEachVersion(t, func(t *testing.T, version string) {
		r := newRig(t, fakeqbittorrent.Client{Version: version, MoveFor: 5 * time.Second})
		c := r.session()
		orchard := r.releases[0]
		r.add(c, orchard, "")
		time.Sleep(12 * time.Second)
		if got := r.status(c, orchard); got.Activity != clients.Moving || got.Finished || got.Done != size {
			t.Fatalf("moving its files: %+v, want moving and not finished", got)
		}
		time.Sleep(5 * time.Second)
		if got := r.status(c, orchard); got.Activity != clients.Seeding || !got.Finished {
			t.Fatalf("after the move: %+v, want finished", got)
		}
	})
}

func TestQBittorrentWarnsAndRemovesAtItsSeedingLimit(t *testing.T) {
	inEachVersion(t, func(t *testing.T, version string) {
		r := newRig(t, fakeqbittorrent.Client{Version: version, SeedRatio: 1, AtLimit: fakeqbittorrent.RemoveAtLimit})
		c := r.session()
		tested, err := c.Test(t.Context(), slipstream)
		if err != nil || !slices.Equal(tested.Warnings, []clients.Warning{clients.RemovesAtLimit}) {
			t.Fatalf("Test = %+v, %v, want the RemovesAtLimit warning", tested, err)
		}
		orchard := r.releases[0]
		r.add(c, orchard, "/downloads/Movies")
		time.Sleep(15 * time.Second)
		r.status(c, orchard)
		time.Sleep(10 * time.Second)
		if s, ok := r.poll(c, orchard); ok {
			t.Fatalf("at the seed ratio qBittorrent still lists %+v, want it removed", s)
		}
		if _, err := os.Stat(filepath.Join(r.client.Dir, "Movies", "orchard.mkv")); err != nil {
			t.Fatalf("removed at the limit, its file: %v", err)
		}
	})
}

func TestQBittorrentPausesAndResumesInItsVersionsWords(t *testing.T) {
	inEachVersion(t, func(t *testing.T, version string) {
		r := newRig(t, fakeqbittorrent.Client{Version: version, Check: 2 * time.Second})
		c := r.session()
		orchard := r.releases[0]
		r.add(c, orchard, "")
		r.act(c, clients.Pause, orchard)
		time.Sleep(5 * time.Second)
		if got := r.status(c, orchard); got.Run != clients.Paused || got.Activity != clients.Downloading || got.Done != 0 || got.Finished {
			t.Fatalf("paused mid-download: %+v, want paused with nothing done and not finished", got)
		}
		for _, a := range []clients.Act{clients.Resume, clients.Recheck, clients.Announce} {
			r.act(c, a, orchard)
		}
		if got := r.status(c, orchard); got.Activity != clients.Checking || got.Run != clients.Started {
			t.Fatalf("rechecking after a resume: %+v, want checking and started", got)
		}
		time.Sleep(5 * time.Second)
		if got := r.status(c, orchard); got.Run != clients.Started || got.Activity != clients.Downloading || got.Done != 300 {
			t.Fatalf("resumed: %+v, want 300 bytes after the 2s check", got)
		}
	})
}

func TestQBittorrentReportsACheckedTorrentAsItWasAsked(t *testing.T) {
	inEachVersion(t, func(t *testing.T, version string) {
		r := newRig(t, fakeqbittorrent.Client{Version: version, Check: 2 * time.Second})
		c := r.session()
		orchard := r.releases[0]
		r.add(c, orchard, "")
		if got := r.status(c, orchard); got.Activity != clients.Checking || got.Run != clients.Started {
			t.Fatalf("checking after its add: %+v, want checking and started", got)
		}
		r.act(c, clients.Pause, orchard)
		if got := r.status(c, orchard); got.Activity != clients.Checking || got.Run != clients.Paused {
			t.Fatalf("paused while checking: %+v, want checking and paused", got)
		}
		time.Sleep(5 * time.Second)
		if got := r.status(c, orchard); got.Run != clients.Paused || got.Activity != clients.Downloading {
			t.Fatalf("paused after checking: %+v", got)
		}
	})
}

func TestQBittorrentPollsAHundredHashesARequest(t *testing.T) {
	inEachVersion(t, func(t *testing.T, version string) {
		r := newRig(t, fakeqbittorrent.Client{Version: version})
		c := r.session()
		orchard, lantern := r.releases[0], r.releases[1]
		r.add(c, orchard, "")
		r.add(c, lantern, "")
		hashes := []bencode.InfoHash{orchard.hash}
		for range 248 {
			var h bencode.InfoHash
			_, _ = rand.Read(h[:])
			hashes = append(hashes, h)
		}
		hashes = append(hashes, lantern.hash)
		r.wire.take()
		got, err := c.Poll(t.Context(), hashes)
		if _, ok := got[lantern.hash]; err != nil || len(got) != 2 || !ok {
			t.Fatalf("Poll of 250 hashes = %v, %v, want both torrents", got, err)
		}
		var sizes []int
		for _, e := range r.wire.take() {
			sizes = append(sizes, e.hashes)
		}
		if !slices.Equal(sizes, []int{100, 100, 50}) {
			t.Fatalf("Poll of 250 hashes asked for %v a request, want 100, 100 and 50", sizes)
		}
	})
}

func TestQBittorrentRemovesWithAndWithoutData(t *testing.T) {
	inEachVersion(t, func(t *testing.T, version string) {
		r := newRig(t, fakeqbittorrent.Client{Version: version})
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
		got, err := c.Poll(t.Context(), []bencode.InfoHash{orchard.hash, lantern.hash})
		if err != nil || len(got) != 0 {
			t.Fatalf("Poll after the removes = %v, %v, want neither listed", got, err)
		}
		movies := filepath.Join(r.client.Dir, "Movies")
		if _, err := os.Stat(filepath.Join(movies, "orchard.mkv")); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("removed with data, its file: %v", err)
		}
		if _, err := os.Stat(filepath.Join(movies, "lantern.mkv")); err != nil {
			t.Errorf("removed without data, its file: %v", err)
		}
	})
}

func TestQBittorrentsOwnRemovesAndLateAddsShowInPolls(t *testing.T) {
	inEachVersion(t, func(t *testing.T, version string) {
		r := newRig(t, fakeqbittorrent.Client{Version: version})
		c := r.session()
		orchard, lantern := r.releases[0], r.releases[1]
		r.add(c, orchard, "")
		r.cue("remove", map[string]any{"release": orchard.name})
		if _, ok := r.poll(c, orchard); ok {
			t.Fatal("Poll lists a torrent qBittorrent removed")
		}
		r.cue("drop-add", map[string]any{})
		r.add(c, orchard, "")
		if _, ok := r.poll(c, orchard); ok {
			t.Fatal("Poll lists a torrent qBittorrent dropped")
		}
		r.cue("late-add", map[string]any{"after": "30s"})
		r.add(c, lantern, "")
		if _, ok := r.poll(c, lantern); ok {
			t.Fatal("Poll lists a late add at once")
		}
		time.Sleep(30 * time.Second)
		r.status(c, lantern)
		r.cue("empty", map[string]any{})
		if got, err := c.Poll(t.Context(), []bencode.InfoHash{orchard.hash, lantern.hash}); err != nil || len(got) != 0 {
			t.Fatalf("Poll of an emptied qBittorrent = %v, %v", got, err)
		}
	})
}

func TestQBittorrentSpeaksOverLoopback(t *testing.T) {
	for _, version := range fakeqbittorrent.Versions {
		t.Run(version, func(t *testing.T) {
			q := fakeqbittorrent.Client{Node: node, Version: version, Username: username, Password: password, Dir: t.TempDir(), Folder: "/downloads", Rate: 1 << 20}
			catalog := faketest.Catalog(t, size, "orchard")
			f := faketest.Loopback(t, fakeqbittorrent.Playing(q), scenario.Scenario{Seed: "qbittorrent"}, faketest.WithReleases(catalog), faketest.As(node))
			a, err := clients.ParseAddress(f.Address.URL(""), clients.QBittorrent)
			if err != nil {
				t.Fatal(err)
			}
			c := qbittorrent.Open(clients.Session{
				HTTP: f.Client, Address: a, Username: username, Secrets: clients.Secrets{Password: password},
				Call: 10 * time.Second, Log: slog.New(slog.DiscardHandler),
			})
			rel := catalog.Releases[0]
			if err := c.Add(t.Context(), clients.Add{Metainfo: rel.Metainfo, Hash: rel.InfoHash, Label: slipstream}); err != nil {
				t.Fatal(err)
			}
			got, err := c.Poll(t.Context(), []bencode.InfoHash{rel.InfoHash})
			if err != nil || got[rel.InfoHash].Name != "orchard.mkv" {
				t.Fatalf("Poll = %+v, %v", got, err)
			}
		})
	}
}
