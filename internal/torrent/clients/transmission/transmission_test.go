package transmission_test

import (
	"bytes"
	"errors"
	"fmt"
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

	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/scenario"
	faketransmission "github.com/jatassi/slipstream/fakes/transmission"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/torrent/clients"
	"github.com/jatassi/slipstream/internal/torrent/clients/transmission"
)

const (
	node = "Media server"
	size = 1000
)

type release struct {
	name     string
	hash     bencode.InfoHash
	metainfo []byte
}

var slipstream, _ = clients.ParseLabel(clients.DefaultLabel, clients.Transmission)

// rig is one Node's fake Transmission and a session with it.
type rig struct {
	t        *testing.T
	served   faketest.Served
	daemon   faketransmission.Daemon
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
	json2  bool
	status int
}

func (w *wire) RoundTrip(req *http.Request) (*http.Response, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, fmt.Errorf("read the request body: %w", err)
	}
	req.Body = io.NopCloser(bytes.NewReader(body))
	res, err := w.inner.RoundTrip(req)
	if err != nil {
		return nil, fmt.Errorf("send the request: %w", err)
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.sent = append(w.sent, exchange{json2: bytes.Contains(body, []byte(`"jsonrpc":"2.0"`)), status: res.StatusCode})
	return res, nil
}

func (w *wire) take() []exchange {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := w.sent
	w.sent = nil
	return out
}

func newRig(t *testing.T, d faketransmission.Daemon) *rig {
	t.Helper()
	d.Node, d.Dir, d.Folder = node, t.TempDir(), "/downloads"
	if d.Rate == 0 {
		d.Rate = 100
	}
	catalog := faketest.Catalog(t, size, "orchard", "lantern")
	r := &rig{t: t, daemon: d}
	for _, rel := range catalog.Releases {
		r.releases = append(r.releases, release{name: rel.Name, hash: rel.InfoHash, metainfo: rel.Metainfo})
	}
	r.served = faketest.InMemory(t, faketransmission.Playing(d), scenario.Scenario{Seed: "transmission"},
		faketest.WithReleases(catalog), faketest.As(node))
	r.wire = &wire{inner: r.served.Client.Transport}
	return r
}

// open is a session at address, or at the fake's own address for "".
func (r *rig) open(address, username, password string) clients.Client {
	r.t.Helper()
	if address == "" {
		address = r.served.Address.URL("/transmission/rpc")
	}
	a, err := clients.ParseAddress(address, clients.Transmission)
	if err != nil {
		r.t.Fatal(err)
	}
	return transmission.Open(clients.Session{
		HTTP: &http.Client{Transport: r.wire}, Address: a, Username: username, Secrets: clients.Secrets{Password: password},
		Call: 30 * time.Second, Log: slog.New(slog.NewTextHandler(r.t.Output(), &slog.HandlerOptions{Level: slog.LevelDebug})),
	})
}

func (r *rig) client() clients.Client { return r.open("", r.daemon.Username, r.daemon.Password) }

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
		r.t.Fatalf("Transmission doesn't list %s", rel.name)
	}
	return s
}

func (r *rig) cue(name string, args map[string]any) {
	r.t.Helper()
	args["node"] = node
	r.served.Cue(r.t, "transmission-"+name, args)
}

