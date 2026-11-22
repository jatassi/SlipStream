package torrentclient_test

import (
	"cmp"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/internal/releases"
	"github.com/jatassi/slipstream/fakes/internal/torrentclient"
)

// placed writes the first n bytes of r's payload where the client saves
// it under dir/Movies.
func placed(t *testing.T, dir string, r releases.Release, n int) {
	t.Helper()
	file := filepath.Join(dir, "Movies", r.Files[0].Path)
	if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, payload(t, r)[:n], 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestACompleteAddAdoptsTheFilesOnDisk(t *testing.T) {
	for name, c := range map[string]struct {
		have  int
		done  int64
		state torrentclient.State
	}{
		"whole":   {size, size, torrentclient.Seeding},
		"short":   {400, 400, torrentclient.Downloading},
		"missing": {0, 0, torrentclient.Downloading},
	} {
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				dir := t.TempDir()
				e, cat := engine(t, torrentclient.Config{Dir: dir, Name: "/downloads", Check: time.Minute}, "orchard")
				r := cat.Releases[0]
				if c.have > 0 {
					placed(t, dir, r, c.have)
				}
				got, _, err := e.Add(torrentclient.Add{Metainfo: r.Metainfo, Folder: "/downloads/Movies", Complete: true})
				if err != nil {
					t.Fatal(err)
				}
				if got.Done != c.done || got.State != c.state {
					t.Fatalf("added complete over %d bytes: done %d, state %d, want %d bytes, state %d", c.have, got.Done, got.State, c.done, c.state)
				}
			})
		})
	}
}

func TestAVerifyTakesWhatTheFilesOnDiskHold(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dir := t.TempDir()
		e, cat := engine(t, torrentclient.Config{Dir: dir, Name: "/downloads", Check: 2 * time.Second}, "orchard")
		r := cat.Releases[0]
		placed(t, dir, r, size)
		if _, _, err := e.Add(torrentclient.Add{Metainfo: r.Metainfo, Folder: "/downloads/Movies", Paused: true}); err != nil {
			t.Fatal(err)
		}
		time.Sleep(3 * time.Second)
		if got := get(t, e, r); got.Done != 0 {
			t.Fatalf("added paused over the files: done %d, want 0 before a verify", got.Done)
		}
		if err := e.Verify(r.InfoHash); err != nil {
			t.Fatal(err)
		}
		time.Sleep(3 * time.Second)
		if err := e.Start(r.InfoHash); err != nil {
			t.Fatal(err)
		}
		if got := get(t, e, r); got.Done != size || got.State != torrentclient.Seeding {
			t.Fatalf("verified and started: done %d, state %d, want complete and seeding", got.Done, got.State)
		}
	})
}

// fakeTracker answers every announce with a 30 second interval, or with
// failure when it is set, and keeps each announce's event.
type fakeTracker struct {
	events  []string
	failure string
}

func (f *fakeTracker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f.events = append(f.events, cmp.Or(q.Get("event"), "regular")+" left "+q.Get("left"))
	if f.failure != "" {
		_, _ = w.Write([]byte("d14:failure reason" + strconv.Itoa(len(f.failure)) + ":" + f.failure + "e"))
		return
	}
	_, _ = w.Write([]byte("d8:completei3e10:incompletei1e8:intervali30e5:peers0:e"))
}

func TestTheEngineAnnouncesToItsTorrentsTrackers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tr := &fakeTracker{}
		e, cat := engine(t, torrentclient.Config{Tracker: tr, PeerID: "-qB4670-0123456789ab"}, "orchard")
		r := cat.Releases[0]
		start := time.Now()
		add(t, e, r, "")
		line := get(t, e, r).Lines[0]
		if !line.AnnouncedAt.Equal(start) || !line.SucceededAt.Equal(start) || !line.NextAt.Equal(start.Add(30*time.Second)) || line.Seeders != 3 {
			t.Fatalf("after the add the line is %+v, want announced now with the next in 30s", line)
		}
		time.Sleep(31 * time.Second)
		if got := get(t, e, r).Lines[0]; !got.AnnouncedAt.Equal(start.Add(31 * time.Second)) {
			t.Fatalf("31s on the line was last announced at %v, want 31s", got.AnnouncedAt.Sub(start))
		}
		time.Sleep(time.Minute)
		get(t, e, r)
		if err := e.Stop(r.InfoHash); err != nil {
			t.Fatal(err)
		}
		if got := get(t, e, r).Lines[0]; !got.NextAt.IsZero() {
			t.Fatalf("stopped, the line's next announce is at %v, want none", got.NextAt)
		}
		tr.failure = "Unregistered torrent"
		if err := e.Start(r.InfoHash); err != nil {
			t.Fatal(err)
		}
		if got := get(t, e, r).Lines[0]; got.Failure != "Unregistered torrent" || !got.SucceededAt.Equal(start.Add(91*time.Second)) {
			t.Fatalf("a refused started left the line %+v, want the failure kept and the last success at 91s", got)
		}
		if err := e.Remove(r.InfoHash, false); err != nil {
			t.Fatal(err)
		}
		want := []string{"started left 1000", "completed left 0", "regular left 0", "stopped left 0", "started left 0", "stopped left 0"}
		if !slices.Equal(tr.events, want) {
			t.Fatalf("the tracker got %q, want %q", tr.events, want)
		}
	})
}
