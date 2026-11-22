package grabs_test

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	historykeep "github.com/jatassi/slipstream/internal/history"
	"github.com/jatassi/slipstream/internal/housekeeping"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/tasks"
	"github.com/jatassi/slipstream/internal/works"
)

// pastKeep is past Keep history for's default.
var pastKeep = historykeep.KeepFor.Default + 24*time.Hour

type keepDefault struct{}

func (keepDefault) Fields(context.Context) ([]setting.Field, error) { return nil, nil }
func (keepDefault) Apply(context.Context, setting.Patch) error      { return nil }
func (keepDefault) Keep(context.Context) (time.Duration, error) {
	return historykeep.KeepFor.Default, nil
}

// aged is a pipeline whose library holds Grabs written straight to its
// rows, each at the clock's time when written.
type aged struct {
	*pipeline
	item   works.ItemID
	target targets.TargetID
	// gone is a Torrent its client removed, which an imported Grab not
	// seeding brought.
	gone storage.UUID
}

func newAged(t *testing.T) *aged {
	p := newPipeline(t)
	a := &aged{pipeline: p, target: targets.TargetID{UUID: storage.NewUUID()}}
	if err := p.lib.QueryRowContext(t.Context(), `SELECT id FROM works_item WHERE work_id = ?`, p.work).Scan(&a.item); err != nil {
		t.Fatal(err)
	}
	run(t, p.lib, sql(`INSERT INTO targets_target (id, name, created_at) VALUES (?, 'HD', 0)`, a.target))
	a.gone = a.torrent()
	a.removed(a.gone)
	return a
}

// torrent is a Torrent in a Download Client.
func (a *aged) torrent() storage.UUID {
	id := storage.NewUUID()
	run(a.t, a.lib, sql(`INSERT INTO downloadclients_torrent (id, node_id, info_hash, metainfo, name, size, private, trackers, content_type, wanted,
		added_order, added_at) VALUES (?, ?, randomblob(20), x'00', 'Attic.WEB-DL', 1, 0, '[]', 'fixture', 'started', abs(random() % 1000000), 0)`, id, a.node))
	return id
}

func (a *aged) removed(torrent storage.UUID) {
	run(a.t, a.lib, sql(`UPDATE downloadclients_torrent SET removed_at = 1, removed_reason = 'admin' WHERE id = ?`, torrent))
}

// finished is a Grab that finished now, failed with code unless code is
// "", holding the cell when cell is true, of torrent, or of a removed one
// when it is zero and the Grab imported.
type finished struct {
	code    grabs.Code
	torrent storage.UUID
	cell    bool
	// created is how long before now it was made.
	created time.Duration
}

func (a *aged) finish(f finished) grabs.GrabID {
	a.t.Helper()
	id := grabs.GrabID{UUID: storage.NewUUID()}
	stage, code, at := "imported", any(nil), time.Now()
	if f.code != "" {
		stage, code = "failed", string(f.code)
	}
	var torrent any
	switch {
	case !f.torrent.IsZero():
		torrent = f.torrent
	case f.code == "":
		torrent = a.gone
	}
	run(a.t, a.lib, sql(`INSERT INTO grabs_grab (id, work_id, item_id, edition, source, node_id, release_title, release_size, release_magnet,
		stage, failure_code, failed_stage, failure_message, torrent_id, created_at, updated_at)
		VALUES (?, ?, ?, 'any', 'search', ?, 'Attic.WEB-DL', 1, 'magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567',
		?, ?, iif(? IS NULL, NULL, 'downloading'), iif(? IS NULL, NULL, 'The import failed.'), ?, ?, ?)`,
		id, a.work, a.item, a.node, stage, code, code, code, torrent, at.Add(-f.created).UnixMicro(), at.UnixMicro()))
	if f.cell {
		run(a.t, a.lib, sql(`INSERT INTO grabs_target (grab_id, position, item_id, target_id) VALUES (?, 0, ?, ?)`, id, a.item, a.target))
	}
	return id
}

