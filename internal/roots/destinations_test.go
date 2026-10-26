package roots_test

import (
	"context"
	"testing"

	storagev1 "github.com/jatassi/slipstream/internal/pb/slipstream/storage/v1"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
)

func TestADestinationReadsItsRootsVolumeOnceFoundAtItsPlace(t *testing.T) {
	r := rig(t, false)
	r.run(t)
	if d := r.destination(t); d.State != roots.Checking || d.Volume != nil {
		t.Fatalf("an unreported root reads %q on %v, want checking on no volume", d.State, d.Volume)
	}
	r.arrive(t, r.report(storagev1.RootState_ROOT_STATE_READY, r.root.Place))
	if d := r.destination(t); d.State != roots.Ready || d.Volume == nil || d.Volume.Mount != "/media" || d.Volume.Free != 40 {
		t.Fatalf("a ready root reads %q on %+v, want ready on /media with 40 free", d.State, d.Volume)
	}
}

func (r *storageRig) destination(t *testing.T) roots.Destination {
	t.Helper()
	var got map[roots.ID]roots.Destination
	err := r.lib.Read(t.Context(), func(ctx context.Context, rd storage.Reader[storage.Library]) error {
		var err error
		got, err = roots.DestinationsIn(ctx, rd, []roots.ID{r.root.ID}, r.storage, roster{})
		return err
	})
	if err != nil || len(got) != 1 {
		t.Fatalf("the destinations = %v, %v; want the rig's root", got, err)
	}
	return got[r.root.ID]
}
