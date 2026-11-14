package queue_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers/queue"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
)

var at = time.Date(2026, 11, 14, 14, 2, 3, 0, time.UTC)

// table is one server's mediaservers_change rows. It reads the open rows a
// change meets as the service does, and applies the ops under the partial
// unique indexes.
type table struct {
	t      *testing.T
	server link.ServerID
	root   roots.ID
	files  map[string]files.ID
	rows   []queue.Change
}

func newTable(t *testing.T) *table {
	return &table{t: t, server: link.ServerID{UUID: storage.NewUUID()}, root: roots.ID{UUID: storage.NewUUID()}, files: map[string]files.ID{}}
}

func (tb *table) file(name string) files.ID {
	if name == "" {
		return files.ID{}
	}
	if _, ok := tb.files[name]; !ok {
		tb.files[name] = files.ID{UUID: storage.NewUUID()}
	}
	return tb.files[name]
}

func rel(t *testing.T, s string) storing.Rel {
	t.Helper()
	if s == "" {
		return storing.Rel{}
	}
	r, err := storing.ParseRel(s)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// change is a new due change. path is "P" or "X>P" for a carry from X.
func (tb *table) change(kind link.ChangeKind, path, file string) queue.Change {
	from, to, carry := strings.Cut(path, ">")
	if !carry {
		from, to = "", path
	}
	return queue.Change{
		ID: link.ChangeID{UUID: storage.NewUUID()}, Server: tb.server, Kind: kind, Root: tb.root,
		Path: rel(tb.t, to), From: rel(tb.t, from), File: tb.file(file), State: queue.Due, Step: link.StepNone,
		NextAt: at, Created: at.Add(time.Duration(len(tb.rows)) * time.Second),
	}
}

func (tb *table) upgrade(c queue.Change) queue.Change {
	c.Upgrade = true
	return c
}

// doomed is c, the removal of an upgrade's old file, replaced by "new".
func (tb *table) doomed(c queue.Change) queue.Change {
	c.ReplacedBy = tb.file("new")
	return c
}

func (tb *table) open(in queue.Change) queue.Open {
	o := queue.Open{AtPath: map[storing.Rel]queue.Change{}}
	for _, r := range tb.rows {
		if r.Foldable() && r.Carry() && r.File == in.File {
			o.Carry = &r
		}
	}
	paths := in.Paths()
	if o.Carry != nil {
		paths = append(paths, o.Carry.From)
	}
	for _, r := range tb.rows {
		if !r.Foldable() {
			continue
		}
		if !r.Carry() && slices.Contains(paths, r.Path) {
			o.AtPath[r.Path] = r
		}
		if in.Kind == link.ChangeRemoved && !in.ReplacedBy.IsZero() && r.Kind == link.ChangeAdded && r.Upgrade && r.File == in.ReplacedBy ||
			in.Kind == link.ChangeAdded && r.Kind == link.ChangeRemoved && r.ReplacedBy == in.File {
			o.Partner = &r
		}
	}
	return o
}

func (tb *table) fold(in ...queue.Change) {
	tb.t.Helper()
	for _, c := range in {
		tb.apply(queue.Fold(tb.open(c), c))
	}
}

func (tb *table) apply(ops []queue.Op) {
	tb.t.Helper()
	for _, op := range ops {
		i := slices.IndexFunc(tb.rows, func(r queue.Change) bool { return r.ID == op.Change.ID })
		switch {
		case op.Do == queue.Insert && i < 0:
			tb.rows = append(tb.rows, op.Change)
		case op.Do == queue.Update && i >= 0:
			tb.rows[i] = op.Change
		case op.Do == queue.Delete && i >= 0:
			tb.rows = slices.Delete(tb.rows, i, i+1)
		default:
			tb.t.Fatalf("op %v on %s, which the table %s", op.Do, show(tb, op.Change), map[bool]string{true: "lacks", false: "holds"}[i < 0])
		}
		tb.indexes()
	}
}

// indexes fails the test when the rows break mediaservers_change_open_path
// or mediaservers_change_open_carry.
func (tb *table) indexes() {
	tb.t.Helper()
	paths, carries := map[storing.Rel]int{}, map[files.ID]int{}
	for _, r := range tb.rows {
		switch {
		case !r.Foldable():
		case r.Carry():
			carries[r.File]++
		default:
			paths[r.Path]++
		}
	}
	for p, n := range paths {
		if n > 1 {
			tb.t.Fatalf("%d open rows at %q", n, p)
		}
	}
	for _, n := range carries {
		if n > 1 {
			tb.t.Fatalf("%d open carries of one file", n)
		}
	}
}

func show(tb *table, c queue.Change) string {
	name := func(id files.ID) string {
		for n, f := range tb.files {
			if f == id {
				return n
			}
		}
		return "?"
	}
	s := string(c.Kind) + " " + c.Path.String()
	if c.Carry() {
		s = string(c.Kind) + " " + c.From.String() + ">" + c.Path.String()
	}
	if !c.File.IsZero() {
		s += " " + name(c.File)
	}
	if c.Upgrade {
		s += " upgrade"
	}
	if !c.ReplacedBy.IsZero() {
		s += " replaced by " + name(c.ReplacedBy)
	}
	if c.State != queue.Due {
		s += " " + string(c.State)
	}
	if c.Snapshot != nil {
		s += " with snapshot"
	}
	return s
}

func (tb *table) holds(want ...string) {
	tb.t.Helper()
	got := make([]string, len(tb.rows))
	for i, r := range tb.rows {
		got[i] = show(tb, r)
	}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		tb.t.Errorf("the queue holds %q, want %q", got, want)
	}
}