// inEachVersion runs test against a fake Transmission at each version it
// plays, over an in-memory network under synctest.
func inEachVersion(t *testing.T, test func(t *testing.T, version string)) {
	for _, v := range faketransmission.Versions {
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

func TestTransmissionSpeaksTheProtocolItsVersionNames(t *testing.T) {
	wantVersion := map[string]string{"4.0": "4.0.5", "4.1": "4.1.1"}
	inEachVersion(t, func(t *testing.T, version string) {
		r := newRig(t, faketransmission.Daemon{Version: version})
		c := r.client()
		tested, err := c.Test(t.Context(), slipstream)
		if err != nil {
			t.Fatal(err)
		}
		if tested.Version.String() != wantVersion[version] || len(tested.Warnings) != 0 {
			t.Fatalf("Test = %+v, want version %s and no warnings", tested, wantVersion[version])
		}
		json2 := version == "4.1"
		if got := r.wire.take(); !slices.Equal(got, []exchange{{status: http.StatusConflict}, {json2: json2, status: http.StatusOK}}) {
			t.Fatalf("a first call sent %+v, want a 409 and the call again in its version's protocol", got)
		}

		r.poll(c, r.releases[0])
		if got := r.wire.take(); !slices.Equal(got, []exchange{{json2: json2, status: http.StatusOK}}) {
			t.Fatalf("a call on the session sent %+v, want one request", got)
		}

		r.cue("restart", map[string]any{})
		r.poll(c, r.releases[0])
		if got := r.wire.take(); !slices.Equal(got, []exchange{{json2: json2, status: http.StatusConflict}, {json2: json2, status: http.StatusOK}}) {
			t.Fatalf("a call after a restart sent %+v, want a 409 and the call again in the same protocol", got)
		}
	})
}

func TestTransmissionAddsWithTheLabelAndFolder(t *testing.T) {
	inEachVersion(t, func(t *testing.T, version string) {
		r := newRig(t, faketransmission.Daemon{Version: version})
		c := r.client()
		folder, err := c.DefaultFolder(t.Context(), slipstream)
		if err != nil || folder != "/downloads" {
			t.Fatalf("DefaultFolder = %q, %v, want /downloads", folder, err)
		}
		orchard, lantern := r.releases[0], r.releases[1]
		r.add(c, orchard, "/downloads/Movies")
		r.add(c, orchard, "/downloads/Elsewhere")
		got := r.status(c, orchard)
		if got.Name != "orchard.mkv" || got.Label != "slipstream" || got.Folder != "/downloads/Movies" || got.Size != size {
			t.Fatalf("after Add: %+v, want orchard.mkv labeled slipstream in /downloads/Movies, the duplicate add ignored", got)
		}
		if _, ok := r.poll(c, lantern); ok {
			t.Fatal("Poll lists a torrent nobody added")
		}
		r.add(c, lantern, "")
		if got := r.status(c, lantern); got.Folder != "/downloads" {
			t.Fatalf("a torrent added with no folder is in %q, want the default", got.Folder)
		}
	})
}

func TestTransmissionReportsDownloadingQueuedAndFailed(t *testing.T) {
	inEachVersion(t, func(t *testing.T, version string) {
		r := newRig(t, faketransmission.Daemon{Version: version, Slots: 1})
		c := r.client()
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
		if got := r.status(c, lantern); got.LocalError != "No space left on device" || got.Run != clients.Paused || got.Finished || got.StoppedAtLimit {
			t.Fatalf("a local error: %+v, want it stopped with the client's words and not finished", got)
		}
	})
}

func TestTransmissionReportsFinishedUntilTheSeedRatioStopsIt(t *testing.T) {
	inEachVersion(t, func(t *testing.T, version string) {
		r := newRig(t, faketransmission.Daemon{Version: version, SeedRatio: 1})
		c := r.client()
		orchard := r.releases[0]
		start := time.Now()
		r.add(c, orchard, "/downloads/Movies")
		time.Sleep(10 * time.Second)
		got := r.status(c, orchard)
		if !got.Finished || got.Activity != clients.Seeding || got.Done != size || !got.FinishedAt.Equal(start.Add(10*time.Second)) || got.StoppedAtLimit {
			t.Fatalf("complete: %+v, want finished and seeding from 10s", got)
		}
		files, err := c.Files(t.Context(), orchard.hash)
		if err != nil || !slices.Equal(files, []clients.File{{Index: 0, Path: "orchard.mkv", Size: size, Done: size}}) {
			t.Fatalf("Files = %+v, %v", files, err)
		}

		r.cue("tracker", map[string]any{"release": orchard.name, "message": "Tracker is down for maintenance"})
		got = r.status(c, orchard)
		if !got.Finished || !slices.Equal(got.Trackers, []clients.TrackerMessage{{Message: "Tracker is down for maintenance"}}) {
			t.Fatalf("a tracker warning: %+v, want it as a warning on a finished torrent", got)
		}
		r.cue("tracker", map[string]any{"release": orchard.name, "message": "Unregistered torrent", "error": true})
		if got := r.status(c, orchard); !slices.Equal(got.Trackers, []clients.TrackerMessage{{Message: "Unregistered torrent", Error: true}}) || got.LocalError != "" {
			t.Fatalf("a tracker error: %+v", got)
		}

		time.Sleep(10 * time.Second)
		got = r.status(c, orchard)
		if !got.StoppedAtLimit || got.Run != clients.Paused || !got.Finished || got.Counters.Uploaded != size || got.Counters.Seeding != 10*time.Second {
			t.Fatalf("at the seed ratio: %+v, want stopped at the limit and still finished", got)
		}
	})
}

func TestTransmissionReportsAVerifiedTorrentAsItWasAsked(t *testing.T) {
	inEachVersion(t, func(t *testing.T, version string) {
		r := newRig(t, faketransmission.Daemon{Version: version, Check: 2 * time.Second})
		c := r.client()
		orchard := r.releases[0]
		r.add(c, orchard, "")
		if got := r.status(c, orchard); got.Activity != clients.Checking || got.Run != clients.Started {
			t.Fatalf("verifying after its add: %+v, want checking and started", got)
		}
		if err := c.Act(t.Context(), clients.Pause, []bencode.InfoHash{orchard.hash}); err != nil {
			t.Fatal(err)
		}
		if got := r.status(c, orchard); got.Activity != clients.Checking || got.Run != clients.Paused {
			t.Fatalf("paused while verifying: %+v, want checking and paused", got)
		}
		time.Sleep(5 * time.Second)
		if got := r.status(c, orchard); got.Run != clients.Paused || got.Activity != clients.Downloading {
			t.Fatalf("paused after verifying: %+v", got)
		}
	})
}

func TestTransmissionStopsAndStartsTorrents(t *testing.T) {
	inEachVersion(t, func(t *testing.T, version string) {
		r := newRig(t, faketransmission.Daemon{Version: version, Check: 2 * time.Second})
		c := r.client()
		orchard := r.releases[0]
		r.add(c, orchard, "")
		if err := c.Act(t.Context(), clients.Pause, []bencode.InfoHash{orchard.hash}); err != nil {
			t.Fatal(err)
		}
		time.Sleep(5 * time.Second)
		if got := r.status(c, orchard); got.Run != clients.Paused || got.Activity != clients.Downloading || got.Done != 0 || got.Finished {
			t.Fatalf("paused mid-download: %+v, want paused with nothing done and not finished", got)
		}
		for _, a := range []clients.Act{clients.Resume, clients.Recheck, clients.Announce} {
			if err := c.Act(t.Context(), a, []bencode.InfoHash{orchard.hash}); err != nil {
				t.Fatalf("Act(%d): %v", a, err)
			}
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

func TestTransmissionRemovesWithAndWithoutData(t *testing.T) {
	inEachVersion(t, func(t *testing.T, version string) {
		r := newRig(t, faketransmission.Daemon{Version: version})
		c := r.client()
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
		movies := filepath.Join(r.daemon.Dir, "Movies")
		if _, err := os.Stat(filepath.Join(movies, "orchard.mkv")); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("removed with data, its file: %v", err)
		}
		if _, err := os.Stat(filepath.Join(movies, "lantern.mkv")); err != nil {
			t.Errorf("removed without data, its file: %v", err)
		}
	})
}

func TestTransmissionsOwnRemovesAndLateAddsShowInPolls(t *testing.T) {
	inEachVersion(t, func(t *testing.T, version string) {
		r := newRig(t, faketransmission.Daemon{Version: version})
		c := r.client()
		orchard, lantern := r.releases[0], r.releases[1]
		r.add(c, orchard, "")
		r.cue("remove", map[string]any{"release": orchard.name})
		if _, ok := r.poll(c, orchard); ok {
			t.Fatal("Poll lists a torrent Transmission removed")
		}
		r.cue("drop-add", map[string]any{})
		r.add(c, orchard, "")
		if _, ok := r.poll(c, orchard); ok {
			t.Fatal("Poll lists a torrent Transmission dropped")
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
			t.Fatalf("Poll of an emptied Transmission = %v, %v", got, err)
		}
	})
}

func TestTransmissionRefusesAWrongPasswordOnce(t *testing.T) {
	inEachVersion(t, func(t *testing.T, version string) {
		r := newRig(t, faketransmission.Daemon{Version: version, Username: "jackson", Password: "orchard", Lockout: 2})
		wrong, right := r.open("", "jackson", "wrong"), r.client()
		_, err := wrong.Test(t.Context(), slipstream)
		if problem(t, err, clients.WrongCredentials).Cause.State() != clients.SignInRefused {
			t.Fatal("a wrong password isn't a refused sign-in")
		}
		if got := r.wire.take(); len(got) != 1 || got[0].status != http.StatusUnauthorized {
			t.Fatalf("a wrong password sent %+v, want one request", got)
		}
		if _, err := right.Test(t.Context(), slipstream); err != nil {
			t.Fatalf("the right password after one failure: %v", err)
		}
		for range 2 {
			_, err = wrong.Poll(t.Context(), nil)
			problem(t, err, clients.WrongCredentials)
		}
		for _, c := range []clients.Client{wrong, right} {
			_, err = c.Poll(t.Context(), nil)
			if problem(t, err, clients.LockedOut).Cause.State() != clients.SignInRefused {
				t.Fatal("the lockout isn't a refused sign-in")
			}
		}
	})
}

func TestTransmissionsAddressAndHostChecksNeedAttention(t *testing.T) {
	inEachVersion(t, func(t *testing.T, version string) {
		listed := newRig(t, faketransmission.Daemon{Version: version, Whitelist: []string{"192.168.1.*"}})
		_, err := listed.client().Test(t.Context(), slipstream)
		if problem(t, err, clients.AddressRefused).Cause.State() != clients.NeedsAttention {
			t.Fatal("the rpc-whitelist refusal doesn't need attention")
		}

		hosts := newRig(t, faketransmission.Daemon{Version: version})
		_, err = hosts.open("http://transmission.test:9091", "", "").Test(t.Context(), slipstream)
		if p := problem(t, err, clients.HostRefused); p.Host != "transmission.test" || p.Cause.State() != clients.NeedsAttention {
			t.Fatalf("the host check: %+v, want transmission.test refused, needing attention", p)
		}
		if _, err := hosts.open("http://localhost:9091", "", "").Test(t.Context(), slipstream); err != nil {
			t.Fatalf("the host check refused localhost: %v", err)
		}

		signed := newRig(t, faketransmission.Daemon{Version: version, Username: "jackson", Password: "orchard"})
		if _, err := signed.open("http://transmission.test:9091", "jackson", "orchard").Test(t.Context(), slipstream); err != nil {
			t.Fatalf("with a password any host passes: %v", err)
		}
	})
}

func TestTransmissionSpeaksOverLoopback(t *testing.T) {
	for _, version := range faketransmission.Versions {
		t.Run(version, func(t *testing.T) {
			d := faketransmission.Daemon{Node: node, Version: version, Dir: t.TempDir(), Folder: "/downloads", Rate: 1 << 20}
			catalog := faketest.Catalog(t, size, "orchard")
			f := faketest.Loopback(t, faketransmission.Playing(d), scenario.Scenario{Seed: "transmission"}, faketest.WithReleases(catalog), faketest.As(node))
			a, err := clients.ParseAddress(f.Address.URL(""), clients.Transmission)
			if err != nil {
				t.Fatal(err)
			}
			c := transmission.Open(clients.Session{HTTP: f.Client, Address: a, Call: 10 * time.Second, Log: slog.New(slog.DiscardHandler)})
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
