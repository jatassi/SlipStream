package node

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/backups"

	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/housekeeping"
	"github.com/jatassi/slipstream/internal/imports"
	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/node/surface"
	cv1 "github.com/jatassi/slipstream/internal/pb/slipstream/converting/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/transfers"
)

// The registry's sources are built from an empty half: only the
// categories' declarations are read.
func TestTheHealthRegistryMatchesItsIDsEventsAndTasks(t *testing.T) {
	cats := healthCategories(&half{})
	ids := []health.ID{health.StorageID}
	events := storageParts(nil, &half{}).RootEvents
	for _, v := range storageParts(nil, &half{}).Volumes {
		events = append(events, v.Events...)
	}
	for _, c := range cats {
		ids = append(ids, c.ID)
		events = append(events, c.Events...)
	}
	if !slices.Equal(ids, health.IDs()) {
		t.Errorf("the registry binds %q, want health.IDs %q in order", ids, health.IDs())
	}
	for _, e := range events {
		if !slices.Contains(surface.Events, e) {
			t.Errorf("a Health category follows %s, which surface.Events doesn't declare", e.Name())
		}
	}
	for _, task := range catalog(housekeeping.Config{}, metadata.RefreshConfig{}, imports.New(imports.Config{Transfers: transfers.New(transfers.ControllerConfig{})}), &backups.Backups{}) {
		if task.Health != "" && !slices.Contains(health.IDs(), health.ID(task.Health)) {
			t.Errorf("task %s warns in Health category %q, which nobody registers", task.ID, task.Health)
		}
	}
}

type convertingRoster struct {
	serving []role.NodeID
	inStep  map[role.NodeID]bool
}

func (r convertingRoster) Serving(name role.Name) []role.NodeID {
	if name != role.Converting {
		return nil
	}
	return r.serving
}

func (r convertingRoster) InStep(node role.NodeID) bool { return r.inStep[node] }

func TestStorageHealthListsEachScratchVolumeNetOfReservations(t *testing.T) {
	attic, den, cellar := role.NodeID{UUID: storage.NewUUID()}, role.NodeID{UUID: storage.NewUUID()}, role.NodeID{UUID: storage.NewUUID()}
	sampled := time.UnixMilli(1_700_000_000_000)
	scratch := map[role.NodeID]*cv1.Volume{
		attic: {Id: "fs-a", Name: "/srv", Mount: "/srv", Total: 1000, Free: 400, SampledAtUnixMs: sampled.UnixMilli()},
		den:   {Id: "fs-d", Name: `D: (Scratch)`, Mount: `D:\`, Total: 500, Free: 100, SampledAtUnixMs: sampled.UnixMilli()},
	}
	reserved := map[role.NodeID]map[string]int64{attic: {"fs-a": 150}, den: {"fs-d": 300}}
	s := scratchVolumes{
		scratch: func(n role.NodeID) *cv1.Volume { return scratch[n] },
		roster:  convertingRoster{serving: []role.NodeID{attic, den, cellar}, inStep: map[role.NodeID]bool{attic: true}},
		names: func(context.Context) (map[role.NodeID]string, error) {
			return map[role.NodeID]string{attic: "Attic", den: "Den", cellar: "Cellar"}, nil
		},
		reserved: func(_ context.Context, n role.NodeID, volume string) (int64, error) { return reserved[n][volume], nil },
	}.source()
	got, err := s.Volumes(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	want := []health.Volume{
		{Node: attic, NodeName: "Attic", Mount: "/srv", Name: "/srv", Total: 1000, Free: 250, SampledAt: sampled, Live: true},
		{Node: den, NodeName: "Den", Mount: `D:\`, Name: `D: (Scratch)`, Total: 500, Free: 0, SampledAt: sampled},
	}
	if !slices.Equal(got, want) {
		t.Errorf("scratch volumes = %+v, want %+v", got, want)
	}
	if err := s.Sample(t.Context(), attic); err != nil {
		t.Errorf("Sample of a Node in step = %v, want nil", err)
	}
	if err := s.Sample(t.Context(), den); !errors.Is(err, health.ErrUnreachable) {
		t.Errorf("Sample of a Node away = %v, want health.ErrUnreachable", err)
	}
}
