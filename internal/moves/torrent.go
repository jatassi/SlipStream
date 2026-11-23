package moves

import (
	"context"
	"path"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/moves/internal/db"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/transfers"
	"github.com/jatassi/slipstream/internal/transfers/source"
)

// sourceIn is the torrent u moves, read in r.
func sourceIn(ctx context.Context, r storage.Reader[file], u Unit) (downloadclients.MoveSource, error) {
	got, err := downloadclients.MoveSources(ctx, r, []downloadclients.ID{u.Torrent.Torrent})
	return got[u.Torrent.Torrent], err
}

// torrentCopy is the Transfer of u's payload into its destination
// client's folder on the other Node.
func (m *Moves) torrentCopy(u Unit, src downloadclients.MoveSource) transfers.Request {
	t := u.Torrent
	to := transfers.DownloadFolder{Node: t.To.Node, InfoHash: src.InfoHash, ContentType: downloadclients.TypeFolder(m.cfg.Works, src.ContentType)}
	if t.To.Client != nil {
		to.Client = new([16]byte(t.To.Client.Bytes()))
	}
	r := transfers.Request{Purpose: transfers.Move, Job: u.ID.String(), Source: t.From.Node, Destination: to, Priority: transfers.Low}
	for _, f := range u.Files {
		r.Files = append(r.Files, transfers.FileRequest{
			Group: groupOf(0), Name: path.Base(f.Sits.String()), Source: source.Torrent{Hash: src.InfoHash, File: f.Index}, Size: f.Size,
		})
	}
	return r
}

// takeTorrentLane starts torrent unit u on its lane: its Place between two
// clients of one Node, its payload's Transfer across Nodes.
func (m *Moves) takeTorrentLane(ctx context.Context, u Unit) error {
	now := m.now()
	_, err := m.update(ctx, u.ID, PhaseQueued, func(ctx context.Context, tx *storage.Tx[file], cu *Unit) error {
		held, err := db.New(tx).LaneHeld(ctx, string(cu.Lane))
		if err != nil {
			return err
		}
		if held {
			cu.Stage, cu.Due = waiting(cu.Stage, WaitPairBusy, now), nil
			return nil
		}
		src, err := sourceIn(ctx, tx, *cu)
		if err != nil {
			return err
		}
		if src.Removed {
			return m.fail(ctx, tx, cu, SourceGone, "")
		}
		if len(cu.Files) == 0 {
			if err := filesOfPayload(ctx, db.New(tx), cu, src, now); err != nil {
				return err
			}
		}
		cu.Planned.At = now
		if cu.Lane.sameNode() {
			cu.Stage, cu.Due = Placing{}, &now
			return nil
		}
		id, err := m.cfg.Transfers.Want(ctx, tx, m.torrentCopy(*cu, src))
		cu.Stage, cu.Due = Copying{Transfer: id}, nil
		return err
	})
	m.cfg.Transfers.Touch()
	return err
}

// filesOfPayload writes, in q, the files of src's payload as u's: a
// drain's torrent that finished downloading after its plan.
func filesOfPayload(ctx context.Context, q *db.Queries, u *Unit, src downloadclients.MoveSource, now time.Time) error {
	u.Files, _ = payloadFiles(src, now)
	for _, f := range u.Files {
		if err := q.InsertUnitFile(ctx, insertFileColumns(u.ID, f)); err != nil {
			return err
		}
	}
	return nil
}

// RecordPlace is the step that records a torrent's place in an arriving
// Torrent, once the destination Node placed its payload.
const RecordPlace = "record_place"

