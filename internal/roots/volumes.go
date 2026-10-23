package roots

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/nodes"
	storagev1 "github.com/jatassi/slipstream/internal/pb/slipstream/storage/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots/internal/db"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
)

// VolumesChanged names the volumes of a Node whose figures changed, came
// or went.
type VolumesChanged struct {
	NodeID    string   `json:"nodeId" format:"uuid"`
	VolumeIDs []string `json:"volumeIds" nullable:"false"`
}

var VolumesChangedEvent = live.AdminEvent[VolumesChanged]("volumes.changed")

type VolumesPublisher interface {
	Publish(ctx context.Context, v VolumesChanged)
}

// NodeVolumes is one Storage Node's volumes that hold a root.
type NodeVolumes struct {
	Node role.NodeID
	// Live: the Node is in step, so its figures are current; else each
	// reads as of its SampledAt.
	Live    bool
	Volumes []VolumeView
}

type VolumeView struct {
	storing.Volume
	Roots []ID
}

func (r *Roots) Volumes(ctx context.Context, _ identity.Admin) ([]NodeVolumes, error) {
	return r.volumes(ctx)
}

func (r *Roots) volumes(ctx context.Context) ([]NodeVolumes, error) {
	var out []NodeVolumes
	err := r.cfg.Library.Read(ctx, func(ctx context.Context, rd storage.Reader[file]) error {
		rows, err := db.New(rd).ListVolumes(ctx)
		if err != nil {
			return err
		}
		on, err := rootsOnVolumes(ctx, rd)
		if err != nil {
			return err
		}
		for _, row := range rows {
			v, err := volumeOf(row)
			if err != nil {
				return err
			}
			ids := on[volumeAt{row.NodeID, row.ID}]
			if len(ids) == 0 {
				continue
			}
			if len(out) == 0 || out[len(out)-1].Node != row.NodeID {
				out = append(out, NodeVolumes{Node: row.NodeID, Live: r.cfg.Roster.InStep(row.NodeID)})
			}
			last := &out[len(out)-1]
			last.Volumes = append(last.Volumes, VolumeView{Volume: v, Roots: ids})
		}
		return nil
	})
	return out, err
}

type volumeAt struct {
	node role.NodeID
	id   db.VolumeID
}

// rootsOnVolumes is the roots its Node last found ready on each volume, at
// the root's place.
func rootsOnVolumes(ctx context.Context, rd storage.Reader[file]) (map[volumeAt][]ID, error) {
	q := db.New(rd)
	all, err := q.ListRoots(ctx)
	if err != nil {
		return nil, err
	}
	seen, err := q.ListSeen(ctx)
	if err != nil {
		return nil, err
	}
	at := make(map[ID]db.RootsSeen, len(seen))
	for _, s := range seen {
		at[s.RootID] = s
	}
	out := map[volumeAt][]ID{}
	for _, root := range all {
		if s, ok := at[root.ID]; ok && s.Place == root.Place {
			key := volumeAt{root.NodeID, s.VolumeID}
			out[key] = append(out[key], root.ID)
		}
	}
	return out, nil
}

func volumeOf(row db.RootsVolume) (storing.Volume, error) {
	kind, err := storing.ParseKindWord(row.Kind)
	if err != nil {
		return storing.Volume{}, fmt.Errorf("volume %s: %w", row.ID, err)
	}
	return storing.Volume{
		ID: string(row.ID), Name: row.Name, Mount: row.Mount, Kind: kind, Total: row.Total, Free: row.Free, SampledAt: row.SampledAt.Time(),
	}, nil
}

