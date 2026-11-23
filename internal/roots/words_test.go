package roots_test

import (
	"strings"
	"testing"

	storagev1 "github.com/jatassi/slipstream/internal/pb/slipstream/storage/v1"
	"github.com/jatassi/slipstream/internal/roots"
)

func TestWordsNameARootByItsPathOnlyBesideATwin(t *testing.T) {
	r := rig(t, false)
	got, err := r.svc.Words(t.Context())
	if w := got[r.root.ID]; err != nil || len(got) != 1 || strings.Contains(w, r.root.Path) || !strings.Contains(w, "Alphas") || !strings.Contains(w, "Media server") {
		t.Fatalf("Words = %v, %v; want the root by its type and Node, without its path", got, err)
	}
	other := r.second(t)
	got, err = r.svc.Words(t.Context())
	if err != nil || len(got) != 2 {
		t.Fatalf("Words with two Alphas roots on one Node = %v, %v", got, err)
	}
	for _, root := range []roots.Root{r.root, other} {
		if w := got[root.ID]; !strings.Contains(w, root.Path) || !strings.Contains(w, "Media server") {
			t.Errorf("Words with two Alphas roots on one Node name %s as %q, want its path %s and its Node", root.ID, w, root.Path)
		}
	}
}

func TestTroubleWordsOnlyAStateThatIsWrong(t *testing.T) {
	for _, s := range []roots.State{roots.Missing, roots.Unmounted, roots.Foreign, roots.Unwritable, roots.Unresponsive, roots.WrittenOff} {
		if sentence, fix, ok := roots.Trouble(s); !ok || sentence == "" || fix == "" {
			t.Errorf("Trouble(%s) = %q, %q, %v; want a sentence and a fix", s, sentence, fix, ok)
		}
	}
	for _, s := range []roots.State{roots.Ready, roots.Checking, roots.Offline} {
		if _, _, ok := roots.Trouble(s); ok {
			t.Errorf("Trouble(%s) is in trouble, want not", s)
		}
	}
}

func TestReadyListsRootsReadyAtTheirCurrentPlace(t *testing.T) {
	r := rig(t, false)
	other := r.second(t)
	stale := r.second(t)
	r.storage.send(r.node, &storagev1.Facts{Roots: []*storagev1.RootFacts{
		ready(r.root),
		{Id: other.ID.Bytes(), Place: other.Place, State: storagev1.RootState_ROOT_STATE_UNMOUNTED},
		{Id: stale.ID.Bytes(), Place: stale.Place - 1, State: storagev1.RootState_ROOT_STATE_READY},
	}})
	got, err := r.svc.Ready(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != r.root.ID {
		t.Fatalf("Ready = %+v, want only the root reported ready at its place, not the unmounted one or one ready at an old place", got)
	}
}
