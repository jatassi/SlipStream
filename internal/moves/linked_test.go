package moves

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/moves/internal/db"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
)

// paired writes torrent unit t and Work unit w placing its first two files
// from t's payload, the first already placed, waiting for t.
func paired(t *testing.T) (lib *storage.DB[storage.Library], torrent, work UnitID) {
	t.Helper()
	lib, batch := seeded(t)
	err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		tu := Unit{
			ID: id[UnitID](), Batch: batch, Seq: 1, Lane: "pair:a>b", Stage: Queued{}, Planned: Planned{At: at},
			Torrent: &TorrentSubject{Torrent: id[downloadclients.ID](), From: Side{Node: id[role.NodeID]()}, To: Side{Node: id[role.NodeID]()}},
		}
		if err := insertUnit(ctx, q, tu, ""); err != nil {
			return err
		}
		wu, err := seedUnit(ctx, tx, batch, 2, "pair:a>b", "")
		if err != nil {
			return err
		}
		if err := q.DeleteUnitFiles(ctx, wu.ID); err != nil {
			return err
		}
		if err := tx.QueryRowContext(ctx, `UPDATE moves_unit SET pairs_with_unit_id = ?, waiting = 'linked_torrent', waiting_at = 0 WHERE id = ? RETURNING id`,
			tu.ID, wu.ID).Scan(&wu.ID); err != nil {
			return fmt.Errorf("pair the Work unit: %w", err)
		}
		for i, f := range []File{
			{Kind: KindLibrary, Via: ViaArriving, ID: id[files.ID](), Sits: rel(t, "Film/Film.mkv"), Size: 10, Index: 0},
			{Kind: KindLibrary, Via: ViaArriving, ID: id[files.ID](), Sits: rel(t, "Film/Film.part2.mkv"), Size: 20, Index: 1},
			{Kind: KindCompanion, Sits: rel(t, "Film/Film.nfo"), Size: 1},
		} {
			f.Position, f.Seen = i, storing.Seen{Modified: at}
			if err := q.InsertUnitFile(ctx, insertFileColumns(wu.ID, f)); err != nil {
				return err
			}
		}
		torrent, work = tu.ID, wu.ID
		return saveFiles(ctx, tx, placedFirst(ctx, tx, t, wu.ID))
	})
	if err != nil {
		t.Fatal(err)
	}
	return lib, torrent, work
}

func placedFirst(ctx context.Context, tx *storage.Tx[file], t *testing.T, id UnitID) Unit {
	u, err := load(ctx, tx, id)
	if err != nil {
		t.Fatal(err)
	}
	u.Files[0].Placed = &storing.Seen{Modified: at}
	return u
}

func rel(t *testing.T, s string) storing.Rel {
	r, err := storing.ParseRel(s)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func reload(t *testing.T, lib *storage.DB[storage.Library], id UnitID) Unit {
	t.Helper()
	var u Unit
	err := lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		u, err = load(ctx, r, id)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return u
}

// TestAPairedWorkWaitsForItsTorrentsPlace has a Work unit paired with a
// torrent unit hold its linked_torrent wait, sends only its unlinked
// files, and takes its lane once the torrent's Place ends.
func TestAPairedWorkWaitsForItsTorrentsPlace(t *testing.T) {
	lib, torrent, work := paired(t)
	w := reload(t, lib, work)
	if a := next(w, Facts{Head: true, NodesUp: true}, at, Rules{}); a != (Hold{Wait: WaitLinkedTorrent}) {
		t.Errorf("the waiting Work unit does %#v, want it held for its torrent", a)
	}
	var sent []string
	for _, f := range copyRequest(w).Files {
		sent = append(sent, f.Name)
	}
	if !slices.Equal(sent, []string{"Film.nfo"}) {
		t.Errorf("the Work unit sends %v, want only its companion", sent)
	}
	err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[file]) error { return releasePair(ctx, tx, torrent, at) })
	if err != nil {
		t.Fatal(err)
	}
	w = reload(t, lib, work)
	if s, ok := w.Stage.(Queued); !ok || s.Wait != "" || w.Due == nil || w.PairsWith == nil {
		t.Errorf("after the torrent's Place the Work unit is %+v due %v paired %v, want it free, due and still paired", s, w.Due, w.PairsWith)
	}
}

// TestAFailedTorrentLeavesItsWorkTheLibrarysFiles fails the torrent unit
// paired with a Work unit: the Work's unplaced linked file comes from its
// library, the placed one stays, and the Work unit is unpaired and due.
func TestAFailedTorrentLeavesItsWorkTheLibrarysFiles(t *testing.T) {
	lib, torrent, work := paired(t)
	err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[file]) error { return unpair(ctx, tx, torrent, at) })
	if err != nil {
		t.Fatal(err)
	}
	w := reload(t, lib, work)
	if got := []Via{w.Files[0].Via, w.Files[1].Via}; !slices.Equal(got, []Via{ViaArriving, ViaLibrary}) {
		t.Errorf("the Work unit's linked files come via %v, want the placed one arriving and the other from the library", got)
	}
	if s, ok := w.Stage.(Queued); !ok || s.Wait != "" || w.Due == nil || w.PairsWith != nil {
		t.Errorf("the Work unit is %+v due %v paired %v, want it unpaired, free and due", s, w.Due, w.PairsWith)
	}
	if got := copyRequest(w).Files; len(got) != 2 {
		t.Errorf("the Work unit sends %d files, want its unlinked file and its companion", len(got))
	}
}
