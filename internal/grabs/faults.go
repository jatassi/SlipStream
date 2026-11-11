package grabs

import (
	"context"
	"errors"
	"slices"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/storage"
)

// causeOfFault is the Blocklist cause of each release fault.
var causeOfFault = map[downloadclients.FaultKind]BlockCause{
	downloadclients.FaultUnregistered: CauseUnregistered,
	downloadclients.FaultNoPayload:    CauseNoPayload,
	downloadclients.FaultEngineError:  CauseEngineError,
}

// settleFaults blocklists the Release of each faulted Torrent a Grab used
// that grabs hasn't settled, one Write per Torrent: the entry, and, while
// a Grab of it downloads, the Torrent's removal with reason release_fault.
// Its downloading Grabs then fail through failOfGone. Grabs past
// downloading carry on. While Reconciling it settles nothing, and the
// sticky fault waits.
func (gr *Grabs) settleFaults(ctx context.Context) error {
	faults, err := gr.unsettledFaults(ctx)
	if err != nil || len(faults) == 0 {
		return err
	}
	var (
		settled bool
		touched []downloadclients.Touched
	)
	for _, f := range faults {
		var t *downloadclients.Touched
		err = gr.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
			var err error
			t, err = gr.settleFault(ctx, tx, f)
			return err
		})
		if err != nil {
			break
		}
		settled = true
		if t != nil {
			touched = append(touched, *t)
		}
	}
	if len(touched) > 0 {
		gr.cfg.Downloads.Touched(touched...)
	}
	if settled {
		gr.cfg.BlocklistChanged.Publish(ctx, BlocklistChanged{})
	}
	if errors.Is(err, ErrReconciling) {
		return nil
	}
	return err
}

// unsettledFaults is each faulted Torrent a Grab used whose fault grabs
// hasn't settled, oldest fault first.
func (gr *Grabs) unsettledFaults(ctx context.Context) ([]downloadclients.FaultedTorrent, error) {
	var out []downloadclients.FaultedTorrent
	err := gr.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		faulted, err := downloadclients.Faulted(ctx, r)
		if err != nil || len(faulted) == 0 {
			return err
		}
		ids := make([]downloadclients.ID, len(faulted))
		for i, f := range faulted {
			ids[i] = f.ID
		}
		unsettled, err := db.New(r).ListUnsettledFaultTorrents(ctx, ids)
		if err != nil {
			return err
		}
		for _, f := range faulted {
			if slices.Contains(unsettled, f.ID) {
				out = append(out, f)
			}
		}
		return nil
	})
	return out, err
}

// settleFault blocklists f's Release and, while a Grab of it downloads,
// removes f with its download copy, in tx, and is what the removal
// touched, nil for none. Block holds the Reconciling fence, so it refuses
// before anything is removed.
func (gr *Grabs) settleFault(ctx context.Context, tx *storage.Tx[file], f downloadclients.FaultedTorrent) (*downloadclients.Touched, error) {
	qs := db.New(tx)
	used, err := qs.ListGrabsOfTorrent(ctx, f.ID)
	if err != nil || len(used) == 0 {
		// None when the Work and its Grabs were deleted since the read.
		return nil, err
	}
	e, err := entryOfFault(ctx, tx, f, used[0])
	if err != nil {
		return nil, err
	}
	if err := (BlocklistOn{Reconciling: gr.cfg.Reconciling}).Block(ctx, tx, e); err != nil {
		return nil, err
	}
	var touched *downloadclients.Touched
	if !f.Removed && slices.ContainsFunc(used, func(g db.ListGrabsOfTorrentRow) bool { return g.Stage == Downloading }) {
		_, t, err := gr.cfg.Downloads.RemoveIn(ctx, tx, []downloadclients.ID{f.ID}, downloadclients.ReleaseFaulted, true)
		if err != nil {
			return nil, err
		}
		touched = &t
	}
	return touched, qs.SettleFault(ctx, db.SettleFaultParams{TorrentID: f.ID, SettledAt: storage.Now()})
}

// entryOfFault is the Blocklist entry of f's release fault: newest, the
// Torrent's newest Grab, names the Release, and the fault is the cause.
// A fault has no actor.
func entryOfFault(ctx context.Context, r storage.Reader[file], f downloadclients.FaultedTorrent, newest db.ListGrabsOfTorrentRow) (BlockEntry, error) {
	e := BlockEntry{
		InfoHash: f.InfoHash, Title: newest.ReleaseTitle, Size: newest.ReleaseSize, Indexer: newest.IndexerID,
		Tracker: newest.TrackerID, Work: newest.WorkID, Grab: newest.ID, Cause: causeOfFault[f.Fault.Kind],
	}
	if !newest.IndexerID.IsZero() {
		idx, err := indexers.Get(ctx, r, newest.IndexerID)
		if err != nil {
			return BlockEntry{}, err
		}
		e.IndexerName = idx.Name
	}
	named, err := nodes.Names(ctx, r)
	if err != nil {
		return BlockEntry{}, err
	}
	var node string
	if i := slices.IndexFunc(named, func(n nodes.Named) bool { return n.ID == f.Node }); i >= 0 {
		node = named[i].Name
	}
	e.Client = node + "'s built-in client"
	if f.Client != nil {
		e.Client = f.Client.Name + " on " + node
	}
	return e, nil
}