func TestRefreshesOnOnePathFoldByTheMergeTable(t *testing.T) {
	for _, prefix := range []string{"", "sidecar_"} {
		for _, row := range []struct{ earlier, later, want string }{
			{"added", "added", "added"},
			{"added", "replaced", "added"},
			{"added", "removed", ""},
			{"replaced", "added", "replaced"},
			{"replaced", "replaced", "replaced"},
			{"replaced", "removed", "removed"},
			{"removed", "added", "replaced"},
			{"removed", "replaced", "replaced"},
			{"removed", "removed", "removed"},
		} {
			name := fmt.Sprintf("%s%s then %s%s", prefix, row.earlier, prefix, row.later)
			t.Run(name, func(t *testing.T) {
				tb := newTable(t)
				tb.fold(tb.change(link.ChangeKind(prefix+row.earlier), "Heat/Heat.mkv", "f1"),
					tb.change(link.ChangeKind(prefix+row.later), "Heat/Heat.mkv", "f1"))
				if row.want == "" {
					tb.holds()
					return
				}
				tb.holds(prefix + row.want + " Heat/Heat.mkv f1")
			})
		}
	}
}

func TestTheSpecsMergeRules(t *testing.T) {
	for _, c := range []struct {
		name  string
		steps [][3]string
		want  []string
	}{
		{"removed then added on one path is replaced", [][3]string{{"removed", "A", "old"}, {"added", "A", "new"}}, []string{"replaced A new"}},
		{"added then removed on one path is nothing", [][3]string{{"added", "A", "f"}, {"removed", "A", "f"}}, nil},
		{"moved A to B then B to C is moved A to C", [][3]string{{"moved", "A>B", "f"}, {"moved", "B>C", "f"}}, []string{"moved A>C f"}},
		{"moved A to B then removed B is removed A", [][3]string{{"moved", "A>B", "f"}, {"removed", "B", "f"}}, []string{"removed A f"}},
		{"moved A to B then B to A is nothing", [][3]string{{"moved", "A>B", "f"}, {"moved", "B>A", "f"}}, nil},
		{"added then moved is added at the new path", [][3]string{{"added", "A", "f"}, {"moved", "A>B", "f"}}, []string{"added B f"}},
		{"replaced then moved is a replaced carry", [][3]string{{"replaced", "A", "f"}, {"moved", "A>B", "f"}}, []string{"replaced A>B f"}},
		{"moved then replaced at its path is a replaced carry", [][3]string{{"moved", "A>B", "f"}, {"replaced", "B", "f"}}, []string{"replaced A>B f"}},
		{"moved then added at its path is the moved change", [][3]string{{"moved", "A>B", "f"}, {"added", "B", "f"}}, []string{"moved A>B f"}},
		{"a replaced carry back to its origin is replaced there", [][3]string{{"replaced", "A", "f"}, {"moved", "A>B", "f"}, {"moved", "B>A", "f"}}, []string{"replaced A f"}},
		{"moved then removed keeps an earlier removal at its path", [][3]string{{"removed", "B", "doomed"}, {"moved", "A>B", "f"}, {"removed", "B", "f"}}, []string{"removed B doomed", "removed A f"}},
		{"a removed carry folds before the row at its origin", [][3]string{{"moved", "A>B", "f"}, {"added", "A", "g"}, {"removed", "B", "f"}}, []string{"replaced A g"}},
		{"sidecars on one path fold apart from the video", [][3]string{{"added", "A.mkv", "f"}, {"sidecar_added", "A.en.srt", "s"}, {"sidecar_removed", "A.en.srt", "s"}}, []string{"added A.mkv f"}},
		{"library refreshes of a root fold into one", [][3]string{{"library", "", ""}, {"library", "", ""}}, []string{"library "}},
	} {
		t.Run(c.name, func(t *testing.T) {
			tb := newTable(t)
			for _, s := range c.steps {
				tb.fold(tb.change(link.ChangeKind(s[0]), s[1], s[2]))
			}
			tb.holds(c.want...)
		})
	}
}