// replaceVolumes makes node's recorded volumes the ones it reported, and
// answers the ids of those whose figures changed, came or went.
func replaceVolumes(ctx context.Context, q *db.Queries, node role.NodeID, reported []*storagev1.Volume) ([]string, error) {
	rows, err := q.ListVolumesOf(ctx, node)
	if err != nil {
		return nil, err
	}
	was := make(map[string]storing.Volume, len(rows))
	for _, row := range rows {
		v, err := volumeOf(row)
		if err != nil {
			return nil, err
		}
		was[v.ID] = v
	}
	if err := q.DeleteVolumesOf(ctx, node); err != nil {
		return nil, err
	}
	var changed []string
	for _, rv := range reported {
		kind, err := storing.ParseKind(rv.GetKind())
		if err != nil {
			return nil, fmt.Errorf("volume %s: %w", rv.GetId(), err)
		}
		v := storing.Volume{ID: rv.GetId(), Name: rv.GetName(), Mount: rv.GetMount(), Kind: kind, Total: rv.GetTotal(), Free: rv.GetFree()}
		err = q.InsertVolume(ctx, db.InsertVolumeParams{
			NodeID: node, ID: db.VolumeID(v.ID), Name: v.Name, Mount: v.Mount, Kind: kind.String(),
			Total: v.Total, Free: v.Free, SampledAt: storage.FromTime(time.UnixMilli(rv.GetSampledAtUnixMs())),
		})
		if err != nil {
			return nil, err
		}
		old, ok := was[v.ID]
		old.SampledAt = time.Time{}
		if !ok || old != v {
			changed = append(changed, v.ID)
		}
		delete(was, v.ID)
	}
	for id := range was {
		changed = append(changed, id)
	}
	slices.Sort(changed)
	return changed, nil
}

// LostIn is every root in rd whose Node's Storage the Admin wrote off, or
// whose Node the Admin removed. Its files count missing until a re-home
// takes it to a live Node.
func LostIn(ctx context.Context, rd storage.Reader[file]) (map[ID]bool, error) {
	writtenOff, err := nodes.WrittenOffIn(ctx, rd, role.Storage)
	if err != nil {
		return nil, err
	}
	all, err := All(ctx, rd)
	if err != nil {
		return nil, err
	}
	out := map[ID]bool{}
	for _, root := range all {
		if writtenOff[root.Node] {
			out[root.ID] = true
		}
	}
	return out, nil
}

func (r *Roots) Figures(ctx context.Context, rd storage.Reader[file], node role.NodeID) (nodes.StorageFigures, error) {
	q := db.New(rd)
	rows, err := q.ListRootsOnNode(ctx, node)
	if err != nil {
		return nodes.StorageFigures{}, err
	}
	on := make([]Root, len(rows))
	for i, row := range rows {
		on[i] = rootOf(row)
	}
	views, err := r.views(ctx, rd, on)
	if err != nil {
		return nodes.StorageFigures{}, err
	}
	var out nodes.StorageFigures
	holding := map[db.VolumeID]bool{}
	for _, v := range views {
		out.Roots = append(out.Roots, nodes.StorageRoot{ID: v.ID.UUID, ContentType: string(v.ContentType), Path: v.Path, State: string(v.State), VolumeID: v.VolumeID})
		if v.VolumeID != "" {
			holding[db.VolumeID(v.VolumeID)] = true
		}
	}
	volumes, err := q.ListVolumesOf(ctx, node)
	if err != nil {
		return nodes.StorageFigures{}, err
	}
	for _, row := range volumes {
		if !holding[row.ID] {
			continue
		}
		v, err := volumeOf(row)
		if err != nil {
			return nodes.StorageFigures{}, err
		}
		out.Volumes = append(out.Volumes, v)
	}
	return out, nil
}

var SampleEvery = setting.Duration{
	Key:     "storage.sampleEvery",
	Label:   "Check storage every",
	Default: 5 * time.Minute,
	Min:     time.Minute,
	Max:     24 * time.Hour,
	Step:    time.Minute,
}

func (r *Roots) SampleEveryField(ctx context.Context) (setting.DurationField, error) {
	var every *int64
	err := r.cfg.Library.Read(ctx, func(ctx context.Context, rd storage.Reader[file]) error {
		var err error
		every, err = db.New(rd).GetSampleEvery(ctx)
		return err
	})
	if err != nil {
		return setting.DurationField{}, fmt.Errorf("read how often storage is checked: %w", err)
	}
	return SampleEvery.Field(every), nil
}

// SetSampleEvery stores how often Storage Nodes sample their roots'
// volumes, and Touches each Node that holds a root so its intent carries
// it.
func (r *Roots) SetSampleEvery(ctx context.Context, v setting.Stored[int64]) error {
	var holders []role.NodeID
	err := r.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		if err := q.SetSampleEvery(ctx, v.Override()); err != nil {
			return err
		}
		var err error
		holders, err = q.ListRootNodes(ctx)
		return err
	})
	if err != nil {
		return fmt.Errorf("set how often storage is checked: %w", err)
	}
	r.cfg.Storage.Touch(holders...)
	return nil
}
