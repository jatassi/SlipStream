package moves

import (
	"context"
	"errors"
	"fmt"
	"path"
	"slices"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/moves/internal/db"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/transfers"
	"github.com/jatassi/slipstream/internal/works"
)

// step runs a's IO and the Write that records it.
func (m *Moves) step(ctx context.Context, u Unit, a Action) error {
	if u.Torrent != nil {
		switch a := a.(type) {
		case TakeLane:
			return m.takeTorrentLane(ctx, u)
		case PlaceTorrent:
			return m.placeTorrent(ctx, u, a.Transfer)
		case Undo:
			return m.undoTorrent(ctx, u)
		}
	}
	switch a := a.(type) {
	case TakeLane:
		return m.takeLane(ctx, u)
	case PlaceFile:
		return m.placeFile(ctx, u, a.Pos)
	case RemoveDropped:
		return m.removeDropped(ctx, u, a.Pos)
	case FinalPass:
		return m.finalPass(ctx, u)
	case Snapshot:
		return m.snapshot(ctx, u)
	case SwitchWork:
		return m.switchWork(ctx, u)
	case RemoveSource:
		return m.removeSource(ctx, u, a.Pos)
	case FinishCleanup:
		return m.finishCleanup(ctx, u)
	case Undo:
		return m.undo(ctx, u)
	case Reconcile:
		return m.reconcile(ctx, u)
	}
	panic(fmt.Sprintf("moves: %T runs no IO", a))
}

func actionName(a Action) string {
	switch a.(type) {
	case TakeLane:
		return "take_lane"
	case PlaceFile:
		return "place_file"
	case RemoveDropped:
		return "remove_dropped"
	case FinalPass:
		return "final_pass"
	case Snapshot:
		return "snapshot"
	case SwitchWork:
		return "switch"
	case RemoveSource:
		return "remove_source"
	case FinishCleanup:
		return "finish_cleanup"
	case Undo:
		return "undo"
	case Reconcile:
		return "reconcile"
	case PlaceTorrent:
		return "place_torrent"
	case Arrived:
		return "arrived"
	case Stop:
		return "stop"
	case Resume:
		return "resume"
	case AskStart:
		return "ask_start"
	case SwitchTorrent:
		return "switch_torrent"
	case Rollback:
		return "rollback"
	case Parted:
		return "parted"
	}
	panic(fmt.Sprintf("moves: %T is no step", a))
}

// update re-reads unit id in one Write and, while it is still in phase,
// has fn change it, then writes it and its files.
func (m *Moves) update(ctx context.Context, id UnitID, phase Phase, fn func(ctx context.Context, tx *storage.Tx[file], u *Unit) error) (Unit, error) {
	var u Unit
	var ran bool
	err := m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		var err error
		ran = false
		if u, err = load(ctx, tx, id); err != nil {
			return err
		}
		if u.Stage.Phase() != phase {
			return nil
		}
		if err := fn(ctx, tx, &u); err != nil {
			return err
		}
		ran = true
		if err := save(ctx, tx, u); err != nil {
			return err
		}
		return saveFiles(ctx, tx, u)
	})
	if err == nil && ran {
		m.cfg.Changed.Publish(ctx, changedOf(u))
	}
	return u, err
}

func changedOf(us ...Unit) Changed {
	c := Changed{BatchIDs: []string{}, UnitIDs: []string{}, WorkIDs: []string{}}
	for _, u := range us {
		c.BatchIDs = appendOnce(c.BatchIDs, u.Batch.String())
		c.UnitIDs = append(c.UnitIDs, u.ID.String())
		if u.Torrent == nil {
			c.WorkIDs = appendOnce(c.WorkIDs, u.Subject.Work.String())
		}
	}
	return c
}

// saveFiles writes u's files, each row again.
func saveFiles(ctx context.Context, tx *storage.Tx[file], u Unit) error {
	q := db.New(tx)
	if err := q.DeleteUnitFiles(ctx, u.ID); err != nil {
		return err
	}
	for _, f := range u.Files {
		if err := q.InsertUnitFile(ctx, insertFileColumns(u.ID, f)); err != nil {
			return err
		}
		if err := q.UpdateUnitFile(ctx, fileColumns(u.ID, f)); err != nil {
			return err
		}
	}
	return nil
}

