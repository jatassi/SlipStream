package roots_test

import (
	"maps"
	"testing"

	storagev1 "github.com/jatassi/slipstream/internal/pb/slipstream/storage/v1"
	"github.com/jatassi/slipstream/internal/roots"
)

func TestWordsNameARootByItsPathOnlyBesideATwin(t *testing.T) {
	r := rig(t, false)
	got, err := r.svc.Words(t.Context())
	if want := map[roots.ID]string{r.root.ID: "Alphas library root on Media server"}; err != nil || !maps.Equal(got, want) {
		t.Fatalf("Words = %v, %v; want %v", got, err, want)
	}
	other := r.second(t)
	got, err = r.svc.Words(t.Context())
	want := map[roots.ID]string{
		r.root.ID: "Alphas library root at " + r.root.Path + " on Media server",
		other.ID:  "Alphas library root at " + other.Path + " on Media server",
	}
	if err != nil || !maps.Equal(got, want) {
		t.Fatalf("Words with two Alphas roots on one Node = %v, %v; want each by its path %v", got, err, want)
	}
}

func TestTroubleWordsOnlyAStateThatIsWrong(t *testing.T) {
	for _, s := range []roots.State{roots.Missing, roots.Unmounted, roots.Foreign, roots.Unwritable, roots.Unresponsive, roots.WrittenOff} {
		if sentence, fix, ok := roots.Trouble(s); !ok || sentence == "" || fix == "" {
			t.Errorf("Trouble(%s) = %q, %q, %v; want a sentence and a fix", s, sentence, fix, ok)
		}
	}
	if sentence, fix, ok := roots.Trouble(roots.Unmounted); sentence != "The folder is there but looks unmounted." || fix != "Plug in or mount the disk." || !ok {
		t.Errorf("Trouble(unmounted) = %q, %q, %v; want the Storage item's sentence and fix", sentence, fix, ok)
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
