package libtorrent_test

import (
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/torrent/engine"
)

// announces is a tracker on loopback that queues each announce's event on
// events and holds its answer to started until answer closes.
type announces struct {
	events chan string
	answer chan struct{}
}

func (a *announces) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	event := r.URL.Query().Get("event")
	a.events <- event
	if event == "started" {
		select {
		case <-a.answer:
		case <-r.Context().Done():
			return
		}
	}
	_, _ = w.Write([]byte("d8:intervali1800e5:peers0:e"))
}

// next is the tracker's next announce event, empty when none comes within
// wait.
func (a *announces) next(wait time.Duration) string {
	select {
	case e := <-a.events:
		return e
	case <-time.After(wait):
		return ""
	}
}

// seedTracked has a new engine seed a payload named name, announcing to a
// new tracker, once its started announce reached the tracker.
func seedTracked(t *testing.T, name string) (engine.Engine, engine.AddParams, engine.Handle, *announces) {
	t.Helper()
	tracker := &announces{events: make(chan string, 8), answer: make(chan struct{})}
	srv := httptest.NewServer(tracker)
	t.Cleanup(srv.Close)
	dir := t.TempDir()
	payload := make([]byte, 1<<20)
	_, _ = rand.NewChaCha8([32]byte{name[0]}).Read(payload)
	if err := os.WriteFile(filepath.Join(dir, name), payload, 0o600); err != nil {
		t.Fatal(err)
	}
	e := open(t)
	add := engine.AddParams{Metainfo: trackedTorrent(t, name, payload, 64<<10, srv.URL+"/announce"), SavePath: dir, SeedMode: true}
	h, err := e.Add(add)
	if err != nil {
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
				close(tracker.answer)
				waitFor(t, e.Events(), func(r engine.TrackerReplied) bool { return r.Torrent == h })
			}
			if err := e.Remove(h); err != nil {
				t.Fatal(err)
			}
			if !tc.answered {
				close(tracker.answer)
			}
			if got := tracker.next(30 * time.Second); got != "stopped" {
				t.Fatalf("the removed torrent announced %q, want stopped", got)
			}
			waitFor(t, e.Events(), func(r engine.Removed) bool { return r.Torrent == h })
		})
	}
}
