package moves

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/moves/internal/db"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/transfers"
	"github.com/jatassi/slipstream/internal/works"
)

var at = time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

const kind = works.TypeID("kind")

func id[T ~struct{ storage.UUID }]() T { return T{UUID: storage.NewUUID()} }

// seeded is a library holding moves' tables and one batch.
func seeded(t testing.TB) (lib *storage.DB[storage.Library], batch BatchID) {
	t.Helper()
	lib = storagetest.Library(t, Schema())
	batch = id[BatchID]()
	err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[file]) error {
		return db.New(tx).InsertBatch(ctx, db.InsertBatchParams{ID: batch, Origin: string(OriginWork), CreatedAt: storage.FromTime(at), PlanTokenHash: []byte("t"), Notes: "{}"})
	})
	if err != nil {
		t.Fatal(err)
	}
	return lib, batch
}

func seedUnit(ctx context.Context, tx *storage.Tx[file], batch BatchID, seq int64, lane Lane, wait Wait) (Unit, error) {
	typ := kind
	u := Unit{
		ID: id[UnitID](), Batch: batch, Seq: seq, Lane: lane, Stage: Queued{}, Planned: Planned{At: at},
		Subject: WorkSubject{Work: id[works.WorkID](), Type: typ, From: End{Root: id[roots.ID](), Node: id[role.NodeID]()}, To: End{Root: id[roots.ID](), Node: id[role.NodeID]()}},
	}
	p := db.InsertUnitParams{
		ID: u.ID, BatchID: batch, Seq: seq, Kind: "work", WorkID: u.Subject.Work, ContentType: &typ, FromRootID: u.Subject.From.Root, FromNodeID: u.Subject.From.Node,
		ToRootID: u.Subject.To.Root, ToNodeID: u.Subject.To.Node, Lane: string(lane), PlannedAt: storage.FromTime(at), Notes: "{}",
	}
	if wait != "" {
		p.Waiting, p.WaitingAt = new(string(wait)), storage.FromTime(at)
		u.Stage = Queued{Wait: wait, Since: at}
	}
	return u, db.New(tx).InsertUnit(ctx, p)
}

func TestStageRoundTrip(t *testing.T) {
	lib, batch := seeded(t)
	later := at.Add(time.Hour)
	transfer := transfers.ID{UUID: storage.NewUUID()}
	stages := []Stage{
		Queued{Wait: WaitRenames, Since: at, Pass: 2},
		Copying{Transfer: transfer, Pass: 1, Wait: WaitNoRoute, Since: at},
		Placing{Pass: 1, Transfer: &transfer, Wait: WaitSpace, Since: at},
		Placing{},
		Switching{Wait: WaitPlaying, Since: at, SwitchNow: &later},
		Cleaning{SwitchedAt: at, Wait: WaitMediaServers, Since: later},
		Done{SwitchedAt: at, CleanedAt: later},
		Failed{Code: DestinationCheckFailed, At: PhaseSwitch, Detail: "gone", Undone: &later},
		Canceled{At: PhasePlace},
	}
	for i, st := range stages {
		t.Run(fmt.Sprintf("%T", st), func(t *testing.T) {
			got := roundTrip(t, lib, batch, i, st)
			if !reflect.DeepEqual(got.Stage, st) {
				t.Errorf("stage read back is %#v, want %#v", got.Stage, st)
			}
			if !got.Notes.Released || len(got.Notes.Skipped) != 1 || (got.Due == nil) != (st.Phase() == PhaseDone) {
				t.Errorf("notes %+v and due %v read back, want them as written", got.Notes, got.Due)
			}
		})
	}
}

