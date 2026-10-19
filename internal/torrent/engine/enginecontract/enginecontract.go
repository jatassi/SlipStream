// Package enginecontract is the one suite every engine.Engine passes: the
// libtorrent session over real sockets and enginetest's in-memory World
// inside a synctest bubble.
package enginecontract

import (
	"crypto/sha1" //nolint:gosec // BitTorrent v1 piece hashes are SHA-1 by spec (BEP 3)
	"crypto/sha256"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/torrent/engine"
	"github.com/jatassi/slipstream/internal/torrent/engine/enginetest"
)

// Harness is an engine's world as the suite needs it.
type Harness interface {
	// Within runs f where the harness's clock rules: a synctest bubble for
	// enginetest, as is for libtorrent.
	Within(t *testing.T, f func(*testing.T))
	// Open opens an engine, on loopback's any port unless cfg names a listen
	// interface, closed when t ends.
	Open(t *testing.T, cfg engine.Config) engine.Engine
	Tracker(t *testing.T, rules enginetest.TrackerRules) Tracker
	// Seed has a peer serve the release whose metainfo is meta and whose
	// files lie under dir, and returns what has a torrent on an engine
	// reach that peer.
	Seed(t *testing.T, meta []byte, dir string) (reach func(engine.Engine, engine.Handle))
	// TakePort holds a port on loopback for as long as t runs.
	TakePort(t *testing.T) netip.AddrPort
	// Incoming has a peer connect to e's listener, waits until e took the
	// connection, and returns the peer's address.
	Incoming(t *testing.T, e engine.Engine) netip.Addr
	// Until waits for cond on the harness's clock, failing t when it takes
	// too long: what says what it waited for.
	Until(t *testing.T, what string, cond func() bool)
}

type Tracker struct {
	// URL ends in /announce, which libtorrent requires of a loopback tracker.
	URL string
	// Announces lists the announces the tracker received, oldest first.
	Announces func() []enginetest.Announce
	// Answer answers the started announces the tracker holds, and every
	// later one at once.
	Answer func()
}

func Run(t *testing.T, h Harness) {
	for _, tc := range []struct {
		name string
		test func(*testing.T, Harness)
	}{
		{"AddWaitsPausedUntilResume", addWaitsPausedUntilResume},
		{"AnAddFromResumeDataSkipsTheCheck", anAddFromResumeDataSkipsTheCheck},
		{"TheBaselineRaisesCounters", theBaselineRaisesCounters},
		{"PauseSendsStoppedOnlyAfterStartedWasAnswered", pauseSendsStoppedOnlyAfterStartedWasAnswered},
		{"RemoveKeepsTheFiles", removeKeepsTheFiles},
		{"RemoveWithFilesDeletesTheDownloadCopyAndItsFolder", removeWithFilesDeletesTheDownloadCopyAndItsFolder},
		{"UpdatesCarryOnlyChangedTorrents", updatesCarryOnlyChangedTorrents},
		{"UpdatesCarryEveryTorrentAfterDropped", updatesCarryEveryTorrentAfterDropped},
		{"TrackerListCarriesNextAndMinAnnounce", trackerListCarriesNextAndMinAnnounce},
		{"TheAnnounceFloorLengthensAShortInterval", theAnnounceFloorLengthensAShortInterval},
		{"AWarningIsReportedAndListed", aWarningIsReportedAndListed},
		{"AFailureWithoutIntervalWaits1800s", aFailureWithoutIntervalWaits1800s},
		{"ReannounceMovesMinAnnounce", reannounceMovesMinAnnounce},
		{"ReannounceIgnoringMinIntervalAnnouncesNow", reannounceIgnoringMinIntervalAnnouncesNow},
		{"RecheckSendsNoAnnounceAndEndsPaused", recheckSendsNoAnnounceAndEndsPaused},
		{"ATakenPortStaysTakenAndTheSessionRuns", aTakenPortStaysTakenAndTheSessionRuns},
		{"ConfigureAppliesLive", configureAppliesLive},
		{"StatsCountThePayload", statsCountThePayload},
		{"AnIncomingPeerIsReportedWhileWatched", anIncomingPeerIsReportedWhileWatched},
		{"TheAnnouncedPeerIDAndUserAgentAreTheVersions", theAnnouncedPeerIDAndUserAgentAreTheVersions},
		{"CloseSendsStopped", closeSendsStopped},
	} {
		t.Run(tc.name, func(t *testing.T) { h.Within(t, func(t *testing.T) { tc.test(t, h) }) })
	}
}

