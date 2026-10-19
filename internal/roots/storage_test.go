package roots_test

import (
	"context"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/nodes"
	storagev1 "github.com/jatassi/slipstream/internal/pb/slipstream/storage/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/roots/rootstest"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

const alpha works.TypeID = "alpha"

type control struct {
	mu      sync.Mutex
	facts   map[role.NodeID]*storagev1.Facts
	touched []role.NodeID
}

func (c *control) Touch(ids ...role.NodeID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.touched = append(c.touched, ids...)
}

func (c *control) Facts(node role.NodeID) (*storagev1.Facts, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	f, ok := c.facts[node]
	return f, ok
}

func (c *control) send(node role.NodeID, f *storagev1.Facts) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.facts[node] = f
}

func (c *control) touches() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.touched)
}

type roster struct{ away role.NodeID }

func (roster) Holds(role.Name, role.NodeID) bool { return true }
func (r roster) InStep(node role.NodeID) bool    { return node != r.away }

type changes chan roots.Changed

func (c changes) Publish(_ context.Context, ev roots.Changed) { c <- ev }

type volumeChanges chan roots.VolumesChanged

func (c volumeChanges) Publish(_ context.Context, ev roots.VolumesChanged) { c <- ev }

type storageRig struct {
	lib          *storage.DB[storage.Library]
	svc          *roots.Roots
	storage      *control
	changed      changes
	volumeEvents volumeChanges
	arrived      *roots.Arrivals
	node         role.NodeID
	root         roots.Root
}

