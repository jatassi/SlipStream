package storing

import (
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/storage"
)

// rootSample is one root's volume as its last sample read it.
type rootSample struct {
	root   storage.UUID
	volume Volume
}

// volumesOf groups roots' samples into volumes by id, so two roots on one
// filesystem share one volume, the latest sample of it. It answers the
// volumes by id and each sampled root's volume id.
func volumesOf(samples []rootSample) (vols []Volume, of map[storage.UUID]string) {
	byID := map[string]Volume{}
	of = map[storage.UUID]string{}
	for _, s := range samples {
		of[s.root] = s.volume.ID
		if v, ok := byID[s.volume.ID]; !ok || s.volume.SampledAt.After(v.SampledAt) {
			byID[s.volume.ID] = s.volume
		}
	}
	vols = make([]Volume, 0, len(byID))
	for _, v := range byID {
		vols = append(vols, v)
	}
	slices.SortFunc(vols, func(a, b Volume) int { return strings.Compare(a.ID, b.ID) })
	return vols, of
}
