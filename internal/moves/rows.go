package moves

import (
	"cmp"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/moves/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
)

// unitOf parses a row into its Unit, the one place a Stage is built from
// columns. The schema's CHECKs refuse a row this can't parse.
func unitOf(row db.MovesUnit) (Unit, error) {
	u := Unit{
		ID: row.ID, Batch: row.BatchID, Seq: row.Seq, Position: int(row.Position), Lane: Lane(row.Lane),
		Planned:   Planned{Bytes: row.PlannedBytes, At: row.PlannedAt.Time()},
		Reconcile: row.Reconcile == 1,
		Due:       timeOf(row.DueAt),
	}
	if !row.PairsWithUnitID.IsZero() {
		u.PairsWith = &row.PairsWithUnitID
	}
	if row.Kind == "torrent" {
		u.Torrent = torrentOf(row)
	} else {
		u.Subject = WorkSubject{
			Work: row.WorkID, Type: *row.ContentType,
			From: End{Root: row.FromRootID, Node: row.FromNodeID}, To: End{Root: row.ToRootID, Node: row.ToNodeID},
		}
	}
	if err := json.Unmarshal([]byte(row.Notes), &u.Notes); err != nil {
		return Unit{}, fmt.Errorf("unit %s's notes: %w", row.ID, err)
	}
	wait, since := Wait(deref(row.Waiting)), row.WaitingAt.Time()
	pass := int(row.Pass)
	switch Phase(row.Phase) {
	case PhaseQueued:
		u.Stage = Queued{Wait: wait, Since: since, Pass: pass}
	case PhaseTransfer:
		u.Stage = Copying{Transfer: row.TransferID, Pass: pass, Wait: wait, Since: since}
	case PhasePlace:
		p := Placing{Pass: pass, Wait: wait, Since: since}
		if !row.TransferID.IsZero() {
			p.Transfer = &row.TransferID
		}
		u.Stage = p
	case PhaseSwitch:
		sw := Switching{Wait: wait, Since: since, SwitchNow: timeOf(row.SwitchNowAt)}
		if row.Handoff != nil {
			sw.Handoff = &HandoffAt{Step: Handoff(*row.Handoff), At: row.HandoffAt.Time()}
		}
		u.Stage = sw
	case PhaseCleanup:
		u.Stage = Cleaning{SwitchedAt: row.SwitchedAt.Time(), Wait: wait, Since: since}
	case PhaseDone:
		u.Stage = Done{SwitchedAt: row.SwitchedAt.Time(), CleanedAt: row.CleanedAt.Time()}
	case PhaseFailed:
		u.Stage = Failed{
			Code: FailureCode(*row.FailureCode), At: Phase(*row.FailedPhase), Detail: deref(row.FailureDetail), KeepsCopy: row.KeepsCopy == 1,
			Undone: timeOf(row.UndoneAt),
		}
	case PhaseCanceled:
		u.Stage = Canceled{At: Phase(*row.FailedPhase), Undone: timeOf(row.UndoneAt)}
	default:
		return Unit{}, fmt.Errorf("unit %s is in phase %q", row.ID, row.Phase)
	}
	return u, nil
}

func torrentOf(row db.MovesUnit) *TorrentSubject {
	t := &TorrentSubject{
		Torrent: downloadclients.ID{UUID: row.TorrentID}, Folder: deref(row.ToFolder),
		From: Side{Node: row.FromNodeID, Client: clientOf(row.FromClientID)}, To: Side{Node: row.ToNodeID, Client: clientOf(row.ToClientID)},
	}
	if !row.DestTorrentID.IsZero() {
		t.Arriving = &downloadclients.ID{UUID: row.DestTorrentID}
	}
	return t
}

func clientOf(id storage.UUID) *downloadclients.ClientID {
	if id.IsZero() {
		return nil
	}
	return &downloadclients.ClientID{UUID: id}
}