// importing is an unfinished Import of grab.
func (a *aged) importing(grab grabs.GrabID) {
	run(a.t, a.lib, sql(`INSERT INTO imports_import (id, source, grab_id, node_id, info_hash, content_type, hint, created_at)
		VALUES (?, 'grab', ?, ?, randomblob(20), 'fixture', '{}', 0)`, storage.NewUUID(), grab, a.node))
}

// housekeeping runs the Finished Grabs step as Housekeeping does.
func (a *aged) housekeeping() {
	a.t.Helper()
	task := housekeeping.Task(housekeeping.Config{Reconciling: func(context.Context) (bool, error) { return false, nil }, Keep: keepDefault{}, Batch: 1000, Steps: []housekeeping.Step{a.g.Retention()}})
	if err := task.Run(a.t.Context(), tasks.Unit{Key: "finished-grabs"}, &tasks.Progress{}); err != nil {
		a.t.Fatal(err)
	}
}

func (a *aged) exists(id grabs.GrabID) bool {
	a.t.Helper()
	_, err := a.g.Get(a.t.Context(), identity.Admin{}, id)
	if err != nil && !errors.Is(err, grabs.ErrNoGrab) {
		a.t.Fatal(err)
	}
	return err == nil
}

// cell is what the cell reads of its Grabs.
func (a *aged) cell() files.GrabCell {
	a.t.Helper()
	var got map[files.CellKey]files.GrabCell
	err := a.lib.Read(a.t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		got, err = grabs.GrabCells{InStep: func(role.NodeID) bool { return true }, Router: byRoot{}}.GrabCells(ctx, r, works.Only([]works.WorkID{a.work}))
		return err
	})
	if err != nil {
		a.t.Fatal(err)
	}
	return got[files.CellKey{Item: a.item, Edition: targets.AnyEdition, Target: a.target}]
}

func TestAnOldGrabStillInADownloadClientStaysWithItsImportUntilTheTorrentGoes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		a := newAged(t)
		torrent := a.torrent()
		id := a.finish(finished{torrent: torrent})
		run(t, a.lib, sql(`INSERT INTO imports_import (id, source, grab_id, node_id, info_hash, content_type, hint, created_at, finished_at)
			VALUES (?, 'grab', ?, ?, randomblob(20), 'fixture', '{}', 0, 1)`, storage.NewUUID(), id, a.node))
		time.Sleep(pastKeep)
		a.housekeeping()
		if v := a.view(id); v.Import == nil {
			t.Fatalf("the seeding Grab reads %+v, want it kept with its Import", v)
		}
		a.removed(torrent)
		a.housekeeping()
		if a.exists(id) {
			t.Fatal("the Grab stays once its torrent was removed")
		}
	})
}

func TestAnOldFailedGrabThatNeedsTheAdminStaysUntilItsCellIsDealtWith(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		a := newAged(t)
		failed := a.finish(finished{code: grabs.ImportFailed, cell: true})
		time.Sleep(pastKeep)
		a.housekeeping()
		if !a.exists(failed) || a.cell().State != files.GrabFailed {
			t.Fatalf("the failed Grab its cell reads went or its cell reads %+v, want it kept and the cell Failed", a.cell())
		}
		imported := a.finish(finished{cell: true})
		a.housekeeping()
		if a.exists(failed) || !a.exists(imported) {
			t.Fatal("once a newer Grab of its cell imported, want the old failed Grab gone and the new one kept")
		}
	})
}

func TestAnOldFailedGrabNoCellReadsFromGoes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		a := newAged(t)
		alone := a.finish(finished{code: grabs.ImportFailed})
		time.Sleep(pastKeep)
		a.housekeeping()
		if a.exists(alone) {
			t.Fatal("an old failed Grab no cell reads from stays")
		}
	})
}

func TestAnOldGrabAnUnfinishedImportNamesStays(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		a := newAged(t)
		id := a.finish(finished{})
		a.importing(id)
		time.Sleep(pastKeep)
		a.housekeeping()
		if !a.exists(id) {
			t.Fatal("a Grab an unfinished Import names went")
		}
	})
}

