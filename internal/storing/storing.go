// Package storing is the Storage Role on every Node: the roots it holds,
// judged in its own path dialect, and every filesystem act on them. The
// Controller reaches it only through the StorageService and FolderService,
// and names files by root-relative Rel paths.
package storing

import (
	"maps"
	"time"

	storagev1 "github.com/jatassi/slipstream/internal/pb/slipstream/storage/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

var Def = role.Def[*storagev1.Intent, *storagev1.Facts, *storagev1.Result]{Name: role.Storage, Notable: statesOrVolumesDiffer}

type Holder = role.Holder[*storagev1.Intent, *storagev1.Facts, *storagev1.Result]

// statesOrVolumesDiffer is the Storage Role's notability: a change in any
// root's state or volume, or in the set of volumes, is notable; free space
// alone is not.
func statesOrVolumesDiffer(last, next *storagev1.Facts) bool {
	type verdict struct {
		state  storagev1.RootState
		volume string
	}
	roots := func(f *storagev1.Facts) map[storage.UUID]verdict {
		m := map[storage.UUID]verdict{}
		for _, r := range f.GetRoots() {
			id, _ := storage.UUIDFromBytes(r.GetId())
			m[id] = verdict{r.GetState(), r.GetVolumeId()}
		}
		return m
	}
	volumes := func(f *storagev1.Facts) map[string]bool {
		m := map[string]bool{}
		for _, v := range f.GetVolumes() {
			m[v.GetId()] = true
		}
		return m
	}
	return !maps.Equal(roots(last), roots(next)) || !maps.Equal(volumes(last), volumes(next))
}

type Timings struct {
	// Stat bounds one filesystem stat, listing or volume read; a miss makes
	// the roots behind it unresponsive and never blocks the loop.
	Stat time.Duration `yaml:"stat"`
}

func DefaultTimings() Timings { return Timings{Stat: 10 * time.Second} }

type At struct {
	Node role.NodeID
	Root storage.UUID
}

const (
	MarkerName = ".slipstream-root"
	HoldingDir = ".slipstream-incoming"
	StagingDir = ".slipstream-staging"
)