// columns is u's mutable columns, its Stage's and its Due, as UpdateUnit
// writes them.
func columns(u Unit) (db.UpdateUnitParams, error) {
	notes, err := json.Marshal(u.Notes)
	if err != nil {
		return db.UpdateUnitParams{}, fmt.Errorf("moves: the notes of unit %s: %w", u.ID, err)
	}
	p := db.UpdateUnitParams{
		ID: u.ID, Phase: string(u.Stage.Phase()), DueAt: atOf(u.Due), PlannedBytes: u.Planned.Bytes, PlannedAt: storage.FromTime(u.Planned.At),
		Notes: string(notes),
	}
	if u.Reconcile {
		p.Reconcile = 1
	}
	if t := u.Torrent; t != nil {
		if t.Folder != "" {
			p.ToFolder = &t.Folder
		}
		if t.Arriving != nil {
			p.DestTorrentID = t.Arriving.UUID
		}
	}
	waiting := func(w Wait, since time.Time) {
		if w != "" {
			p.Waiting, p.WaitingAt = new(string(w)), storage.FromTime(since)
		}
	}
	switch s := u.Stage.(type) {
	case Queued:
		waiting(s.Wait, s.Since)
		p.Pass = int64(s.Pass)
	case Copying:
		waiting(s.Wait, s.Since)
		p.Pass, p.TransferID = int64(s.Pass), s.Transfer
	case Placing:
		waiting(s.Wait, s.Since)
		p.Pass = int64(s.Pass)
		if s.Transfer != nil {
			p.TransferID = *s.Transfer
		}
	case Switching:
		waiting(s.Wait, s.Since)
		p.SwitchNowAt = atOf(s.SwitchNow)
		if s.Handoff != nil {
			p.Handoff, p.HandoffAt = new(string(s.Handoff.Step)), storage.FromTime(s.Handoff.At)
		}
	case Cleaning:
		waiting(s.Wait, s.Since)
		p.SwitchedAt = storage.FromTime(s.SwitchedAt)
	case Done:
		p.SwitchedAt, p.CleanedAt = storage.FromTime(s.SwitchedAt), storage.FromTime(s.CleanedAt)
	case Failed:
		p.FailureCode, p.FailedPhase, p.UndoneAt = new(string(s.Code)), new(string(s.At)), atOf(s.Undone)
		if s.KeepsCopy {
			p.KeepsCopy = 1
		}
		if s.Detail != "" {
			p.FailureDetail = &s.Detail
		}
	case Canceled:
		p.FailedPhase, p.UndoneAt = new(string(s.At)), atOf(s.Undone)
	}
	return p, nil
}

func fileOf(row db.MovesUnitFile) (File, error) {
	sits, err := storing.ParseRel(row.SitsRel)
	if err != nil {
		return File{}, fmt.Errorf("file %d of unit %s: %w", row.Position, row.UnitID, err)
	}
	f := File{
		Position: int(row.Position), Kind: FileKind(row.Kind), Via: Via(row.Via), ID: row.FileID, Sits: sits, Size: row.Size,
		Seen: seenOf(row.ModifiedAt, row.NodeFileID, row.NodeVolumeID), Ledger: row.Ledger == 1,
		Cleanup: Cleanup(deref(row.Cleanup)), Copy: row.CopyBytes, Dropped: timeOf(row.DroppedAt),
	}
	if row.PayloadIndex != nil {
		f.Index = int(*row.PayloadIndex)
	}
	if row.Placed == 1 {
		f.Placed = new(seenOf(row.PlacedModifiedAt, row.PlacedNodeFileID, row.PlacedNodeVolumeID))
	}
	return f, nil
}

func fileColumns(u UnitID, f File) db.UpdateUnitFileParams {
	p := db.UpdateUnitFileParams{
		UnitID: u, Position: int64(f.Position), Size: f.Size, CopyBytes: f.Copy, DroppedAt: atOf(f.Dropped),
	}
	p.ModifiedAt, p.NodeFileID, p.NodeVolumeID = seenColumns(f.Seen)
	if f.Placed != nil {
		p.Placed = 1
		p.PlacedModifiedAt, p.PlacedNodeFileID, p.PlacedNodeVolumeID = seenColumns(*f.Placed)
	}
	if f.Cleanup != "" {
		p.Cleanup = new(string(f.Cleanup))
	}
	return p
}

func insertFileColumns(u UnitID, f File) db.InsertUnitFileParams {
	p := db.InsertUnitFileParams{UnitID: u, Position: int64(f.Position), Kind: string(f.Kind), Via: string(cmp.Or(f.Via, ViaLibrary)), SitsRel: f.Sits.String(), Size: f.Size}
	if !f.ID.IsZero() {
		p.FileID = f.ID
	}
	p.ModifiedAt, p.NodeFileID, p.NodeVolumeID = seenColumns(f.Seen)
	if f.Ledger {
		p.Ledger = 1
	}
	if f.Kind == KindPayload || f.Via == ViaArriving {
		p.PayloadIndex = new(int64(f.Index))
	}
	return p
}

func seenOf(modified storage.Time, file, volume *string) storing.Seen {
	s := storing.Seen{Modified: modified.Time()}
	if file != nil {
		s.ID = storing.FileID{File: *file, Volume: deref(volume)}
	}
	return s
}

func seenColumns(s storing.Seen) (modified storage.Time, file, volume *string) {
	if s.ID.IsZero() {
		return storage.FromTime(s.Modified), nil, nil
	}
	return storage.FromTime(s.Modified), &s.ID.File, &s.ID.Volume
}

func timeOf(t storage.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return new(t.Time())
}

func atOf(t *time.Time) storage.Time {
	if t == nil {
		return storage.Time{}
	}
	return storage.FromTime(*t)
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