func fileAt(u *Unit, pos int) *File {
	i := slices.IndexFunc(u.Files, func(f File) bool { return f.Position == pos })
	return &u.Files[i]
}

// fail ends u with code in tx, its undo due now.
func (m *Moves) fail(ctx context.Context, tx *storage.Tx[file], u *Unit, code FailureCode, detail string) error {
	if u.Torrent != nil {
		return m.rollback(ctx, tx, u, code, detail)
	}
	s := Failed{Code: code, At: u.Stage.Phase(), Detail: detail}
	u.Stage, u.Due = s, new(m.now())
	return m.recordFailed(ctx, tx, *u, s)
}

// probe reads rels of root on node: as sources with their companions, or
// as destinations with their occupants, in pages of probe_page.
func (m *Moves) probe(ctx context.Context, node role.NodeID, root roots.ID, rels []storing.Rel, source bool) (storing.Probed, error) {
	out := storing.Probed{
		Files: map[storing.Library]storing.ProbedFile{}, Companions: map[storing.Library][]storing.ProbedFile{},
		Occupants: map[storing.Library][]storing.ProbedFile{},
	}
	for page := range slices.Chunk(rels, m.cfg.Timings.ProbePage) {
		req := storing.ProbeMove{}
		pr := []storing.ProbeRoot{{Root: root.UUID, Rels: page}}
		if source {
			req.Sources = pr
		} else {
			req.Destinations = pr
		}
		pctx, cancel := context.WithTimeout(ctx, m.cfg.Timings.ProbeDeadline)
		got, err := m.cfg.Disk.ProbeMove(pctx, node, req)
		cancel()
		if err != nil {
			return storing.Probed{}, err
		}
		for k, v := range got.Files {
			out.Files[k] = v
		}
		for k, v := range got.Companions {
			out.Companions[k] = v
		}
		for k, v := range got.Occupants {
			out.Occupants[k] = v
		}
		out.Missing = append(out.Missing, got.Missing...)
	}
	return out, nil
}

// fresh is the Work's files as the source probe read them now: each live
// row and its companions. missing is each live row the probe didn't find.
func (m *Moves) fresh(ctx context.Context, u Unit) (out []File, missing []files.ID, ringing bool, err error) {
	var sits map[files.ID]files.Sitting
	err = m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		sits, err = files.Sits(ctx, r, []works.WorkID{u.Subject.Work})
		return err
	})
	if err != nil {
		return nil, nil, false, err
	}
	ids := make([]files.ID, 0, len(sits))
	for id, s := range sits {
		ids = append(ids, id)
		ringing = ringing || s.Ringing
	}
	slices.SortFunc(ids, func(a, b files.ID) int { return a.Compare(b.UUID) })
	rels := make([]storing.Rel, len(ids))
	for i, id := range ids {
		rels[i] = sits[id].Sits
	}
	p, err := m.probe(ctx, u.Subject.From.Node, u.Subject.From.Root, rels, true)
	if err != nil {
		return nil, nil, false, err
	}
	for _, id := range ids {
		at := storing.Library{Root: u.Subject.From.Root.UUID, Rel: sits[id].Sits}
		got, ok := p.Files[at]
		if !ok {
			missing = append(missing, id)
			continue
		}
		out = append(out, File{Kind: KindLibrary, ID: id, Sits: sits[id].Sits, Size: got.Size, Seen: got.Seen})
		for _, c := range p.Companions[at] {
			kind := KindCompanion
			if c.Ledger {
				kind = KindSidecar
			}
			out = append(out, File{Kind: kind, Sits: c.Rel, Size: c.Size, Seen: c.Seen, Ledger: c.Ledger})
		}
	}
	return out, missing, ringing, nil
}

// sameFile is whether a and b are one file with the same bytes.
func sameFile(a, b File) bool {
	return a.Kind == b.Kind && a.ID == b.ID && a.Sits == b.Sits && a.Size == b.Size && a.Seen.Same(b.Seen)
}

