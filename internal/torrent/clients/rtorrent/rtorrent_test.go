package rtorrent_test

import (
	"bytes"
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
	fakertorrent "github.com/jatassi/slipstream/fakes/rtorrent"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/torrent/clients"
	"github.com/jatassi/slipstream/internal/torrent/clients/rtorrent"
)

const (
	node = "Media server"
	size = 1000
)

var slipstream, _ = clients.ParseLabel(clients.DefaultLabel, clients.RTorrent)

type release struct {
	name     string
	hash     bencode.InfoHash
	metainfo []byte
}

// rig is one Node's fake rTorrent and sessions with it.
type rig struct {
	t        *testing.T
	served   faketest.Served
	client   fakertorrent.Client
	releases []release
	wire     *wire
}

// wire counts the requests sessions sent.
type wire struct {
	mu    sync.Mutex
	inner http.RoundTripper
	sent  int
}

func (w *wire) RoundTrip(req *http.Request) (*http.Response, error) {
	w.mu.Lock()
	w.sent++
	w.mu.Unlock()
	return w.inner.RoundTrip(req) //nolint:wrapcheck // the session wraps it
}

func (w *wire) take() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	n := w.sent
	w.sent = 0
	return n
}

func newRig(t *testing.T, c fakertorrent.Client) *rig {
	t.Helper()
	c.Node, c.Dir, c.Folder = node, t.TempDir(), "/downloads"
	if c.Rate == 0 {
		c.Rate = 100
	}
	catalog := faketest.Catalog(t, size, "orchard", "lantern")
	r := &rig{t: t, client: c}
	for _, rel := range catalog.Releases {
		r.releases = append(r.releases, release{name: rel.Name, hash: rel.InfoHash, metainfo: rel.Metainfo})
	}
	r.served = faketest.InMemory(t, fakertorrent.Playing(c), scenario.Scenario{Seed: "rtorrent"}, faketest.WithReleases(catalog), faketest.As(node))
	r.wire = &wire{inner: r.served.Client.Transport}
	return r
}

func (r *rig) open(username, password string) clients.Client {
	r.t.Helper()
	a, err := clients.ParseAddress(r.served.Address.URL(""), clients.RTorrent)
	if err != nil {
		r.t.Fatal(err)
	}
	return rtorrent.Open(clients.Session{
		HTTP: &http.Client{Transport: r.wire}, Address: a, Username: username, Secrets: clients.Secrets{Password: password},
		Call: 30 * time.Second, Log: slog.New(slog.NewTextHandler(r.t.Output(), &slog.HandlerOptions{Level: slog.LevelDebug})),
	})
}

func (r *rig) session() clients.Client { return r.open(r.client.Username, r.client.Password) }

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
		r.t.Fatalf("rTorrent doesn't list %s", rel.name)
	}
	return s
}

func (r *rig) cue(name string, args map[string]any) {
	r.t.Helper()
	args["node"] = node
	r.served.Cue(r.t, "rtorrent-"+name, args)
}

func (r *rig) act(c clients.Client, a clients.Act, rel release) {
	r.t.Helper()
	if err := c.Act(r.t.Context(), a, []bencode.InfoHash{rel.hash}); err != nil {
		r.t.Fatalf("Act(%d): %v", a, err)
	}
}

// inRTorrent is a person calling method on rel's torrent in ruTorrent or
// rTorrent's own screen.
func (r *rig) inRTorrent(method string, rel release) {
	r.t.Helper()
	body := `<?xml version="1.0"?><methodCall><methodName>` + method + `</methodName><params><param><value><string>` +
		rel.hash.String() + `</string></value></param></params></methodCall>`
	req, err := http.NewRequestWithContext(r.t.Context(), http.MethodPost, r.served.Address.URL("/RPC2"), strings.NewReader(body))
	if err != nil {
		r.t.Fatal(err)
	}
	req.SetBasicAuth(r.client.Username, r.client.Password)
	res, err := r.served.Client.Do(req)
	if err != nil {
		r.t.Fatal(err)
	}
	_ = res.Body.Close()
}