// roundTrip saves a unit at stage st, with notes and due, and loads it.
func roundTrip(t *testing.T, lib *storage.DB[storage.Library], batch BatchID, i int, st Stage) Unit {
	t.Helper()
	later := at.Add(time.Hour)
	var got Unit
	err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[file]) error {
		u, err := seedUnit(ctx, tx, batch, int64(i), Lane(fmt.Sprintf("vol:a:%d>2", i)), "")
		if err != nil {
			return err
		}
		u.Stage, u.Notes = st, Notes{Skipped: []string{"a.srt"}, Released: true}
		if st.Phase() != PhaseDone {
			u.Due = &later
		}
		if err := save(ctx, tx, u); err != nil {
			return err
		}
		got, err = load(ctx, tx, u.ID)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestEveryWaitHasARule(t *testing.T) {
	for _, w := range Waits() {
		r, ok := waits[w]
		switch {
		case !ok:
			t.Errorf("wait %s has no rule", w)
		case len(r.on) == 0 && !r.laneFreed && !r.every:
			t.Errorf("wait %s has a rule nothing ends", w)
		}
	}
	if len(waits) != len(Waits()) {
		t.Errorf("the registry holds %d waits, want the %d stored ones", len(waits), len(Waits()))
	}
}

func TestEveryActionRuns(t *testing.T) {
	for _, a := range []Action{
		Hold{}, TakeLane{}, PlaceFile{}, RemoveDropped{}, FinalPass{}, Snapshot{}, SwitchWork{}, RemoveSource{}, ReleaseServers{},
		FinishCleanup{}, Undo{}, Reconcile{}, Nothing{},
	} {
		if io(a) && actionName(a) == "" {
			t.Errorf("%T runs IO and has no step", a)
		}
	}
}

func TestDefaultTimingsCheck(t *testing.T) {
	if err := DefaultTimings().Check(); err != nil {
		t.Errorf("the default Timings fail their check: %v", err)
	}
	late := DefaultTimings()
	late.HandoffBound = late.ReplyWait + late.BoundMargin
	if err := late.Check(); err == nil {
		t.Error("Timings whose local window ends at the rollback pass their check")
	}
}

func placed(f File) File {
	f.Placed = &storing.Seen{Modified: at}
	return f
}

func TestNext(t *testing.T) {
	r := Rules{PlexBound: 30 * time.Minute, PlayingEvery: 30 * time.Second}
	library := File{Position: 0, Kind: KindLibrary}
	companion := File{Position: 1, Kind: KindCompanion}
	server := link.ServerID{UUID: storage.NewUUID()}
	now := at.Add(10 * time.Minute)
	for _, c := range []struct {
		name string
		u    Unit
		f    Facts
		want Action
	}{
		{"a restored unit reconciles first", Unit{Reconcile: true, Stage: Switching{}}, Facts{}, Reconcile{}},
		{"a queued unit behind its lane's head waits", Unit{Stage: Queued{}}, Facts{NodesUp: true}, Hold{Wait: WaitPairBusy}},
		{"a head whose Node is away waits", Unit{Stage: Queued{}}, Facts{Head: true}, Hold{Wait: WaitNodeOffline}},
		{"a head takes its lane", Unit{Stage: Queued{}}, Facts{Head: true, NodesUp: true}, TakeLane{}},
		{"a copying unit mirrors its Transfer", Unit{Stage: Copying{}}, Facts{Transfer: &transfers.View{State: transfers.NeedsSpace}}, Hold{Wait: WaitSpace}},
		{"a dropped file goes first", Unit{Stage: Placing{}, Files: []File{library, func() File { f := placed(companion); f.Dropped = &at; return f }()}}, Facts{}, RemoveDropped{Pos: 1}},
		{"an unplaced file places", Unit{Stage: Placing{}, Files: []File{placed(library), companion}}, Facts{}, PlaceFile{Pos: 1}},
		{"every file placed passes once more", Unit{Stage: Placing{}, Files: []File{placed(library)}}, Facts{}, FinalPass{}},
		{"an Import holds the switch", Unit{Stage: Switching{SwitchNow: &at}}, Facts{Importing: true}, Hold{Wait: WaitImporting}},
		{"open files hold the switch", Unit{Stage: Switching{SwitchNow: &at}}, Facts{OpenFiles: true}, Hold{Wait: WaitOpenFiles}},
		{"Switch now passes playing", Unit{Stage: Switching{SwitchNow: &at}}, Facts{Playing: []link.ServerID{server}}, SwitchWork{}},
		{"playing waits and reads again", Unit{Stage: Switching{}}, Facts{Playing: []link.ServerID{server}}, Hold{Wait: WaitPlaying, Until: now.Add(r.PlayingEvery)}},
		{"a missing snapshot is taken", Unit{Stage: Switching{}}, Facts{Snapshotted: map[link.ServerID]bool{server: false}}, Snapshot{}},
		{"a snapshot still missing waits", Unit{Stage: Switching{Wait: WaitMediaServers}}, Facts{Snapshotted: map[link.ServerID]bool{server: false}}, Hold{Wait: WaitMediaServers}},
		{"nothing holding switches", Unit{Stage: Switching{}}, Facts{Snapshotted: map[link.ServerID]bool{server: true}}, SwitchWork{}},
		{"Plex not safe waits until the bound", Unit{Stage: Cleaning{SwitchedAt: at}}, Facts{Safe: map[link.ServerID]bool{server: false}}, Hold{Wait: WaitMediaServers, Until: at.Add(r.PlexBound)}},
		{"past the bound the servers are released", Unit{Stage: Cleaning{SwitchedAt: at.Add(-time.Hour)}}, Facts{Safe: map[link.ServerID]bool{server: false}}, ReleaseServers{}},
		{"a pending source file is removed", Unit{Stage: Cleaning{SwitchedAt: at}, Files: []File{{Position: 3, Cleanup: CleanupPending}}}, Facts{}, RemoveSource{Pos: 3}},
		{"every source file done finishes", Unit{Stage: Cleaning{SwitchedAt: at}, Files: []File{{Cleanup: CleanupDone}}}, Facts{}, FinishCleanup{}},
		{"an Import from a torrent holds its guard", Unit{Torrent: &TorrentSubject{}, Stage: Switching{Handoff: &HandoffAt{Step: HandoffGuard}}},
			Facts{Importing: true, Handoff: downloadclients.Handoff{Wanted: downloadclients.Paused}}, Hold{Wait: WaitImporting}},
		{"a failed unit undoes", Unit{Stage: Failed{}}, Facts{}, Undo{}},
		{"an undone cancel rests", Unit{Stage: Canceled{Undone: &at}}, Facts{}, Nothing{}},
		{"a done unit rests", Unit{Stage: Done{}}, Facts{}, Nothing{}},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := next(c.u, c.f, now, r); !reflect.DeepEqual(got, c.want) {
				t.Errorf("next is %#v, want %#v", got, c.want)
			}
		})
	}
}