func addWaitsPausedUntilResume(t *testing.T, h Harness) {
	tr := h.Tracker(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
	e := h.Open(t, engine.Config{})
	r := record(e)
	rel := release(t, "Waits", tr.URL)
	th := add(t, e, engine.AddParams{Metainfo: rel.meta, SavePath: rel.dir, SeedMode: true})
	if s := awaitStatus(t, h, e, r, th, func(engine.Status) bool { return true }); !s.Paused {
		t.Fatalf("the added torrent reads %+v, want paused", s)
	}
	if got := tr.Announces(); len(got) != 0 {
		t.Fatalf("before Resume the tracker received %+v, want nothing", got)
	}
	if err := e.Resume(th); err != nil {
		t.Fatal(err)
	}
	h.Until(t, "the tracker hears from the resumed torrent", func() bool { return len(tr.Announces()) > 0 })
	if got := tr.Announces()[0]; got.Event != "started" || got.InfoHash != rel.hash {
		t.Fatalf("after Resume the tracker first received %+v, want started for %s", got, rel.hash)
	}
}

func anAddFromResumeDataSkipsTheCheck(t *testing.T, h Harness) {
	e := h.Open(t, engine.Config{})
	r := record(e)
	baseline := engine.Totals{Uploaded: 1000, Downloaded: 2000, Active: time.Hour, Seeding: 30 * time.Minute}
	rel, data := resumeData(t, h, e, r, baseline)

	th := add(t, e, engine.AddParams{Resume: data, SavePath: rel.dir})
	if err := e.Resume(th); err != nil {
		t.Fatal(err)
	}
	s := awaitStatus(t, h, e, r, th, func(s engine.Status) bool { return s.State == engine.StateSeeding })
	if checked(r.seen, th) {
		t.Fatalf("the torrent added from resume data checked its files: events %+v", r.seen)
	}
	if s.Done != int64(len(rel.payload)) || s.Wanted != s.Done {
		t.Fatalf("the torrent added from resume data reads %d of %d bytes done, want all %d", s.Done, s.Wanted, len(rel.payload))
	}
	if !atLeast(s.Totals, baseline) || s.Totals.Uploaded != baseline.Uploaded || s.Totals.Downloaded != baseline.Downloaded {
		t.Fatalf("the torrent added from resume data counts %+v, want the %+v its resume data holds", s.Totals, baseline)
	}
}

func theBaselineRaisesCounters(t *testing.T, h Harness) {
	e := h.Open(t, engine.Config{})
	r := record(e)
	saved := engine.Totals{Uploaded: 1000, Downloaded: 2000, Active: time.Hour, Seeding: 30 * time.Minute}
	rel, data := resumeData(t, h, e, r, saved)

	baseline := engine.Totals{Uploaded: 5000, Downloaded: 100, Active: 2 * time.Hour}
	th := add(t, e, engine.AddParams{Resume: data, SavePath: rel.dir, Baseline: baseline})
	s := awaitStatus(t, h, e, r, th, func(engine.Status) bool { return true })
	want := engine.Totals{Uploaded: 5000, Downloaded: 2000, Active: 2 * time.Hour, Seeding: 30 * time.Minute}
	if !atLeast(s.Totals, want) || s.Totals.Uploaded != want.Uploaded || s.Totals.Downloaded != want.Downloaded {
		t.Fatalf("resume data counting %+v added with baseline %+v counts %+v, want the larger of each: %+v", saved, baseline, s.Totals, want)
	}
}

func pauseSendsStoppedOnlyAfterStartedWasAnswered(t *testing.T, h Harness) {
	tr := h.Tracker(t, enginetest.TrackerRules{Interval: 30 * time.Minute, HoldStarted: true})
	e := h.Open(t, engine.Config{})
	r := record(e)
	rel := release(t, "Pausing", tr.URL)
	th := add(t, e, engine.AddParams{Metainfo: rel.meta, SavePath: rel.dir, SeedMode: true})
	if err := e.Resume(th); err != nil {
		t.Fatal(err)
	}
	h.Until(t, "the tracker hears from the resumed torrent", func() bool { return len(tr.Announces()) > 0 })
	if err := e.Pause(th); err != nil {
		t.Fatal(err)
	}
	if got := events(tr.Announces()); !slices.Equal(got, []string{"started"}) {
		t.Fatalf("paused while the tracker holds its answer to started, the torrent announced %q, want only started", got)
	}
	tr.Answer()
	h.Until(t, "the tracker hears stopped", func() bool { return slices.Contains(events(tr.Announces()), "stopped") })
	if got := events(tr.Announces()); !slices.Equal(got, []string{"started", "stopped"}) {
		t.Fatalf("once started was answered the paused torrent announced %q, want started then stopped", got)
	}
	await(t, h, r, func(p engine.Paused) bool { return p.Torrent == th })
}

func removeKeepsTheFiles(t *testing.T, h Harness) {
	e := h.Open(t, engine.Config{})
	r := record(e)
	rel := release(t, "Kept", "")
	th := add(t, e, engine.AddParams{Metainfo: rel.meta, SavePath: rel.dir, SeedMode: true})
	if err := e.Resume(th); err != nil {
		t.Fatal(err)
	}
	if err := e.Remove(th, false); err != nil {
		t.Fatal(err)
	}
	await(t, h, r, func(rm engine.Removed) bool { return rm.Torrent == th })
	if got, err := os.ReadFile(filepath.Join(rel.dir, rel.files[0].path())); err != nil || len(got) != len(rel.files[0].data) {
		t.Fatalf("the removed torrent's file holds %d bytes, %v; want its %d", len(got), err, len(rel.files[0].data))
	}
}

func removeWithFilesDeletesTheDownloadCopyAndItsFolder(t *testing.T, h Harness) {
	e := h.Open(t, engine.Config{})
	r := record(e)
	remove := func(rel released) {
		t.Helper()
		th := add(t, e, engine.AddParams{Metainfo: rel.meta, SavePath: rel.dir, SeedMode: true})
		if err := e.Resume(th); err != nil {
			t.Fatal(err)
		}
		if err := e.Remove(th, true); err != nil {
			t.Fatal(err)
		}
		deleted := await(t, h, r, func(d engine.FilesDeleted) bool { return d.Torrent == th })
		if deleted.Err != nil {
			t.Fatalf("deleting the torrent's files: %v", deleted.Err)
		}
		if i, j := index(r.seen, engine.Removed{Torrent: th}), index(r.seen, engine.Event(deleted)); i < 0 || i > j {
			t.Fatalf("events %+v, want Removed before FilesDeleted", r.seen)
		}
		for _, f := range rel.files {
			if _, err := os.Stat(filepath.Join(rel.dir, f.path())); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("after the delete %s is %v, want gone", f.path(), err)
			}
		}
	}

	rel := release(t, "Deleted", "", file{[]string{"feature.mkv"}, 40 << 10}, file{[]string{"Extras", "trailer.mkv"}, 20 << 10})
	sibling := filepath.Join(filepath.Dir(rel.dir), "sibling.mkv")
	if err := os.WriteFile(sibling, []byte("not the torrent's"), 0o600); err != nil {
		t.Fatal(err)
	}
	remove(rel)
	if _, err := os.Stat(rel.dir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("after the delete the save folder is %v, want gone", err)
	}
	if _, err := os.Stat(sibling); err != nil {
		t.Fatalf("the delete took a file beside the save folder: %v", err)
	}

	shared := release(t, "Shared", "")
	stranger := filepath.Join(shared.dir, "notes.txt")
	if err := os.WriteFile(stranger, []byte("not the torrent's"), 0o600); err != nil {
		t.Fatal(err)
	}
	remove(shared)
	if _, err := os.Stat(stranger); err != nil {
		t.Fatalf("the delete took a file in the save folder the torrent does not list: %v", err)
	}
}

