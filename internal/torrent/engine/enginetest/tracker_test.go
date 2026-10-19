package enginetest_test

import (
	"reflect"
	"strconv"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/torrent/engine"
	"github.com/jatassi/slipstream/internal/torrent/engine/enginetest"
)

const tracked = "https://tracker.example/announce"

// trackedSingle is single announcing to tracked.
var trackedSingle = "d8:announce" + strconv.Itoa(len(tracked)) + ":" + tracked + single[1:]

// announced is when each announce reached tracked, as offsets from start,
// with its event.
type announced struct {
	After time.Duration
	Event string
}

func announces(w *enginetest.World, start time.Time) []announced {
	var out []announced
	for _, a := range w.Announces(tracked) {
		out = append(out, announced{a.At.Sub(start), a.Event})
	}
	return out
}

// running adds trackedSingle, seeding unless payload is set, and resumes
// it in an engine whose announce floor is libtorrent's default, 5 minutes.
func running(t *testing.T, w *enginetest.World, payload []byte) (engine.Engine, engine.Handle) {
	t.Helper()
	m, err := bencode.DecodeMetainfo([]byte(trackedSingle))
	if err != nil {
		t.Fatal(err)
	}
	if payload != nil {
		w.Payload(m.InfoHash, payload)
	}
	e, err := w.Open(engine.Config{AnnounceFloor: 5 * time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = e.Close() })
	h, err := e.Add(engine.AddParams{Metainfo: []byte(trackedSingle), SavePath: t.TempDir(), SeedMode: payload == nil})
	if err != nil {
		t.Fatal(err)
	}
	if err := e.Resume(h); err != nil {
		t.Fatal(err)
	}
	return e, h
}

// TestAnnouncesFollowLibtorrentsIntervals checks the World's trackers
// against libtorrent's rules: the next announce waits for the larger of
// the interval and the 5-minute floor, and a seed added complete never
// sends completed.
func TestAnnouncesFollowLibtorrentsIntervals(t *testing.T) {
	for _, tc := range []struct {
		name  string
		rules enginetest.TrackerRules
		want  []announced
	}{
		{"an interval under the floor", enginetest.TrackerRules{Interval: time.Minute}, []announced{{0, "started"}, {5 * time.Minute, ""}, {10 * time.Minute, ""}, {15 * time.Minute, ""}}},
		{"an interval over the floor", enginetest.TrackerRules{Interval: 7 * time.Minute}, []announced{{0, "started"}, {7 * time.Minute, ""}, {14 * time.Minute, ""}}},
		{"a min interval over the interval", enginetest.TrackerRules{Interval: 5 * time.Minute, MinInterval: 8 * time.Minute}, []announced{{0, "started"}, {8 * time.Minute, ""}, {16 * time.Minute, ""}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				w := enginetest.NewWorld()
				w.Tracker(tracked, tc.rules)
				start := time.Now()
				running(t, w, nil)
				time.Sleep(16*time.Minute + time.Second)
				if got := announces(w, start); !reflect.DeepEqual(got, tc.want) {
					t.Fatalf("announces = %+v, want %+v", got, tc.want)
				}
			})
		})
	}
}

// TestAFailureWithoutIntervalRetriesAfter1800s: libtorrent reads 1800 s
// from a failure that names no interval, and sends started again since
// none was answered.
func TestAFailureWithoutIntervalRetriesAfter1800s(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := enginetest.NewWorld()
		w.Tracker(tracked, enginetest.TrackerRules{Interval: time.Minute, FailWithoutInterval: true})
		start := time.Now()
		e, h := running(t, w, nil)
		time.Sleep(time.Hour + time.Second)
		want := []announced{{0, "started"}, {1800 * time.Second, "started"}, {3600 * time.Second, "started"}}
		if got := announces(w, start); !reflect.DeepEqual(got, want) {
			t.Fatalf("announces = %+v, want %+v", got, want)
		}
		failed := 0
		for len(e.Events()) > 0 {
			if f, ok := (<-e.Events()).(engine.TrackerFailed); ok && f.Torrent == h {
				failed++
			}
		}
		if failed != 3 {
			t.Fatalf("the engine sent %d TrackerFailed, want one per announce, 3", failed)
		}
	})
}

// TestADownloadAnnouncesCompletedWhenItFinishes: a torrent that downloads
// while running sends completed at once, past its min interval.
func TestADownloadAnnouncesCompletedWhenItFinishes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := enginetest.NewWorld()
		w.Tracker(tracked, enginetest.TrackerRules{Interval: 10 * time.Minute, MinInterval: 10 * time.Minute})
		start := time.Now()
		running(t, w, []byte("hello"))
		time.Sleep(15 * time.Minute)
		want := []announced{{0, "started"}, {0, "completed"}, {10 * time.Minute, ""}}
		if got := announces(w, start); !reflect.DeepEqual(got, want) {
			t.Fatalf("announces = %+v, want %+v", got, want)
		}
	})
}

// TestAPausedTorrentStartsAgainOnResume: a pause sends stopped and resets
// the tracker, so the resume sends started.
func TestAPausedTorrentStartsAgainOnResume(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := enginetest.NewWorld()
		w.Tracker(tracked, enginetest.TrackerRules{Interval: 30 * time.Minute})
		start := time.Now()
		e, h := running(t, w, nil)
		time.Sleep(time.Minute)
		if err := e.Pause(h); err != nil {
			t.Fatal(err)
		}
		time.Sleep(time.Minute)
		if err := e.Resume(h); err != nil {
			t.Fatal(err)
		}
		want := []announced{{0, "started"}, {time.Minute, "stopped"}, {2 * time.Minute, "started"}}
		if got := announces(w, start); !reflect.DeepEqual(got, want) {
			t.Fatalf("announces = %+v, want %+v", got, want)
		}
	})
}

// TestARemovalWaitsForTheAnswerToStarted removes a torrent while the
// tracker holds its answer to started: stopped follows the answer, and so
// does Removed.
func TestARemovalWaitsForTheAnswerToStarted(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := enginetest.NewWorld()
		w.Tracker(tracked, enginetest.TrackerRules{Interval: 30 * time.Minute, HoldStarted: true})
		start := time.Now()
		e, h := running(t, w, nil)
		if err := e.Remove(h, false); err != nil {
			t.Fatal(err)
		}
		if _, err := e.Status(h); err == nil {
			t.Fatal("a torrent whose removal waits still reads a status, want unknown")
		}
		time.Sleep(time.Minute)
		w.Answer(tracked)
		want := []announced{{0, "started"}, {time.Minute, "stopped"}}
		if got := announces(w, start); !reflect.DeepEqual(got, want) {
			t.Fatalf("announces = %+v, want %+v", got, want)
		}
		var removed bool
		for len(e.Events()) > 0 {
			if r, ok := (<-e.Events()).(engine.Removed); ok && r.Torrent == h {
				removed = true
			}
		}
		if !removed {
			t.Fatal("the engine never sent Removed")
		}
	})
}