// placeTorrent has the destination Node put u's payload in its client's
// folder, from the staging of Transfer tr or from the client on that Node
// that holds it, then adds the arriving Torrent. A built-in destination
// starts its handoff at once; an external one first checks the files.
func (m *Moves) placeTorrent(ctx context.Context, u Unit, tr *transfers.ID) error {
	t := u.Torrent
	var src downloadclients.MoveSource
	var staged transfers.Transfer
	err := m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		if src, err = sourceIn(ctx, r, u); err != nil || tr == nil {
			return err
		}
		staged, _, err = m.cfg.Transfers.Find(ctx, r, transfers.Move, u.ID.String())
		return err
	})
	if err != nil {
		return err
	}
	req, positions := m.placeRequest(u, src, staged, tr)
	res, err := m.cfg.Downloading(t.To.Node).PlacePayload(ctx, req)
	if err != nil {
		return err
	}
	if m.cfg.BeforeStep != nil {
		m.cfg.BeforeStep(ctx, Step{Unit: u.ID, Action: RecordPlace})
	}
	now := m.now()
	_, err = m.update(ctx, u.ID, PhasePlace, func(ctx context.Context, tx *storage.Tx[file], cu *Unit) error {
		a := downloadclients.Arriving{Unit: [16]byte(cu.ID.Bytes()), From: t.Torrent, Node: t.To.Node, Client: t.To.Client}
		if t.To.Client != nil {
			a.Folder = res.GetFolder()
		}
		id, err := downloadclients.InsertArriving(ctx, tx, a)
		if err != nil {
			return err
		}
		for _, pos := range positions {
			if err := m.cfg.Transfers.Placed(ctx, tx, *tr, pos); err != nil {
				return err
			}
		}
		cu.Torrent.Arriving, cu.Torrent.Folder = &id, a.Folder
		for i := range cu.Files {
			cu.Files[i].Placed = &storing.Seen{Modified: now}
		}
		if t.To.Client == nil {
			return m.handOff(ctx, tx, cu, now)
		}
		cu.Stage, cu.Due = Placing{Transfer: tr, Wait: WaitDownloading, Since: now}, new(now.Add(m.cfg.Timings.DownloadingEvery))
		return nil
	})
	m.cfg.Clients.Touch(t.From.Node, t.To.Node)
	return err
}

// placeRequest is the Place of u's payload: from the positions staged
// holds when it copied by Transfer tr, else from the client on the
// destination's Node that holds it. It answers the positions it places.
func (m *Moves) placeRequest(u Unit, src downloadclients.MoveSource, staged transfers.Transfer, tr *transfers.ID) (req *dlv1.PlacePayloadRequest, positions []int) {
	req = &dlv1.PlacePayloadRequest{InfoHash: src.InfoHash[:], AllowCopy: true, ContentType: downloadclients.TypeFolder(m.cfg.Works, src.ContentType)}
	if c := u.Torrent.To.Client; c != nil {
		req.ClientId = c.Bytes()
	}
	if tr == nil {
		p := &dlv1.HeldPayload{}
		for _, f := range u.Files {
			p.Files = append(p.Files, &dlv1.HeldFile{Index: int32(f.Index), Path: f.Sits.String()}) //nolint:gosec // G115: a torrent's file index fits
		}
		req.From = &dlv1.PlacePayloadRequest_Held{Held: p}
		return req, nil
	}
	p := &dlv1.StagedPayload{TransferId: tr.Bytes()}
	for _, f := range u.Files {
		pos := stagedAt(staged, f, src)
		positions = append(positions, pos)
		p.Files = append(p.Files, &dlv1.StagedFile{Position: int32(pos), Path: f.Sits.String()}) //nolint:gosec // G115: a position counts a torrent's files
	}
	req.From = &dlv1.PlacePayloadRequest_Staged{Staged: p}
	return req, positions
}

// stagedAt is the position of f's verified copy in t.
func stagedAt(t transfers.Transfer, f File, src downloadclients.MoveSource) int {
	ref := source.Torrent{Hash: src.InfoHash, File: f.Index}
	for _, tf := range t.Files {
		if tf.Source == ref && tf.Verified {
			return tf.Position
		}
	}
	return 0
}

// handOff has placed torrent unit u wait out its guard in tx, its source
// watched.
func (m *Moves) handOff(ctx context.Context, tx *storage.Tx[file], u *Unit, now time.Time) error {
	u.Stage, u.Due = Switching{Since: now, Handoff: &HandoffAt{Step: HandoffGuard, At: now}}, &now
	if err := releasePair(ctx, tx, u.ID, now); err != nil {
		return err
	}
	return m.cfg.Clients.WatchIn(ctx, tx, u.Torrent.Torrent, true)
}