// takeLane probes the source and starts the unit on its lane: its Place
// on one Node, its Transfer across Nodes. A file mid-ring in holding has
// the unit wait for its renames.
func (m *Moves) takeLane(ctx context.Context, u Unit) error {
	now := m.now()
	got, missing, ringing, err := m.fresh(ctx, u)
	if err != nil {
		return err
	}
	_, err = m.update(ctx, u.ID, PhaseQueued, func(ctx context.Context, tx *storage.Tx[file], cu *Unit) error {
		held, err := db.New(tx).LaneHeld(ctx, string(cu.Lane))
		switch {
		case err != nil:
			return err
		case ringing:
			cu.Stage, cu.Due = waiting(cu.Stage, WaitRenames, now), nil
			return nil
		case held:
			cu.Stage, cu.Due = waiting(cu.Stage, WaitPairBusy, now), nil
			return nil
		case len(missing) > 0:
			return m.fail(ctx, tx, cu, SourceGone, fmt.Sprintf("%d files are gone from the source root", len(missing)))
		}
		cu.Files = merged(cu.Files, got)
		cu.Planned.At = now
		if cu.Lane.sameNode() {
			cu.Stage, cu.Due = Placing{Pass: cu.pass()}, &now
			return nil
		}
		return m.copy(ctx, tx, cu)
	})
	m.cfg.Transfers.Touch()
	return err
}

// merged is fresh with each file's place kept from was when its bytes are
// the same, and was's other placed files dropped.
func merged(was, fresh []File) []File {
	out := make([]File, 0, len(was)+len(fresh))
	next := 0
	for _, f := range was {
		if f.Placed != nil && !slices.ContainsFunc(fresh, func(g File) bool { return sameFile(f, g) }) {
			if f.Dropped == nil {
				f.Dropped = new(f.Placed.Modified)
			}
			out = append(out, f)
			next = max(next, f.Position+1)
		}
	}
	for _, f := range fresh {
		if i := slices.IndexFunc(was, func(g File) bool { return sameFile(f, g) && g.Dropped == nil }); i >= 0 {
			f.Placed, f.Position, f.Via, f.Index = was[i].Placed, was[i].Position, was[i].Via, was[i].Index
		} else {
			f.Position = next
		}
		for slices.ContainsFunc(out, func(g File) bool { return g.Position == f.Position }) {
			f.Position++
		}
		next = max(next, f.Position+1)
		out = append(out, f)
	}
	slices.SortFunc(out, func(a, b File) int { return a.Position - b.Position })
	return out
}

// placeFile puts one file at its Rel in the destination root through its
// holding name: from its Transfer's staged copy across Nodes, else a
// link, a clone, or a copy with a grant.
func (m *Moves) placeFile(ctx context.Context, u Unit, pos int) error {
	f := *fileAt(&u, pos)
	var sc *stagedCopy
	if s, ok := u.Stage.(Placing); ok && s.Transfer != nil {
		var err error
		if sc, err = m.staged(ctx, u, f); err != nil {
			return err
		}
	}
	placed, done, err := m.put(ctx, u, f, holdingOf(u.ID, pos, f.Sits), sc)
	if done || err != nil {
		return err
	}
	now := m.now()
	_, err = m.update(ctx, u.ID, PhasePlace, func(ctx context.Context, tx *storage.Tx[file], cu *Unit) error {
		g := fileAt(cu, pos)
		g.Placed, g.Copy = &placed.Seen, 0
		cu.Due = &now
		if sc != nil {
			return m.cfg.Transfers.Placed(ctx, tx, sc.transfer, sc.position)
		}
		return nil
	})
	if sc != nil {
		m.cfg.Transfers.Touch()
	}
	return err
}