func updatesCarryOnlyChangedTorrents(t *testing.T, h Harness) {
	e := h.Open(t, engine.Config{})
	r := record(e)
	a, b := settled(t, h, e, r)
	if err := e.Resume(a); err != nil {
		t.Fatal(err)
	}
	from := r.next
	posting(t, h, e, r, func(u engine.Updates) bool { _, ok := statusOf(u, a); return ok })
	for _, ev := range r.seen[from:r.next] {
		if u, ok := ev.(engine.Updates); ok {
			if s, ok := statusOf(u, b); ok {
				t.Fatalf("after only %d resumed, Updates carried the untouched %d: %+v", a, b, s)
			}
		}
	}
}

func updatesCarryEveryTorrentAfterDropped(t *testing.T, h Harness) {
	e := h.Open(t, engine.Config{})
	r := record(e)
	a, b := settled(t, h, e, r)
	// More than libtorrent's alert queue holds: high_performance_seed's 10000,
	// doubled for state_update's priority, with every buffer after it full.
	for range 50000 {
		if err := e.PostUpdates(); err != nil {
			t.Fatal(err)
		}
	}
	from := r.next
	h.Until(t, "Updates carry both torrents after Dropped", func() bool {
		r.poll()
		dropped := false
		for _, ev := range r.seen[from:] {
			switch ev := ev.(type) {
			case engine.Dropped:
				dropped = true
			case engine.Updates:
				_, hasA := statusOf(ev, a)
				_, hasB := statusOf(ev, b)
				if hasA && hasB {
					if !dropped {
						t.Fatalf("Updates carried the unchanged %d and %d before any Dropped", a, b)
					}
					return true
				}
			}
		}
		if err := e.PostUpdates(); err != nil {
			t.Fatal(err)
		}
		return false
	})
}

func trackerListCarriesNextAndMinAnnounce(t *testing.T, h Harness) {
	tr := h.Tracker(t, enginetest.TrackerRules{Interval: 30 * time.Minute, MinInterval: 10 * time.Minute})
	e := h.Open(t, engine.Config{})
	r := record(e)
	th := running(t, e, "Listed", tr.URL)
	if a := await(t, h, r, func(a engine.TrackerAnnounced) bool { return a.Torrent == th }); a.Tracker != 0 || a.Event != engine.AnnounceStarted {
		t.Fatalf("the resumed torrent first reported %+v, want started to tracker 0", a)
	}
	replied := await(t, h, r, func(rp engine.TrackerReplied) bool { return rp.Torrent == th })
	got := onlyTracker(t, trackerList(t, h, e, r, th))
	want := engine.Tracker{URL: tr.URL, Endpoints: 1, NextAnnounce: got.NextAnnounce, MinAnnounce: got.MinAnnounce, StartSent: true}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("after the tracker answered started the tracker list holds %+v, want %+v", got, want)
	}
	if !near(got.NextAnnounce, replied.At.Add(30*time.Minute)) || !near(got.MinAnnounce, replied.At.Add(10*time.Minute)) {
		t.Fatalf("answered at %s with a 30m interval and 10m min interval, the tracker reads next %s and min %s",
			replied.At, got.NextAnnounce, got.MinAnnounce)
	}
}