func TestAFoldedRowKeepsTheEarlierRowsPlace(t *testing.T) {
	tb := newTable(t)
	first := tb.change(link.ChangeRemoved, "A", "old")
	tb.fold(first)
	tb.fold(tb.change(link.ChangeAdded, "A", "new"))
	if len(tb.rows) != 1 || tb.rows[0].ID != first.ID || !tb.rows[0].Created.Equal(first.Created) {
		t.Errorf("the queue holds %+v, want the first row's id and age", tb.rows)
	}
}

func TestAPreparedRowBecomesTheRename(t *testing.T) {
	tb := newTable(t)
	prepared := tb.change(link.ChangeMoved, "A>B", "f")
	prepared.State, prepared.Step, prepared.StepAt, prepared.Snapshot = queue.Prepared, link.StepSnapshotted, at, &link.Snapshot{}
	tb.rows = append(tb.rows, prepared)
	tb.fold(tb.change(link.ChangeMoved, "A>B", "f"))
	tb.holds("moved A>B f with snapshot")

	tb = newTable(t)
	prepared.File = tb.file("f")
	tb.rows = append(tb.rows, prepared)
	tb.fold(tb.change(link.ChangeRemoved, "A", "f"))
	tb.holds("removed A f")
}

func TestAnUpgradeGoesAsOneReplaced(t *testing.T) {
	for _, c := range []struct {
		name string
		do   func(tb *table)
		want []string
	}{
		{"same name, removed first", func(tb *table) {
			tb.fold(tb.doomed(tb.change(link.ChangeRemoved, "A", "old")), tb.change(link.ChangeAdded, "A", "new"))
		}, []string{"replaced A new"}},
		{"new name, added first", func(tb *table) {
			tb.fold(tb.upgrade(tb.change(link.ChangeAdded, "B", "new")), tb.doomed(tb.change(link.ChangeRemoved, "A", "old")))
		}, []string{"replaced A>B new"}},
		{"new name, removed first", func(tb *table) {
			tb.fold(tb.doomed(tb.change(link.ChangeRemoved, "A", "old")), tb.upgrade(tb.change(link.ChangeAdded, "B", "new")))
		}, []string{"replaced A>B new"}},
		{"of a file the server never saw", func(tb *table) {
			tb.fold(tb.change(link.ChangeAdded, "A", "old"), tb.upgrade(tb.change(link.ChangeAdded, "B", "new")),
				tb.doomed(tb.change(link.ChangeRemoved, "A", "old")))
		}, []string{"added B new"}},
		{"of a renamed file", func(tb *table) {
			tb.fold(tb.change(link.ChangeMoved, "X>A", "old"), tb.upgrade(tb.change(link.ChangeAdded, "B", "new")),
				tb.doomed(tb.change(link.ChangeRemoved, "A", "old")))
		}, []string{"replaced X>B new"}},
		{"under a new name prepared before the old bytes went", func(tb *table) {
			tb.fold(tb.upgrade(tb.change(link.ChangeAdded, "B", "new")))
			p := tb.doomed(tb.change(link.ChangeReplaced, "A>B", "old"))
			p.State, p.Step, p.StepAt, p.Snapshot = queue.Prepared, link.StepSnapshotted, at, &link.Snapshot{}
			tb.rows = append(tb.rows, p)
			tb.fold(tb.doomed(tb.change(link.ChangeRemoved, "A", "old")))
		}, []string{"replaced A>B new with snapshot"}},
		{"whose partner hasn't come waits as a pairing half", func(tb *table) {
			tb.fold(tb.doomed(tb.change(link.ChangeRemoved, "A", "old")))
			if !tb.rows[0].Pairing() {
				tb.t.Error("the removed half doesn't wait for its partner")
			}
		}, []string{"removed A old replaced by new"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			tb := newTable(t)
			c.do(tb)
			tb.holds(c.want...)
		})
	}
}