// put places f at its Rel through hold, from whichever source fits what
// the destination root holds. done is a step whose Write already recorded
// it.
func (m *Moves) put(ctx context.Context, u Unit, f File, hold storing.Rel, sc *stagedCopy) (placed storing.Placed, done bool, err error) {
	to := u.Subject.To.Root
	p, err := m.probe(ctx, u.Subject.To.Node, to, []storing.Rel{f.Sits, hold}, false)
	if err != nil {
		return storing.Placed{}, false, err
	}
	at, held := occupant(p, to, f.Sits), occupant(p, to, hold)
	switch {
	case at != nil && at.Rel == f.Sits && at.Size == f.Size:
		return storing.Placed{Size: at.Size, Seen: at.Seen}, false, nil
	case at != nil:
		return storing.Placed{}, true, m.failIn(ctx, u.ID, PhasePlace, DestinationCheckFailed, "another file is at "+f.Sits.String())
	case held != nil && held.Size == f.Size:
		placed, err = m.move(ctx, to, hold, f)
		return placed, false, err
	case sc != nil:
		if _, err = m.cfg.Disk.Move(ctx, to, transfers.StagedRel(sc.transfer, sc.position), hold, false); err != nil {
			return storing.Placed{}, false, err
		}
		placed, err = m.move(ctx, to, hold, f)
		return placed, false, err
	case f.Via == ViaArriving:
		var from storing.Arriving
		err := m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
			var err error
			from, err = m.arrivingSource(ctx, r, u, f)
			return err
		})
		if err != nil {
			return storing.Placed{}, false, err
		}
		return m.placeFrom(ctx, u, f, hold, from)
	case !u.Lane.sameNode():
		return storing.Placed{}, true, m.recopy(ctx, u.ID)
	default:
		return m.placeFrom(ctx, u, f, hold, storing.Library{Root: u.Subject.From.Root.UUID, Rel: f.Sits})
	}
}

// placeFrom places f from a file on the destination Node, its source
// root's or its paired torrent's payload: a link, a clone, or a copy once
// granted. done is a step that ended in a grant or a wait for space,
// whose Write already recorded it.
func (m *Moves) placeFrom(ctx context.Context, u Unit, f File, hold storing.Rel, from storing.Source) (placed storing.Placed, done bool, err error) {
	to := u.Subject.To.Root
	if f.Ledger {
		_, err = m.cfg.Disk.PlaceSidecar(ctx, to, hold, from, f.Copy > 0)
	} else {
		_, err = m.cfg.Disk.Place(ctx, to, hold, from, f.Copy > 0)
	}
	if needs, ok := errors.AsType[*storing.NeedsCopyError](err); ok {
		return storing.Placed{}, true, m.grant(ctx, u.ID, f.Position, needs.Size)
	}
	if _, ok := errors.AsType[*storing.NeedsSpaceError](err); ok {
		return storing.Placed{}, true, m.waitSpace(ctx, u.ID)
	}
	if err != nil {
		return storing.Placed{}, false, err
	}
	placed, err = m.move(ctx, to, hold, f)
	return placed, false, err
}

// recopy has unit id send its unplaced files again: its Transfer holds no
// staged copy of one, after a restore or a re-home of the destination.
func (m *Moves) recopy(ctx context.Context, id UnitID) error {
	_, err := m.update(ctx, id, PhasePlace, func(ctx context.Context, tx *storage.Tx[file], cu *Unit) error {
		return m.copy(ctx, tx, cu)
	})
	m.cfg.Transfers.Touch()
	return err
}

func (m *Moves) move(ctx context.Context, root roots.ID, hold storing.Rel, f File) (storing.Placed, error) {
	if f.Ledger {
		return m.cfg.Disk.MoveSidecar(ctx, root, hold, f.Sits)
	}
	return m.cfg.Disk.Move(ctx, root, hold, f.Sits, false)
}

// occupant is the file a destination probe found at rel, or one whose
// name differs from rel's only in case, nil for none.
func occupant(p storing.Probed, root roots.ID, rel storing.Rel) *storing.ProbedFile {
	occ := p.Occupants[storing.Library{Root: root.UUID, Rel: rel}]
	if i := slices.IndexFunc(occ, func(o storing.ProbedFile) bool { return o.Rel == rel }); i >= 0 {
		return &occ[i]
	}
	if len(occ) > 0 {
		return &occ[0]
	}
	return nil
}

func (m *Moves) failIn(ctx context.Context, id UnitID, phase Phase, code FailureCode, detail string) error {
	_, err := m.update(ctx, id, phase, func(ctx context.Context, tx *storage.Tx[file], u *Unit) error {
		return m.fail(ctx, tx, u, code, detail)
	})
	return err
}

// grant grants file pos a copy of size bytes when its destination volume
// has room for it, else has the unit wait for space.
func (m *Moves) grant(ctx context.Context, id UnitID, pos int, size int64) error {
	now := m.now()
	_, err := m.update(ctx, id, PhasePlace, func(ctx context.Context, tx *storage.Tx[file], u *Unit) error {
		room, known, err := m.cfg.Imports.CopyRoom(ctx, tx, u.Subject.To.Root)
		if err != nil {
			return err
		}
		if known && size > room {
			u.Stage, u.Due = waiting(u.Stage, WaitSpace, now), nil
			return nil
		}
		fileAt(u, pos).Copy = size
		u.Stage, u.Due = waiting(u.Stage, "", now), &now
		return nil
	})
	return err
}

