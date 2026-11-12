package imports

import (
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/works"
)

func TestANotAnUpgradeFileOffersForceNamingTheFileItDeletesAndSaysEachTargetsVerdict(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	ctx := t.Context()
	target := w.add(t, "w1", w.root.ID)
	w.pend(t, pend{Folder: "A", Files: []File{found("A/a.mkv")}})
	w.pend(t, pend{Folder: "B", Files: []File{found("B/a.mkv")}})
	a, b := w.at(t, "A"), w.at(t, "B")
	item := []works.ItemID{target.Items[0].ID}
	w.reconcile(t)
	if _, err := w.im.Into(ctx, identity.Admin{}, a.ID, target.ID, []Chosen{{Position: position(a, "A/a.mkv"), Items: item}}); err != nil {
		t.Fatal(err)
	}
	w.until(t, "A lands", a.ID, func(imp *Import) bool { return fileAt(imp, position(a, "A/a.mkv")).Outcome == Placed })
	kept := w.filesOf(t, target.ID)
	if _, err := w.im.Into(ctx, identity.Admin{}, b.ID, target.ID, []Chosen{{Position: position(b, "B/a.mkv"), Items: item}}); err != nil {
		t.Fatal(err)
	}
	at := position(b, "B/a.mkv")
	got := w.until(t, "B waits", b.ID, func(imp *Import) bool { return fileAt(imp, at).Outcome == Waiting })
	if f := fileAt(got, at); f.Reason.Code != NotAnUpgrade {
		t.Fatalf("B's file waits %+v, want not_an_upgrade", f.Reason)
	}

	words, err := w.im.Words(ctx, identity.Admin{}, *got)
	if err != nil {
		t.Fatal(err)
	}
	said := words[at]
	notBetter := "This isn't better than the file you have for HD."
	if said.Sentence != notBetter {
		t.Errorf("the row says %q, want %q", said.Sentence, notBetter)
	}
	if want := []TargetVerdict{{Target: w.hd, Sentence: notBetter}}; !slices.Equal(said.Verdicts, want) {
		t.Errorf("Details lists the verdicts %+v, want %+v", said.Verdicts, want)
	}
	if len(said.Forces) != 1 || said.Forces[0].Cell.Item != item[0] || said.Forces[0].Cell.Target != w.hd || !slices.Equal(said.Forces[0].Deletes, kept) {
		t.Fatalf("Force offers %+v, want HD for the Item deleting %v", said.Forces, kept)
	}

	if _, err := w.im.Act(ctx, identity.Admin{}, FileRef{Import: b.ID, Position: at}, ForceFile{Cell: said.Forces[0].Cell}); err != nil {
		t.Fatal(err)
	}
	forced := fileAt(w.until(t, "B lands by Force", b.ID, func(imp *Import) bool { return fileAt(imp, at).Outcome == Placed }), at).FileID
	l, err := w.files.WorkFiles(ctx, identity.Admin{}, target.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(l.Files) != 1 || l.Files[0].ID != forced {
		t.Errorf("after the Force the Work holds %+v, want B's file %s alone, %v deleted as the option said", l.Files, forced, kept)
	}
}
