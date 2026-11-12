package imports

import (
	"context"
	"math/rand/v2"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/library"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// folder records a folder Import into work on w's Node of one unprobed
// pending file per Item, named for it.
func (w *world) folder(t *testing.T, work works.Tree) Import {
	t.Helper()
	imp := Import{
		ID: ImportID{UUID: storage.NewUUID()}, Source: FromFolder, Node: w.node, Folder: "/downloads/Shelf",
		ContentType: contenttypetest.ID, Work: work.ID, Created: storage.Now(),
	}
	for i, it := range work.Items {
		imp.Files = append(imp.Files, File{
			Position: i, Path: "/downloads/Shelf/" + it.Label + " " + strconv.Itoa(i) + ".mkv", Size: 1 << 30,
			Items: []works.ItemID{it.ID}, Outcome: Pending,
		})
	}
	err := w.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return insertImport(ctx, tx, imp)
	})
	if err != nil {
		t.Fatal(err)
	}
	return imp
}

// tenItems stocks Work key with one box of ten clips and adds it.
func (w *world) tenItems(t *testing.T, key works.Key) works.Tree {
	t.Helper()
	w.stock(key)
	tree := w.add(t, key, w.root.ID)
	for len(tree.Items) < 10 {
		tree.Items = append(tree.Items, tree.Items...)
	}
	tree.Items = tree.Items[:10]
	return tree
}

func TestTenProbesInAnyOrderDecideOnceTheLastArrives(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	imp := w.folder(t, w.tenItems(t, "w1"))
	order := rand.New(rand.NewPCG(7, 7)).Perm(len(imp.Files)) //nolint:gosec // a fixed order the test replays
	for k, position := range order {
		if err := w.im.probeFile(t.Context(), imp, imp.Files[position]); err != nil {
			t.Fatal(err)
		}
		got := w.stored(t, imp.ID)
		pending := slices.IndexFunc(got.Files, func(f File) bool { return f.Outcome != Pending })
		if k < len(order)-1 && pending >= 0 {
			t.Fatalf("after %d of 10 probes file %d is %s, want every file pending until the last", k+1, pending, got.Files[pending].Outcome)
		}
		if k == len(order)-1 && slices.ContainsFunc(got.Files, func(f File) bool { return f.Outcome == Pending || f.Probe == nil }) {
			t.Fatalf("after the tenth probe the files are %v, want each probed and decided", outcomesOf(got))
		}
	}
}

func outcomesOf(imp *Import) []Outcome {
	out := make([]Outcome, len(imp.Files))
	for i, f := range imp.Files {
		out[i] = f.Outcome
	}
	return out
}

// stopAt holds the first IO of import at at until ctx ends, and answers
// once it is held.
func (w *world) stopAt(id ImportID, at StepAt) <-chan struct{} {
	held := make(chan struct{})
	stop := func(ctx context.Context, s Step) {
		if s.Import == id && s.At == at {
			w.stop.Store(nil)
			close(held)
			<-ctx.Done()
		}
	}
	w.stop.Store(&stop)
	return held
}

// reconcile runs a reconciler over w until the test ends, as a restarted
// Controller would.
func (w *world) reconcile(t *testing.T) {
	cfg := w.cfg
	cfg.BeforeImport = nil
	im := w.newImports(cfg)
	ctx, cancel := context.WithCancel(context.WithoutCancel(t.Context()))
	done := make(chan struct{})
	go func() {
		im.reconcile(ctx)
		close(done)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})
}

func (w *world) until(t *testing.T, what string, id ImportID, ok func(*Import) bool) *Import {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		got := w.stored(t, id)
		if ok(got) {
			return got
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s: not within 10s; the files are %v", what, outcomesOf(got))
		}
		time.Sleep(time.Millisecond)
	}
}

func TestARestartAtThePlaceOrTheLandingEndsTheSame(t *testing.T) {
	t.Parallel()
	into := func(t *testing.T, w *world) (ImportID, works.Tree, []Chosen) {
		target := w.add(t, "w1", w.root.ID)
		w.pend(t, pend{Folder: "Odd", Files: []File{found("Odd/a.mkv")}})
		odd := w.at(t, "Odd")
		return odd.ID, target, []Chosen{{Position: position(odd, "Odd/a.mkv"), Items: []works.ItemID{target.Items[1].ID}}}
	}
	base := newWorld(t)
	id, target, chosen := into(t, base)
	if _, err := base.im.Into(t.Context(), identity.Admin{}, id, target.ID, chosen); err != nil {
		t.Fatal(err)
	}
	want := base.filesOf(t, target.ID)
	for _, at := range []StepAt{AtPlace, AtLand} {
		t.Run(string(at), func(t *testing.T) {
			t.Parallel()
			w := newWorld(t)
			id, target, chosen := into(t, w)
			ctx, crash := context.WithCancel(t.Context())
			held := w.stopAt(id, at)
			go func() {
				<-held
				crash()
			}()
			if _, err := w.im.Into(ctx, identity.Admin{}, id, target.ID, chosen); err == nil {
				t.Fatalf("Into ran on past the crash at %s", at)
			}
			if f := fileAt(w.stored(t, id), chosen[0].Position); f.Outcome != Placing {
				t.Fatalf("the crash at %s left the file %s, want placing", at, f.Outcome)
			}
			w.reconcile(t)
			got := w.until(t, "the restarted reconciler lands it", id, func(imp *Import) bool { return !imp.Finished.IsZero() })
			if f := fileAt(got, chosen[0].Position); f.Outcome != Placed || f.PlacedBy != InPlace || f.FileID.IsZero() {
				t.Errorf("after the restart the file is %s by %q as %s, want placed in_place", f.Outcome, f.PlacedBy, f.FileID)
			}
			if got := w.filesOf(t, target.ID); !slices.Equal(got, want) {
				t.Errorf("after the restart the Work holds %v, want %v as without one", got, want)
			}
		})
	}
}

