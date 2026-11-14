package link_test

import (
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
)

var t0 = time.Date(2026, 11, 14, 14, 2, 0, 0, time.UTC)

type world struct {
	t      *testing.T
	movies roots.ID
	shows  roots.ID
	due    link.Due
}

func newWorld(t *testing.T) *world {
	w := &world{t: t, movies: roots.ID{UUID: storage.NewUUID()}, shows: roots.ID{UUID: storage.NewUUID()}}
	w.due = link.Due{
		Now:     t0,
		Rows:    make([]link.Row, 0, 8),
		Timings: link.DefaultTimings(),
		Mappings: []link.PlanMapping{
			{Root: w.movies, Library: "1", Folder: path(t, "/media/movies"), Shown: true, Confirmed: true, Scanned: true},
			{Root: w.shows, Library: "2", Folder: path(t, `D:\Shows`), Shown: true, Confirmed: true, Scanned: true},
		},
	}
	return w
}

// row adds a row created age before now: kind at to, or a carry from -> to
// when from isn't empty.
func (w *world) row(root roots.ID, kind link.ChangeKind, from, to string, age time.Duration) *link.Row {
	r := link.Row{
		JobChange: link.JobChange{ID: link.ChangeID{UUID: storage.NewUUID()}, Kind: kind},
		Root:      root,
		Created:   t0.Add(-age),
		NextAt:    t0.Add(-age),
		Step:      link.StepNone,
	}
	if kind != link.ChangeLibrary {
		r.Rel = rel(w.t, to)
	}
	if from != "" {
		r.FromRel = rel(w.t, from)
	}
	w.due.Rows = append(w.due.Rows, r)
	return &w.due.Rows[len(w.due.Rows)-1]
}

func (w *world) ready(wait time.Duration) (link.Pick, []link.ChangeID) {
	p := w.due.Ready(wait)
	ids := make([]link.ChangeID, len(p.Rows))
	for i, r := range p.Rows {
		ids[i] = r.ID
	}
	return p, ids
}

func ids(rs ...*link.Row) []link.ChangeID {
	out := make([]link.ChangeID, len(rs))
	for i, r := range rs {
		out[i] = r.ID
	}
	return out
}

func TestReadyHoldsRowsUnderAMappingNotShownOrConfirmed(t *testing.T) {
	w := newWorld(t)
	w.due.Mappings[1].Confirmed = false
	extra := roots.ID{UUID: storage.NewUUID()}
	w.due.Mappings = append(w.due.Mappings, link.PlanMapping{Root: extra, Library: "3", Folder: path(t, "/media/music"), Confirmed: true})
	a := w.row(w.movies, link.ChangeAdded, "", "Heat/Heat.mkv", time.Minute)
	w.row(w.shows, link.ChangeAdded, "", "Show/S01E01.mkv", time.Minute)
	w.row(extra, link.ChangeAdded, "", "Album/01.flac", time.Minute)
	w.row(roots.ID{UUID: storage.NewUUID()}, link.ChangeAdded, "", "Unmapped.mkv", time.Minute)
	if _, got := w.ready(0); !slices.Equal(got, ids(a)) {
		t.Fatalf("ready %v, want only the row under the confirmed, shown mapping %v", got, ids(a))
	}
}

func TestReadyHoldsNewRowsUntilTheOldestHasWaited(t *testing.T) {
	w := newWorld(t)
	a := w.row(w.movies, link.ChangeAdded, "", "Heat/Heat.mkv", 4*time.Second)
	b := w.row(w.movies, link.ChangeAdded, "", "Ran/Ran.mkv", time.Second)
	p, got := w.ready(10 * time.Second)
	if len(got) != 0 || !p.Next.Equal(t0.Add(6*time.Second)) {
		t.Fatalf("ready %v next %v, want none until %v", got, p.Next, t0.Add(6*time.Second))
	}
	w.due.Now = t0.Add(6 * time.Second)
	if _, got := w.ready(10 * time.Second); !slices.Equal(got, ids(a, b)) {
		t.Fatalf("ready %v, want both once the oldest waited %v", got, ids(a, b))
	}
}

func TestReadyDoesntHoldARetryOrAStartedStepForTheWait(t *testing.T) {
	w := newWorld(t)
	retry := w.row(w.movies, link.ChangeRemoved, "", "Heat/Heat.mkv", time.Second)
	retry.Attempts = 2
	started := w.row(w.movies, link.ChangeMoved, "Ran/Ran.mkv", "Ran (1985)/Ran (1985).mkv", time.Second)
	started.Step = link.StepOldRemoved
	w.row(w.movies, link.ChangeAdded, "", "Alien/Alien.mkv", time.Second)
	if _, got := w.ready(10 * time.Second); !slices.Equal(got, ids(retry, started)) {
		t.Fatalf("ready %v, want the retry and the started carry %v", got, ids(retry, started))
	}
}

