package files_test

import (
	"context"
	"fmt"
	"maps"
	"reflect"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

func TestFilesFromBeforeReleaseNamesReadTheirGrabsTitleOrTheirOwnNameAtStart(t *testing.T) {
	ctx := t.Context()
	open := opener(t)
	before := open(nodes.Schema(), works.Schema(), targets.Schema().Through(1), roots.Schema().Through(1), indexers.Schema().Through(2), grabs.Schema().Through(1), files.Schema().Through(1))
	var at seeds
	var grabbed, named string
	err := before.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		var err error
		if at, err = seedLibrary(ctx, tx); err != nil {
			return err
		}
		if grabbed, err = seedFile(ctx, tx, 1, "The.Quiet.Orchard.2021.1080p.WEB-DL.REPACK-FKW", frames[2], at); err != nil {
			return err
		}
		if named, err = seedNamedFile(ctx, tx, at, "Orchard (2021)/Orchard.2021.2160p.BluRay.x265-GRP.mkv"); err != nil {
			return err
		}
		target := storage.NewUUID()
		for _, stmt := range []string{
			`INSERT INTO targets_target VALUES (?, 'Small', 1)`,
			`INSERT INTO targets_resolution VALUES (?, 1, '1080p')`,
			`INSERT INTO files_fill SELECT id, ?, item_id, 'standard' FROM files_file WHERE grab_id IS NOT NULL`,
		} {
			if _, err := tx.ExecContext(ctx, stmt, target); err != nil {
				return fmt.Errorf("%s: %w", stmt, err)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := before.Close(); err != nil {
		t.Fatal(err)
	}

	lib := open(nodes.Schema(), works.Schema(), targets.Schema(), roots.Schema(), downloadclients.Schema(), grabs.Schema(), indexers.Schema(), files.Schema())
	t.Cleanup(func() { _ = lib.Close() })
	var unread, fills, scored int
	if err := lib.QueryRowContext(ctx, `SELECT count(*) FROM files_file WHERE release_name IS NULL AND title_read IS NULL`).Scan(&unread); err != nil {
		t.Fatal(err)
	}
	if err := lib.QueryRowContext(ctx, `SELECT count(*), count(*) FILTER (WHERE score <> 0) FROM files_fill`).Scan(&fills, &scored); err != nil {
		t.Fatal(err)
	}
	if unread != 2 || fills != 1 || scored != 0 {
		t.Fatalf("after the migration %d files lack a release name and %d of %d fills score other than 0, want 2 and 0 of 1", unread, scored, fills)
	}

	want := map[string]string{
		grabbed: "The.Quiet.Orchard.2021.1080p.WEB-DL.REPACK-FKW by FKW v2",
		named:   "Orchard.2021.2160p.BluRay.x265-GRP.mkv by GRP v1",
	}
	for _, start := range []string{"the first start", "a second start"} {
		if got := readsOf(t, openOver(t, lib), works.WorkID{UUID: at.work}); !maps.Equal(got, want) {
			t.Fatalf("files read after %s = %v\nwant the Grab's title and the file's own name, each read: %v", start, got, want)
		}
	}
	distinct(t, lib, `SELECT coalesce(source_name, 'none') FROM files_file WHERE grab_id IS NOT NULL`, "none")
	distinct(t, lib, `SELECT source_name FROM files_file WHERE grab_id IS NULL`, "Orchard.2021.2160p.BluRay.x265-GRP.mkv")
}

// readsOf is each file of work by path: its release name, and the group
// and revision its Title reads.
func readsOf(t *testing.T, f *files.Files, work works.WorkID) map[string]string {
	t.Helper()
	listed, err := f.WorkFiles(t.Context(), admin, work)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, l := range listed.Files {
		group, _ := l.Title.Facts.Group.Get()
		out[l.Path] = fmt.Sprintf("%s by %s v%d", l.ReleaseName, group, max(l.Title.Facts.Revision.Version, 1))
	}
	return out
}

// seedNamedFile adds an Item of the Work and a file of it at path that no
// Grab brought, and returns the path.
func seedNamedFile(ctx context.Context, tx *storage.Tx[storage.Library], at seeds, path string) (string, error) {
	item := storage.NewUUID()
	if _, err := tx.ExecContext(ctx, `INSERT INTO works_item (id, work_id, key, label, position) VALUES (?, ?, 'named', 'Item', 99)`, item, at.work); err != nil {
		return "", err
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO files_file VALUES (?, ?, ?, 'standard', ?, ?, 1, '{"width":3840,"height":2160}', NULL)`, storage.NewUUID(), at.work, item, at.root, path)
	return path, err
}

// openOver opens the files owner over lib, a library of the fixture type,
// which reads every unread file.
func openOver(t *testing.T, lib *storage.DB[storage.Library]) *files.Files {
	t.Helper()
	fx := contenttypetest.New()
	w, err := works.Open(t.Context(), works.Config{
		Library: lib, Catalogs: map[works.TypeID]works.Catalog{contenttypetest.ID: fx}, Source: fx.Source(), Artwork: fx,
		Changed: discard[works.Changed]{}, TypesChanged: discard[works.ContentTypesChanged]{},
	})
	if err != nil {
		t.Fatal(err)
	}
	goals, err := targets.Open(t.Context(), targets.Config{
		Library: lib, Works: w, Types: map[works.TypeID]targets.Facet{contenttypetest.ID: fx}, Changed: discard[targets.Changed]{},
	})
	if err != nil {
		t.Fatal(err)
	}
	f, err := files.Open(t.Context(), files.Config{
		Library: lib, Targets: goals, Grabs: grabs.GrabCells{InStep: func(role.NodeID) bool { return true }},
		Names: map[works.TypeID]release.Names{contenttypetest.ID: fx}, Today: time.Now,
	})
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestAnImportedFileKeepsItsGrabsTitleAndItsReading(t *testing.T) {
	l := open(t)
	svc := l.files(files.NoGrabs{})
	tree := l.work(t, "w1", true, nil)
	name := "The.Quiet.Orchard.2021.1080p.WEB-DL.DDP5.1.Atmos.HDR.x265.REPACK-FKW"
	l.grabbed(t, svc, tree, 0, release.WEBDL1080p, name)
	listed, err := svc.WorkFiles(t.Context(), admin, tree.ID)
	if err != nil {
		t.Fatal(err)
	}
	if want := release.Title(name, l.fx, nil); len(listed.Files) != 1 || listed.Files[0].ReleaseName != name || !reflect.DeepEqual(listed.Files[0].Title, want) {
		t.Fatalf("the imported file lists as %+v, want release name %q read as %+v", listed.Files, name, want)
	}
}
