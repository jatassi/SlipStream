package files_test

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/roots/rootstest"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// put writes content at root-relative path p in l's root.
func (l *library) put(t *testing.T, p, content string) {
	t.Helper()
	at := filepath.Join(l.root.Path, filepath.FromSlash(p))
	if err := os.MkdirAll(filepath.Dir(at), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(at, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

// download imports a downloaded file of item holding content, as a Grab's
// import does: its bytes go to the Grab's holding path first, then the
// file commits.
func (l *library) download(t *testing.T, svc *files.Files, tree works.Tree, item int, q release.Quality, content string) files.File {
	t.Helper()
	return l.downloadProbed(t, svc, tree, item, q, content, files.Probe{Width: 1920, Height: 1080})
}

// downloadProbed is download of a file whose probe read probe.
func (l *library) downloadProbed(t *testing.T, svc *files.Files, tree works.Tree, item int, q release.Quality, content string, probe files.Probe) files.File {
	t.Helper()
	grab := storage.NewUUID()
	holding := files.HoldingOf(grab, ".mkv")
	l.put(t, holding, content)
	var f files.File
	err := l.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		var err error
		f, _, err = svc.Import(ctx, tx, files.Incoming{
			WorkID: tree.ID, ItemID: tree.Items[item].ID, Cut: release.Standard, RootID: l.root.ID, Holding: holding,
			Size: int64(len(content)), Probe: probe, Quality: q,
		})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return f
}

// queued counts the disk queue's rows.
func (l *library) queued(t *testing.T) int {
	t.Helper()
	var n int
	err := l.lib.QueryRowContext(t.Context(), `SELECT (SELECT count(*) FROM files_move) + (SELECT count(*) FROM files_doomed)`).Scan(&n)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// drain runs svc's disk queue until it is empty or Run returns.
func (l *library) drain(ctx context.Context, t *testing.T, svc *files.Files, cancel context.CancelFunc) {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- svc.Run(ctx) }()
	svc.Poke()
	deadline := time.After(10 * time.Second)
	for {
		select {
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
			return
		case <-deadline:
			cancel()
			t.Fatalf("the disk queue still holds %d rows", l.queued(t))
		case <-time.After(time.Millisecond):
			if l.queued(t) == 0 {
				cancel()
				if err := <-done; err != nil {
					t.Fatal(err)
				}
				return
			}
		}
	}
}

func (l *library) settle(t *testing.T, svc *files.Files) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	l.drain(ctx, t, svc, cancel)
}

// onDisk is every file under l's root and what it holds.
func (l *library) onDisk(t *testing.T) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(l.root.Path, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() == storing.MarkerName {
			return err
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err //nolint:wrapcheck // the test reports it as is
		}
		rel, err := filepath.Rel(l.root.Path, p)
		out[filepath.ToSlash(rel)] = string(data)
		return err //nolint:wrapcheck // the test reports it as is
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// paths is each file's path in the library by what it holds on disk.
func (l *library) paths(t *testing.T, ws ...works.WorkID) map[string]string {
	t.Helper()
	out := map[string]string{}
	disk := l.onDisk(t)
	for _, w := range ws {
		_, cells, err := l.files(files.NoGrabs{}).WorkCells(t.Context(), admin, w)
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range cells {
			if c.File != nil {
				out[c.File.Path] = disk[c.File.Path]
			}
		}
	}
	return out
}

func TestOneFileFillsEveryTargetItWinsAndSitsOnDiskOnce(t *testing.T) {
	l := open(t)
	best := l.target(t, "Best", []release.Quality{release.Bluray1080p}, []release.Quality{release.WEBDL1080p})
	web := l.target(t, "Web", []release.Quality{release.WEBDL1080p})
	p := l.profile(t, "Both", best, web)
	tree := l.work(t, "w1", true, &p)
	svc := l.files(files.NoGrabs{})
	f := l.download(t, svc, tree, 0, release.WEBDL1080p, "webdl")
	l.settle(t, svc)

	_, cells, err := svc.WorkCells(t.Context(), admin, tree.ID)
	if err != nil {
		t.Fatal(err)
	}
	want := "Scarf any Best upgradable wanted=true Shelf w1/Shelf w1 - Scarf - Best.mkv webdl-1080p 1920x1080\n" +
		"Scarf any Web met wanted=false Shelf w1/Shelf w1 - Scarf - Best.mkv webdl-1080p 1920x1080\n" +
		"Mittens any Best missing wanted=true\n" +
		"Mittens any Web missing wanted=true\n"
	if got := lines(cells); got != want || cells[0].File.ID != f.ID || cells[1].File.ID != f.ID {
		t.Fatalf("cells =\n%swant\n%s", got, want)
	}
	if got := l.onDisk(t); !maps.Equal(got, map[string]string{"Shelf w1/Shelf w1 - Scarf - Best.mkv": "webdl"}) {
		t.Fatalf("the root holds %v, want the one file at its name", got)
	}
}

func TestAnUpgradeTakesThePathOfTheFileItLeavesFillingNothing(t *testing.T) {
	l := open(t)
	best := l.target(t, "Best", []release.Quality{release.Bluray1080p}, []release.Quality{release.WEBDL1080p})
	p := l.profile(t, "Best", best)
	tree := l.work(t, "w1", true, &p)
	svc := l.files(files.NoGrabs{})
	l.download(t, svc, tree, 0, release.WEBDL1080p, "webdl")
	l.settle(t, svc)

	var placement files.Placement
	err := l.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		incoming := files.Incoming{WorkID: tree.ID, ItemID: tree.Items[0].ID, Cut: release.Standard, RootID: l.root.ID, Holding: ".slipstream-incoming/g.mkv", Quality: release.WEBRip1080p}
		_, err := svc.Place(ctx, r, incoming)
		if _, nothing := errors.AsType[*files.FillsNothingError](err); !nothing {
			return fmt.Errorf("placing a WEBRip beside the tied WEB-DL: err = %w, want it fills nothing", err)
		}
		incoming.Quality = release.Bluray1080p
		placement, err = svc.Place(ctx, r, incoming)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := (files.Placement{Fills: []targets.TargetID{best.ID}, Path: "Shelf w1/Shelf w1 - Scarf - Best.mkv"}); placement.Path != want.Path || len(placement.Fills) != 1 || placement.Fills[0] != best.ID || placement.Kept {
		t.Fatalf("Place of a Bluray = %+v, want %+v", placement, want)
	}

	bluray := l.download(t, svc, tree, 0, release.Bluray1080p, "bluray")
	l.settle(t, svc)
	if got := l.onDisk(t); !maps.Equal(got, map[string]string{"Shelf w1/Shelf w1 - Scarf - Best.mkv": "bluray"}) {
		t.Fatalf("after the upgrade the root holds %v, want the Bluray alone at the name", got)
	}
	_, cells, err := svc.WorkCells(t.Context(), admin, tree.ID)
	if err != nil || cells[0].File == nil || cells[0].File.ID != bluray.ID || cells[0].State != files.Met || fileCount(t, l) != 1 {
		t.Fatalf("after the upgrade cells = %s, %v; want the Bluray meeting Best and one file", lines(cells), err)
	}
}

func TestAFilePlannedIntoARootItsWorkLeftIsRefused(t *testing.T) {
	l := open(t)
	tree := l.work(t, "w1", true, nil)
	other := rootstest.Root(t, l.lib, l.root.Node, contenttypetest.ID, t.TempDir())
	svc := l.files(files.NoGrabs{})
	err := l.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, _, err := svc.Import(ctx, tx, files.Incoming{
			WorkID: tree.ID, ItemID: tree.Items[0].ID, Cut: release.Standard, RootID: other.ID, Holding: files.HoldingOf(storage.NewUUID(), ".mkv"),
			Size: 5, Probe: files.Probe{Width: 1920, Height: 1080}, Quality: release.WEBDL1080p,
		})
		return err
	})
	if !errors.Is(err, files.ErrRootChanged) {
		t.Fatalf("Import into a root the Work isn't in = %v, want ErrRootChanged", err)
	}
	if n := l.queued(t); n != 0 {
		t.Fatalf("the refused import queued %d moves, want none", n)
	}
	l.download(t, svc, tree, 0, release.WEBDL1080p, "webdl")
}

func TestAFileThatFillsNothingLandsKept(t *testing.T) {
	l := open(t)
	web := l.target(t, "Web", []release.Quality{release.WEBDL1080p, release.WEBRip1080p})
	p := l.profile(t, "Web", web)
	tree := l.work(t, "w1", true, &p)
	svc := l.files(files.NoGrabs{})
	l.download(t, svc, tree, 0, release.WEBDL1080p, "webdl")
	kept := l.download(t, svc, tree, 0, release.WEBRip1080p, "webrip")
	l.settle(t, svc)
	if !kept.Kept() || kept.Label != "Web" || kept.Extra != 1 {
		t.Fatalf("a tied second file imported as %+v, want Kept as Web (extra 1)", kept)
	}
	want := map[string]string{"Shelf w1/Shelf w1 - Scarf - Web.mkv": "webdl", "Shelf w1/Shelf w1 - Scarf - Web (extra 1).mkv": "webrip"}
	if got := l.onDisk(t); !maps.Equal(got, want) {
		t.Fatalf("the root holds %v, want %v", got, want)
	}
}

func fileCount(t *testing.T, l *library) int {
	t.Helper()
	var n int
	if err := l.lib.QueryRowContext(t.Context(), `SELECT count(*) FROM files_file`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// crashing is a Disk that crashes the queue at its nth Move or Remove:
// before doing it, or after.
type crashing struct {
	files.Disk
	n     int
	after bool
	seen  int
	crash context.CancelFunc
}

var errCrashed = errors.New("the queue crashed")

func (d *crashing) at(do func() error) error {
	d.seen++
	if d.seen != d.n {
		return do()
	}
	if d.after {
		_ = do()
	}
	d.crash()
	return errCrashed
}

func (d *crashing) Move(ctx context.Context, root roots.ID, from, to storing.Rel) (int64, error) {
	var size int64
	err := d.at(func() error {
		var err error
		size, err = d.Disk.Move(ctx, root, from, to)
		return err
	})
	return size, err
}

func (d *crashing) Remove(ctx context.Context, root roots.ID, rel storing.Rel) error {
	return d.at(func() error { return d.Disk.Remove(ctx, root, rel) })
}

// crashWorld is a library whose disk queue holds a same-path upgrade of
// a placed file, a new file in w2 and one in w1, and two files of w1
// trading names, all committed.
type crashWorld struct {
	l      *library
	w1, w2 works.Tree
}

func newCrashWorld(t *testing.T) crashWorld {
	t.Helper()
	l := open(t)
	best := l.target(t, "Best", []release.Quality{release.Bluray1080p}, []release.Quality{release.WEBDL1080p})
	web := l.target(t, "Web", []release.Quality{release.WEBDL1080p}, []release.Quality{release.Bluray1080p})
	both, one := l.profile(t, "Both", best, web), l.profile(t, "Best", best)
	c := crashWorld{l: l, w1: l.work(t, "w1", true, &both), w2: l.work(t, "w2", true, &one)}
	svc := l.files(files.NoGrabs{})

	l.put(t, "Shelf w1/Shelf w1 - Scarf - Best.mkv", "x")
	l.put(t, "Shelf w1/Shelf w1 - Scarf - Web.mkv", "y")
	err := l.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		seed := func(path string, q release.Quality, fill targets.Target) error {
			_, err := files.Add(ctx, tx, files.File{
				WorkID: c.w1.ID, ItemID: c.w1.Items[0].ID, Cut: release.Standard, RootID: l.root.ID, Path: path, Size: 1, Quality: q,
			}, []targets.TargetID{fill.ID})
			return err
		}
		if err := seed("Shelf w1/Shelf w1 - Scarf - Best.mkv", release.WEBDL1080p, best); err != nil {
			return err
		}
		return seed("Shelf w1/Shelf w1 - Scarf - Web.mkv", release.Bluray1080p, web)
	})
	if err != nil {
		t.Fatal(err)
	}
	l.download(t, svc, c.w2, 0, release.WEBDL1080p, "w")
	l.settle(t, svc)

	l.download(t, svc, c.w2, 0, release.Bluray1080p, "z")
	l.download(t, svc, c.w2, 1, release.WEBDL1080p, "w2")
	l.download(t, svc, c.w1, 1, release.WEBDL1080p, "v")
	return c
}

// later commits, while the queue is down, an upgrade of the file w2 that
// the crashed queue left wherever it stopped.
func (c crashWorld) later(t *testing.T) {
	t.Helper()
	c.l.download(t, c.l.files(files.NoGrabs{}), c.w2, 1, release.Bluray1080p, "z2")
}

var converged = map[string]string{
	"Shelf w1/Shelf w1 - Scarf - Best.mkv":   "y",
	"Shelf w1/Shelf w1 - Scarf - Web.mkv":    "x",
	"Shelf w1/Shelf w1 - Mittens - Best.mkv": "v",
	"Shelf w2/Shelf w2 - Scarf - Best.mkv":   "z",
	"Shelf w2/Shelf w2 - Mittens - Best.mkv": "z2",
}

func (c crashWorld) check(t *testing.T) {
	t.Helper()
	if got := c.l.onDisk(t); !maps.Equal(got, converged) {
		t.Fatalf("the root holds\n%v\nwant\n%v", got, converged)
	}
	if got := c.l.paths(t, c.w1.ID, c.w2.ID); !maps.Equal(got, converged) {
		t.Fatalf("the library's files hold\n%v\nwant\n%v", got, converged)
	}
}

func TestTheDiskQueueConvergesFromACrashAtEveryStep(t *testing.T) {
	steps := 0
	for n := 1; ; n++ {
		reached := false
		for _, after := range []bool{false, true} {
			t.Run(fmt.Sprintf("crash at step %d, done %t", n, after), func(t *testing.T) {
				c := newCrashWorld(t)
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				disk := &crashing{Disk: c.l.held, n: n, after: after, crash: cancel}
				c.l.drain(ctx, t, c.l.on(files.NoGrabs{}, disk), cancel)
				reached = disk.seen >= n
				c.later(t)
				c.l.settle(t, c.l.files(files.NoGrabs{}))
				c.check(t)
				steps = max(steps, disk.seen)
			})
		}
		if !reached {
			break
		}
	}
	if steps < 10 {
		t.Fatalf("the queue took %d steps, want at least the ten moves and the removals it holds", steps)
	}
}

func TestTheQueueSweepsHeldFilesNoFileOrGrabNames(t *testing.T) {
	l := open(t)
	tree := l.work(t, "w1", true, nil)
	svc := l.files(files.NoGrabs{})
	live := l.download(t, svc, tree, 0, release.WEBDL1080p, "live")
	grab := storage.NewUUID()
	staged := storing.StagingDir + "/" + grab.String() + ".part"
	l.put(t, files.HoldingOf(grab, ".mkv"), "a Grab's download")
	l.put(t, staged, "a Grab's Transfer")
	l.put(t, files.HoldingOf(storage.NewUUID(), ".mkv.part"), "a failed Grab's copy")
	l.put(t, storing.StagingDir+"/"+storage.NewUUID().String()+".part", "a failed Grab's Transfer")
	l.put(t, storing.HoldingDir+"/stray.txt", "not SlipStream's")
	l.settle(t, l.on(unfinished{grab}, l.held))
	want := map[string]string{live.Path: "live", files.HoldingOf(grab, ".mkv"): "a Grab's download", staged: "a Grab's Transfer"}
	if got := l.onDisk(t); !maps.Equal(got, want) {
		t.Fatalf("after the sweep the root holds %v, want %v", got, want)
	}
}

type unfinished []storage.UUID

func (unfinished) GrabCells(context.Context, storage.Reader[storage.Library], []works.WorkID) (map[files.CellKey]files.GrabState, error) {
	return nil, nil
}

func (u unfinished) Unfinished(context.Context, storage.Reader[storage.Library]) ([]storage.UUID, error) {
	return u, nil
}

func (unfinished) Settle(context.Context, *storage.Tx[storage.Library], works.WorkID, map[files.CellKey]bool, files.Unwanted) (int, error) {
	return 0, nil
}