func TestVerdictOf(t *testing.T) {
	from, to := id[roots.ID](), id[roots.ID]()
	node := id[role.NodeID]()
	server := link.ServerID{UUID: storage.NewUUID()}
	vol := &storing.Volume{ID: "1"}
	unit := PlanUnit{
		Subject: WorkSubject{Type: kind, From: End{Root: from, Node: node}, To: End{Root: to, Node: node}},
		Lane:    laneOf(End{Node: node}, End{Node: node}, "1", "2"), Files: []File{{Kind: KindLibrary}}, Bytes: 10,
	}
	ready := func(root roots.ID) roots.Destination {
		return roots.Destination{Root: roots.Root{ID: root, ContentType: kind, Node: node}, State: roots.Ready, Volume: vol}
	}
	green := planFacts{from: ready(from), to: ready(to), budget: 200, longest: 40}
	for _, c := range []struct {
		name  string
		u     PlanUnit
		f     func(f planFacts) planFacts
		check Check
		color Color
	}{
		{"a destination not ready", unit, func(f planFacts) planFacts { f.to.State = roots.Offline; return f }, CheckDestination, Red},
		{"a path past the destination's budget", unit, func(f planFacts) planFacts { f.longest = 201; return f }, CheckNames, Red},
		{"already moving", unit, func(f planFacts) planFacts { f.moving = true; return f }, CheckMoving, Red},
		{"a probe that missed its deadline", unit, func(f planFacts) planFacts { f.probe = context.DeadlineExceeded; return f }, CheckProbe, Red},
		{"a missing source file", unit, func(f planFacts) planFacts { f.missing = 1; return f }, CheckProbe, Red},
		{"an occupied destination", unit, func(f planFacts) planFacts { f.occupied = 1; return f }, CheckFolder, Red},
		{"no room", unit, func(f planFacts) planFacts { f.room = new(int64(9)); return f }, CheckSpace, Red},
		{"a server showing only the source", unit, func(f planFacts) planFacts {
			f.covers = []mediaservers.Cover{{Server: server, Root: from, State: mediaservers.MapConfirmed}, {Server: server, Root: to, State: mediaservers.MapNone}}
			return f
		}, CheckMediaServers, Red},
		{"a server unconfirmed at the destination", unit, func(f planFacts) planFacts {
			f.covers = []mediaservers.Cover{{Server: server, Root: from, State: mediaservers.MapConfirmed}, {Server: server, Root: to, State: mediaservers.MapUnconfirmed}}
			return f
		}, CheckMediaServers, Amber},
		{"playing now", unit, func(f planFacts) planFacts { f.playing = true; return f }, CheckWaits, Amber},
	} {
		t.Run(c.name, func(t *testing.T) {
			v := verdictOf(c.u, c.f(green))
			if v.Color != c.color || len(v.Reasons) != 1 || v.Reasons[0].Check != c.check || v.Reasons[0].Words == "" {
				t.Errorf("verdict is %+v, want one %s reason from %s", v, c.color, c.check)
			}
		})
	}
	if v := verdictOf(unit, green); v.Color != Green || len(v.Reasons) != 0 {
		t.Errorf("a unit with nothing in its way is %+v, want green", v)
	}
	across := unit
	across.Subject.To.Node = id[role.NodeID]()
	across.Lane = laneOf(across.Subject.From, across.Subject.To, "1", "1")
	if v := verdictOf(across, green); v.Color != Green {
		t.Errorf("a unit to another Node's root is %+v, want green", v)
	}
}

