package libtorrent_test

import (
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/torrent/engine"
	"github.com/jatassi/slipstream/internal/torrent/engine/enginetest"
)

// announces is a tracker on loopback that logs each announce and answers
// it as its rules say, holding each answer to started or to a regular
// announce until answer closes when they hold it.
type announces struct {
	rules  enginetest.TrackerRules
	answer chan struct{}
	once   sync.Once

	mu   sync.Mutex
	log  []enginetest.Announce
	read int
}

// loopbackTracker serves a tracker following rules, at a URL ending in
// /announce as libtorrent's ssrf_mitigation requires of a loopback tracker.
func loopbackTracker(t *testing.T, rules enginetest.TrackerRules) (a *announces, url string) {
	t.Helper()
	a = &announces{rules: rules, answer: make(chan struct{})}
	srv := httptest.NewServer(a)
	t.Cleanup(srv.Close)
	return a, srv.URL + "/announce"
}

func (a *announces) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var hash bencode.InfoHash
	copy(hash[:], q.Get("info_hash"))
	event := q.Get("event")
	a.mu.Lock()
	a.log = append(a.log, enginetest.Announce{InfoHash: hash, Event: event, At: time.Now(), PeerID: q.Get("peer_id"), UserAgent: r.UserAgent()})
	a.mu.Unlock()
	if event == "started" && a.rules.HoldStarted || event == "" && a.rules.HoldRegular {
		select {
		case <-a.answer:
		case <-r.Context().Done():
			return
		}
	}
	reply := map[string]any{"interval": int64(a.rules.Interval / time.Second), "min interval": int64(a.rules.MinInterval / time.Second), "peers": ""}
	switch {
	case a.rules.Failure != "":
		reply = map[string]any{"failure reason": a.rules.Failure}
	case a.rules.Warning != "":
		reply["warning message"] = a.rules.Warning
	}
	body, err := bencode.Encode(reply)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(body)
}

func (a *announces) Answer() { a.once.Do(func() { close(a.answer) }) }

func (a *announces) Announces() []enginetest.Announce {
	a.mu.Lock()
	defer a.mu.Unlock()
	return slices.Clone(a.log)
}

// next is the tracker's next announce event, empty when none comes within
// wait.
func (a *announces) next(wait time.Duration) string {
	for deadline := time.Now().Add(wait); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		a.mu.Lock()
		if a.read < len(a.log) {
			a.read++
			event := a.log[a.read-1].Event
			a.mu.Unlock()
			return event
		}
		a.mu.Unlock()
	}
	return ""
}

// seedTracked has a new engine seed a payload named name, announcing to a
// new tracker, once its started announce reached the tracker.
func seedTracked(t *testing.T, name string) (engine.Engine, engine.AddParams, engine.Handle, *announces) {
	t.Helper()
	tracker, url := loopbackTracker(t, enginetest.TrackerRules{Interval: 30 * time.Minute, HoldStarted: true})
	dir := t.TempDir()
	payload := make([]byte, 1<<20)
	_, _ = rand.NewChaCha8([32]byte{name[0]}).Read(payload)
	if err := os.WriteFile(filepath.Join(dir, name), payload, 0o600); err != nil {
		t.Fatal(err)
	}
	e := open(t)
	add := engine.AddParams{Metainfo: trackedTorrent(t, name, payload, 64<<10, url), SavePath: dir, SeedMode: true}
	h, err := e.Add(add)
	if err != nil {
		t.Fatal(err)
	}
	if err := e.Resume(h); err != nil {
		t.Fatal(err)
	}
	if got := tracker.next(60 * time.Second); got != "started" {
		t.Fatalf("the added torrent announced %q, want started", got)
	}
	return e, add, h, tracker
}

// TestARemovedTorrentAnnouncesStopped removes a seeding torrent from the
// engine once the tracker answered its started announce, and while that
// answer is still on its way: either way the tracker hears stopped, so a
// private tracker never counts the Node as a peer after it let go.
func TestARemovedTorrentAnnouncesStopped(t *testing.T) {
	for _, tc := range []struct {
		name     string
		answered bool
	}{
		{"after the tracker answered started", true},
		{"while the tracker's answer to started is on its way", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e, _, h, tracker := seedTracked(t, "Tracked.mkv")
			if tc.answered {
				tracker.Answer()
				waitFor(t, e.Events(), func(r engine.TrackerReplied) bool { return r.Torrent == h })
			}
			if err := e.Remove(h, false); err != nil {
				t.Fatal(err)
			}
			if !tc.answered {
				tracker.Answer()
			}
			if got := tracker.next(30 * time.Second); got != "stopped" {
				t.Fatalf("the removed torrent announced %q, want stopped", got)
			}
			waitFor(t, e.Events(), func(r engine.Removed) bool { return r.Torrent == h })
		})
	}
}

// TestATorrentAddedAgainWhileItsRemovalWaitsStays removes a seeding
// torrent while the tracker's answer to started is on its way, then adds
// and resumes it again: the engine keeps it under the same handle, and the
// answer neither removes nor pauses it.
func TestATorrentAddedAgainWhileItsRemovalWaitsStays(t *testing.T) {
	e, add, h, tracker := seedTracked(t, "Again.mkv")
	if err := e.Remove(h, false); err != nil {
		t.Fatal(err)
	}
	again, err := e.Add(add)
	if err != nil || again != h {
		t.Fatalf("added again while its removal waits, the torrent is %d, %v; want %d", again, err, h)
	}
	if err := e.Resume(again); err != nil {
		t.Fatal(err)
	}
	tracker.Answer()
	waitFor(t, e.Events(), func(r engine.TrackerReplied) bool { return r.Torrent == h })
	if _, err := e.Files(h); err != nil {
		t.Fatalf("after the tracker answered, the torrent added again reads %v, want its files", err)
	}
	if got := tracker.next(2 * time.Second); got != "" {
		t.Fatalf("after the tracker answered, the torrent added again announced %q, want nothing", got)
	}
}