func theAnnounceFloorLengthensAShortInterval(t *testing.T, h Harness) {
	tr := h.Tracker(t, enginetest.TrackerRules{Interval: time.Minute})
	e := h.Open(t, engine.Config{AnnounceFloor: 10 * time.Minute})
	r := record(e)
	th := running(t, e, "Floored", tr.URL)
	replied := await(t, h, r, func(rp engine.TrackerReplied) bool { return rp.Torrent == th })
	if got := onlyTracker(t, trackerList(t, h, e, r, th)); !near(got.NextAnnounce, replied.At.Add(10*time.Minute)) {
		t.Fatalf("answered at %s with a 1m interval under a 10m floor, the next announce is at %s, want 10m on", replied.At, got.NextAnnounce)
	}
}

func aWarningIsReportedAndListed(t *testing.T, h Harness) {
	const warning = "Your client is out of date"
	tr := h.Tracker(t, enginetest.TrackerRules{Interval: 30 * time.Minute, Warning: warning})
	e := h.Open(t, engine.Config{})
	r := record(e)
	th := running(t, e, "Warned", tr.URL)
	if w := await(t, h, r, func(w engine.TrackerWarned) bool { return w.Torrent == th }); w.Tracker != 0 || w.Message != warning {
		t.Fatalf("the tracker warned %q, and the engine reported %+v", warning, w)
	}
	await(t, h, r, func(rp engine.TrackerReplied) bool { return rp.Torrent == th })
	if got := onlyTracker(t, trackerList(t, h, e, r, th)); got.Message != warning || got.LastError != nil {
		t.Fatalf("the tracker that warned %q lists %+v", warning, got)
	}
}

func aFailureWithoutIntervalWaits1800s(t *testing.T, h Harness) {
	const reason = "Unregistered torrent"
	tr := h.Tracker(t, enginetest.TrackerRules{Failure: reason})
	e := h.Open(t, engine.Config{})
	r := record(e)
	th := running(t, e, "Unregistered", tr.URL)
	failed := await(t, h, r, func(f engine.TrackerFailed) bool { return f.Torrent == th })
	if failed.Tracker != 0 || failed.Times != 1 || failed.Reason != reason || !errors.Is(failed.Err, errTrackerFailure) {
		t.Fatalf("the tracker answered %q, and the engine reported %+v", reason, failed)
	}
	got := onlyTracker(t, trackerList(t, h, e, r, th))
	if got.Fails != 1 || got.Message != reason || !errors.Is(got.LastError, errTrackerFailure) || got.StartSent {
		t.Fatalf("the tracker that answered %q lists %+v", reason, got)
	}
	if !near(got.NextAnnounce, failed.At.Add(1800*time.Second)) {
		t.Fatalf("failed at %s with no interval, the next announce is at %s, want 1800s on", failed.At, got.NextAnnounce)
	}
}

func reannounceMovesMinAnnounce(t *testing.T, h Harness) {
	tr := h.Tracker(t, enginetest.TrackerRules{Interval: 30 * time.Minute, MinInterval: 20 * time.Minute})
	e := h.Open(t, engine.Config{})
	r := record(e)
	th := running(t, e, "Moved", tr.URL)
	replied := await(t, h, r, func(rp engine.TrackerReplied) bool { return rp.Torrent == th })

	if err := e.Reannounce(th, 0, time.Minute, false); err != nil {
		t.Fatal(err)
	}
	got := onlyTracker(t, trackerList(t, h, e, r, th))
	if want := replied.At.Add(20*time.Minute + time.Second); !near(got.NextAnnounce, want) || !near(got.MinAnnounce, got.NextAnnounce) {
		t.Fatalf("a reannounce in 1m under a min interval to %s reads next %s and min %s, want both a second past the min",
			replied.At.Add(20*time.Minute), got.NextAnnounce, got.MinAnnounce)
	}

	asked := time.Now()
	if err := e.Reannounce(th, 0, 25*time.Minute, false); err != nil {
		t.Fatal(err)
	}
	got = onlyTracker(t, trackerList(t, h, e, r, th))
	if want := asked.Add(25*time.Minute + time.Second); !near(got.NextAnnounce, want) || !near(got.MinAnnounce, got.NextAnnounce) {
		t.Fatalf("a reannounce in 25m asked at %s reads next %s and min %s, want both at %s", asked, got.NextAnnounce, got.MinAnnounce, want)
	}
	if got := events(tr.Announces()); !slices.Equal(got, []string{"started"}) {
		t.Fatalf("after two reannounces held to the min interval the tracker received %q, want only started", got)
	}
}

