package storing

import (
	"testing"

	storagev1 "github.com/jatassi/slipstream/internal/pb/slipstream/storage/v1"
	"github.com/jatassi/slipstream/internal/storage"
)

func TestNotable(t *testing.T) {
	a, b := storage.NewUUID().Bytes(), storage.NewUUID().Bytes()
	facts := func(stateA storagev1.RootState, volA string, free int64, vols ...string) *storagev1.Facts {
		f := &storagev1.Facts{Roots: []*storagev1.RootFacts{
			{Id: a, Place: 1, State: stateA, VolumeId: volA},
			{Id: b, Place: 1, State: storagev1.RootState_ROOT_STATE_READY, VolumeId: "fs-1"},
		}}
		for _, v := range vols {
			f.Volumes = append(f.Volumes, &storagev1.Volume{Id: v, Free: free, Total: 100})
		}
		return f
	}
	ready := storagev1.RootState_ROOT_STATE_READY
	last := facts(ready, "fs-1", 50, "fs-1")
	for _, c := range []struct {
		name string
		next *storagev1.Facts
		want bool
	}{
		{"free space alone", facts(ready, "fs-1", 10, "fs-1"), false},
		{"the same", facts(ready, "fs-1", 50, "fs-1"), false},
		{"a root's state", facts(storagev1.RootState_ROOT_STATE_UNMOUNTED, "fs-1", 50, "fs-1"), true},
		{"a root's volume", facts(ready, "fs-2", 50, "fs-1", "fs-2"), true},
		{"a new volume", facts(ready, "fs-1", 50, "fs-1", "fs-3"), true},
		{"a root gone", &storagev1.Facts{Roots: last.GetRoots()[1:], Volumes: last.GetVolumes()}, true},
	} {
		if got := statesOrVolumesDiffer(last, c.next); got != c.want {
			t.Errorf("%s: notable = %v, want %v", c.name, got, c.want)
		}
	}
	if !statesOrVolumesDiffer(&storagev1.Facts{}, last) {
		t.Error("first facts with roots aren't notable")
	}
}
