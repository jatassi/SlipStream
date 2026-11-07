package roots_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/nodes"
	storagev1 "github.com/jatassi/slipstream/internal/pb/slipstream/storage/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/roots/rootstest"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

var sampled = time.UnixMilli(1_760_000_000_000).UTC()

func volume(id string, free int64, at time.Time) *storagev1.Volume {
	return &storagev1.Volume{Id: id, Name: "/mnt/" + id, Mount: "/mnt/" + id, Kind: storagev1.VolumeKind_VOLUME_KIND_LOCAL, Total: 100, Free: free, SampledAtUnixMs: at.UnixMilli()}
}

func ready(root roots.Root) *storagev1.RootFacts {
	return &storagev1.RootFacts{Id: root.ID.Bytes(), Place: root.Place, State: storagev1.RootState_ROOT_STATE_READY, VolumeId: "a"}
}

func (r *storageRig) second(t *testing.T) roots.Root {
	t.Helper()
	v, err := r.svc.Create(t.Context(), identity.Admin{}, roots.NewRoot{ContentType: alpha, Node: r.node, Path: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	<-r.changed
	return v.Root
}

func (r *storageRig) volumesChanged(t *testing.T, f *storagev1.Facts, want ...string) {
	t.Helper()
	r.storage.send(r.node, f)
	r.arrived.Note(r.node, string(role.Storage))
	select {
	case ev := <-r.volumeEvents:
		if ev.NodeID != r.node.String() || !slices.Equal(ev.VolumeIDs, want) {
			t.Fatalf("volumes.changed = %+v, want node %s's %v", ev, r.node, want)
		}
	case <-time.After(10 * time.Second):
		t.Fatalf("volumes.changed never named %v", want)
	}
}

func (r *storageRig) listVolumes(t *testing.T) []roots.NodeVolumes {
	t.Helper()
	out, err := r.svc.Volumes(t.Context(), identity.Admin{})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestTwoRootsOnOneVolumeAreListedOnce(t *testing.T) {
	r := rig(t, false)
	other := r.second(t)
	r.run(t)
	facts := func(freeA, freeB int64, at time.Time) *storagev1.Facts {
		return &storagev1.Facts{Roots: []*storagev1.RootFacts{ready(r.root), ready(other)}, Volumes: []*storagev1.Volume{volume("a", freeA, at), volume("b", freeB, at)}}
	}
	r.volumesChanged(t, facts(40, 90, sampled), "a", "b")
	want := []roots.NodeVolumes{{Node: r.node, Live: true, Volumes: []roots.VolumeView{{
		Volume: storing.Volume{ID: "a", Name: "/mnt/a", Mount: "/mnt/a", Kind: storing.Local, Total: 100, Free: 40, SampledAt: sampled},
		Roots:  []roots.ID{r.root.ID, other.ID},
	}}}}
	slices.SortFunc(want[0].Volumes[0].Roots, func(a, b roots.ID) int { return a.Compare(b.UUID) })
	if got := r.listVolumes(t); !equalVolumes(got, want) {
		t.Fatalf("Volumes = %+v, want %+v", got, want)
	}

	later := sampled.Add(5 * time.Minute)
	r.storage.send(r.node, facts(40, 90, later))
	r.arrived.Note(r.node, string(role.Storage))
	deadline := time.After(10 * time.Second)
	for r.listVolumes(t)[0].Volumes[0].SampledAt != later {
		select {
		case <-deadline:
			t.Fatal("the recorder never recorded the second sample")
		case <-time.After(time.Millisecond):
		}
	}
	r.volumesChanged(t, facts(40, 80, later), "b")
}

func equalVolumes(a, b []roots.NodeVolumes) bool {
	return slices.EqualFunc(a, b, func(x, y roots.NodeVolumes) bool {
		return x.Node == y.Node && x.Live == y.Live && slices.EqualFunc(x.Volumes, y.Volumes, func(v, w roots.VolumeView) bool {
			return v.ID == w.ID && v.Name == w.Name && v.Mount == w.Mount && v.Kind == w.Kind && v.Total == w.Total && v.Free == w.Free &&
				v.SampledAt.Equal(w.SampledAt) && slices.Equal(v.Roots, w.Roots)
		})
	})
}

func TestAnOfflineNodesVolumesKeepTheirLastFigures(t *testing.T) {
	r := rig(t, true)
	r.run(t)
	r.volumesChanged(t, &storagev1.Facts{Roots: []*storagev1.RootFacts{ready(r.root)}, Volumes: []*storagev1.Volume{volume("a", 40, sampled)}}, "a")
	got := r.listVolumes(t)
	if len(got) != 1 || got[0].Live || len(got[0].Volumes) != 1 || got[0].Volumes[0].Free != 40 || !got[0].Volumes[0].SampledAt.Equal(sampled) {
		t.Fatalf("Volumes of an offline Node = %+v, want its one volume as of %s, not live", got, sampled)
	}
}

func TestTheTopologyFiguresAreTheNodesRootsAndTheirVolumes(t *testing.T) {
	r := rig(t, false)
	r.run(t)
	r.volumesChanged(t, &storagev1.Facts{Roots: []*storagev1.RootFacts{ready(r.root)}, Volumes: []*storagev1.Volume{volume("a", 40, sampled), volume("b", 90, sampled)}}, "a", "b")
	var got nodes.StorageFigures
	err := r.lib.Read(t.Context(), func(ctx context.Context, rd storage.Reader[storage.Library]) error {
		var err error
		got, err = r.svc.Figures(ctx, rd, r.node)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	wantRoot := nodes.StorageRoot{ID: r.root.ID.UUID, ContentType: string(alpha), Path: r.root.Path, State: string(roots.Ready), VolumeID: "a"}
	if len(got.Roots) != 1 || got.Roots[0] != wantRoot || len(got.Volumes) != 1 || got.Volumes[0].ID != "a" || got.Volumes[0].Free != 40 {
		t.Fatalf("Figures = %+v, want %+v on volume a alone", got, wantRoot)
	}
}

func TestCheckStorageEveryReachesTheIntent(t *testing.T) {
	r := rig(t, false)
	every := func() int64 {
		t.Helper()
		var in *storagev1.Intent
		err := r.lib.Read(t.Context(), func(ctx context.Context, rd storage.Reader[storage.Library]) error {
			var err error
			in, err = roots.StorageIntent(files.RootContents{})(ctx, rd, r.node)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		return in.GetSampleEverySeconds()
	}
	field, err := r.svc.SampleEveryField(t.Context())
	if err != nil || field.Value != 300 || !field.IsDefault || every() != 300 {
		t.Fatalf("the default interval = %+v, %v, intent %ds; want 5 minutes everywhere", field, err, every())
	}
	minute := int64(60)
	v, err := roots.SampleEvery.Parse(&minute)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.svc.SetSampleEvery(t.Context(), v); err != nil {
		t.Fatal(err)
	}
	if field, _ := r.svc.SampleEveryField(t.Context()); field.Value != 60 || every() != 60 || r.storage.touches() != 2 {
		t.Fatalf("after setting a minute the field reads %ds, the intent %ds, touched %d; want 60, 60, and the Node touched", field.Value, every(), r.storage.touches())
	}
	if err := r.svc.SetSampleEvery(t.Context(), setting.Stored[int64]{}); err != nil {
		t.Fatal(err)
	}
	if every() != 300 {
		t.Fatalf("the reset intent samples every %ds, want 300", every())
	}
}

func TestAWrittenOffNodesRootsAreLost(t *testing.T) {
	r := rig(t, false)
	writeOff(t, r.lib, r.node)
	var lost map[roots.ID]bool
	err := r.lib.Read(t.Context(), func(ctx context.Context, rd storage.Reader[storage.Library]) error {
		var err error
		lost, err = roots.LostIn(ctx, rd)
		return err
	})
	if err != nil || !lost[r.root.ID] || len(lost) != 1 {
		t.Fatalf("LostIn = %v, %v; want the root alone", lost, err)
	}
	if v := r.view(t); v.State != roots.WrittenOff {
		t.Fatalf("the root reads %q, want written_off", v.State)
	}
}

func writeOff(t *testing.T, lib *storage.DB[storage.Library], node role.NodeID) {
	t.Helper()
	err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		if _, err := tx.ExecContext(ctx, "INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, randomblob(32), 'Media server', 0)", node); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, "INSERT INTO nodes_forfeit (node_id, role, declared_at) VALUES (?, ?, 0)", node, role.Storage)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestAPlacementIntoADrainingNodesRootWaitsInWords(t *testing.T) {
	lib := storagetest.Library(t, nodes.Schema(), works.Schema(), targets.Schema(), roots.Schema(), downloadclients.Schema(), grabs.Schema(), indexers.Schema(), files.Schema())
	node := role.NodeID{UUID: storage.NewUUID()}
	err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, "INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, randomblob(32), 'seedbox', 0)", node)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	root := rootstest.Root(t, lib, node, alpha, t.TempDir())
	download := filepath.Join(t.TempDir(), "orchard.mkv")
	if err := os.WriteFile(download, []byte("orchard"), 0o600); err != nil {
		t.Fatal(err)
	}
	svc := rootstest.Draining(t, lib, node, payload(download), alpha)
	to, err := storing.ParseRel(storing.HoldingDir + "/g.mkv")
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Place(t.Context(), root.ID, to, storing.Payload{})
	var waiting *roots.WaitingError
	var notReady *storing.NotReady
	const words = "seedbox is draining Storage, so the library root there takes no new files."
	if !errors.As(err, &waiting) || err.Error() != words || !errors.As(err, &notReady) || !notReady.Draining {
		t.Fatalf("Place into a draining Node's root = %v, want NotReady draining in the words %q", err, words)
	}
}