func TestAFileWaitsBusyWhileAnotherImportPlacesItsItemThenDecides(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	ctx := t.Context()
	target := w.add(t, "w1", w.root.ID)
	w.pend(t, pend{Folder: "A", Files: []File{found("A/a.mkv")}})
	w.pend(t, pend{Folder: "B", Files: []File{found("B/a.mkv")}})
	a, b := w.at(t, "A"), w.at(t, "B")
	item := []works.ItemID{target.Items[0].ID}
	landing, release := context.WithCancel(ctx)
	held := w.stopAt(a.ID, AtLand)
	placed := make(chan error, 1)
	go func() {
		<-held
		_, err := w.im.Into(ctx, identity.Admin{}, b.ID, target.ID, []Chosen{{Position: position(b, "B/a.mkv"), Items: item}})
		release()
		placed <- err
	}()
	if _, err := w.im.Into(landing, identity.Admin{}, a.ID, target.ID, []Chosen{{Position: position(a, "A/a.mkv"), Items: item}}); err == nil {
		t.Fatal("A's Into ran on past its held landing")
	}
	if err := <-placed; err != nil {
		t.Fatal(err)
	}
	f := fileAt(w.stored(t, b.ID), position(b, "B/a.mkv"))
	if f.Outcome != Waiting || f.Reason.Code != Busy || !slices.Equal(f.Reason.Reading, item) {
		t.Fatalf("B's file is %s %+v while A places its Item, want waiting busy for it", f.Outcome, f.Reason)
	}
	w.reconcile(t)
	w.until(t, "A lands", a.ID, func(imp *Import) bool { return fileAt(imp, position(a, "A/a.mkv")).Outcome == Placed })
	got := w.until(t, "B decides once A's Item is free", b.ID, func(imp *Import) bool { return !fileAt(imp, position(b, "B/a.mkv")).busy() })
	if f := fileAt(got, position(b, "B/a.mkv")); f.Outcome != Waiting || f.Reason.Code != NotAnUpgrade {
		t.Errorf("B's file is %s %+v after A placed the same quality, want waiting not_an_upgrade", f.Outcome, f.Reason)
	}
}

func TestARestartAtTheProbeEndsTheSame(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	imp := w.folder(t, w.add(t, "w1", w.root.ID))
	uncut := newWorld(t)
	plain := uncut.folder(t, uncut.add(t, "w1", uncut.root.ID))
	for _, f := range plain.Files {
		if err := uncut.im.probeFile(t.Context(), plain, f); err != nil {
			t.Fatal(err)
		}
	}
	ctx, crash := context.WithCancel(t.Context())
	held := w.stopAt(imp.ID, AtProbe)
	go func() {
		<-held
		crash()
	}()
	if err := w.im.probeFile(ctx, imp, imp.Files[0]); err == nil {
		t.Fatal("the probe ran on past the crash")
	}
	w.reconcile(t)
	got := w.until(t, "the restarted reconciler probes and decides", imp.ID, func(imp *Import) bool {
		return !slices.ContainsFunc(imp.Files, func(f File) bool { return f.Outcome == Pending })
	})
	if a, b := outcomesOf(got), outcomesOf(uncut.stored(t, plain.ID)); !slices.Equal(a, b) {
		t.Errorf("after the restart the files are %v, want %v as without one", a, b)
	}
}

func TestAWorkDeleteDeclinesItsImportsFilesWorkDeleted(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	work := w.add(t, "w1", w.root.ID)
	imp := w.folder(t, work)
	if _, _, err := w.library.Delete(t.Context(), identity.Admin{}, []works.WorkID{work.ID}, library.Deleting{Files: library.DeleteFiles}); err != nil {
		t.Fatal(err)
	}
	got := w.stored(t, imp.ID)
	if got.Finished.IsZero() || slices.ContainsFunc(got.Files, func(f File) bool { return f.Outcome != Declined || f.Reason.Code != WorkDeleted }) {
		t.Errorf("after its Work's delete the Import is finished at %v with %v, want finished with every file declined work_deleted", got.Finished, outcomesOf(got))
	}
}
