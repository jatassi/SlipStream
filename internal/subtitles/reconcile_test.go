package subtitles_test

import (
	"context"
	"fmt"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/subtitles"
	"github.com/jatassi/slipstream/internal/subtitles/subtitlestest"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

func TestTheBulkPassCoversTheLibraryAChunkAtATime(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := world(t, func(tm *subtitles.Timings) { tm.ReconcileChunk = 500 })
		p := w.Profile("Films", subtitlestest.Wanted(t, "en", false, targets.Allow, targets.Always))
		withTrack := audio("fr")
		withTrack.Subtitles = []media.Subtitle{{Codec: "subrip", Language: "de"}}
		var trees []works.Tree
		for i := range 20 {
			trees = append(trees, w.Work(works.Key(fmt.Sprint("w", i)), p))
		}
		seed(t, w, trees, withTrack)
		if got := w.Reconcile(); got != (subtitles.Reconciled{Chunks: 4, Files: 2000}) {
			t.Fatalf("the bulk pass covered %+v, want 2,000 files in 4 chunks", got)
		}
		if got := query(t, w, `SELECT count(*) FROM subtitles_want WHERE unmet = 1`); got[0] != "2000" {
			t.Fatalf("the pass wrote %s unmet wants, want one per file", got[0])
		}
		if got := query(t, w, `SELECT count(*) FROM subtitles_track_ignored`); got[0] != "0" {
			t.Fatalf("an ignore naming a track that changed is still there")
		}
	})
}

func TestOpenClearsEveryLease(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := world(t, nil)
		p := w.Profile("Films", subtitlestest.Wanted(t, "en", false, targets.Allow, targets.Always))
		w.Place(w.Work("heat", p), "Heat/Heat.mkv", audio("fr"))
		w.Write(func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			_, err := tx.ExecContext(ctx, `UPDATE subtitles_want SET attempt_id = ?`, storage.NewUUID())
			return err
		})
		w.Reopen()
		if got := query(t, w, `SELECT count(*) FROM subtitles_want WHERE attempt_id IS NOT NULL`); got[0] != "0" {
			t.Fatalf("after Open %s wants keep a lease", got[0])
		}
	})
}

// seed adds 2,000 files over trees, the first with a subtitles row whose
// ignore names a track the file no longer has.
func seed(t *testing.T, w *subtitlestest.World, trees []works.Tree, m media.Record) {
	t.Helper()
	err := w.Lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		for i := range 2000 {
			id, err := w.Add(ctx, tx, trees[i%len(trees)], fmt.Sprintf("W%d/F%04d.mkv", i%len(trees), i), m)
			if err != nil {
				return err
			}
			if i > 0 {
				continue
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO subtitles_file (file_id, work_id, root_id, map_since) VALUES (?, ?, ?, 1)`, id, trees[0].ID, w.Root.ID); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO subtitles_track_ignored (file_id, stream_index, codec, language, forced) VALUES (?, 0, 'subrip', 'nl', 0)`, id); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestAProviderGainedMakesDueEveryMissingWantSearchedBefore(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := world(t, nil)
		p := w.Profile("Films", subtitlestest.Wanted(t, "en", false, targets.Allow, targets.Always))
		w.Place(w.Work("heat", p), "Heat/Heat.mkv", audio("fr"))
		w.Place(w.Work("ronin", p), "Ronin/Ronin.mkv", audio("fr"))
		now := storage.FromTime(time.Now())
		exec(t, w, `UPDATE subtitles_want SET last_searched_at = 1, due_at = ?`, storage.FromTime(time.Now().Add(time.Hour)))
		exec(t, w, `UPDATE subtitles_provider SET gained_at = ?`, now)
		w.Reconcile()
		if got := query(t, w, `SELECT count(*) FROM subtitles_want WHERE unmet = 1 AND due_at <= ?`, now); got[0] != "2" {
			t.Fatalf("%s of 2 missing wants are due after a provider gained, want both", got[0])
		}
	})
}