func (m *Moves) waitSpace(ctx context.Context, id UnitID) error {
	now := m.now()
	_, err := m.update(ctx, id, PhasePlace, func(_ context.Context, _ *storage.Tx[file], u *Unit) error {
		u.Stage, u.Due = waiting(u.Stage, WaitSpace, now), nil
		return nil
	})
	return err
}

// removeDropped removes the placed copy of a file the source no longer
// has as planned, then forgets the file.
func (m *Moves) removeDropped(ctx context.Context, u Unit, pos int) error {
	f := *fileAt(&u, pos)
	if _, err := m.cfg.Disk.RemoveIf(ctx, u.Subject.To.Root, f.Sits, f.Size, f.Placed.Modified); err != nil {
		return err
	}
	now := m.now()
	_, err := m.update(ctx, u.ID, PhasePlace, func(_ context.Context, _ *storage.Tx[file], cu *Unit) error {
		cu.Files = slices.DeleteFunc(cu.Files, func(g File) bool { return g.Position == pos })
		cu.Due = &now
		return nil
	})
	return err
}

// finalPass compares the source with the unit's files once more: the
// same files move on to the switch, any difference is placed first, sent
// across Nodes in the next pass's group.
func (m *Moves) finalPass(ctx context.Context, u Unit) error {
	now := m.now()
	got, missing, _, err := m.fresh(ctx, u)
	if err != nil {
		return err
	}
	_, err = m.update(ctx, u.ID, PhasePlace, func(ctx context.Context, tx *storage.Tx[file], cu *Unit) error {
		if len(missing) > 0 {
			return m.fail(ctx, tx, cu, SourceGone, fmt.Sprintf("%d files are gone from the source root", len(missing)))
		}
		files := merged(cu.Files, got)
		cu.Due, cu.Planned.At = &now, now
		if slices.EqualFunc(files, cu.Files, func(a, b File) bool {
			return sameFile(a, b) && a.Position == b.Position && (a.Dropped == nil) == (b.Dropped == nil)
		}) {
			cu.Stage = Switching{}
			return nil
		}
		cu.Files = files
		cu.Stage = Placing{Pass: cu.pass() + 1}
		if cu.Lane.sameNode() {
			return nil
		}
		return m.copy(ctx, tx, cu)
	})
	m.cfg.Transfers.Touch()
	return err
}

// snapshot has each Jellyfin and Emby server that shows the source root
// save the files' watch data, then judges the switch again.
func (m *Moves) snapshot(ctx context.Context, u Unit) error {
	m.cfg.MediaServers.PrepareMove(ctx, libraryFiles(u), u.Subject.From.Root, u.Subject.To.Root)
	now := m.now()
	_, err := m.update(ctx, u.ID, PhaseSwitch, func(_ context.Context, _ *storage.Tx[file], cu *Unit) error {
		cu.Stage, cu.Due = waiting(cu.Stage, WaitMediaServers, now), &now
		return nil
	})
	return err
}

// switchWork verifies every placed file at the destination, then switches
// the Work's rows to it in one Write. A Work that changed meanwhile goes
// back to its lane for another final pass.
func (m *Moves) switchWork(ctx context.Context, u Unit) error {
	rels := make([]storing.Rel, len(u.Files))
	for i, f := range u.Files {
		rels[i] = f.Sits
	}
	p, err := m.probe(ctx, u.Subject.To.Node, u.Subject.To.Root, rels, false)
	if err != nil {
		return err
	}
	if f := unplaced(p, u); f != nil {
		return m.failIn(ctx, u.ID, PhaseSwitch, DestinationCheckFailed, f.Sits.String()+" isn't the file SlipStream placed")
	}
	return m.switchFiles(ctx, u, false)
}