func reannounceIgnoringMinIntervalAnnouncesNow(t *testing.T, h Harness) {
	tr := h.Tracker(t, enginetest.TrackerRules{Interval: 30 * time.Minute, MinInterval: 20 * time.Minute})
	e := h.Open(t, engine.Config{})
	r := record(e)
	th := running(t, e, "Forced", tr.URL)
	await(t, h, r, func(rp engine.TrackerReplied) bool { return rp.Torrent == th })
	asked := time.Now()
	if err := e.Reannounce(th, 0, 0, true); err != nil {
		t.Fatal(err)
	}
	if a := await(t, h, r, func(a engine.TrackerAnnounced) bool { return a.Torrent == th }); a.Event != engine.AnnounceNone || !near(a.At, asked.Add(time.Second)) {
		t.Fatalf("a reannounce ignoring the min interval asked at %s sent %+v, want a regular announce a second later", asked, a)
	}
	h.Until(t, "the tracker hears the reannounce", func() bool { return len(tr.Announces()) == 2 })
	if got := events(tr.Announces()); !slices.Equal(got, []string{"started", ""}) {
		t.Fatalf("a reannounce ignoring the min interval reached the tracker as %q, want started then a regular announce", got)
	}
}

func recheckSendsNoAnnounceAndEndsPaused(t *testing.T, h Harness) {
	tr := h.Tracker(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
	e := h.Open(t, engine.Config{})
	r := record(e)
	rel := release(t, "Rechecked", tr.URL)
	th := add(t, e, engine.AddParams{Metainfo: rel.meta, SavePath: rel.dir, SeedMode: true})
	if err := e.Resume(th); err != nil {
		t.Fatal(err)
	}
	await(t, h, r, func(rp engine.TrackerReplied) bool { return rp.Torrent == th })
	if err := e.Pause(th); err != nil {
		t.Fatal(err)
	}
	await(t, h, r, func(p engine.Paused) bool { return p.Torrent == th })
	h.Until(t, "the tracker hears stopped", func() bool { return slices.Contains(events(tr.Announces()), "stopped") })
	corrupt(t, filepath.Join(rel.dir, rel.files[0].path()))

	if err := e.Recheck(th); err != nil {
		t.Fatal(err)
	}
	await(t, h, r, func(c engine.Checked) bool { return c.Torrent == th })
	await(t, h, r, func(p engine.Paused) bool { return p.Torrent == th })
	s := awaitStatus(t, h, e, r, th, func(engine.Status) bool { return true })
	if want := int64(len(rel.payload) - pieceLen); !s.Paused || s.Done != want {
		t.Fatalf("rechecked with its first piece corrupt, the torrent reads %+v, want paused with %d bytes done", s, want)
	}
	if got := events(tr.Announces()); !slices.Equal(got, []string{"started", "stopped"}) {
		t.Fatalf("around a recheck of the paused torrent the tracker received %q, want only the started and stopped before it", got)
	}
}

// corrupt flips the first byte of file.
func corrupt(t *testing.T, file string) {
	t.Helper()
	f, err := os.OpenFile(file, os.O_RDWR, 0) //nolint:gosec // a file of a release the test wrote
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	b := make([]byte, 1)
	if _, err := f.ReadAt(b, 0); err != nil {
		t.Fatal(err)
	}
	b[0] ^= 0xff
	if _, err := f.WriteAt(b, 0); err != nil {
		t.Fatal(err)
	}
}

// aTakenPortStaysTakenAndTheSessionRuns: an engine that does not listen
// checks and holds torrents, and announces nothing, having no endpoint to
// announce from.
func aTakenPortStaysTakenAndTheSessionRuns(t *testing.T, h Harness) {
	taken := h.TakePort(t)
	e := h.Open(t, engine.Config{ListenInterface: taken})
	r := record(e)
	await(t, h, r, func(engine.ListenFailed) bool { return true })
	if ap, err := e.ListenAddr(); !errors.Is(err, engine.ErrNotListening) {
		t.Fatalf("on the taken %s the engine listens on %s, %v; want %v", taken, ap, err, engine.ErrNotListening)
	}
	tr := h.Tracker(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
	rel := release(t, "Unbound", tr.URL)
	th := add(t, e, engine.AddParams{Metainfo: rel.meta, SavePath: rel.dir})
	if err := e.Resume(th); err != nil {
		t.Fatal(err)
	}
	awaitStatus(t, h, e, r, th, func(s engine.Status) bool { return s.State == engine.StateSeeding })
	if got := onlyTracker(t, trackerList(t, h, e, r, th)); got.Endpoints != 0 || len(tr.Announces()) != 0 {
		t.Fatalf("the engine that does not listen lists %+v, and its tracker received %+v; want no endpoint and nothing", got, tr.Announces())
	}
	if ap, err := e.ListenAddr(); !errors.Is(err, engine.ErrNotListening) {
		t.Fatalf("once it ran a torrent, the engine whose %s was taken listens on %s, %v; want %v", taken, ap, err, engine.ErrNotListening)
	}
}

func configureAppliesLive(t *testing.T, h Harness) {
	const limit = 32 << 10
	e := h.Open(t, engine.Config{})
	r := record(e)
	rel := release(t, "Limited", "", file{[]string{"feature.mkv"}, 1 << 20})
	reach := h.Seed(t, rel.meta, rel.dir)
	if err := e.Configure(engine.Limits{DownloadLimit: limit, Connections: 50, UTP: true}); err != nil {
		t.Fatal(err)
	}
	th := add(t, e, engine.AddParams{Metainfo: rel.meta, SavePath: t.TempDir()})
	if err := e.Resume(th); err != nil {
		t.Fatal(err)
	}
	reach(e, th)
	first := awaitStatus(t, h, e, r, th, func(s engine.Status) bool { return s.Done > 0 })
	began := time.Now()
	h.Until(t, "three seconds pass", func() bool { return time.Since(began) >= 3*time.Second })
	s := awaitStatus(t, h, e, r, th, func(engine.Status) bool { return true })
	took := time.Since(began)
	if most := int64(limit * (took.Seconds() + 2)); s.Done-first.Done > most || s.Done == s.Wanted {
		t.Fatalf("under a %d B/s limit the torrent went from %d to %d of %d bytes in %s, want at most %d more", limit, first.Done, s.Done, s.Wanted, took, most)
	}
	if err := e.Configure(engine.Limits{Connections: 50, UTP: true}); err != nil {
		t.Fatal(err)
	}
	await(t, h, r, func(f engine.Finished) bool { return f.Torrent == th })
}

func statsCountThePayload(t *testing.T, h Harness) {
	e := h.Open(t, engine.Config{})
	r := record(e)
	rel := release(t, "Counted", "", file{[]string{"feature.mkv"}, 256 << 10})
	reach := h.Seed(t, rel.meta, rel.dir)
	th := add(t, e, engine.AddParams{Metainfo: rel.meta, SavePath: t.TempDir()})
	if err := e.Resume(th); err != nil {
		t.Fatal(err)
	}
	reach(e, th)
	await(t, h, r, func(f engine.Finished) bool { return f.Torrent == th })
	asked := time.Now()
	if err := e.PostStats(); err != nil {
		t.Fatal(err)
	}
	st := await(t, h, r, func(engine.Stats) bool { return true })
	if st.RecvPayload < int64(len(rel.payload)) || !near(st.At, asked) {
		t.Fatalf("after a %d byte download the engine's stats asked at %s read %+v, want that much received", len(rel.payload), asked, st)
	}
}

func anIncomingPeerIsReportedWhileWatched(t *testing.T, h Harness) {
	e := h.Open(t, engine.Config{})
	r := record(e)
	running(t, e, "Watched", "")

	h.Incoming(t, e)
	if err := e.PostStats(); err != nil {
		t.Fatal(err)
	}
	await(t, h, r, func(engine.Stats) bool { return true })
	if i := slices.IndexFunc(r.seen, func(ev engine.Event) bool { _, ok := ev.(engine.IncomingPeer); return ok }); i >= 0 {
		t.Fatalf("with incoming peers unwatched the engine reported %+v", r.seen[i])
	}

	if err := e.Configure(engine.Limits{Connections: 50, UTP: true, WatchIncoming: true}); err != nil {
		t.Fatal(err)
	}
	from := h.Incoming(t, e)
	if p := await(t, h, r, func(engine.IncomingPeer) bool { return true }); p.Addr != from {
		t.Fatalf("a peer at %s connected and the engine reported %+v", from, p)
	}
}

func theAnnouncedPeerIDAndUserAgentAreTheVersions(t *testing.T, h Harness) {
	tr := h.Tracker(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
	e := h.Open(t, engine.Config{})
	v := e.Version()
	parts := strings.Split(v.Libtorrent, ".")
	if len(parts) != 3 {
		t.Fatalf("the engine's version %q is not major.minor.tiny", v.Libtorrent)
	}
	prefix := []byte("-LT")
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 || n >= 36 {
			t.Fatalf("the engine's version %q has a part libtorrent's fingerprint cannot encode", v.Libtorrent)
		}
		prefix = append(prefix, "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"[n])
	}
	if want := string(prefix) + "0-"; v.PeerIDPrefix != want || v.UserAgent != "libtorrent/"+v.Libtorrent+".0" {
		t.Fatalf("version %+v, want peer ID prefix %s and user agent libtorrent/%s.0, libtorrent's own", v, want, v.Libtorrent)
	}
	running(t, e, "Identified", tr.URL)
	h.Until(t, "the tracker hears from the torrent", func() bool { return len(tr.Announces()) > 0 })
	if a := tr.Announces()[0]; len(a.PeerID) != 20 || !strings.HasPrefix(a.PeerID, v.PeerIDPrefix) || a.UserAgent != v.UserAgent {
		t.Fatalf("the torrent announced as peer %q with user agent %q, want a 20-byte peer ID starting %s and %s", a.PeerID, a.UserAgent, v.PeerIDPrefix, v.UserAgent)
	}
}

func closeSendsStopped(t *testing.T, h Harness) {
	tr := h.Tracker(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
	e := h.Open(t, engine.Config{})
	r := record(e)
	th := running(t, e, "Closed", tr.URL)
	await(t, h, r, func(rp engine.TrackerReplied) bool { return rp.Torrent == th })
	if err := e.Close(); err != nil {
		t.Fatal(err)
	}
	h.Until(t, "the tracker hears stopped", func() bool { return slices.Contains(events(tr.Announces()), "stopped") })
	if got := events(tr.Announces()); !slices.Equal(got, []string{"started", "stopped"}) {
		t.Fatalf("a torrent running when its engine closed announced %q, want started then stopped", got)
	}
}

// errTrackerFailure is libtorrent's tracker_failure, a reply with a
// failure reason.
var errTrackerFailure = &engine.Error{Category: engine.CategoryLibtorrent, Code: 173}

// running adds a seeding release named name announcing to announce, and
// resumes it.
func running(t *testing.T, e engine.Engine, name, announce string) engine.Handle {
	t.Helper()
	rel := release(t, name, announce)
	th := add(t, e, engine.AddParams{Metainfo: rel.meta, SavePath: rel.dir, SeedMode: true})
	if err := e.Resume(th); err != nil {
		t.Fatal(err)
	}
	return th
}

func trackerList(t *testing.T, h Harness, e engine.Engine, r *recorder, th engine.Handle) engine.TrackerList {
	t.Helper()
	if err := e.PostTrackers(th); err != nil {
		t.Fatal(err)
	}
	return await(t, h, r, func(l engine.TrackerList) bool { return l.Torrent == th })
}

func onlyTracker(t *testing.T, l engine.TrackerList) engine.Tracker {
	t.Helper()
	if len(l.Trackers) != 1 {
		t.Fatalf("the torrent's tracker list holds %+v, want its one tracker", l.Trackers)
	}
	return l.Trackers[0]
}

// near is whether got is within libtorrent's whole-second announce clock
// of want.
func near(got, want time.Time) bool {
	d := got.Sub(want)
	return !got.IsZero() && d > -2*time.Second && d < 2*time.Second
}

// settled adds two torrents and posts updates until, once both came, one
// carries neither.
func settled(t *testing.T, h Harness, e engine.Engine, r *recorder) (a, b engine.Handle) {
	t.Helper()
	ra, rb := release(t, "Alpha", ""), release(t, "Bravo", "")
	a = add(t, e, engine.AddParams{Metainfo: ra.meta, SavePath: ra.dir, SeedMode: true})
	b = add(t, e, engine.AddParams{Metainfo: rb.meta, SavePath: rb.dir, SeedMode: true})
	var cameA, cameB bool
	posting(t, h, e, r, func(u engine.Updates) bool {
		_, hasA := statusOf(u, a)
		_, hasB := statusOf(u, b)
		cameA, cameB = cameA || hasA, cameB || hasB
		return cameA && cameB && len(u.Statuses) == 0
	})
	return a, b
}

// resumeData has e check a release it was added without resume data for,
// and returns the resume data it then saves.
func resumeData(t *testing.T, h Harness, e engine.Engine, r *recorder, baseline engine.Totals) (rel released, data []byte) {
	t.Helper()
	rel = release(t, "Resumed", "")
	th := add(t, e, engine.AddParams{Metainfo: rel.meta, SavePath: rel.dir, Baseline: baseline})
	if err := e.Resume(th); err != nil {
		t.Fatal(err)
	}
	awaitStatus(t, h, e, r, th, func(s engine.Status) bool { return s.State == engine.StateSeeding })
	if !checked(r.seen, th) {
		t.Fatalf("the torrent added without resume data never checked its files: events %+v", r.seen)
	}
	if err := e.SaveResume(th); err != nil {
		t.Fatal(err)
	}
	saved := await(t, h, r, func(s engine.ResumeSaved) bool { return s.Torrent == th })
	if err := e.Remove(th, false); err != nil {
		t.Fatal(err)
	}
	await(t, h, r, func(rm engine.Removed) bool { return rm.Torrent == th })
	return rel, saved.Data
}

func add(t *testing.T, e engine.Engine, p engine.AddParams) engine.Handle {
	t.Helper()
	th, err := e.Add(p)
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	return th
}

// checked is whether events show torrent th checking its files.
func checked(events []engine.Event, th engine.Handle) bool {
	for _, ev := range events {
		switch ev := ev.(type) {
		case engine.StateChanged:
			if ev.Torrent == th && ev.State == engine.StateCheckingFiles {
				return true
			}
		case engine.Updates:
			if s, ok := statusOf(ev, th); ok && s.State == engine.StateCheckingFiles {
				return true
			}
		}
	}
	return false
}

func atLeast(got, want engine.Totals) bool {
	return got.Uploaded >= want.Uploaded && got.Downloaded >= want.Downloaded && got.Active >= want.Active && got.Seeding >= want.Seeding
}

func statusOf(u engine.Updates, th engine.Handle) (engine.Status, bool) {
	i := slices.IndexFunc(u.Statuses, func(s engine.Status) bool { return s.Torrent == th })
	if i < 0 {
		return engine.Status{}, false
	}
	return u.Statuses[i], true
}

func events(announces []enginetest.Announce) []string {
	out := make([]string, len(announces))
	for i, a := range announces {
		out[i] = a.Event
	}
	return out
}

func index(events []engine.Event, ev engine.Event) int {
	return slices.IndexFunc(events, func(e engine.Event) bool { return e == ev })
}

// recorder keeps every event an engine sent, read without blocking so a
// synctest bubble's clock moves while the suite waits.
type recorder struct {
	events <-chan engine.Event
	seen   []engine.Event
	// next is where the next await looks from.
	next int
}

func record(e engine.Engine) *recorder { return &recorder{events: e.Events()} }

func (r *recorder) poll() {
	for {
		select {
		case ev, ok := <-r.events:
			if !ok {
				return
			}
			r.seen = append(r.seen, ev)
		default:
			return
		}
	}
}

// await waits for the first event of type E after the last one an await
// returned that matches.
func await[E engine.Event](t *testing.T, h Harness, r *recorder, match func(E) bool) E {
	t.Helper()
	var got E
	h.Until(t, fmt.Sprintf("the engine sends the %T awaited", got), func() bool {
		r.poll()
		for ; r.next < len(r.seen); r.next++ {
			if ev, ok := r.seen[r.next].(E); ok && match(ev) {
				got = ev
				r.next++
				return true
			}
		}
		return false
	})
	return got
}

// awaitStatus posts updates until one carries th in a status that matches.
func awaitStatus(t *testing.T, h Harness, e engine.Engine, r *recorder, th engine.Handle, match func(engine.Status) bool) engine.Status {
	t.Helper()
	u := posting(t, h, e, r, func(u engine.Updates) bool {
		s, ok := statusOf(u, th)
		return ok && match(s)
	})
	s, _ := statusOf(u, th)
	return s
}

// posting posts updates on each poll until Updates that match arrive,
// after the last event an await returned.
func posting(t *testing.T, h Harness, e engine.Engine, r *recorder, match func(engine.Updates) bool) engine.Updates {
	t.Helper()
	var got engine.Updates
	h.Until(t, "posted Updates carry what is awaited", func() bool {
		r.poll()
		for ; r.next < len(r.seen); r.next++ {
			if u, ok := r.seen[r.next].(engine.Updates); ok && match(u) {
				got = u
				r.next++
				return true
			}
		}
		if err := e.PostUpdates(); err != nil {
			t.Fatal(err)
		}
		return false
	})
	return got
}

type file struct {
	path []string
	size int
}

type releaseFile struct {
	name []string
	data []byte
}

func (f releaseFile) path() string { return filepath.Join(f.name...) }

// released is a release's files written under dir, its save folder, which
// is a folder of its own inside the test's temporary folder.
type released struct {
	dir     string
	meta    []byte
	hash    bencode.InfoHash
	files   []releaseFile
	payload []byte
}

// release writes a release named name and returns its metainfo, which
// announces to announce unless it is empty. With no files it is one file
// named name.
const pieceLen = 16 << 10

func release(t *testing.T, name, announce string, files ...file) released {
	t.Helper()
	dir := filepath.Join(t.TempDir(), name)
	single := len(files) == 0
	if single {
		files = []file{{nil, 48<<10 + 123}}
	}
	rel := released{dir: dir}
	var list []any
	for i, f := range files {
		data := fill(f.size, fmt.Sprintf("%s/%d", name, i))
		rf := releaseFile{name: append([]string{name}, f.path...), data: data}
		if single {
			rf.name = []string{name}
		}
		rel.files = append(rel.files, rf)
		rel.payload = append(rel.payload, data...)
		p := make([]any, len(f.path))
		for j, s := range f.path {
			p[j] = s
		}
		list = append(list, map[string]any{"length": f.size, "path": p})
		full := filepath.Join(dir, rf.path())
		if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	var pieces []byte
	for off := 0; off < len(rel.payload); off += pieceLen {
		sum := sha1.Sum(rel.payload[off:min(off+pieceLen, len(rel.payload))]) //nolint:gosec // BitTorrent v1 piece hashes are SHA-1 by spec (BEP 3)
		pieces = append(pieces, sum[:]...)
	}
	info := map[string]any{"name": name, "piece length": pieceLen, "pieces": pieces}
	if single {
		info["length"] = len(rel.payload)
	} else {
		info["files"] = list
	}
	top := map[string]any{"info": info}
	if announce != "" {
		top["announce"] = announce
	}
	meta, err := bencode.Encode(top)
	if err != nil {
		t.Fatal(err)
	}
	m, err := bencode.DecodeMetainfo(meta)
	if err != nil {
		t.Fatal(err)
	}
	rel.meta, rel.hash = meta, m.InfoHash
	return rel
}

// fill is size bytes that depend only on seed: SHA-256 blocks of the seed and
// a counter, so every run writes the same release.
func fill(size int, seed string) []byte {
	data := make([]byte, 0, size+sha256.Size)
	for n := 0; len(data) < size; n++ {
		sum := sha256.Sum256(fmt.Appendf(nil, "%s/%d", seed, n))
		data = append(data, sum[:]...)
	}
	return data[:size]
}