// applyTorrent changes torrent unit u in tx by a, an Action that needs no
// IO, answering the Nodes to Touch once tx commits.
func (m *Moves) applyTorrent(ctx context.Context, tx *storage.Tx[file], u *Unit, a Action, now time.Time) ([]role.NodeID, error) {
	t := u.Torrent
	from, to := t.From.Node, t.To.Node
	switch a := a.(type) {
	case Arrived:
		return []role.NodeID{from}, m.handOff(ctx, tx, u, now)
	case Stop:
		if _, _, err := m.cfg.Clients.WantIn(ctx, tx, []downloadclients.ID{t.Torrent}, downloadclients.Paused); err != nil {
			return nil, err
		}
		m.handoffTo(u, HandoffStopping, now, now.Add(m.cfg.Timings.StopBound))
		return []role.NodeID{from}, nil
	case Resume:
		if err := m.cfg.Clients.ResumeIn(ctx, tx, t.Torrent); err != nil {
			return nil, err
		}
		m.handoffTo(u, HandoffGuard, now, now)
		u.Stage = waiting(u.Stage, WaitNodeOffline, now)
		return []role.NodeID{from}, nil
	case AskStart:
		asked := now.Truncate(time.Millisecond)
		tm := m.cfg.Timings
		if err := m.cfg.Clients.StartIn(ctx, tx, *t.Arriving, t.Torrent, asked, asked.Add(tm.HandoffBound-tm.ReplyWait-tm.BoundMargin)); err != nil {
			return nil, err
		}
		m.handoffTo(u, HandoffStarting, asked, asked.Add(tm.HandoffBound))
		return []role.NodeID{to}, nil
	case SwitchTorrent:
		return []role.NodeID{from, to}, m.switchTorrent(ctx, tx, u, now)
	case Rollback:
		return []role.NodeID{from, to}, m.rollback(ctx, tx, u, a.Code, a.Detail)
	case Parted:
		u.Stage, u.Due = Done{SwitchedAt: a.SwitchedAt, CleanedAt: now}, nil
		return nil, m.finishBatch(ctx, tx, u.Batch, storage.FromTime(now), u.ID)
	}
	return nil, nil
}

// handoffTo moves u's handoff to s at at, due at due, or sooner once a
// report of either Torrent wakes its guard wait.
func (m *Moves) handoffTo(u *Unit, s Handoff, at, due time.Time) {
	u.Stage, u.Due = Switching{Wait: WaitGuard, Since: at, Handoff: &HandoffAt{Step: s, At: at}}, &due
}

// switchTorrent hands u's torrent to its arriving Torrent in tx: the
// source is removed moved, its Grabs and its goal run follow, and the
// unit waits for the source to leave its Node.
func (m *Moves) switchTorrent(ctx context.Context, tx *storage.Tx[file], u *Unit, now time.Time) error {
	t := u.Torrent
	if err := m.cfg.Clients.SwitchIn(ctx, tx, t.Torrent, *t.Arriving); err != nil {
		return err
	}
	if err := m.cfg.Repoint(ctx, tx, t.Torrent, *t.Arriving, t.To.Node); err != nil {
		return err
	}
	if err := m.cfg.CopyGoalRun(ctx, tx, t.Torrent, *t.Arriving); err != nil {
		return err
	}
	if err := m.endCopy(ctx, tx, u.ID); err != nil {
		return err
	}
	u.Stage, u.Due = Cleaning{SwitchedAt: now, Wait: WaitDownloading, Since: now}, new(now.Add(m.cfg.Timings.DownloadingEvery))
	return m.recordTorrentMoved(ctx, tx, *u)
}

// rollback ends u with code in tx, its source seeding again. A refused or
// unconfirmed start keeps the paused arriving Torrent, so a Retry hands
// off again without another copy.
func (m *Moves) rollback(ctx context.Context, tx *storage.Tx[file], u *Unit, code FailureCode, detail string) error {
	if err := m.unwind(ctx, tx, *u); err != nil {
		return err
	}
	if err := unpair(ctx, tx, u.ID, m.now()); err != nil {
		return err
	}
	keeps := u.Torrent.Arriving != nil && (code == DestinationUnreachable || code == SwitchRefused)
	s := Failed{Code: code, At: u.Stage.Phase(), Detail: detail, KeepsCopy: keeps}
	u.Stage, u.Due = s, new(m.now())
	return m.recordFailed(ctx, tx, *u, s)
}

// unwind has a torrent unit's source seed as before its handoff, in tx.
func (m *Moves) unwind(ctx context.Context, tx *storage.Tx[file], u Unit) error {
	sw, ok := u.Stage.(Switching)
	switch {
	case !ok:
		return nil
	case sw.Handoff.Step == HandoffGuard:
		return m.cfg.Clients.WatchIn(ctx, tx, u.Torrent.Torrent, false)
	}
	return m.cfg.Clients.RollbackIn(ctx, tx, u.Torrent.Torrent, *u.Torrent.Arriving)
}