func TestAnOlderFailedGrabDoesntTurnItsCellFailedWhenTheNewerImportedOneAgesOutFirst(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		a := newAged(t)
		imported := a.finish(finished{cell: true})
		time.Sleep(time.Hour)
		failed := a.finish(finished{code: grabs.ImportFailed, cell: true, created: 2 * time.Hour})
		time.Sleep(historykeep.KeepFor.Default - 30*time.Minute)
		a.housekeeping()
		if !a.exists(imported) || !a.exists(failed) {
			t.Fatal("with only the imported Grab past Keep history for, want both Grabs kept")
		}
		if c := a.cell(); c.State == files.GrabFailed {
			t.Fatalf("the cell reads %+v, want it not Failed", c)
		}
		time.Sleep(time.Hour)
		a.housekeeping()
		if a.exists(imported) || a.exists(failed) {
			t.Fatal("with both Grabs past Keep history for, want both gone")
		}
	})
}

func TestTheWalkPassesFifteenHundredSeedingGrabsToTheTenItCanDelete(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		a := newAged(t)
		torrent := a.torrent()
		a.many(1500, torrent)
		var gone []grabs.GrabID
		for range 10 {
			gone = append(gone, a.finish(finished{}))
		}
		time.Sleep(pastKeep)
		tally, err := a.g.PruneOlderThan(t.Context(), time.Now().Add(-historykeep.KeepFor.Default), 1000, func() {})
		if err != nil {
			t.Fatal(err)
		}
		if tally != (housekeeping.Tally{Deleted: 10, Kept: 1500}) {
			t.Fatalf("the prune deleted and kept %+v, want 10 deleted past 1,500 kept", tally)
		}
		for _, id := range gone {
			if a.exists(id) {
				t.Fatalf("Grab %s stays behind the seeding ones", id)
			}
		}
	})
}

func TestAPruneOfManyBatchesFinishesWhileAnImportCommitsBetweenThem(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		a := newAged(t)
		torrent := a.torrent()
		a.many(1200, a.gone)
		seeding := []grabs.GrabID{a.finish(finished{torrent: torrent}), a.finish(finished{torrent: torrent})}
		a.many(1200, a.gone)
		late := a.finish(finished{})
		time.Sleep(pastKeep)
		batches := 0
		tally, err := a.g.PruneOlderThan(t.Context(), time.Now().Add(-historykeep.KeepFor.Default), 1000, func() {
			batches++
			if batches == 1 {
				a.importing(late)
			}
		})
		if err != nil {
			t.Fatal(err)
		}
		if tally != (housekeeping.Tally{Deleted: 2400, Kept: 3}) || batches != 3 {
			t.Fatalf("the prune deleted and kept %+v in %d batches, want 2,400 deleted and 3 kept in 3", tally, batches)
		}
		for _, id := range append(seeding, late) {
			if !a.exists(id) {
				t.Fatalf("Grab %s went, want the seeding ones and the one an Import named between batches kept", id)
			}
		}
	})
}

// many finishes n imported Grabs of torrent in one Write, one microsecond
// apart from now on, and moves the clock past them.
func (a *aged) many(n int, torrent storage.UUID) {
	a.t.Helper()
	at := time.Now().UnixMicro()
	run(a.t, a.lib, sql(`WITH RECURSIVE k(i) AS (SELECT 1 UNION ALL SELECT i + 1 FROM k WHERE i < ?)
		INSERT INTO grabs_grab (id, work_id, item_id, edition, source, node_id, release_title, release_size, release_magnet,
		stage, torrent_id, created_at, updated_at)
		SELECT randomblob(16), ?, ?, 'any', 'search', ?, 'Attic.WEB-DL', 1, 'magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567',
		'imported', ?, ? - i, ? + i FROM k`, n, a.work, a.item, a.node, torrent, at, at))
	time.Sleep(time.Duration(n+1) * time.Microsecond)
}
