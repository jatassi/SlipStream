package trackers_test

import (
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/trackers/seeding"
)

// failedGrabOf plants a Grab on torrent that failed after it downloaded.
func (l *library) failedGrabOf(torrent downloadclients.ID) {
	l.t.Helper()
	at := storage.Now()
	l.exec(`INSERT INTO grabs_grab (id, work_id, item_id, edition, source, node_id, release_title, release_size, release_magnet,
		stage, torrent_id, failure_code, created_at, updated_at)
		VALUES (?, ?, ?, 'any', 'search', ?, 'Orchard.1080p', 1, 'magnet:?xt=urn:btih:00', 'failed', ?, 'import_failed', ?, ?)`,
		storage.NewUUID(), l.work, l.item, l.node, torrent, at, at)
}

func TestThePreviewListsWhatTheNextRunWouldDoWithTheSameEvaluation(t *testing.T) {
	l := newLibrary(t)
	seedpool := l.tracker("seedpool", indexers.Private)
	nyaa := l.tracker("Nyaa", indexers.Public)
	start := time.Now().Add(-time.Hour)
	l.torrent("Harbor", &nyaa, false, start, 8*gib, 4*gib)
	meadow := l.torrent("Meadow", &nyaa, false, start, 8*gib, 4*gib)
	l.failedGrabOf(meadow)
	orchard := l.torrent("Orchard", &seedpool, true, start, 8*gib, 4*gib)
	l.grabOf(seedpool, orchard, ptr(1.0), nil)
	attic := l.torrent("Attic", &seedpool, true, start, 0, 4*gib)
	l.grabOf(seedpool, attic, ptr(1.0), nil)

	planned, err := l.x.Preview(t.Context(), admin)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]seeding.Plan{}
	for _, p := range planned {
		got[p.Name] = p.Plan
	}
	want := map[string]seeding.Plan{
		"Harbor":  seeding.Act{Do: seeding.RemoveDelete},
		"Meadow":  seeding.Keep{Why: seeding.KeptNotImported},
		"Orchard": seeding.Wait{Why: seeding.GrabDownloading},
	}
	if len(got) != len(want) {
		t.Fatalf("the preview plans %v, want %v: Attic still owes seedpool, so its goal isn't met", got, want)
	}
	for name, plan := range want {
		if got[name] != plan {
			t.Errorf("the preview plans %#v for %s, want %#v", got[name], name, plan)
		}
	}
	if rows := l.rows(); !rows["Harbor"].Seeding.GoalMet || !rows["Orchard"].Seeding.GoalMet || rows["Attic"].Seeding.GoalMet {
		t.Errorf("the rows read goal met %v, %v and %v for Harbor, Orchard and Attic, want the preview's", rows["Harbor"].Seeding.GoalMet,
			rows["Orchard"].Seeding.GoalMet, rows["Attic"].Seeding.GoalMet)
	}
}
