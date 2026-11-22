package moves

import (
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/transfers"
)

// Facts are what a pass read for one unit, owned elsewhere.
type Facts struct {
	// Head is whether the unit is the first ready unit of a lane no unit
	// holds.
	Head bool
	// NodesUp is whether both its Nodes are connected and in step.
	NodesUp bool
	// Transfer is its Transfer's view while it copies, nil when transfers
	// has no record of it.
	Transfer *transfers.View
	// Importing and OpenFiles hold its switch: an Import of its Items, or
	// a Transfer, Speech Map or Sidecar write on its files.
	Importing, OpenFiles bool
	// Playing is the servers someone plays its files on now.
	Playing []link.ServerID
	// Snapshotted is each server that snapshots first and shows the
	// source root, and whether it holds a fresh snapshot of every file.
	Snapshotted map[link.ServerID]bool
	// Safe is each Plex server that showed the source root, and whether it
	// found every file at its new path.
	Safe map[link.ServerID]bool
}

// Rules are the settings and timings next reads.
type Rules struct {
	PlexBound    time.Duration
	PlayingEvery time.Duration
}

// Action is exactly one of the types below. Each Write that applies one
// sets the unit's due_at from it: Hold's Until, now for an Action that
// leads to another step, or NULL when only an event can end the wait.
type Action interface{ action() }

type (
	// Hold writes the wait when it changed, and has the unit due at
	// Until, or by its wait's events when Until is zero.
	Hold struct {
		Wait  Wait
		Until time.Time
	}
	// TakeLane probes the source and starts the copy on the lane.
	TakeLane struct{}
	// PlaceFile puts one file at its Rel in the destination root.
	PlaceFile struct{ Pos int }
	// RemoveDropped removes the placed copy of a file a final pass found
	// gone or changed at the source.
	RemoveDropped struct{ Pos int }
	// FinalPass compares the source with the unit's files once more.
	FinalPass struct{}
	// Snapshot saves each Jellyfin and Emby server's watch data of the
	// files before the switch.
	Snapshot struct{}
	// SwitchWork verifies the destination files and switches the Work's
	// rows in one Write.
	SwitchWork struct{}
	// RemoveSource removes one source file after the switch.
	RemoveSource struct{ Pos int }
	// ReleaseServers tells the Media Servers the source is gone at the
	// bound, no IO.
	ReleaseServers struct{}
	// FinishCleanup removes the emptied source folders and ends the unit.
	FinishCleanup struct{}
	// Undo removes what a canceled or failed unit left at the destination.
	Undo struct{}
	// Reconcile judges a restored unit against both roots once
	// Reconciling ends.
	Reconcile struct{}
	Nothing   struct{}
)

func (Hold) action()           {}
func (TakeLane) action()       {}
func (PlaceFile) action()      {}
func (RemoveDropped) action()  {}
func (FinalPass) action()      {}
func (Snapshot) action()       {}
func (SwitchWork) action()     {}
func (RemoveSource) action()   {}
func (ReleaseServers) action() {}
func (FinishCleanup) action()  {}
func (Undo) action()           {}
func (Reconcile) action()      {}
func (Nothing) action()        {}

// io is whether a runs a step on a Node, or in another owner outside a
// Write, before the Write that records it.
func io(a Action) bool {
	switch a.(type) {
	case Hold, ReleaseServers, Nothing:
		return false
	}
	return true
}

// next is the one rule of what a unit does now. It is pure: it reads the
// unit, the facts the pass read for it, the time and the rules. u.Files
// are loaded for every unit past its lane.
func next(u Unit, f Facts, now time.Time, r Rules) Action {
	if u.Reconcile {
		return Reconcile{}
	}
	switch s := u.Stage.(type) {
	case Queued:
		switch {
		case !f.Head:
			return Hold{Wait: WaitPairBusy}
		case !f.NodesUp:
			return Hold{Wait: WaitNodeOffline}
		}
		return TakeLane{}
	case Copying:
		return Hold{Wait: waitOfTransfer(f.Transfer)}
	case Placing:
		return nextPlace(u.Files)
	case Switching:
		return nextSwitch(s, f, now, r)
	case Cleaning:
		return nextCleanup(u, s, f, now, r)
	case Failed:
		if s.Undone == nil {
			return Undo{}
		}
	case Canceled:
		if s.Undone == nil {
			return Undo{}
		}
	}
	return Nothing{}
}

// nextPlace removes each dropped copy, then places each file, then runs
// the final pass.
func nextPlace(fs []File) Action {
	for _, file := range fs {
		if file.Dropped != nil {
			return RemoveDropped{Pos: file.Position}
		}
	}
	for _, file := range fs {
		if file.Placed == nil {
			return PlaceFile{Pos: file.Position}
		}
	}
	return FinalPass{}
}

// nextCleanup waits for the Media Servers to stop reading the source, at
// most PlexBound after the switch, then removes each source file.
func nextCleanup(u Unit, s Cleaning, f Facts, now time.Time, r Rules) Action {
	bound := s.SwitchedAt.Add(r.PlexBound)
	if !all(f.Safe) && now.Before(bound) {
		return Hold{Wait: WaitMediaServers, Until: bound}
	}
	if !now.Before(bound) && !u.Notes.Released {
		return ReleaseServers{}
	}
	for _, file := range u.Files {
		if file.Cleanup == CleanupPending {
			return RemoveSource{Pos: file.Position}
		}
	}
	return FinishCleanup{}
}

// nextSwitch judges the switch's waits in order: the first that holds is
// the unit's wait. Switch now passes playing and media_servers, never
// importing or open_files.
func nextSwitch(s Switching, f Facts, now time.Time, r Rules) Action {
	switch {
	case f.Importing:
		return Hold{Wait: WaitImporting}
	case f.OpenFiles:
		return Hold{Wait: WaitOpenFiles}
	case s.SwitchNow != nil:
		return SwitchWork{}
	case len(f.Playing) > 0:
		return Hold{Wait: WaitPlaying, Until: now.Add(r.PlayingEvery)}
	case !all(f.Snapshotted) && s.Wait == WaitMediaServers:
		return Hold{Wait: WaitMediaServers}
	case !all(f.Snapshotted):
		return Snapshot{}
	}
	return SwitchWork{}
}

func all(of map[link.ServerID]bool) bool {
	for _, ok := range of {
		if !ok {
			return false
		}
	}
	return true
}

// waitOfTransfer is the wait a copying unit mirrors from its Transfer's
// view: "" while bytes move.
func waitOfTransfer(v *transfers.View) Wait {
	if v == nil {
		return ""
	}
	switch v.State {
	case transfers.Queued:
		return WaitLowPriority
	case transfers.NeedsSpace:
		return WaitSpace
	case transfers.Waiting:
		switch v.Wait.Kind {
		case transfers.NoRoute, transfers.Connecting:
			return WaitNoRoute
		case transfers.NodeAway, transfers.NodeOtherBuild:
			return WaitNodeOffline
		case transfers.RootNotReady, transfers.StorageDraining, transfers.SourceChecking, transfers.SourceNotReady,
			transfers.FolderNotReady, transfers.Reconciling:
		}
	case transfers.Copying, transfers.Failed:
	}
	return ""
}