func TestReadyHoldsARowUntilItsNextAttempt(t *testing.T) {
	w := newWorld(t)
	r := w.row(w.movies, link.ChangeAdded, "", "Heat/Heat.mkv", time.Minute)
	r.Attempts, r.NextAt = 1, t0.Add(30*time.Second)
	p, got := w.ready(0)
	if len(got) != 0 || !p.Next.Equal(r.NextAt) {
		t.Fatalf("ready %v next %v, want none until %v", got, p.Next, r.NextAt)
	}
}

func TestReadyHoldsEitherHalfOfAnUpgradeForThePairingBoundEvenWithNoWait(t *testing.T) {
	w := newWorld(t)
	removed := w.row(w.movies, link.ChangeRemoved, "", "Heat/Heat (1995) - 1080p.mkv", 30*time.Second)
	removed.Pairing = true
	added := w.row(w.movies, link.ChangeAdded, "", "Alien/Alien - 2160p.mkv", 10*time.Second)
	added.Pairing = true
	plain := w.row(w.movies, link.ChangeAdded, "", "Ran/Ran.mkv", 10*time.Second)
	bound := w.due.Timings.UpgradePairing
	p, got := w.ready(0)
	if !slices.Equal(got, ids(plain)) || !p.Next.Equal(removed.Created.Add(bound)) {
		t.Fatalf("ready %v next %v, want only the plain row until %v", got, p.Next, removed.Created.Add(bound))
	}
	w.due.Now = removed.Created.Add(bound)
	if _, got := w.ready(0); !slices.Equal(got, ids(removed, plain)) {
		t.Fatalf("ready %v, want the removed once its bound passed %v", got, ids(removed, plain))
	}
}

func TestReadyHoldsRowsIntoAFreshMappingBehindOneFreshJob(t *testing.T) {
	w := newWorld(t)
	requested := t0.Add(-time.Minute)
	w.due.Mappings[0].Scanned, w.due.Mappings[0].Requested = false, requested
	w.row(w.movies, link.ChangeAdded, "", "Heat/Heat.mkv", time.Minute)
	w.row(w.movies, link.ChangeAdded, "", "Ran/Ran.mkv", time.Minute)
	library := w.row(w.movies, link.ChangeLibrary, "", "", time.Minute)
	shows := w.row(w.shows, link.ChangeAdded, "", "Show/S01E01.mkv", time.Minute)
	p, got := w.ready(0)
	if !slices.Equal(got, ids(library, shows)) {
		t.Errorf("ready %v, want the library refresh and the scanned mapping's row %v", got, ids(library, shows))
	}
	want := []link.Fresh{{Library: "1", Folder: w.due.Mappings[0].Folder, Requested: requested}}
	if !slices.Equal(p.Fresh, want) {
		t.Errorf("fresh %+v, want %+v", p.Fresh, want)
	}
}

func TestReadySendsAnEarlierRefreshAtACarrysPathInTheSamePass(t *testing.T) {
	w := newWorld(t)
	removed := w.row(w.movies, link.ChangeRemoved, "", "Heat (1995)/Heat (1995).mkv", time.Minute)
	carry := w.row(w.movies, link.ChangeMoved, "Heat/Heat.mkv", "Heat (1995)/Heat (1995).mkv", time.Minute)
	if _, got := w.ready(0); !slices.Equal(got, ids(removed, carry)) {
		t.Fatalf("ready %v, want the removal and the carry %v", got, ids(removed, carry))
	}
}

func TestReadyHoldsAChangeOnAPathAnEarlierCarryTouches(t *testing.T) {
	w := newWorld(t)
	carry := w.row(w.movies, link.ChangeMoved, "Heat/Heat.mkv", "Heat (1995)/Heat (1995).mkv", time.Minute)
	w.row(w.movies, link.ChangeAdded, "", "Heat/Heat.mkv", time.Minute)
	w.row(w.movies, link.ChangeRemoved, "", "Heat (1995)/Heat (1995).mkv", time.Minute)
	elsewhere := w.row(w.shows, link.ChangeAdded, "", "Heat/Heat.mkv", time.Minute)
	if _, got := w.ready(0); !slices.Equal(got, ids(carry, elsewhere)) {
		t.Fatalf("ready %v, want the carry and the same path in another root %v", got, ids(carry, elsewhere))
	}
}

func TestReadyHoldsAChangeBehindAnEarlierRowThatWaits(t *testing.T) {
	w := newWorld(t)
	reported := w.row(w.movies, link.ChangeAdded, "", "Heat/Heat.mkv", time.Minute)
	reported.Step, reported.NextAt = link.StepReported, t0.Add(time.Minute)
	w.row(w.movies, link.ChangeRemoved, "", "Heat/Heat.mkv", time.Minute)
	w.row(w.movies, link.ChangeMoved, "Heat/Heat.mkv", "Heat (1995)/Heat.mkv", time.Minute)
	other := w.row(w.movies, link.ChangeMoved, "Ran/Ran.mkv", "Ran (1985)/Ran.mkv", time.Minute)
	if _, got := w.ready(0); !slices.Equal(got, ids(other)) {
		t.Fatalf("ready %v, want only the carry on another path %v", got, ids(other))
	}
}