// cancelIn cancels u in tx before its switch, its handoff unwound first.
// A failed unit that keeps its copy is canceled too, so its undo deletes
// the copy.
func (m *Moves) cancelIn(ctx context.Context, tx *storage.Tx[file], u *Unit, now time.Time) error {
	if f, ok := u.Stage.(Failed); ok && f.KeepsCopy {
		u.Stage, u.Due = Canceled{At: f.At}, &now
		return nil
	}
	if !u.beforeSwitch() {
		return nil
	}
	if u.Torrent != nil {
		if err := m.unwind(ctx, tx, *u); err != nil {
			return err
		}
		if err := unpair(ctx, tx, u.ID, now); err != nil {
			return err
		}
	}
	u.Stage, u.Due = Canceled{At: u.Stage.Phase()}, &now
	return nil
}

// touchTorrents has the Nodes of us' torrent units derive their
// downloading intents again.
func (m *Moves) touchTorrents(us ...Unit) {
	for _, u := range us {
		if t := u.Torrent; t != nil {
			m.cfg.Clients.Touch(t.From.Node, t.To.Node)
		}
	}
}

// undoTorrent ends u's Transfer and removes its arriving Torrent with its
// files, unless the failed unit keeps it for a Retry. A unit that ended
// placing before any arriving Torrent recorded its place has the
// destination discard what the place may have put there.
func (m *Moves) undoTorrent(ctx context.Context, u Unit) error {
	if u.Torrent.Arriving == nil && endedAt(u.Stage) == PhasePlace {
		if err := m.discardPlaced(ctx, u); err != nil {
			return err
		}
	}
	now := m.now()
	var touched downloadclients.Touched
	_, err := m.update(ctx, u.ID, u.Stage.Phase(), func(ctx context.Context, tx *storage.Tx[file], cu *Unit) error {
		if err := m.endCopy(ctx, tx, cu.ID); err != nil {
			return err
		}
		keeps := false
		switch s := cu.Stage.(type) {
		case Failed:
			keeps = s.KeepsCopy
			s.Undone = &now
			cu.Stage = s
		case Canceled:
			s.Undone = &now
			cu.Stage = s
		}
		if t := cu.Torrent; t.Arriving != nil && !keeps {
			var err error
			if _, touched, err = m.cfg.Clients.RemoveIn(ctx, tx, []downloadclients.ID{*t.Arriving}, downloadclients.Canceled, true); err != nil {
				return err
			}
		}
		cu.Due = nil
		return m.finishBatch(ctx, tx, cu.Batch, storage.FromTime(now), cu.ID)
	})
	if err == nil {
		m.cfg.Clients.Touched(touched)
	}
	m.cfg.Transfers.Touch()
	return err
}

// endedAt is the phase a failed or canceled unit ended in.
func endedAt(s Stage) Phase {
	switch s := s.(type) {
	case Failed:
		return s.At
	case Canceled:
		return s.At
	}
	return s.Phase()
}

// discardPlaced has u's destination Node remove each payload file a
// place may have put in its client's folder.
func (m *Moves) discardPlaced(ctx context.Context, u Unit) error {
	var src downloadclients.MoveSource
	err := m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		src, err = sourceIn(ctx, r, u)
		return err
	})
	if err != nil {
		return err
	}
	t := u.Torrent
	req := &dlv1.DiscardPayloadRequest{InfoHash: src.InfoHash[:], Held: u.Lane.sameNode(), ContentType: downloadclients.TypeFolder(m.cfg.Works, src.ContentType)}
	if t.To.Client != nil {
		req.ClientId = t.To.Client.Bytes()
	}
	for _, f := range u.Files {
		req.Files = append(req.Files, &dlv1.DiscardFile{Index: int32(f.Index), Path: f.Sits.String(), Size: f.Size}) //nolint:gosec // G115: a torrent's file index fits
	}
	_, err = m.cfg.Downloading(t.To.Node).DiscardPayload(ctx, req)
	return err
}

// retryIn has failed torrent unit u, which kept its copy, hand off
// again from its guard in tx, its source watched. Any other failed unit
// queues again.
func (m *Moves) retryIn(ctx context.Context, tx *storage.Tx[file], u *Unit, s Failed) error {
	now := m.now()
	if s.KeepsCopy {
		return m.handOff(ctx, tx, u, now)
	}
	u.Stage, u.Due = Queued{Wait: WaitPairBusy, Since: now}, nil
	return nil
}