func TestPassQueryBudget(t *testing.T) {
	lib, batch := seeded(t)
	const lanes, queued, parked = 90, 15_000, 5_000
	seedPass(t, lib, batch, lanes, queued, parked)
	m := New(Config{Timings: DefaultTimings()})
	var units []Unit
	var heads map[UnitID]bool
	err := lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		units, heads, err = m.dueUnits(ctx, r, at)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(heads) != lanes || len(units) > lanes+m.cfg.Timings.PassPage {
		t.Errorf("a pass read %d units, %d of them heads, want the %d heads and at most %d due", len(units), len(heads), lanes, m.cfg.Timings.PassPage)
	}
	for _, u := range units {
		if _, parked := u.Stage.(Switching); parked {
			t.Fatalf("a pass read unit %s, parked at its switch with no due time", u.ID)
		}
	}
}

// seedPass seeds queued units on lanes lanes, each lane's first at its
// head, and parked units waiting at their switch with no due time.
func seedPass(t *testing.T, lib *storage.DB[storage.Library], batch BatchID, lanes, queued, parked int) {
	t.Helper()
	err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[file]) error {
		for i := range queued {
			wait := WaitPairBusy
			if i < lanes {
				wait = ""
			}
			if _, err := seedUnit(ctx, tx, batch, int64(i), Lane(fmt.Sprintf("vol:n:%02d>x", i%lanes)), wait); err != nil {
				return err
			}
		}
		for i := range parked {
			u, err := seedUnit(ctx, tx, batch, int64(queued+i), Lane(fmt.Sprintf("vol:m:%04d>y", i)), "")
			if err != nil {
				return err
			}
			u.Stage = Switching{Wait: WaitMediaServers, Since: at}
			if err := save(ctx, tx, u); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestMergedKeepsPlacesAndDropsChanged(t *testing.T) {
	a, b := id[files.ID](), id[files.ID]()
	rel := func(s string) storing.Rel { r, _ := storing.ParseRel(s); return r }
	was := []File{
		placed(File{Position: 0, Kind: KindLibrary, ID: a, Sits: rel("M/a.mkv"), Size: 1}),
		placed(File{Position: 1, Kind: KindLibrary, ID: b, Sits: rel("M/b.mkv"), Size: 1}),
	}
	got := merged(was, []File{
		{Kind: KindLibrary, ID: a, Sits: rel("M/a.mkv"), Size: 1},
		{Kind: KindLibrary, ID: b, Sits: rel("M/b.mkv"), Size: 2},
	})
	if len(got) != 3 || got[0].Placed == nil || got[1].Dropped == nil || got[2].Position != 2 || got[2].Placed != nil || got[2].Size != 2 {
		t.Errorf("merged is %+v, want a kept, b's old copy dropped and b again at position 2", got)
	}
}

func TestReconcileOf(t *testing.T) {
	from, to := id[roots.ID](), id[roots.ID]()
	rel := func(s string) storing.Rel { r, _ := storing.ParseRel(s); return r }
	a, b := rel("M/a.mkv"), rel("M/a.srt")
	u := Unit{Subject: WorkSubject{From: End{Root: from}, To: End{Root: to}}, Files: []File{{Position: 0, Sits: a, Size: 10}, {Position: 1, Sits: b, Size: 2}}}
	at := func(root roots.ID, rels ...storing.Rel) storing.Probed {
		p := storing.Probed{Files: map[storing.Library]storing.ProbedFile{}, Occupants: map[storing.Library][]storing.ProbedFile{}}
		for _, r := range rels {
			size := int64(10)
			if r == b {
				size = 2
			}
			k := storing.Library{Root: root.UUID, Rel: r}
			p.Files[k] = storing.ProbedFile{Rel: r, Size: size}
			p.Occupants[k] = []storing.ProbedFile{{Rel: r, Size: size}}
		}
		return p
	}
	for _, c := range []struct {
		name     string
		src, dst storing.Probed
		want     reconciled
	}{
		{"a whole source moves again", at(from, a, b), at(to, a), again},
		{"a gone source with every file placed switches", at(from), at(to, a, b), switchRestored},
		{"a gone source with a file missing at the destination fails", at(from), at(to, a), lost},
		{"half a source fails", at(from, a), at(to, a, b), lost},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := reconcileOf(u, c.src, c.dst); got != c.want {
				t.Errorf("reconcileOf is %d, want %d", got, c.want)
			}
		})
	}
}
