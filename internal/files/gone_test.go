package files_test

import (
	"context"
	"errors"
	"maps"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/roots/rootstest"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// goneRoot is l with a Work holding one gone file and one live file under
// HD, and l's files owner.
func goneRoot(t *testing.T) (*library, *files.Files, works.Tree) {
	t.Helper()
	l := open(t)
	hd := l.target(t, "HD", []release.Quality{release.WEBDL1080p})
	p := l.profile(t, "HD", hd)
	tree := l.work(t, "w1", true, &p)
	svc := l.files(files.NoGrabs{})
	l.place(t, tree.Items[0].ID, tree.ID, "Shelf w1/Scarf.mkv", release.WEBDL1080p, hd.ID)
	l.place(t, tree.Items[1].ID, tree.ID, "Shelf w1/Mittens.mkv", release.WEBDL1080p, hd.ID)
	l.markGone(t, svc, tree.ID, "Shelf w1/Scarf.mkv")
	return l, svc, tree
}

func TestAGoneFileCountsForNoneOfARootsChecks(t *testing.T) {
	l, _, tree := goneRoot(t)
	var (
		use      map[roots.ID]roots.Use
		proof    []string
		recorded []string
		has      bool
	)
	err := l.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		c := files.RootContents{}
		var err error
		if use, err = c.Use(ctx, r, []roots.ID{l.root.ID}); err != nil {
			return err
		}
		if proof, err = c.Proof(ctx, r, l.root.ID); err != nil {
			return err
		}
		rs, err := c.Recorded(ctx, r, l.root.ID)
		for _, rec := range rs {
			recorded = append(recorded, rec.Path.String())
		}
		if err != nil {
			return err
		}
		has, err = c.HasFiles(ctx, r, tree.ID)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	live := []string{"Shelf w1/Mittens.mkv"}
	if use[l.root.ID] != (roots.Use{Files: 1, Bytes: 1024}) || !slices.Equal(proof, live) || !slices.Equal(recorded, live) || !has {
		t.Fatalf("the root counts %+v, proves %v and re-home verifies %v, has files %t; want the live file alone", use[l.root.ID], proof, recorded, has)
	}
}

func TestDeletingARootWithOnlyGoneFilesForgetsThem(t *testing.T) {
	l, svc, tree := goneRoot(t)
	l.markGone(t, svc, tree.ID, "Shelf w1/Mittens.mkv")

	if err := l.held.Delete(t.Context(), admin, l.root.ID, nil); err != nil {
		t.Fatalf("deleting a root whose files are all gone = %v, want it to go", err)
	}
	if n := fileCount(t, l); n != 0 {
		t.Fatalf("%d file rows remain, want the root's gone files forgotten", n)
	}
}

func TestAWorkWhoseFilesAreAllGoneChangesRootAndForgetsThem(t *testing.T) {
	l, svc, tree := goneRoot(t)
	other := rootstest.Root(t, l.lib, l.root.Node, contenttypetest.ID, t.TempDir())

	if _, refused, err := l.held.HomeAll(t.Context(), admin, []works.WorkID{tree.ID}, other.ID); err != nil || !errors.Is(refused[tree.ID], roots.ErrHasFiles) {
		t.Fatalf("moving a Work with a live file = %v, %v; want ErrHasFiles", refused, err)
	}
	l.markGone(t, svc, tree.ID, "Shelf w1/Mittens.mkv")
	moved, refused, err := l.held.HomeAll(t.Context(), admin, []works.WorkID{tree.ID}, other.ID)
	if err != nil || len(refused) != 0 || !slices.Equal(moved, []works.WorkID{tree.ID}) {
		t.Fatalf("moving a Work whose files are all gone = %v, %v, %v; want it moved", moved, refused, err)
	}
	if n := fileCount(t, l); n != 0 {
		t.Fatalf("%d file rows remain, want the Work's gone files forgotten", n)
	}
}

func TestForgetDeletesGoneFilesAndRefusesLiveOnes(t *testing.T) {
	l, svc, tree := goneRoot(t)
	ids := map[string]files.ID{}
	for _, f := range must[files.Listing](t)(svc.WorkFiles(t.Context(), admin, tree.ID)).Files {
		ids[f.Path] = f.ID
	}
	gone, live := ids["Shelf w1/Scarf.mkv"], ids["Shelf w1/Mittens.mkv"]

	_, err := svc.Forget(t.Context(), admin, []files.ID{gone, live})
	if notGone, ok := errors.AsType[*files.NotGoneError](err); !ok || notGone.Live != 1 || fileCount(t, l) != 2 {
		t.Fatalf("forgetting a gone and a live file = %v, want *NotGoneError{1} and both kept", err)
	}
	touched, err := svc.Forget(t.Context(), admin, []files.ID{gone})
	if err != nil || !slices.Equal(touched, []works.WorkID{tree.ID}) || fileCount(t, l) != 1 {
		t.Fatalf("forgetting the gone file = %v, %v with %d files left; want its Work and the live file alone", touched, err, fileCount(t, l))
	}
}

func TestHealthAndTheLibraryCountGoneFiles(t *testing.T) {
	l, _, tree := goneRoot(t)
	var (
		byRoot map[roots.ID]files.Tally
		byWork map[works.WorkID]int
	)
	err := l.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		if byRoot, err = files.GoneOf(ctx, r); err != nil {
			return err
		}
		byWork, err = files.GoneOfWorks(ctx, r, works.Every[works.WorkID](nil))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	tally := byRoot[l.root.ID]
	if tally.Count != 1 || len(tally.First) != 1 || tally.First[0].Path != "Shelf w1/Scarf.mkv" || !maps.Equal(byWork, map[works.WorkID]int{tree.ID: 1}) {
		t.Fatalf("gone files by root %+v and by Work %v, want Scarf's one file", byRoot, byWork)
	}
}

func TestLeaveFilesListsTheLiveFilesAtTheirPaths(t *testing.T) {
	l, svc, tree := goneRoot(t)
	l.download(t, svc, tree, 0, release.WEBDL1080p, "moving")
	var left []files.File
	err := l.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		left, err = files.LeftIn(ctx, r, []works.WorkID{tree.ID})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(left) != 1 || left[0].Path != "Shelf w1/Mittens.mkv" {
		t.Fatalf("a Leave files delete would list %+v, want the live file at its path alone, not the gone one or the one still moving", left)
	}
}

func TestDeletingAWorksFilesRemovesOnlyTheLiveOnesBytes(t *testing.T) {
	l, _, tree := goneRoot(t)
	err := l.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return files.DoomIn(ctx, tx, []works.WorkID{tree.ID})
	})
	if err != nil {
		t.Fatal(err)
	}
	var doomed []string
	rows, err := l.lib.QueryContext(t.Context(), `SELECT path FROM files_doomed`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			t.Fatal(err)
		}
		doomed = append(doomed, p)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if n := fileCount(t, l); n != 0 || !slices.Equal(doomed, []string{"Shelf w1/Mittens.mkv"}) {
		t.Fatalf("Delete files left %d rows and queued %v for removal, want no rows and the live file's bytes alone", n, doomed)
	}
}