// handoffFacts reads, into out, the Handoff of each torrent unit of us
// past its copy.
func handoffFacts(ctx context.Context, r storage.Reader[file], us []Unit, out []Facts) error {
	var pairs []downloadclients.Pair
	for _, u := range us {
		if u.Torrent == nil {
			continue
		}
		switch u.Stage.(type) {
		case Queued, Placing, Switching, Cleaning:
			pairs = append(pairs, downloadclients.Pair{From: u.Torrent.Torrent, To: u.Torrent.Arriving})
		}
	}
	if len(pairs) == 0 {
		return nil
	}
	got, err := downloadclients.Handoffs(ctx, r, pairs)
	if err != nil {
		return err
	}
	importing, err := importingFrom(ctx, r, us)
	for i, u := range us {
		if u.Torrent != nil {
			out[i].Handoff = got[u.Torrent.Torrent]
			out[i].Importing = importing[u.Torrent.Torrent]
		}
	}
	return err
}

// importingFrom is which sources of us, the torrent units at their guard,
// a Grab imports from now: its Import reads the source the switch
// removes.
func importingFrom(ctx context.Context, r storage.Reader[file], us []Unit) (map[downloadclients.ID]bool, error) {
	var ids []downloadclients.ID
	for _, u := range us {
		if s, ok := u.Stage.(Switching); ok && u.Torrent != nil && s.Handoff.Step == HandoffGuard {
			ids = append(ids, u.Torrent.Torrent)
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}
	uses, err := grabs.TorrentUses{}.Of(ctx, r, ids)
	out := map[downloadclients.ID]bool{}
	for id, us := range uses {
		out[id] = slices.ContainsFunc(us, func(u downloadclients.Use) bool { return u.Stage == downloadclients.UseImporting })
	}
	return out, err
}

// Moving is each of ids a unit moves before its switch, or keeps a copy
// of, and where: downloadclients' and trackers' Moving.
func (m *Moves) Moving(ctx context.Context, r storage.Reader[file], ids []downloadclients.ID) (map[downloadclients.ID]downloadclients.MovingTo, error) {
	rows, err := db.New(r).ListMovingTorrents(ctx, uuidsOf(ids))
	out := make(map[downloadclients.ID]downloadclients.MovingTo, len(rows))
	for _, row := range rows {
		out[downloadclients.ID{UUID: row.TorrentID}] = downloadclients.MovingTo{Node: row.ToNodeID, Client: clientOf(row.ToClientID)}
	}
	return out, err
}

// CancelIn cancels the Move of Torrent id in tx, its handoff unwound
// first, so a seeding goal's action on it wins: trackers' Moving.
func (m *Moves) CancelIn(ctx context.Context, tx *storage.Tx[file], id downloadclients.ID) error {
	q := db.New(tx)
	rows, err := q.ListMovingTorrents(ctx, []storage.UUID{id.UUID})
	if err != nil {
		return err
	}
	us, err := unitsOf(ctx, q, rows)
	if err != nil {
		return err
	}
	now := m.now()
	for i := range us {
		if err := m.cancelIn(ctx, tx, &us[i], now); err != nil {
			return err
		}
		if err := save(ctx, tx, us[i]); err != nil {
			return err
		}
	}
	m.Poke()
	return nil
}

func uuidsOf(ids []downloadclients.ID) []storage.UUID {
	out := make([]storage.UUID, len(ids))
	for i, id := range ids {
		out[i] = id.UUID
	}
	return out
}

// MovesOf is the Move of each of ids that moves, or keeps a failed
// Move's copy, read in r: downloadclients' Moving.
func (m *Moves) MovesOf(ctx context.Context, r storage.Reader[file], ids []downloadclients.ID) (map[downloadclients.ID]downloadclients.Move, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	q := db.New(r)
	rows, err := q.ListMovingTorrents(ctx, uuidsOf(ids))
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	us, err := unitsOf(ctx, q, rows)
	if err != nil {
		return nil, err
	}
	vs, err := m.views(ctx, r, us)
	out := make(map[downloadclients.ID]downloadclients.Move, len(vs))
	for _, v := range vs {
		out[*v.Torrent] = downloadclients.Move{
			Unit: v.Unit.UUID, Batch: v.Batch.UUID, Phase: string(v.Phase), Waiting: string(v.Waiting),
			ToNode: v.To.Node, ToNodeName: v.To.NodeName, Words: v.Words,
		}
	}
	return out, err
}

// UsesClient is how many units that haven't ended move a torrent into or
// out of client, read in r: downloadclients' Moving.
func (m *Moves) UsesClient(ctx context.Context, r storage.Reader[file], client downloadclients.ClientID) (int64, error) {
	return db.New(r).CountClientUnits(ctx, client.UUID)
}
