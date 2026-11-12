package files_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// nowhere is a Disk that finds nothing at a new path and moves bytes
// nowhere, so a benchmark times the library's side of a rename alone.
type nowhere struct{ files.Disk }

func (nowhere) Move(context.Context, roots.ID, storing.Rel, storing.Rel, bool) (storing.Placed, error) {
	return storing.Placed{}, nil
}

func (nowhere) Survey(_ context.Context, _ roots.ID, planned []storing.Planned, _ bool) ([]storing.Surveyed, error) {
	return make([]storing.Surveyed, len(planned)), nil
}

func (nowhere) Prune(context.Context, roots.ID, storing.Rel, []storing.Rel) ([]storing.Rel, error) {
	return nil, nil
}

const renamedWorks = 20_000

// misplaced is a library of 20,000 Works, each holding one file filling
// its Target in a folder of found files, and the files owner over it.
func misplaced(b *testing.B) (*library, *files.Files) {
	b.Helper()
	l := open(b)
	web := l.target(b, "Web", []release.Quality{release.WEBDL1080p})
	profile := l.profile(b, "Web", web)
	trees := make([]works.Tree, renamedWorks)
	for i := range trees {
		key := works.Key(fmt.Sprintf("w%d", i))
		l.fx.Put(key, contenttypetest.Record{Tree: works.Fetched{Label: fmt.Sprintf("Shelf %d", i), Children: []works.Entry{
			{Key: "b1", Label: "Winter", Children: []works.Entry{{Key: "c1", Label: "Scarf"}}},
		}}})
		var err error
		trees[i], _, err = l.works.Add(b.Context(), admin, contenttypetest.ID, key, works.Adding{
			Monitor: true,
			Also: func(ctx context.Context, tx *storage.Tx[storage.Library], w works.Summary) error {
				if err := targets.Assign(ctx, tx, w, &profile); err != nil {
					return err
				}
				return roots.Assign(ctx, tx, w, nil)
			},
		})
		if err != nil {
			b.Fatal(err)
		}
	}
	err := l.lib.Write(b.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		for i, t := range trees {
			_, err := files.Add(ctx, tx, files.File{
				WorkID: t.ID, Items: []works.ItemID{t.Items[0].ID}, Cut: release.Standard, RootID: l.root.ID, Path: fmt.Sprintf("Found/%d.mkv", i),
				Size: 4 << 30, Media: media.Record{Video: media.Video{Width: 1920, Height: 1080}}, Quality: release.WEBDL1080p,
			}, []targets.TargetID{web.ID})
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		b.Fatal(err)
	}
	svc := l.on(files.NoGrabs{}, nowhere{l.held})
	return l, svc
}

// misplace puts every file back in the found files' folder, with nothing
// queued.
func (l *library) misplace(b *testing.B) {
	b.Helper()
	err := l.lib.Write(b.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		for _, q := range []string{
			`DELETE FROM files_move`, `DELETE FROM files_renamed`, `DELETE FROM files_prune`,
			`UPDATE files_file SET path = 'Found/' || hex(id) || '.mkv'`,
		} {
			if _, err := tx.ExecContext(ctx, q); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		b.Fatal(err)
	}
}

// confirmed is the library plan, every Work of it confirmed as shown.
func confirmed(b *testing.B, svc *files.Files) []files.Confirm {
	b.Helper()
	ps, err := svc.Plan(b.Context(), admin, files.LibraryScope(), nil)
	if err != nil {
		b.Fatal(err)
	}
	if len(ps) != renamedWorks {
		b.Fatalf("the plan lists %d Works, want %d", len(ps), renamedWorks)
	}
	return confirm(ps)
}

// BenchmarkRenamingALibrary measures, at the spec's 20,000 Works each
// holding one file outside its folder, the library plan, the card's count
// of files the templates name otherwise, the confirm of every Work, and
// the disk queue moving every confirmed file and recording each rename,
// over a Disk that finds nothing and moves no bytes. Run it with
// -benchtime=1x: the setup alone adds 20,000 Works.
func BenchmarkRenamingALibrary(b *testing.B) {
	l, svc := misplaced(b)
	l.misplace(b)
	b.Run("Plan", func(b *testing.B) {
		for b.Loop() {
			confirmed(b, svc)
		}
	})
	b.Run("Unmatched", func(b *testing.B) {
		for b.Loop() {
			n, err := svc.Unmatched(b.Context(), admin)
			if err != nil || n[contenttypetest.ID] != renamedWorks {
				b.Fatalf("unmatched = %v, %v; want every file", n, err)
			}
		}
	})
	b.Run("Confirm", func(b *testing.B) {
		for b.Loop() {
			b.StopTimer()
			l.misplace(b)
			cs := confirmed(b, svc)
			b.StartTimer()
			done, err := svc.Rename(b.Context(), admin, cs)
			if err != nil || done.Queued != renamedWorks {
				b.Fatalf("confirm = %d queued, %v; want every file", done.Queued, err)
			}
		}
	})
	b.Run("Drain", func(b *testing.B) {
		for b.Loop() {
			b.StopTimer()
			l.misplace(b)
			if _, err := svc.Rename(b.Context(), admin, confirmed(b, svc)); err != nil {
				b.Fatal(err)
			}
			b.StartTimer()
			l.drained(b, svc)
		}
	})
}

// drained runs svc's disk queue until it holds nothing.
func (l *library) drained(b *testing.B, svc *files.Files) {
	b.Helper()
	ctx, cancel := context.WithCancel(b.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- svc.Run(ctx) }()
	svc.Poke()
	for {
		var n int
		err := l.lib.QueryRowContext(b.Context(), `SELECT count(*) FROM files_move`).Scan(&n)
		switch {
		case err != nil:
			b.Fatal(err)
		case n == 0:
			cancel()
			if err := <-done; err != nil {
				b.Fatal(err)
			}
			return
		}
		select {
		case err := <-done:
			b.Fatalf("the disk queue stopped with %d moves left: %v", n, err)
		case <-time.After(50 * time.Millisecond):
		}
	}
}
