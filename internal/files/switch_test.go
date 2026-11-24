package files_test

import (
	"context"
	"errors"
	"maps"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/roots/rootstest"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/works"
)

// moving is a library with one Work's file in l.root and a second root to
// switch it to.
func moving(t *testing.T) (*library, *files.Files, works.Tree, files.ID, roots.Root) {
	t.Helper()
	l := open(t)
	f := l.files(nil)
	tree := l.work(t, "w1", true, nil)
	seen := storing.Seen{Modified: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC), ID: storing.FileID{Volume: "v1", File: "7"}}
	var id files.ID
	err := l.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		added, err := files.Add(ctx, tx, files.File{
			WorkID: tree.ID, Items: []works.ItemID{tree.Items[0].ID}, Cut: release.Standard, RootID: l.root.ID, Path: "Shelf/a.mkv", Size: 1024,
			Quality: release.WEBDL1080p, Seen: &seen,
		}, nil)
		id = added.ID
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	to := rootstest.Root(t, l.lib, role.NodeID{UUID: storage.NewUUID()}, contenttypetest.ID, t.TempDir())
	return l, f, tree, id, to
}

func switchIn(tree works.Tree, id files.ID, from, to roots.ID, size int64, seen storing.Seen, sits string, t *testing.T) files.SwitchIn {
	return files.SwitchIn{Work: tree.ID, From: from, To: to, GoneBefore: storage.FromTime(time.Now()), Files: []files.Switched{{
		ID: id, Size: size, Seen: seen, Sits: rel(t, sits), Placed: storing.Seen{Modified: seen.Modified, ID: storing.FileID{Volume: "v2", File: "9"}},
	}}}
}

func TestSwitchRefusesEachStaleCaseHavingWrittenNothing(t *testing.T) {
	seen := storing.Seen{Modified: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC), ID: storing.FileID{Volume: "v1", File: "7"}}
	for _, c := range []struct {
		name string
		in   func(tree works.Tree, id files.ID, from, to roots.ID) files.SwitchIn
	}{
		{"a changed size", func(tree works.Tree, id files.ID, from, to roots.ID) files.SwitchIn {
			return switchIn(tree, id, from, to, 2048, seen, "Shelf/a.mkv", t)
		}},
		{"a changed identity", func(tree works.Tree, id files.ID, from, to roots.ID) files.SwitchIn {
			return switchIn(tree, id, from, to, 1024, storing.Seen{Modified: seen.Modified}, "Shelf/a.mkv", t)
		}},
		{"bytes sitting elsewhere", func(tree works.Tree, id files.ID, from, to roots.ID) files.SwitchIn {
			return switchIn(tree, id, from, to, 1024, seen, "Shelf/b.mkv", t)
		}},
		{"a file the Move never placed", func(tree works.Tree, _ files.ID, from, to roots.ID) files.SwitchIn {
			return files.SwitchIn{Work: tree.ID, From: from, To: to, GoneBefore: storage.FromTime(time.Now())}
		}},
		{"a placed file whose row is gone", func(tree works.Tree, id files.ID, from, to roots.ID) files.SwitchIn {
			in := switchIn(tree, id, from, to, 1024, seen, "Shelf/a.mkv", t)
			in.Files = append(in.Files, files.Switched{ID: files.ID{UUID: storage.NewUUID()}, Sits: rel(t, "Shelf/c.mkv")})
			return in
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			l, f, tree, id, to := moving(t)
			err := l.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
				return f.Switch(ctx, tx, c.in(tree, id, l.root.ID, to.ID))
			})
			if _, stale := errors.AsType[*files.StaleError](err); !stale {
				t.Fatalf("Switch answered %v, want a StaleError", err)
			}
			assertRootOf(t, l, tree.ID, id, l.root.ID)
			if told := l.told(t); len(told) != 0 {
				t.Errorf("a refused Switch told the Media Servers %+v", told)
			}
		})
	}
}