// switchFiles switches the Work's rows to u's placed files in one Write.
// restored is a restore's reconcile, whose files the caller just found
// placed: rows a scan marked gone at the source switch too.
func (m *Moves) switchFiles(ctx context.Context, u Unit, restored bool) error {
	now := m.now()
	var switched bool
	cu, err := m.update(ctx, u.ID, u.Stage.Phase(), func(ctx context.Context, tx *storage.Tx[file], cu *Unit) error {
		if err := m.fence(ctx, tx); err != nil {
			return err
		}
		importing, err := importingOf(ctx, tx, []works.WorkID{cu.Subject.Work})
		if err != nil {
			return err
		}
		if restored {
			cu.Files = u.Files
			if err := m.endCopy(ctx, tx, cu.ID); err != nil {
				return err
			}
		}
		cu.Reconcile = false
		busy := importing[cu.Subject.Work]
		if !busy {
			err = m.cfg.Files.Switch(ctx, tx, files.SwitchIn{
				Work: cu.Subject.Work, From: cu.Subject.From.Root, To: cu.Subject.To.Root,
				Files: switchedOf(cu.Files), GoneBefore: storage.FromTime(cu.Planned.At), Restored: restored,
			})
		}
		if _, stale := errors.AsType[*files.StaleError](err); busy || stale {
			cu.Stage, cu.Due = Queued{Pass: 1}, nil
			return nil
		}
		if err != nil {
			return err
		}
		for i := range cu.Files {
			cu.Files[i].Cleanup = CleanupPending
		}
		cu.Stage, cu.Due, switched = Cleaning{SwitchedAt: now}, &now, true
		return m.recordMoved(ctx, tx, *cu)
	})
	if err != nil || !switched {
		return err
	}
	m.cfg.FilesChanged.Publish(ctx, files.Changed{WorkID: cu.Subject.Work.String()})
	m.cfg.Files.Poke()
	m.cfg.Touch(cu.Subject.From.Node, cu.Subject.To.Node)
	return nil
}

// unplaced is the first of u's files the destination probe p doesn't
// show as SlipStream placed it, nil when each is.
func unplaced(p storing.Probed, u Unit) *File {
	for i, f := range u.Files {
		at := occupant(p, u.Subject.To.Root, f.Sits)
		if at == nil || at.Rel != f.Sits || at.Size != f.Size || !at.Seen.Same(*f.Placed) {
			return &u.Files[i]
		}
	}
	return nil
}

func switchedOf(fs []File) []files.Switched {
	var out []files.Switched
	for _, f := range fs {
		if f.Kind == KindLibrary {
			out = append(out, files.Switched{ID: f.ID, Size: f.Size, Seen: f.Seen, Sits: f.Sits, Placed: *f.Placed})
		}
	}
	return out
}

// removeSource removes one source file once its size and modified time
// are the ones copied. A file changed since stays, and the unit notes it.
func (m *Moves) removeSource(ctx context.Context, u Unit, pos int) error {
	f := *fileAt(&u, pos)
	got, err := m.cfg.Disk.RemoveIf(ctx, u.Subject.From.Root, f.Sits, f.Size, f.Seen.Modified)
	if err != nil {
		return err
	}
	now := m.now()
	_, err = m.update(ctx, u.ID, PhaseCleanup, func(_ context.Context, _ *storage.Tx[file], cu *Unit) error {
		g := fileAt(cu, pos)
		g.Cleanup = CleanupDone
		if got == storing.Changed {
			g.Cleanup = CleanupSkipped
			cu.Notes.Skipped = append(cu.Notes.Skipped, f.Sits.String())
		}
		cu.Due = &now
		return nil
	})
	return err
}

// folders is each folder of fs's Rels, outermost last.
func folders(fs []File) []storing.Rel {
	var out []storing.Rel
	for _, f := range fs {
		dir := path.Dir(f.Sits.String())
		if dir == "." {
			continue
		}
		if rel, err := storing.ParseRel(dir); err == nil && !slices.Contains(out, rel) {
			out = append(out, rel)
		}
	}
	return out
}

