package files_test

import (
	"context"
	"errors"
	"maps"
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
		landed, err := svc.Import(ctx, tx, files.Landing{
			Batch: files.Batch{Work: tree.ID, Root: l.root.ID, Candidates: []files.Candidate{holdingOf(tree, 0, holding, 5, read1080, release.WEBDL1080p)}},
			Seen:  placed, PlacedBy: "copy",
		})
		f = landed.File
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

func TestAnImportAndAFoundFileRecordTheNamesTheyCameWith(t *testing.T) {
	l := open(t)
	hd := l.target(t, "HD", []release.Quality{release.WEBDL1080p})
	p := l.profile(t, "HD", hd)
	tree := l.work(t, "w1", true, &p)
	svc := l.files(files.NoGrabs{})
	err := l.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		c := holdingOf(tree, 0, files.HoldingOf(storage.NewUUID(), ".mkv"), 5, read1080, release.WEBDL1080p)
		c.SourceName = "Scarf.2020.1080p.WEB-DL.mkv"
		_, err := l.land(ctx, tx, svc, tree.ID, c)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	l.scan(t, svc, files.Scan{Work: tree.ID, Root: l.root.ID, Found: []files.Found{
		found(tree.Items[1].ID, release.WEBDL1080p, walked(t, "Shelf w1/Mittens, as found.mkv", 10, 1)),
	}})

	listing, err := svc.WorkFiles(t.Context(), admin, tree.ID)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, f := range listing.Files {
		got[f.Path] = f.SourceName
	}
	want := map[string]string{"Shelf w1/Shelf w1 - Scarf - HD.mkv": "Scarf.2020.1080p.WEB-DL.mkv", "Shelf w1/Mittens, as found.mkv": "Mittens, as found.mkv"}
	if !maps.Equal(got, want) {
		t.Fatalf("the Work's files came with the names %v, want the download's for the import and its own for the found file, %v", got, want)
	}
}

func TestAnImportedFileListsHowItWasPlacedAndAFoundFileNothing(t *testing.T) {
	l := open(t)
	hd := l.target(t, "HD", []release.Quality{release.WEBDL1080p})
	p := l.profile(t, "HD", hd)
	tree := l.work(t, "w1", true, &p)
	svc := l.files(files.NoGrabs{})
	err := l.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := l.land(ctx, tx, svc, tree.ID, holdingOf(tree, 0, files.HoldingOf(storage.NewUUID(), ".mkv"), 5, read1080, release.WEBDL1080p))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	l.scan(t, svc, files.Scan{Work: tree.ID, Root: l.root.ID, Found: []files.Found{
		found(tree.Items[1].ID, release.WEBDL1080p, walked(t, "Shelf w1/Mittens, as found.mkv", 10, 1)),
	}})

	listing, err := svc.WorkFiles(t.Context(), admin, tree.ID)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, f := range listing.Files {
		got[f.Path] = f.PlacedBy
	}
	want := map[string]string{"Shelf w1/Shelf w1 - Scarf - HD.mkv": "copy", "Shelf w1/Mittens, as found.mkv": ""}
	if !maps.Equal(got, want) {
		t.Fatalf("the Work's files list placed by %v, want the import's copy and nothing for the found file, %v", got, want)
	}
}

func TestAPlannedPathIsTakenOnlyByAFileSlipStreamDoesntTrack(t *testing.T) {
	l := open(t)
	tree := l.work(t, "w1", true, nil)
	svc := l.files(files.NoGrabs{})
	l.put(t, "Shelf w1/Foreign.mkv", "theirs")
	l.put(t, "Shelf w1/Tracked.mkv", "ours")
	l.place(t, tree.Items[0].ID, tree.ID, "Shelf w1/Tracked.mkv", release.WEBDL1080p)
	holding := files.HoldingOf(storage.NewUUID(), ".mkv")

	err := svc.Vacant(t.Context(), l.root.ID, holding, "Shelf w1/FOREIGN.mkv")
	if taken, ok := errors.AsType[*files.NameTakenError](err); !ok || *taken != (files.NameTakenError{Path: "Shelf w1/Foreign.mkv", Foreign: true}) {
		t.Errorf("Vacant at the foreign file's name in another case = %v, want it taken by the file SlipStream doesn't track", err)
	}
	for _, to := range []string{"Shelf w1/tracked.mkv", "Shelf w2/Foreign.mkv"} {
		if err := svc.Vacant(t.Context(), l.root.ID, holding, to); err != nil {
			t.Errorf("Vacant(%s) = %v, want nil for a tracked file and a folder that isn't there", to, err)
		}
	}
}
