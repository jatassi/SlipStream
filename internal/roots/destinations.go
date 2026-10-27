package roots

import (
	"context"
	"slices"

	"github.com/jatassi/slipstream/internal/roots/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
)

// Destination is a root as a Transfer into it reads it: where it is now,
// its State, and the volume its Node last found it on at that place.
type Destination struct {
	Root
	State State
	// Detail is the OS error text behind State.
	Detail string
	// Volume is nil until its Node found the root at its place.
	Volume *storing.Volume
}

// DestinationsIn is each of ids that rd holds, its State judged from s's
// facts and roster as List judges it, and its volume read in rd's
// snapshot. Watch closes when a State this reads moves.
func DestinationsIn(ctx context.Context, rd storage.Reader[file], ids []ID, s Storage, roster Roster) (map[ID]Destination, error) {
	all, err := All(ctx, rd)
	if err != nil {
		return nil, err
	}
	wanted := slices.DeleteFunc(all, func(root Root) bool { return !slices.Contains(ids, root.ID) })
	lost, err := LostIn(ctx, rd)
	if err != nil {
		return nil, err
	}
	states, err := judgeRoots(wanted, lost, s, roster)
	if err != nil {
		return nil, err
	}
	q := db.New(rd)
	seen, err := q.ListSeen(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListVolumes(ctx)
	if err != nil {
		return nil, err
	}
	volumes := make(map[volumeAt]storing.Volume, len(rows))
	for _, row := range rows {
		v, err := volumeOf(row)
		if err != nil {
			return nil, err
		}
		volumes[volumeAt{row.NodeID, row.ID}] = v
	}
	out := make(map[ID]Destination, len(wanted))
	for i, root := range wanted {
		d := Destination{Root: root, State: states[i].state, Detail: states[i].detail}
		at := slices.IndexFunc(seen, func(s db.RootsSeen) bool { return s.RootID == root.ID && s.Place == root.Place })
		if at >= 0 {
			if v, ok := volumes[volumeAt{root.Node, seen[at].VolumeID}]; ok {
				d.Volume = &v
			}
		}
		out[root.ID] = d
	}
	return out, nil
}