func TestAnImportForgetsTheGoneFilesItLeavesNothingToWinBack(t *testing.T) {
	l := open(t)
	best := l.target(t, "Best", []release.Quality{release.Bluray1080p}, []release.Quality{release.WEBDL1080p})
	p := l.profile(t, "Best", best)
	outranked, outranking := l.work(t, "w1", true, &p), l.work(t, "w2", true, &p)
	svc := l.files(files.NoGrabs{})
	l.place(t, outranked.Items[0].ID, outranked.ID, "Shelf w1/Old.WEB-DL.mkv", release.WEBDL1080p, best.ID)
	l.place(t, outranking.Items[0].ID, outranking.ID, "Shelf w2/Old.BluRay.mkv", release.Bluray1080p, best.ID)
	l.markGone(t, svc, outranked.ID, "Shelf w1/Old.WEB-DL.mkv")
	l.markGone(t, svc, outranking.ID, "Shelf w2/Old.BluRay.mkv")

	l.download(t, svc, outranked, 0, release.Bluray1080p, "bluray")
	l.download(t, svc, outranking, 0, release.WEBDL1080p, "webdl")

	if got, want := l.listed(t, svc, outranked.ID), "Shelf w1/Shelf w1 - Scarf - Best.mkv: label=Best extra=0 fills=1"; got != want {
		t.Fatalf("after a Bluray import Shelf w1 lists\n%s\nwant the gone WEB-DL forgotten\n%s", got, want)
	}
	want := "Shelf w2/Old.BluRay.mkv: label=Best extra=0 fills=0 gone\n" +
		"Shelf w2/Shelf w2 - Scarf - Best.mkv: label=Best extra=0 fills=1"
	if got := l.listed(t, svc, outranking.ID); got != want {
		t.Fatalf("after a WEB-DL import Shelf w2 lists\n%s\nwant the gone Bluray kept, as it would win its cell back\n%s", got, want)
	}
}

