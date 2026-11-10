package files_test

import (
	"context"
	"testing"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
)

func TestALibraryPlanListsFoundFilesAndCountsMovingOnes(t *testing.T) {
	l := open(t)
	web := l.target(t, "Web", []release.Quality{release.WEBDL1080p})
	p := l.profile(t, "Web", web)
	svc := l.files(files.NoGrabs{})
	w1, w2 := l.work(t, "w1", true, &p), l.work(t, "w2", true, &p)
	moving := l.download(t, svc, w1, 0, release.WEBDL1080p, "w1")
	found := l.download(t, svc, w2, 0, release.WEBDL1080p, "w2")
	l.settle(t, svc)
	err := l.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, `UPDATE files_file SET path = 'Found/scarf.mkv' WHERE id = ?`, found.ID)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	l.renames(t, map[string]string{moving.Path: "Shelf w1/elsewhere.mkv"})

	got, err := svc.Plans(t.Context(), w1.ID, w2.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Work != w1.ID || got[0].Moving != 1 || len(got[0].Rows) != 0 {
		t.Fatalf("plans = %+v, want w1's moving file counted and not listed", got)
	}
	if rows := got[1].Rows; len(rows) != 1 || rows[0].File != found.ID || rows[0].From != "Found/scarf.mkv" || rows[0].To != found.Path {
		t.Fatalf("w2's rows = %+v, want the found file back at %s", rows, found.Path)
	}
}