// inEachVersion runs test against a fake rTorrent at each version it
// plays, over an in-memory network under synctest.
func inEachVersion(t *testing.T, test func(t *testing.T, version string)) {
	for _, v := range fakertorrent.Versions {
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

// multiFile is a torrent of two files under the folder harbor, whose
// data no peer sends.
func multiFile(t *testing.T) release {
	t.Helper()
	pieces := bytes.Repeat([]byte{0x5a}, 20)
	meta, err := bencode.Encode(map[string]any{
		"announce": "https://127.0.0.1/announce/faketest",
		"info": map[string]any{
			"name": "harbor", "piece length": int64(1 << 14), "pieces": pieces,
			"files": []any{
				map[string]any{"length": int64(600), "path": []any{"harbor.mkv"}},
				map[string]any{"length": int64(40), "path": []any{"subs", "en.srt"}},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	m, err := bencode.DecodeMetainfo(meta)
	if err != nil {
		t.Fatal(err)
	}
	return release{name: "harbor", hash: m.InfoHash, metainfo: meta}
}

func TestRTorrentReadsItsVersionAndDefaultFolder(t *testing.T) {
	wantVersion := map[string]string{"0.9.8": "0.9.8", "0.16": "0.16.27"}
	inEachVersion(t, func(t *testing.T, version string) {
		r := newRig(t, fakertorrent.Client{Version: version})
		c := r.session()
		tested, err := c.Test(t.Context(), slipstream)
		if err != nil {
			t.Fatal(err)
		}
		if tested.Version.String() != wantVersion[version] || len(tested.Warnings) != 0 || tested.MovesAfterFinish {
			t.Fatalf("Test = %+v, want %s and no warnings", tested, wantVersion[version])
		}
		if v, err := c.Version(t.Context()); err != nil || v != tested.Version {
			t.Fatalf("Version = %s, %v, want %s", v, err, tested.Version)
		}
		if folder, err := c.DefaultFolder(t.Context(), slipstream); err != nil || folder != "/downloads" {
			t.Fatalf("DefaultFolder = %q, %v, want directory.default's /downloads", folder, err)
		}
	})
}

func TestRTorrentAddsWithTheLabelAndFolder(t *testing.T) {
	inEachVersion(t, func(t *testing.T, version string) {
		r := newRig(t, fakertorrent.Client{Version: version})
		c := r.session()
		orchard, lantern := r.releases[0], r.releases[1]
		label, err := clients.ParseLabel(`Films, "4K"`, clients.RTorrent)
		if err != nil {
			t.Fatal(err)
		}
		folder := `/downloads/My Movies, \ 4K`
		if err := c.Add(t.Context(), clients.Add{Metainfo: orchard.metainfo, Hash: orchard.hash, Folder: folder, Label: label, Paused: true}); err != nil {
			t.Fatal(err)
		}
		r.add(c, orchard, "/downloads/Elsewhere")
		got := r.status(c, orchard)
		if got.Name != "orchard.mkv" || got.Label != label.String() || got.Folder != folder || got.Size != size || got.Run != clients.Paused || got.Finished {
			t.Fatalf("after a paused Add: %+v, want orchard.mkv paused in %q labeled %q, the duplicate add ignored", got, folder, label)
		}
		if _, ok := r.poll(c, lantern); ok {
			t.Fatal("Poll lists a torrent nobody added")
		}
		r.add(c, lantern, "")
		if got := r.status(c, lantern); got.Folder != "/downloads" || got.Run != clients.Started || got.Label != "slipstream" {
			t.Fatalf("a torrent added with no folder: %+v, want it started in directory.default", got)
		}
	})
}

func TestRTorrentAddsAMultiFileTorrentUnderItsName(t *testing.T) {
	inEachVersion(t, func(t *testing.T, version string) {
		r := newRig(t, fakertorrent.Client{Version: version})
		c := r.session()
		harbor := multiFile(t)
		r.add(c, harbor, "/downloads/TV")
		if got := r.status(c, harbor); got.Name != "harbor" || got.Folder != "/downloads/TV" || got.Size != 640 {
			t.Fatalf("a multi-file torrent: %+v, want its folder as given, as the other kinds report it", got)
		}
		files, err := c.Files(t.Context(), harbor.hash)
		want := []clients.File{{Index: 0, Path: "harbor/harbor.mkv", Size: 600}, {Index: 1, Path: "harbor/subs/en.srt", Size: 40}}
		if err != nil || !slices.Equal(files, want) {
			t.Fatalf("Files = %+v, %v, want %+v", files, err, want)
		}
	})
}

func TestRTorrentListsALateHashOnALaterPoll(t *testing.T) {
	inEachVersion(t, func(t *testing.T, version string) {
		r := newRig(t, fakertorrent.Client{Version: version})
		c := r.session()
		orchard := r.releases[0]
		r.cue("late-add", map[string]any{"after": "30s"})
		r.add(c, orchard, "/downloads/Movies")
		if _, ok := r.poll(c, orchard); ok {
			t.Fatal("Poll lists a late add at once")
		}
		time.Sleep(30 * time.Second)
		if got := r.status(c, orchard); got.Folder != "/downloads/Movies" || got.Label != "slipstream" {
			t.Fatalf("a late add once rTorrent lists it: %+v, want it in its folder with its label", got)
		}
	})
}

func TestRTorrentPollsEveryHashInOneRequest(t *testing.T) {
	inEachVersion(t, func(t *testing.T, version string) {
		r := newRig(t, fakertorrent.Client{Version: version})
		c := r.session()
		orchard, lantern, harbor := r.releases[0], r.releases[1], multiFile(t)
		r.add(c, orchard, "")
		r.add(c, harbor, "")
		r.wire.take()
		got, err := c.Poll(t.Context(), []bencode.InfoHash{orchard.hash, lantern.hash, harbor.hash})
		if err != nil {
			t.Fatal(err)
		}
		if n := r.wire.take(); n != 1 {
			t.Fatalf("a poll of three hashes sent %d requests, want one system.multicall", n)
		}
		if _, ok := got[lantern.hash]; ok || got[orchard.hash].Name != "orchard.mkv" || got[harbor.hash].Name != "harbor" || len(got) != 2 {
			t.Fatalf("Poll = %+v, want orchard and harbor and not lantern, which rTorrent doesn't hold", got)
		}
	})
}

func TestRTorrentReportsDownloadingFinishedAndFailed(t *testing.T) {
	inEachVersion(t, func(t *testing.T, version string) {
		r := newRig(t, fakertorrent.Client{Version: version})
		c := r.session()
		orchard, lantern := r.releases[0], r.releases[1]
		start := time.Now()
		r.add(c, orchard, "/downloads/Movies")
		r.add(c, lantern, "/downloads/Movies")

		time.Sleep(4 * time.Second)
		got := r.status(c, orchard)
		if got.Run != clients.Started || got.Activity != clients.Downloading || got.Done != 400 || got.Finished || got.Rates.Down != 100 || got.Rates.Peers != 1 || got.Rates.Seeds != 1 || !got.FinishedAt.IsZero() {
			t.Fatalf("downloading: %+v", got)
		}
		r.cue("error", map[string]any{"release": lantern.name, "message": "No space left on device"})
		if got := r.status(c, lantern); got.LocalError != "No space left on device" || got.Run != clients.Paused || got.Finished || got.Trackers != nil {
			t.Fatalf("a local error: %+v, want it paused with rTorrent's message and not finished", got)
		}

		time.Sleep(11 * time.Second)
		got = r.status(c, orchard)
		if !got.Finished || got.Activity != clients.Seeding || got.Done != size || !got.FinishedAt.Equal(start.Add(10*time.Second)) ||
			got.Counters.Seeding != 5*time.Second || got.Counters.Uploaded != 500 || got.Counters.Downloaded != size || got.Rates.Up != 100 {
			t.Fatalf("complete: %+v, want finished at 10s and seeding for 5s", got)
		}
		files, err := c.Files(t.Context(), orchard.hash)
		if err != nil || !slices.Equal(files, []clients.File{{Index: 0, Path: "orchard.mkv", Size: size, Done: size}}) {
			t.Fatalf("Files = %+v, %v", files, err)
		}
		if _, err := os.Stat(filepath.Join(r.client.Dir, "Movies", "orchard.mkv")); err != nil {
			t.Fatalf("complete, its file: %v", err)
		}
	})
}

func TestRTorrentReadsTrackerFailuresFromItsMessage(t *testing.T) {
	inEachVersion(t, func(t *testing.T, version string) {
		r := newRig(t, fakertorrent.Client{Version: version})
		c := r.session()
		orchard := r.releases[0]
		r.add(c, orchard, "")
		time.Sleep(15 * time.Second)
		r.cue("tracker", map[string]any{"release": orchard.name, "message": "Unregistered torrent", "error": true})
		if got := r.status(c, orchard); !got.Finished || got.LocalError != "" || !slices.Equal(got.Trackers, []clients.TrackerMessage{{Message: "Unregistered torrent", Error: true}}) {
			t.Fatalf("a tracker's failure: %+v, want it as a tracker error on a finished torrent", got)
		}
		r.cue("tracker", map[string]any{"release": orchard.name, "message": "Slow down"})
		if got := r.status(c, orchard); got.Trackers != nil || got.LocalError != "" {
			t.Fatalf("a tracker's warning: %+v, want nothing, since libtorrent passes no warning on", got)
		}
		r.cue("tracker", map[string]any{"release": orchard.name, "message": "torrent not registered with this tracker"})
		got := r.status(c, orchard)
		if version == "0.9.8" && got.Trackers != nil {
			t.Fatalf("an unregistered warning at 0.9.8: %+v, want nothing", got)
		}
		if version == "0.16" && !slices.Equal(got.Trackers, []clients.TrackerMessage{{Message: "torrent not registered with this tracker", Error: true}}) {
			t.Fatalf("an unregistered warning at 0.16: %+v, want it as a tracker error, as libtorrent 0.16 reads it", got)
		}
	})
}

func TestRTorrentPausesResumesAndRechecks(t *testing.T) {
	inEachVersion(t, func(t *testing.T, version string) {
		r := newRig(t, fakertorrent.Client{Version: version, Check: 2 * time.Second})
		c := r.session()
		orchard := r.releases[0]
		r.add(c, orchard, "")
		if got := r.status(c, orchard); got.Activity != clients.Checking || got.Run != clients.Started {
			t.Fatalf("hashing after its add: %+v", got)
		}
		r.act(c, clients.Pause, orchard)
		time.Sleep(5 * time.Second)
		if got := r.status(c, orchard); got.Run != clients.Paused || got.Activity != clients.Downloading || got.Done != 0 {
			t.Fatalf("stopped: %+v, want paused with nothing done", got)
		}
		for _, a := range []clients.Act{clients.Resume, clients.Recheck, clients.Announce} {
			r.act(c, a, orchard)
		}
		if got := r.status(c, orchard); got.Activity != clients.Checking || got.Run != clients.Started {
			t.Fatalf("rehashing after a resume: %+v", got)
		}
		time.Sleep(5 * time.Second)
		if got := r.status(c, orchard); got.Run != clients.Started || got.Activity != clients.Downloading || got.Done != 300 {
			t.Fatalf("resumed: %+v, want 300 bytes after the 2s rehash", got)
		}

		r.inRTorrent("d.pause", orchard)
		if got := r.status(c, orchard); got.Run != clients.Paused {
			t.Fatalf("paused in rTorrent: %+v, want it paused", got)
		}
		r.act(c, clients.Resume, orchard)
		if got := r.status(c, orchard); got.Run != clients.Started || got.Activity != clients.Downloading {
			t.Fatalf("resumed after a pause in rTorrent: %+v, want it downloading", got)
		}
	})
}

func TestRTorrentRemovesTheTorrentAndKeepsItsData(t *testing.T) {
	inEachVersion(t, func(t *testing.T, version string) {
		r := newRig(t, fakertorrent.Client{Version: version})
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
			t.Fatalf("removing a torrent rTorrent no longer holds: %v", err)
		}
		got, err := c.Poll(t.Context(), []bencode.InfoHash{orchard.hash, lantern.hash})
		if err != nil || len(got) != 0 {
			t.Fatalf("Poll after the removes = %v, %v, want neither listed", got, err)
		}
		for _, rel := range []release{orchard, lantern} {
			if _, err := os.Stat(filepath.Join(r.client.Dir, "Movies", rel.name+".mkv")); err != nil {
				t.Errorf("d.erase deletes no data, so %s's file stays for the Node to delete: %v", rel.name, err)
			}
		}
		if err := c.Act(t.Context(), clients.Pause, []bencode.InfoHash{orchard.hash}); err != nil {
			t.Fatalf("pausing a torrent rTorrent no longer holds: %v", err)
		}
	})
}

func TestRTorrentsOwnRemovesAndDroppedAddsShowInPolls(t *testing.T) {
	inEachVersion(t, func(t *testing.T, version string) {
		r := newRig(t, fakertorrent.Client{Version: version})
		c := r.session()
		orchard := r.releases[0]
		r.add(c, orchard, "")
		r.cue("remove", map[string]any{"release": orchard.name})
		if _, ok := r.poll(c, orchard); ok {
			t.Fatal("Poll lists a torrent rTorrent removed")
		}
		r.cue("drop-add", map[string]any{})
		r.add(c, orchard, "")
		if _, ok := r.poll(c, orchard); ok {
			t.Fatal("Poll lists a torrent rTorrent dropped")
		}
		r.add(c, orchard, "")
		r.cue("empty", map[string]any{})
		if got, err := c.Poll(t.Context(), []bencode.InfoHash{orchard.hash}); err != nil || len(got) != 0 {
			t.Fatalf("Poll of an emptied rTorrent = %v, %v", got, err)
		}
		if _, err := c.Files(t.Context(), orchard.hash); err == nil {
			t.Fatal("Files of a torrent rTorrent doesn't hold is no error")
		}
	})
}

func TestRTorrentSignsInWithBasicAuthWithoutLockingOut(t *testing.T) {
	inEachVersion(t, func(t *testing.T, version string) {
		r := newRig(t, fakertorrent.Client{Version: version, Username: "jackson", Password: "orchard"})
		for _, c := range []clients.Client{r.open("jackson", "wrong"), r.open("", "")} {
			for range 10 {
				_, err := c.Test(t.Context(), slipstream)
				if problem(t, err, clients.WrongCredentials).Cause.State() != clients.SignInRefused {
					t.Fatal("nginx's 401 isn't a refused sign-in")
				}
			}
		}
		if _, err := r.session().Test(t.Context(), slipstream); err != nil {
			t.Fatalf("the right password after twenty refused: %v", err)
		}

		open := newRig(t, fakertorrent.Client{Version: version})
		if _, err := open.open("anyone", "anything").Test(t.Context(), slipstream); err != nil {
			t.Fatalf("credentials for an nginx without auth_basic: %v", err)
		}
	})
}

func TestRTorrentSpeaksOverLoopback(t *testing.T) {
	for _, version := range fakertorrent.Versions {
		t.Run(version, func(t *testing.T) {
			c := fakertorrent.Client{Node: node, Version: version, Username: "jackson", Password: "orchard", Dir: t.TempDir(), Folder: "/downloads", Rate: 1 << 20}
			catalog := faketest.Catalog(t, size, "orchard")
			f := faketest.Loopback(t, fakertorrent.Playing(c), scenario.Scenario{Seed: "rtorrent"}, faketest.WithReleases(catalog), faketest.As(node))
			a, err := clients.ParseAddress(f.Address.URL(""), clients.RTorrent)
			if err != nil {
				t.Fatal(err)
			}
			s := rtorrent.Open(clients.Session{HTTP: f.Client, Address: a, Username: c.Username, Secrets: clients.Secrets{Password: c.Password}, Call: 10 * time.Second, Log: slog.New(slog.DiscardHandler)})
			rel := catalog.Releases[0]
			if err := s.Add(t.Context(), clients.Add{Metainfo: rel.Metainfo, Hash: rel.InfoHash, Label: slipstream}); err != nil {
				t.Fatal(err)
			}
			got, err := s.Poll(t.Context(), []bencode.InfoHash{rel.InfoHash})
			if err != nil || got[rel.InfoHash].Name != "orchard.mkv" || got[rel.InfoHash].Label != "slipstream" {
				t.Fatalf("Poll = %+v, %v", got, err)
			}
		})
	}
}