func TestAnImportNamedOntoAGoneFilesPathForgetsIt(t *testing.T) {
	l := open(t)
	best := l.target(t, "Best", []release.Quality{release.Bluray1080p}, []release.Quality{release.WEBDL1080p})
	p := l.profile(t, "Best", best)
	tree := l.work(t, "w1", true, &p)
	svc := l.files(files.NoGrabs{})
	at := "Shelf w1/Shelf w1 - Scarf - Best.mkv"
	l.place(t, tree.Items[0].ID, tree.ID, at, release.Bluray1080p, best.ID)
	l.markGone(t, svc, tree.ID, at)

	l.download(t, svc, tree, 0, release.WEBDL1080p, "webdl")
	l.settle(t, svc)

	if got, want := l.listed(t, svc, tree.ID), at+": label=Best extra=0 fills=1"; got != want || !maps.Equal(l.onDisk(t), map[string]string{at: "webdl"}) {
		t.Fatalf("the Work lists\n%s\nand the root holds %v; want the WEB-DL at the gone Bluray's path and the Bluray forgotten:\n%s", got, l.onDisk(t), want)
	}
}

func TestAnImportLandsBesideTheFilesAScanFound(t *testing.T) {
	l := open(t)
	web := l.target(t, "Web", []release.Quality{release.WEBDL1080p})
	best := l.target(t, "Best", []release.Quality{release.Bluray1080p}, []release.Quality{release.WEBDL1080p})
	p, upgrading := l.profile(t, "Web", web), l.profile(t, "Best", best)
	flat, foldered, upgraded := l.work(t, "w1", true, &p), l.boxed(t, "w2", true, p), l.work(t, "w3", true, &upgrading)
	svc := l.files(files.NoGrabs{})
	l.scan(t, svc, files.Scan{Work: flat.ID, Root: l.root.ID, Found: []files.Found{
		found(flat.Items[0].ID, release.WEBDL1080p, walked(t, "Found Shelf/Loose/Scarf.mkv", 10, 1)),
	}})
	l.scan(t, svc, files.Scan{Work: foldered.ID, Root: l.root.ID, Found: []files.Found{
		found(itemOf(t, foldered, "Scarf"), release.WEBDL1080p, walked(t, "Boxed Shelf/Winter/Scarf.mkv", 10, 2)),
	}})

	l.scan(t, svc, files.Scan{Work: upgraded.ID, Root: l.root.ID, Found: []files.Found{
		found(upgraded.Items[0].ID, release.WEBDL1080p, walked(t, "Upgraded/Scarf.mkv", 10, 3)),
	}})

	button := slices.IndexFunc(foldered.Items, func(it works.Item) bool { return it.Label == "Button" })
	mittens := l.download(t, svc, flat, 1, release.WEBDL1080p, "mittens")
	spare := l.download(t, svc, foldered, button, release.WEBDL1080p, "button")
	upgrade := l.download(t, svc, upgraded, 0, release.Bluray1080p, "bluray")

	if mittens.Path != "Found Shelf/Loose/Shelf w1 - Mittens - Web.mkv" || spare.Path != "Boxed Shelf/Spare boxes/Shelf w2 - Button - Web.mkv" ||
		upgrade.Path != "Upgraded/Shelf w3 - Scarf - Best.mkv" {
		t.Fatalf("imports landed at %s, %s and %s; want beside Scarf, its sibling, in the found Work folder's level folder, and beside the file the upgrade replaces",
			mittens.Path, spare.Path, upgrade.Path)
	}
}
