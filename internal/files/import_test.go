package files_test

import (
	"context"
	"testing"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
)

func TestAnImportedFileScansUnchangedFromWhatPlacingItRead(t *testing.T) {
	l := open(t)
	hd := l.target(t, "HD", []release.Quality{release.WEBDL1080p})
	p := l.profile(t, "HD", hd)
	tree := l.work(t, "w1", true, &p)
	svc := l.files(files.NoGrabs{})
	holding := files.HoldingOf(storage.NewUUID(), ".mkv")
	l.put(t, holding, "webdl")
	placed := walked(t, holding, 5, 7).Seen
	var f files.File
	err := l.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		var err error
		f, _, err = svc.Import(ctx, tx, files.Incoming{
			WorkID: tree.ID, ItemID: tree.Items[0].ID, Cut: release.Standard, RootID: l.root.ID, Holding: holding,
			Size: 5, Seen: placed, Media: read1080, Quality: release.WEBDL1080p,
		})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	l.settle(t, svc)

	recorded := l.onRoot(t).Files
	if len(recorded) != 1 || recorded[0].Seen == nil || !recorded[0].Seen.Same(placed) {
		t.Fatalf("the root records %+v, want the imported file with the modification time and file id its placing read, %+v", recorded, placed)
	}
	if shares := l.compare(t, walked(t, f.Path, 5, 7)); len(shares) != 0 {
		t.Fatalf("a scan that walks the imported file as placed has %+v to write, want nothing", shares)
	}
}