// finishCleanup removes the source folders the cleanup emptied and ends
// the unit.
func (m *Moves) finishCleanup(ctx context.Context, u Unit) error {
	for _, dir := range folders(u.Files) {
		if _, err := m.cfg.Disk.Prune(ctx, u.Subject.From.Root, dir, nil); err != nil {
			return err
		}
	}
	now := m.now()
	_, err := m.update(ctx, u.ID, PhaseCleanup, func(ctx context.Context, tx *storage.Tx[file], cu *Unit) error {
		if !cu.Notes.Released {
			if err := m.cfg.MediaServers.SourcesGone(ctx, tx, cu.Subject.From.Root, libraryFiles(*cu)); err != nil {
				return err
			}
			cu.Notes.Released = true
		}
		s, ok := cu.Stage.(Cleaning)
		if !ok {
			return fmt.Errorf("moves: unit %s is cleaning in phase %s", cu.ID, cu.Stage.Phase())
		}
		cu.Stage, cu.Due = Done{SwitchedAt: s.SwitchedAt, CleanedAt: now}, nil
		return m.finishBatch(ctx, tx, cu.Batch, storage.FromTime(now), cu.ID)
	})
	return err
}

// finishBatch marks batch finished once no unit but the one being written
// is open, unless it deletes the roots it empties: deleteEmptied finishes
// that one.
func (m *Moves) finishBatch(ctx context.Context, tx *storage.Tx[file], batch BatchID, now storage.Time, writing UnitID) error {
	q := db.New(tx)
	b, err := q.GetBatch(ctx, batch)
	if err != nil || b.DeleteEmptiedRoots == 1 {
		return err
	}
	open, err := q.ListOpenUnitsOfBatch(ctx, batch)
	if err != nil {
		return err
	}
	if slices.ContainsFunc(open, func(row db.MovesUnit) bool { return row.ID != writing }) {
		return nil
	}
	return q.FinishBatch(ctx, db.FinishBatchParams{At: now, ID: batch})
}

// undo removes what a canceled or failed unit left at the destination,
// ends its Transfer, and drops its Media Servers' snapshots.
func (m *Moves) undo(ctx context.Context, u Unit) error {
	if err := m.sweep(ctx, u); err != nil {
		return err
	}
	now := m.now()
	phase := u.Stage.Phase()
	_, err := m.update(ctx, u.ID, phase, func(ctx context.Context, tx *storage.Tx[file], cu *Unit) error {
		if err := m.endCopy(ctx, tx, cu.ID); err != nil {
			return err
		}
		if err := mediaservers.DropMovePrepared(ctx, tx, libraryFiles(*cu)); err != nil {
			return err
		}
		switch s := cu.Stage.(type) {
		case Failed:
			s.Undone = &now
			cu.Stage = s
		case Canceled:
			s.Undone = &now
			cu.Stage = s
		}
		cu.Files = unplacedFiles(cu.Files)
		cu.Due = nil
		return m.finishBatch(ctx, tx, cu.Batch, storage.FromTime(now), cu.ID)
	})
	m.cfg.Transfers.Touch()
	return err
}

// unplacedFiles is fs as planned, none placed, dropped or granted a copy.
func unplacedFiles(fs []File) []File {
	for i := range fs {
		fs[i].Copy, fs[i].Placed, fs[i].Dropped = 0, nil, nil
	}
	return fs
}

// sweep removes what u may have left at its destination, from one probe
// of every file's Rel and holding name. A holding file is the unit's by
// its name. A file at a Rel is removed only while no library row names
// that Rel in the destination root and its size is the planned one, so a
// Place a crash left unrecorded goes too.
func (m *Moves) sweep(ctx context.Context, u Unit) error {
	if len(u.Files) == 0 {
		return nil
	}
	to := u.Subject.To.Root
	var rels, sits []storing.Rel
	for _, f := range u.Files {
		rels = append(rels, f.Sits, holdingOf(u.ID, f.Position, f.Sits))
		sits = append(sits, f.Sits)
	}
	var rowed map[storing.Rel]files.ID
	err := m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		rowed, err = files.IDsAt(ctx, r, to, sits)
		return err
	})
	if err != nil {
		return err
	}
	p, err := m.probe(ctx, u.Subject.To.Node, to, rels, false)
	if err != nil {
		return err
	}
	for _, f := range u.Files {
		hold := holdingOf(u.ID, f.Position, f.Sits)
		if h := occupant(p, to, hold); h != nil && h.Rel == hold {
			if _, err := m.cfg.Disk.RemoveIf(ctx, to, hold, h.Size, h.Seen.Modified); err != nil {
				return err
			}
		}
		at := occupant(p, to, f.Sits)
		if _, owned := rowed[f.Sits]; owned || at == nil || at.Rel != f.Sits || at.Size != f.Size {
			continue
		}
		if _, err := m.cfg.Disk.RemoveIf(ctx, to, f.Sits, at.Size, at.Seen.Modified); err != nil {
			return err
		}
	}
	for _, dir := range folders(u.Files) {
		if _, err := m.cfg.Disk.Prune(ctx, to, dir, nil); err != nil {
			return err
		}
	}
	return nil
}