func TestSwitchMovesTheRowsAndTellsOneMovedChange(t *testing.T) {
	l, f, tree, id, to := moving(t)
	seen := storing.Seen{Modified: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC), ID: storing.FileID{Volume: "v1", File: "7"}}
	in := switchIn(tree, id, l.root.ID, to.ID, 1024, seen, "Shelf/a.mkv", t)
	err := l.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return f.Switch(ctx, tx, in)
	})
	if err != nil {
		t.Fatal(err)
	}
	assertRootOf(t, l, tree.ID, id, to.ID)
	told := l.told(t)
	if len(told) != 1 || told[0].Kind != files.PathMoved || told[0].Root != to.ID || told[0].Path.String() != "Shelf/a.mkv" {
		t.Errorf("the Media Servers were told %+v, want one move to the new root at Shelf/a.mkv", told)
	}
	var sits map[files.ID]files.Sitting
	err = l.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		sits, err = files.Sits(ctx, r, []works.WorkID{tree.ID})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := sits[id].Seen; got == nil || !got.Same(in.Files[0].Placed) {
		t.Errorf("the row's Seen is %+v, want the destination's %+v", got, in.Files[0].Placed)
	}
}

func assertRootOf(t *testing.T, l *library, w works.WorkID, id files.ID, want roots.ID) {
	t.Helper()
	err := l.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		home, err := roots.OfWork(ctx, r, w)
		if err != nil {
			return err
		}
		sits, err := files.Sits(ctx, r, []works.WorkID{w})
		if home.ID != want || sits[id].Root != want {
			t.Errorf("the Work is homed in %s and its file in %s, want both in %s", home.ID, sits[id].Root, want)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

// unitHolds is a files.Moves holding the Works in works and reserving
// reserved in each root, recording each Holds it is asked.
type unitHolds struct {
	mu       sync.Mutex
	works    map[works.WorkID]bool
	reserved map[roots.ID][]storing.Rel
	asked    [][]works.WorkID
}

func (h *unitHolds) Reserved(context.Context, storage.Reader[storage.Library], []roots.ID) (map[roots.ID][]storing.Rel, error) {
	return h.reserved, nil
}

func (h *unitHolds) Holds(_ context.Context, _ storage.Reader[storage.Library], ws []works.WorkID) (map[works.WorkID]bool, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.asked = append(h.asked, slices.Clone(ws))
	return maps.Clone(h.works), nil
}

func (h *unitHolds) release() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.works = nil
}

func (h *unitHolds) askedFor() [][]works.WorkID {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.asked)
}

func TestHoldsStopsARenameUntilItsMoveSwitches(t *testing.T) {
	l := open(t)
	tree, idle := l.work(t, "w1", true, nil), l.work(t, "w2", true, nil)
	l.stock(t, tree, "Shelf/a.mkv")
	l.stock(t, idle, "Other/c.mkv")
	l.renames(t, map[string]string{"Shelf/a.mkv": "Shelf/b.mkv"})
	h := &unitHolds{works: map[works.WorkID]bool{tree.ID: true}}
	l.moving = h
	svc := l.files(files.NoGrabs{})
	l.runUntil(t, svc, "the queue asks whether a Move holds the Work", func() bool { return len(h.askedFor()) > 1 })
	for _, ws := range h.askedFor() {
		if !slices.Equal(ws, []works.WorkID{tree.ID}) {
			t.Fatalf("Holds was asked for %v, want only the Work with a queued rename", ws)
		}
	}
	if _, at := l.onDisk(t)["Shelf/a.mkv"]; !at || l.queued(t) != 1 {
		t.Fatalf("a held Work's rename ran: the root holds %v", l.onDisk(t))
	}
	h.release()
	l.runUntil(t, svc, "the rename runs once the Move switched", func() bool { return l.queued(t) == 0 })
	if _, at := l.onDisk(t)["Shelf/b.mkv"]; !at {
		t.Errorf("after the switch the root holds %v, want the file renamed", l.onDisk(t))
	}
}

func TestTheSweepKeepsAMovesHoldingFile(t *testing.T) {
	l := open(t)
	kept := storing.HoldingDir + "/" + storage.NewUUID().String() + ".0.mkv"
	stray := storing.HoldingDir + "/" + storage.NewUUID().String() + ".1.mkv"
	l.put(t, kept, "a Move's placed file")
	l.put(t, stray, "a Move's file no unit names")
	l.moving = &unitHolds{reserved: map[roots.ID][]storing.Rel{l.root.ID: {rel(t, kept)}}}
	l.runUntil(t, l.files(files.NoGrabs{}), "the sweep removes the unreserved file", func() bool { return l.gone(stray) })
	if got := l.onDisk(t); got[kept] != "a Move's placed file" {
		t.Errorf("after the sweep the root holds %v, want the reserved holding file kept", got)
	}
}