func rig(t *testing.T, offline bool) *storageRig {
	t.Helper()
	lib := storagetest.Library(t, nodes.Schema(), works.Schema(), targets.Schema(), roots.Schema(), downloadclients.Schema(), grabs.Schema(), files.Schema())
	r := &storageRig{lib: lib, storage: &control{facts: map[role.NodeID]*storagev1.Facts{}}, changed: make(changes, 8), volumeEvents: make(volumeChanges, 8), node: role.NodeID{UUID: storage.NewUUID()}}
	cfg := rootstest.Config(t, lib, alpha)
	var away role.NodeID
	if offline {
		away = r.node
	}
	cfg.Roster = roster{away: away}
	cfg.Storage = r.storage
	cfg.Changed = r.changed
	cfg.Volumes = r.volumeEvents
	r.arrived = cfg.Arrivals
	svc, err := roots.Open(t.Context(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	r.svc = svc
	v, err := svc.Create(t.Context(), identity.Admin{}, roots.NewRoot{ContentType: alpha, Node: r.node, Path: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	r.root = v.Root
	<-r.changed
	if r.storage.touches() != 1 {
		t.Fatalf("creating a root touched %d times, want its Node once", r.storage.touches())
	}
	return r
}

func (r *storageRig) run(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- r.svc.Run(ctx) }()
	t.Cleanup(func() {
		cancel()
		if err := <-done; err != nil {
			t.Error(err)
		}
	})
}

func (r *storageRig) report(state storagev1.RootState, place int64) *storagev1.Facts {
	return &storagev1.Facts{
		Roots: []*storagev1.RootFacts{{Id: r.root.ID.Bytes(), Place: place, State: state, VolumeId: "v1"}},
		Volumes: []*storagev1.Volume{{
			Id: "v1", Name: "/media", Mount: "/media", Kind: storagev1.VolumeKind_VOLUME_KIND_LOCAL,
			Total: 100, Free: 40, SampledAtUnixMs: time.Now().UnixMilli(),
		}},
	}
}

func (r *storageRig) arrive(t *testing.T, f *storagev1.Facts) {
	t.Helper()
	r.storage.send(r.node, f)
	r.arrived.Note(r.node, string(role.Storage))
	select {
	case ev := <-r.changed:
		if !slices.Equal(ev.RootIDs, []string{r.root.ID.String()}) {
			t.Fatalf("the recorder published %v, want only the rig's root", ev.RootIDs)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the recorder never published the rig's root")
	}
}

func (r *storageRig) view(t *testing.T) roots.View {
	t.Helper()
	all, err := r.svc.List(t.Context(), identity.Admin{})
	if err != nil || len(all) != 1 {
		t.Fatalf("listing roots = %+v, %v; want the rig's root", all, err)
	}
	return all[0]
}

func (r *storageRig) adopting(t *testing.T) bool {
	t.Helper()
	var in *storagev1.Intent
	err := r.lib.Read(t.Context(), func(ctx context.Context, rd storage.Reader[storage.Library]) error {
		var err error
		in, err = roots.StorageIntent(files.RootContents{})(ctx, rd, r.node)
		return err
	})
	if err != nil || len(in.GetRoots()) != 1 {
		t.Fatalf("the Storage intent = %v, %v; want the rig's root", in, err)
	}
	return in.GetRoots()[0].GetAdopt()
}

func (r *storageRig) volumes(t *testing.T) int {
	t.Helper()
	var n int
	if err := r.lib.QueryRowContext(t.Context(), "SELECT count(*) FROM roots_volume WHERE node_id = ?", r.node).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestARootIsAdoptedOnceItsNodeFindsItReady(t *testing.T) {
	r := rig(t, false)
	r.run(t)
	if v := r.view(t); v.State != roots.Checking || !r.adopting(t) {
		t.Fatalf("a root no Node reported reads %q, adopting %v; want checking and adopting", v.State, r.adopting(t))
	}

	r.arrive(t, r.report(storagev1.RootState_ROOT_STATE_UNWRITABLE, r.root.Place))
	if v := r.view(t); v.State != roots.Unwritable || v.VolumeID != "" || !r.adopting(t) || r.storage.touches() != 1 {
		t.Fatalf("an unwritable root reads %q on %q, adopting %v, touched %d; want unwritable on no volume, adopting, touched once", v.State, v.VolumeID, r.adopting(t), r.storage.touches())
	}
	if n := r.volumes(t); n != 1 {
		t.Fatalf("the library records %d volumes of the Node, want the 1 it reported", n)
	}

	r.arrive(t, r.report(storagev1.RootState_ROOT_STATE_READY, r.root.Place))
	if v := r.view(t); v.State != roots.Ready || v.VolumeID != "v1" || r.adopting(t) || r.storage.touches() != 2 {
		t.Fatalf("a ready root reads %q on %q, adopting %v, touched %d; want ready on v1, not adopting, touched again", v.State, v.VolumeID, r.adopting(t), r.storage.touches())
	}
}

func TestAReportAtAnotherPlaceIsNotTheRoots(t *testing.T) {
	r := rig(t, false)
	r.storage.send(r.node, r.report(storagev1.RootState_ROOT_STATE_READY, r.root.Place+1))
	if v := r.view(t); v.State != roots.Checking || v.VolumeID != "" {
		t.Fatalf("a root reported at another place reads %q on %q, want checking on no volume", v.State, v.VolumeID)
	}
}

func TestARootOnAnOfflineNodeIsOffline(t *testing.T) {
	r := rig(t, true)
	r.storage.send(r.node, r.report(storagev1.RootState_ROOT_STATE_READY, r.root.Place))
	if v := r.view(t); v.State != roots.Offline {
		t.Fatalf("a root on an offline Node reads %q, want offline", v.State)
	}
}

func TestStorageOwnsANodesRootsForever(t *testing.T) {
	r := rig(t, false)
	owner := roots.StorageOwner(files.RootContents{})
	var held nodes.Held
	err := r.lib.Read(t.Context(), func(ctx context.Context, rd storage.Reader[storage.Library]) error {
		var err error
		held, err = owner.Held(ctx, rd, r.node)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if owner.Role != role.Storage || held.Units != 1 || held.Noun.One != "root" || !held.Endless {
		t.Fatalf("Storage's owner = %v holding %+v, want Storage holding 1 root endlessly", owner.Role, held)
	}
}
