package reachability

import (
	"bytes"
	"context"
	"errors"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/reachability/internal/db"
	"github.com/jatassi/slipstream/internal/reachability/judge"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks"
)

// Run follows what moves a route besides the network: Address, the Media
// Server records and the Nodes, and looks up what a suggestion needs. On
// start it checks again every route a release judged with another version.
func (r *Reachability) Run(ctx context.Context) error {
	address := r.cfg.Hub.Watch(ctx, identity.AddressChangedEvent)
	servers := r.cfg.Hub.Watch(ctx, mediaservers.ChangedEvent)
	roster := r.cfg.Hub.Watch(ctx, nodes.ChangedEvent)
	go r.lookUps(ctx)
	r.wakeLookUp()
	r.rejudge(ctx)
	r.serversChanged(ctx)
	r.nodesChanged(ctx)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-address:
			r.addressChanged(ctx)
			r.wakeLookUp()
		case <-servers:
			r.serversChanged(ctx)
			r.wakeLookUp()
		case <-roster:
			r.nodesChanged(ctx)
		}
	}
}

// rejudge starts every unit whose stored result another codec version
// wrote, so a release that changes Judged rewrites each row once.
func (r *Reachability) rejudge(ctx context.Context) {
	var units []tasks.UnitKey
	err := r.cfg.Library.Read(ctx, func(ctx context.Context, rd storage.Reader[storage.Library]) error {
		rows, err := db.New(rd).ListResults(ctx)
		if err != nil {
			return err
		}
		for _, row := range rows {
			res, err := resultOf(row)
			if err != nil {
				return err
			}
			if !res.read {
				units = append(units, Exposure{ID: row.ExposureID}.Unit())
			}
		}
		return nil
	})
	if err != nil {
		r.cfg.Log.WarnContext(ctx, "read the stored routes", "err", err)
		return
	}
	r.start(ctx, units...)
}

func (r *Reachability) addressChanged(ctx context.Context) {
	var units []tasks.UnitKey
	err := r.cfg.Library.Read(ctx, func(ctx context.Context, rd storage.Reader[storage.Library]) error {
		wanted, err := wantedIn(ctx, rd)
		for _, e := range wanted {
			if e.Kind == judge.UIPortal {
				units = append(units, e.Unit())
			}
		}
		return err
	})
	if err != nil {
		r.cfg.Log.WarnContext(ctx, "read the wanted routes", "err", err)
		return
	}
	r.start(ctx, units...)
}

// serversChanged reruns each Media Server Exposure whose record's digest
// moved, and tells every Node and page when a removal took one with it.
func (r *Reachability) serversChanged(ctx context.Context) {
	var (
		units   []tasks.UnitKey
		present map[storage.UUID]bool
	)
	err := r.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		var err error
		units, present, err = movedIn(ctx, tx)
		return err
	})
	if err != nil {
		r.cfg.Log.WarnContext(ctx, "compare the Media Servers with their routes", "err", err)
		return
	}
	r.mu.Lock()
	gone := false
	for id := range r.known {
		gone = gone || !present[id]
	}
	r.known = present
	r.mu.Unlock()
	if gone {
		r.touch(ctx)
		r.changed(ctx)
	}
	r.start(ctx, units...)
}

// movedIn stores the new digest of each Media Server Exposure whose
// record moved and is its unit, with every Media Server Exposure present.
func movedIn(ctx context.Context, tx *storage.Tx[storage.Library]) (units []tasks.UnitKey, present map[storage.UUID]bool, err error) {
	q := db.New(tx)
	rows, err := q.ListExposures(ctx)
	if err != nil {
		return nil, nil, err
	}
	recs, err := mediaservers.ServersIn(ctx, tx)
	if err != nil {
		return nil, nil, err
	}
	byID := map[storage.UUID]mediaservers.Record{}
	for _, rec := range recs {
		byID[rec.ID.UUID] = rec
	}
	present = map[storage.UUID]bool{}
	for _, row := range rows {
		if row.Kind != KindName(judge.MediaServer) {
			continue
		}
		present[row.ID.UUID] = true
		d := digest(byID[row.ServerID.UUID])
		if bytes.Equal(d, row.ServerDigest) {
			continue
		}
		if err := q.SetServerDigest(ctx, db.SetServerDigestParams{ID: row.ID, Digest: d}); err != nil {
			return nil, nil, err
		}
		units = append(units, exposureOf(row).Unit())
	}
	return units, present, nil
}

// nodesChanged deletes the rows of every removed Node. Readers filter
// removed Nodes anyway, so a row read before this Write is never shown.
func (r *Reachability) nodesChanged(ctx context.Context) {
	named, err := r.cfg.Roster.Names(ctx)
	if err != nil {
		r.cfg.Log.WarnContext(ctx, "list the Nodes", "err", err)
		return
	}
	var removed []role.NodeID
	for id, n := range named {
		if n.Removed {
			removed = append(removed, id)
		}
	}
	if len(removed) == 0 {
		return
	}
	err = r.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		q := db.New(tx)
		if err := q.DeleteNodes(ctx, removed); err != nil {
			return err
		}
		return q.DeleteNodeSettings(ctx, removed)
	})
	if err != nil {
		r.cfg.Log.WarnContext(ctx, "forget the removed Nodes' networks", "err", err)
		return
	}
	r.touch(ctx)
	r.changed(ctx)
}

// RunNow checks every route again, or one: Run again. No STUN pass from
// before the press counts.
func (r *Reachability) RunNow(ctx context.Context, by storage.UUID, only *judge.ExposureID) (tasks.RunID, error) {
	var units []tasks.UnitKey
	err := r.cfg.Library.Read(ctx, func(ctx context.Context, rd storage.Reader[storage.Library]) error {
		wanted, err := wantedIn(ctx, rd)
		for _, e := range wanted {
			if only == nil || *only == e.ID {
				units = append(units, e.Unit())
			}
		}
		return err
	})
	switch {
	case err != nil:
		return 0, err
	case len(units) == 0 && only != nil:
		return 0, ErrNotFound
	case len(units) == 0:
		return 0, ErrNothingWanted
	}
	named, err := r.cfg.Roster.Names(ctx)
	if err != nil {
		return 0, err
	}
	r.mu.Lock()
	for id := range named {
		r.freshAfter[id] = r.now()
	}
	r.mu.Unlock()
	st, err := r.cfg.Tasks.Start(ctx, tasks.Start{Task: TaskID, Units: units, Trigger: tasks.ByRunNow, By: by})
	if errors.Is(err, tasks.ErrUnknownTask) {
		return 0, ErrNothingWanted
	}
	return st.Run, err
}

// testersIn is each Exposure's last tester.
func testersIn(ctx context.Context, rd storage.Reader[storage.Library]) (map[judge.ExposureID]role.NodeID, error) {
	rows, err := db.New(rd).ListResults(ctx)
	out := map[judge.ExposureID]role.NodeID{}
	for _, row := range rows {
		out[row.ExposureID] = row.TestedByNodeID
	}
	return out, err
}
