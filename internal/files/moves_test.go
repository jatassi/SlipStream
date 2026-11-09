package files_test

import (
	"context"
	"errors"
	"maps"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// recording is a Disk that records every Move it passes on, and runs
// before, when set, ahead of each.
type recording struct {
	files.Disk
	mu     sync.Mutex
	moves  []string
	before func(from, to storing.Rel) error
}

func (d *recording) Move(ctx context.Context, root roots.ID, from, to storing.Rel, carry bool) (storing.Placed, error) {
	d.mu.Lock()
	before := d.before
	d.moves = append(d.moves, from.String()+" -> "+to.String())
	d.mu.Unlock()
	if before != nil {
		if err := before(from, to); err != nil {
			return storing.Placed{}, err
		}
	}
	return d.Disk.Move(ctx, root, from, to, carry)
}

func (d *recording) moved() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return slices.Clone(d.moves)
}

// held is every recorded move into or out of the holding folder.
func (d *recording) held() []string {
	return slices.DeleteFunc(d.moved(), func(m string) bool { return !strings.Contains(m, storing.HoldingDir+"/") })
}

// renames queues a move of each file at a key of to to its value, as a
// confirmed rename does: in one Write, each row through its holding path.
func (l *library) renames(t *testing.T, to map[string]string) {
	t.Helper()
	err := l.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		for from := range to {
			_, err := tx.ExecContext(ctx, `INSERT INTO files_move (file_id, sits, origin) SELECT id, path, path FROM files_file WHERE path = ?`, from)
			if err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `UPDATE files_file SET path = '.slipstream-incoming/' || hex(id) WHERE path = ?`, from); err != nil {
				return err
			}
		}
		for from, dst := range to {
			_, err := tx.ExecContext(ctx, `UPDATE files_file SET path = ? WHERE id = (SELECT file_id FROM files_move WHERE sits = ?)`, dst, from)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// stock places a Kept file of tree's first Item at each of paths, holding
// the path itself, and answers the files' ids by path.
func (l *library) stock(t *testing.T, tree works.Tree, paths ...string) map[string]files.ID {
	t.Helper()
	for _, p := range paths {
		l.place(t, tree.Items[0].ID, tree.ID, p, release.WEBDL1080p)
		l.put(t, p, p)
	}
	out := map[string]files.ID{}
	for p, id := range l.rows(t) {
		if slices.Contains(paths, p) {
			out[p] = id
		}
	}
	return out
}

// rows is every file's path in the library and its id.
func (l *library) rows(t *testing.T) map[string]files.ID {
	t.Helper()
	rows, err := l.lib.QueryContext(t.Context(), `SELECT path, id FROM files_file`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	out := map[string]files.ID{}
	for rows.Next() {
		var p string
		var id files.ID
		if err := rows.Scan(&p, &id); err != nil {
			t.Fatal(err)
		}
		out[p] = id
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// renamed is every rename the library recorded, as "from -> to".
func (l *library) renamed(t *testing.T) []string {
	t.Helper()
	rows, err := l.lib.QueryContext(t.Context(), `SELECT from_path || ' -> ' || to_path FROM files_renamed ORDER BY from_path`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var r string
		if err := rows.Scan(&r); err != nil {
			t.Fatal(err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestATargetRenameMovesTheFileAndItsCompanionStraightToTheNewName(t *testing.T) {
	s := shelved(t)
	s.put(t, "Shelf w1/Shelf w1 - Scarf - HD.en.srt", "subtitles")
	disk := &recording{Disk: s.held}
	svc := s.on(files.NoGrabs{}, disk)
	full := targets.UpdateTarget{ID: s.hd.ID, Spec: targets.Spec{
		Scores: targets.Scores{Increment: 1}, Name: "Full HD", Tiers: [][]release.Quality{{release.Bluray1080p}, {release.WEBDL1080p}}, Cutoff: release.Bluray1080p,
	}}
	if got := s.change(t, svc, full, keeping); got != (files.Impact{Works: 1, Renamed: 1}) {
		t.Fatalf("renaming HD = %+v, want one Work and one file renamed", got)
	}
	want := map[string]string{"Shelf w1/Shelf w1 - Scarf - Full HD.mkv": "webdl", "Shelf w1/Shelf w1 - Scarf - Full HD.en.srt": "subtitles"}
	if got := s.onDisk(t); !maps.Equal(got, want) {
		t.Fatalf("the root holds %v, want %v", got, want)
	}
	if moves := disk.moved(); !slices.Equal(moves, []string{"Shelf w1/Shelf w1 - Scarf - HD.mkv -> Shelf w1/Shelf w1 - Scarf - Full HD.mkv"}) {
		t.Fatalf("the queue moved %v, want the one file straight to its new name", moves)
	}
	if got := s.renamed(t); !slices.Equal(got, []string{"Shelf w1/Shelf w1 - Scarf - HD.mkv -> Shelf w1/Shelf w1 - Scarf - Full HD.mkv"}) {
		t.Fatalf("the library recorded renames %v, want the one", got)
	}
	if !s.moved.works()[s.tree.ID.String()] {
		t.Fatal("no files.moved named the renamed file's Work")
	}
}

func TestFilesThatTradePathsPassThroughHoldingWithTheirCompanions(t *testing.T) {
	l := open(t)
	tree := l.work(t, "w1", true, nil)
	ids := l.stock(t, tree, "w1/a.mkv", "w1/b.mkv")
	l.put(t, "w1/a.en.srt", "a's subtitles")
	l.renames(t, map[string]string{"w1/a.mkv": "w1/b.mkv", "w1/b.mkv": "w1/a.mkv"})
	disk := &recording{Disk: l.held}
	l.settle(t, l.on(files.NoGrabs{}, disk))

	want := map[string]string{"w1/b.mkv": "w1/a.mkv", "w1/a.mkv": "w1/b.mkv", "w1/b.en.srt": "a's subtitles"}
	if got := l.onDisk(t); !maps.Equal(got, want) {
		t.Fatalf("the root holds %v, want %v", got, want)
	}
	if rows := l.rows(t); rows["w1/b.mkv"] != ids["w1/a.mkv"] || rows["w1/a.mkv"] != ids["w1/b.mkv"] {
		t.Fatalf("the rows are %v, want the two files traded", rows)
	}
	if held := disk.held(); len(held) != 4 {
		t.Fatalf("the queue moved %v through holding, want each file there and out", held)
	}
	if got := l.renamed(t); !slices.Equal(got, []string{"w1/a.mkv -> w1/b.mkv", "w1/b.mkv -> w1/a.mkv"}) {
		t.Fatalf("the library recorded renames %v, want both", got)
	}
}

func TestATradeQueuedAcrossTwoWritesStillLands(t *testing.T) {
	l := open(t)
	tree := l.work(t, "w1", true, nil)
	l.stock(t, tree, "w1/a.mkv", "w1/b.mkv")
	l.renames(t, map[string]string{"w1/b.mkv": "w1/c.mkv"})
	l.renames(t, map[string]string{"w1/a.mkv": "w1/b.mkv"})
	l.exec(t, `UPDATE files_file SET path = 'w1/a.mkv' WHERE path = 'w1/c.mkv'`)
	l.settle(t, l.files(files.NoGrabs{}))

	if got, want := l.onDisk(t), map[string]string{"w1/b.mkv": "w1/a.mkv", "w1/a.mkv": "w1/b.mkv"}; !maps.Equal(got, want) {
		t.Fatalf("the root holds %v, want the two files traded", got)
	}
}

func TestAChainOfRenamesWaitsAndLandsInOrder(t *testing.T) {
	l := open(t)
	tree := l.work(t, "w1", true, nil)
	l.stock(t, tree, "w1/a.mkv", "w1/b.mkv")
	l.renames(t, map[string]string{"w1/a.mkv": "w1/b.mkv", "w1/b.mkv": "w1/c.mkv"})
	disk := &recording{Disk: l.held}
	l.settle(t, l.on(files.NoGrabs{}, disk))

	if got, want := l.onDisk(t), map[string]string{"w1/b.mkv": "w1/a.mkv", "w1/c.mkv": "w1/b.mkv"}; !maps.Equal(got, want) {
		t.Fatalf("the root holds %v, want %v", got, want)
	}
	if moves := disk.moved(); !slices.Equal(moves, []string{"w1/b.mkv -> w1/c.mkv", "w1/a.mkv -> w1/b.mkv"}) {
		t.Fatalf("the queue moved %v, want b to c, then a to b, both straight", moves)
	}
}

func TestATargetTakenAtDiskTimePutsTheWholeChainBack(t *testing.T) {
	l := open(t)
	tree := l.work(t, "w1", true, nil)
	ids := l.stock(t, tree, "w1/a.mkv", "w1/b.mkv", "w1/c.mkv")
	l.renames(t, map[string]string{"w1/a.mkv": "w1/b.mkv", "w1/b.mkv": "w1/c.mkv", "w1/c.mkv": "w1/d.mkv"})
	l.put(t, "w1/d.mkv", "not SlipStream's")
	l.settle(t, l.files(files.NoGrabs{}))

	want := map[string]string{"w1/a.mkv": "w1/a.mkv", "w1/b.mkv": "w1/b.mkv", "w1/c.mkv": "w1/c.mkv", "w1/d.mkv": "not SlipStream's"}
	if got := l.onDisk(t); !maps.Equal(got, want) {
		t.Fatalf("the root holds %v, want every file where it was", got)
	}
	if rows := l.rows(t); !maps.Equal(rows, ids) {
		t.Fatalf("the rows are %v, want each at its origin %v", rows, ids)
	}
	if got := l.renamed(t); len(got) != 0 {
		t.Fatalf("the library recorded renames %v, want none", got)
	}
	if !l.moved.works()[tree.ID.String()] {
		t.Fatal("no files.moved named the Work whose moves went back")
	}
}

func TestAFreshImportWhoseNameIsTakenLandsAtItsOwnHoldingName(t *testing.T) {
	l := open(t)
	tree := l.work(t, "w1", true, nil)
	svc := l.files(files.NoGrabs{})
	f := l.download(t, svc, tree, 0, release.WEBDL1080p, "download")
	l.put(t, f.Path, "not SlipStream's")
	l.settle(t, svc)

	own := files.HoldingOf(f.ID.UUID, ".mkv")
	want := map[string]string{f.Path: "not SlipStream's", own: "download"}
	if got := l.onDisk(t); !maps.Equal(got, want) {
		t.Fatalf("the root holds %v, want %v", got, want)
	}
	if rows := l.rows(t); rows[own] != f.ID {
		t.Fatalf("the rows are %v, want the import at %s", rows, own)
	}
	l.runUntil(t, l.files(files.NoGrabs{}), "a sweep runs", func() bool { return true })
	if got := l.onDisk(t); !maps.Equal(got, want) {
		t.Fatalf("after a sweep the root holds %v, want %v", got, want)
	}
}

func TestATradeWhoseTargetIsTakenMidwayEndsWithEveryRowAtItsBytes(t *testing.T) {
	l := open(t)
	tree := l.work(t, "w1", true, nil)
	ids := l.stock(t, tree, "w1/a.mkv", "w1/b.mkv")
	l.renames(t, map[string]string{"w1/a.mkv": "w1/b.mkv", "w1/b.mkv": "w1/a.mkv"})
	disk := &recording{Disk: l.held}
	taken := false
	disk.before = func(from, to storing.Rel) error {
		if !taken && strings.HasPrefix(from.String(), storing.HoldingDir) {
			taken = true
			l.put(t, to.String(), "not SlipStream's")
		}
		return nil
	}
	l.settle(t, l.on(files.NoGrabs{}, disk))

	disk2 := l.onDisk(t)
	content := map[files.ID]string{ids["w1/a.mkv"]: "w1/a.mkv", ids["w1/b.mkv"]: "w1/b.mkv"}
	foreign := 0
	for _, c := range disk2 {
		if c == "not SlipStream's" {
			foreign++
		}
	}
	if foreign != 1 {
		t.Fatalf("the root holds %v, want the other file untouched", disk2)
	}
	for p, id := range l.rows(t) {
		if disk2[p] != content[id] {
			t.Fatalf("the row at %s holds %q on disk, want its own bytes; the root holds %v", p, disk2[p], disk2)
		}
	}
}

func TestTheQueueRemovesTheFoldersItsMovesEmptied(t *testing.T) {
	l := open(t)
	tree := l.work(t, "w1", true, nil)
	l.stock(t, tree, "Old/Deep/a.mkv", "Kept/b.mkv", "Away/c.mkv", "Wait/d.mkv")
	l.put(t, "Old/Deep/a.en.srt", "a's subtitles")
	l.put(t, "Kept/notes.txt", "not SlipStream's")
	l.renames(t, map[string]string{"Old/Deep/a.mkv": "New/a.mkv", "Kept/b.mkv": "New/b.mkv", "Away/c.mkv": "New/c.mkv", "Wait/d.mkv": "Away/d.mkv"})
	refused := true
	disk := &recording{Disk: l.held}
	var mu sync.Mutex
	disk.before = func(from, _ storing.Rel) error {
		mu.Lock()
		defer mu.Unlock()
		if from.String() == "Wait/d.mkv" && refused {
			return errors.New("the Node refused the move")
		}
		return nil
	}
	svc := l.on(files.NoGrabs{}, disk)
	l.runUntil(t, svc, "the moves that may go land", func() bool { return l.gone("Old", "Away/c.mkv") })
	if l.gone("Away") || l.gone("Kept/notes.txt") {
		t.Fatalf("the root holds %v, want Away kept for the queued move and Kept for its other file", l.onDisk(t))
	}
	mu.Lock()
	refused = false
	mu.Unlock()
	l.runUntil(t, svc, "every move lands", func() bool { return l.queued(t) == 0 && l.gone("Wait") })
	want := map[string]string{
		"New/a.mkv": "Old/Deep/a.mkv", "New/a.en.srt": "a's subtitles", "New/b.mkv": "Kept/b.mkv", "New/c.mkv": "Away/c.mkv",
		"Away/d.mkv": "Wait/d.mkv", "Kept/notes.txt": "not SlipStream's",
	}
	if got := l.onDisk(t); !maps.Equal(got, want) {
		t.Fatalf("the root holds %v, want %v", got, want)
	}
	var left int
	if err := l.lib.QueryRowContext(t.Context(), `SELECT count(*) FROM files_prune`).Scan(&left); err != nil || left != 0 {
		t.Fatalf("%d folders wait to be removed (%v), want none", left, err)
	}
}

func TestAMoveQueuedBeforeRenamesWentStraightStillPassesThroughHolding(t *testing.T) {
	l := open(t)
	tree := l.work(t, "w1", true, nil)
	ids := l.stock(t, tree, "w1/a.mkv")
	l.exec(t, `INSERT INTO files_move (file_id, sits, origin, ring) VALUES (?, 'w1/a.mkv', 'w1/a.mkv', ?)`, ids["w1/a.mkv"], ids["w1/a.mkv"])
	l.exec(t, `UPDATE files_file SET path = 'w1/b.mkv'`)
	on := l.onRoot(t)
	if len(on.Files) != 1 || on.Files[0].Next.String() != files.HoldingOf(ids["w1/a.mkv"].UUID, ".mkv") {
		t.Fatalf("a scan reads %+v, want its next hop the holding path", on.Files)
	}
	disk := &recording{Disk: l.held}
	l.settle(t, l.on(files.NoGrabs{}, disk))
	if got := l.onDisk(t); !maps.Equal(got, map[string]string{"w1/b.mkv": "w1/a.mkv"}) {
		t.Fatalf("the root holds %v, want the file at its new path", got)
	}
	if held := disk.held(); len(held) != 2 {
		t.Fatalf("the queue moved %v through holding, want into it and out", held)
	}
}

// foundWhereRecorded checks that a scan reading the library now finds each
// file's bytes where its record says they sit, or where the disk queue
// takes them next, so it calls no file gone or found.
func (l *library) foundWhereRecorded(t *testing.T) {
	t.Helper()
	disk := l.onDisk(t)
	for _, r := range l.onRoot(t).Files {
		if _, ok := disk[r.Sits.String()]; ok {
			continue
		}
		if _, ok := disk[r.Next.String()]; ok && r.Next != (storing.Rel{}) {
			continue
		}
		t.Fatalf("a scan reads %+v, whose bytes lie at neither; the root holds %v", r, disk)
	}
}

func TestMovingCountsTheFilesTheQueueStillMovesByWorkAndByType(t *testing.T) {
	l := open(t)
	w1, w2 := l.work(t, "w1", true, nil), l.work(t, "w2", true, nil)
	l.stock(t, w1, "w1/a.mkv", "w1/b.mkv")
	l.stock(t, w2, "w2/c.mkv")
	l.renames(t, map[string]string{"w1/a.mkv": "w1/x.mkv", "w1/b.mkv": "w1/y.mkv", "w2/c.mkv": "w2/z.mkv"})
	svc := l.files(files.NoGrabs{})
	count := func(s files.Scope) int {
		t.Helper()
		n, err := svc.Moving(t.Context(), s)
		if err != nil {
			t.Fatal(err)
		}
		return n
	}
	if w, typ, other := count(files.WorkScope(w1.ID)), count(files.TypeScope(contenttypetest.ID)), count(files.TypeScope("other")); w != 2 || typ != 3 || other != 0 {
		t.Fatalf("Moving = %d of w1, %d of the type and %d of another, want 2, 3 and 0", w, typ, other)
	}
	l.settle(t, svc)
	if n := count(files.TypeScope(contenttypetest.ID)); n != 0 {
		t.Fatalf("Moving once the queue drained = %d, want 0", n)
	}
}