// reconciled is what a restored unit does once Reconciling ends.
type reconciled uint8

const (
	// again moves a unit whose source is whole from its lane again.
	again reconciled = iota
	// switchRestored switches a unit whose destination holds every file
	// and whose source is gone.
	switchRestored
	// lost fails a unit neither side holds whole.
	lost
)

// reconcileOf judges u against the probes of its source, src, and its
// destination, dst, each of its files at its Rel with its size.
func reconcileOf(u Unit, src, dst storing.Probed) reconciled {
	from, to := u.Subject.From.Root, u.Subject.To.Root
	whole, none, placed := true, true, true
	for _, f := range u.Files {
		if f.Dropped != nil {
			continue
		}
		got, ok := src.Files[storing.Library{Root: from.UUID, Rel: f.Sits}]
		whole = whole && ok && got.Size == f.Size
		none = none && !ok
		at := occupant(dst, to, f.Sits)
		placed = placed && at != nil && at.Rel == f.Sits && at.Size == f.Size
	}
	switch {
	case whole:
		return again
	case none && placed:
		return switchRestored
	}
	return lost
}

// reconcile judges a restored unit against both roots once Reconciling
// ends. A unit past its switch only clears its mark: its rows already
// name the destination.
func (m *Moves) reconcile(ctx context.Context, u Unit) error {
	if !u.beforeSwitch() {
		return m.unmark(ctx, u)
	}
	var rels []storing.Rel
	for _, f := range u.Files {
		if f.Dropped == nil {
			rels = append(rels, f.Sits)
		}
	}
	src, err := m.probe(ctx, u.Subject.From.Node, u.Subject.From.Root, rels, true)
	if err != nil {
		return err
	}
	dst, err := m.probe(ctx, u.Subject.To.Node, u.Subject.To.Root, rels, false)
	if err != nil {
		return err
	}
	switch reconcileOf(u, src, dst) {
	case again:
		return m.again(ctx, u)
	case switchRestored:
		u.Files = slices.DeleteFunc(u.Files, func(f File) bool { return f.Dropped != nil })
		for i, f := range u.Files {
			at := occupant(dst, u.Subject.To.Root, f.Sits)
			u.Files[i].Placed = &at.Seen
		}
		return m.switchFiles(ctx, u, true)
	case lost:
	}
	_, err = m.update(ctx, u.ID, u.Stage.Phase(), func(ctx context.Context, tx *storage.Tx[file], cu *Unit) error {
		cu.Reconcile = false
		return m.fail(ctx, tx, cu, SourceGone, "the source root no longer holds every file the Move planned")
	})
	return err
}

// again sweeps what a restored unit placed and queues it on its lane, to
// plan its files afresh when it takes the lane.
func (m *Moves) again(ctx context.Context, u Unit) error {
	if err := m.sweep(ctx, u); err != nil {
		return err
	}
	now := m.now()
	_, err := m.update(ctx, u.ID, u.Stage.Phase(), func(ctx context.Context, tx *storage.Tx[file], cu *Unit) error {
		cu.Reconcile, cu.Files = false, unplacedFiles(cu.Files)
		cu.Stage, cu.Due = Queued{Since: now}, &now
		return m.endCopy(ctx, tx, cu.ID)
	})
	m.cfg.Transfers.Touch()
	return err
}

// unmark clears a restored unit's mark: it goes on from its stored Stage.
func (m *Moves) unmark(ctx context.Context, u Unit) error {
	now := m.now()
	_, err := m.update(ctx, u.ID, u.Stage.Phase(), func(_ context.Context, _ *storage.Tx[file], cu *Unit) error {
		cu.Reconcile, cu.Due = false, &now
		return nil
	})
	return err
}