func TestReopenFoldsTheRowThatTookItsPlace(t *testing.T) {
	t.Run("an added the server may have seen, then removed, is removed", func(t *testing.T) {
		tb := newTable(t)
		back := tb.change(link.ChangeAdded, "A", "f")
		back.State = queue.Sending
		newer := tb.change(link.ChangeRemoved, "A", "f")
		tb.rows = []queue.Change{back, newer}
		tb.apply(queue.Reopen(back, &newer))
		tb.holds("removed A f")
	})
	t.Run("a rename back to its origin after a cut-off send is replaced there", func(t *testing.T) {
		tb := newTable(t)
		back := tb.change(link.ChangeMoved, "A>B", "f")
		back.State, back.Step, back.StepAt, back.Snapshot = queue.Sending, link.StepSnapshotted, at, &link.Snapshot{}
		newer := tb.change(link.ChangeMoved, "B>A", "f")
		tb.rows = []queue.Change{back, newer}
		tb.apply(queue.Reopen(back, &newer))
		tb.holds("replaced A f")
	})
	t.Run("a row at a started step goes back to due beside the newer row", func(t *testing.T) {
		tb := newTable(t)
		back := tb.change(link.ChangeMoved, "A>B", "f")
		back.State, back.Step, back.StepAt = queue.Sending, link.StepOldRemoved, at
		newer := tb.change(link.ChangeMoved, "B>C", "f")
		tb.rows = []queue.Change{back, newer}
		tb.apply(queue.Reopen(back, &newer))
		tb.holds("moved A>B f", "moved B>C f")
	})
	t.Run("a row alone goes back to due", func(t *testing.T) {
		tb := newTable(t)
		back := tb.change(link.ChangeAdded, "A", "f")
		back.State = queue.Sending
		tb.rows = []queue.Change{back}
		tb.apply(queue.Reopen(back, nil))
		tb.holds("added A f")
	})
}
