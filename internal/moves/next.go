package moves

import (
	"time"

	"github.com/jatassi/slipstream/internal/downloadclients"
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
	// a Transfer, Speech Map or Sidecar write on its files. A torrent
	// unit's Importing is an Import of a Grab of its source, read at its
	// guard.
	Importing, OpenFiles bool
	// Playing is the servers someone plays its files on now.
	Playing []link.ServerID
	// Snapshotted is each server that snapshots first and shows the
	// source root, and whether it holds a fresh snapshot of every file.
	Snapshotted map[link.ServerID]bool
	// Safe is each Plex server that showed the source root, and whether it
	// found every file at its new path.
	Safe map[link.ServerID]bool
	// Handoff is a torrent unit's source and destination Torrents, read
	// once it is placing.
	Handoff downloadclients.Handoff
}

// Rules are the settings and timings next reads.
type Rules struct {
	PlexBound    time.Duration
	PlayingEvery time.Duration
	// Guard to DownloadingEvery are a torrent handoff's Timings.
	Guard, ReplyWait, StopBound, HandoffBound, DownloadingEvery time.Duration
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
	// PlaceTorrent puts a torrent's payload in its destination client's
	// folder, from Transfer's staging when it copied by one, then adds its
	// arriving Torrent.
	PlaceTorrent struct{ Transfer *transfers.ID }
	// Arrived starts the handoff of a torrent its external destination
	// checked complete, no IO.
	Arrived struct{}
	// Stop pauses the source once the guard holds, no IO.
	Stop struct{}
	// Resume has a source whose pause went unseen seed again and the
	// handoff wait out the guard again, no IO.
	Resume struct{}
	// AskStart asks the destination to start inside its window, no IO.
	AskStart struct{}
	// SwitchTorrent hands the torrent to its destination in one Write,
	// no IO.
	SwitchTorrent struct{}
	// Rollback ends a torrent unit with Code, its source seeding again,
	// no IO.
	Rollback struct {
		Code   FailureCode
		Detail string
	}
	// Parted ends a torrent unit switched at SwitchedAt once its source
	// left its Node, no IO.
	Parted  struct{ SwitchedAt time.Time }
	Nothing struct{}
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
func (PlaceTorrent) action()   {}
func (Arrived) action()        {}
func (Stop) action()           {}
func (Resume) action()         {}
func (AskStart) action()       {}
func (SwitchTorrent) action()  {}
func (Rollback) action()       {}
func (Parted) action()         {}
func (Nothing) action()        {}

// io is whether a runs a step on a Node, or in another owner outside a
// Write, before the Write that records it.
func io(a Action) bool {
	switch a.(type) {
	case Hold, ReleaseServers, Arrived, Stop, Resume, AskStart, SwitchTorrent, Rollback, Parted, Nothing:
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
	if u.Torrent != nil {
		return nextTorrent(u, f, now, r)
	}
	switch s := u.Stage.(type) {
	case Queued:
		switch {
		case s.Wait == WaitLinkedTorrent:
			return Hold{Wait: WaitLinkedTorrent}
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

// nextTorrent is next for a torrent unit: its copy and Place, then the
// handoff, then its source's part.
func nextTorrent(u Unit, f Facts, now time.Time, r Rules) Action {
	h := f.Handoff
	switch s := u.Stage.(type) {
	case Queued:
		switch {
		case !h.Removed && !h.Complete:
			return Hold{Wait: WaitDownloading, Until: now.Add(r.DownloadingEvery)}
		case !f.Head:
			return Hold{Wait: WaitPairBusy}
		case !f.NodesUp:
			return Hold{Wait: WaitNodeOffline}
		}
		return TakeLane{}
	case Copying:
		return Hold{Wait: waitOfTransfer(f.Transfer)}
	case Placing:
		switch {
		case u.Torrent.Arriving == nil:
			return PlaceTorrent{Transfer: s.Transfer}
		case h.Removed:
			return Rollback{Code: SourceGone}
		case h.Gone:
			return Rollback{Code: DestinationGone}
		case h.Arrival != nil && *h.Arrival == downloadclients.ArrivalComplete:
			return Arrived{}
		case h.Arrival != nil && *h.Arrival == downloadclients.ArrivalCheckFailed:
			return Rollback{Code: DestinationCheckFailed, Detail: h.ArrivalWhy}
		}
		return Hold{Wait: WaitDownloading, Until: now.Add(r.DownloadingEvery)}
	case Switching:
		return nextHandoff(*s.Handoff, f, now, r)
	case Cleaning:
		if h.Parted {
			return Parted{SwitchedAt: s.SwitchedAt}
		}
		return Hold{Wait: WaitDownloading, Until: now.Add(r.DownloadingEvery)}
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

// nextHandoff is a placed torrent's switch. The guard waits until no
// Import reads the source. A source paused by the Admin or a goal then
// switches at once and its destination stays paused. Otherwise the guard
// waits until the source's trackers last answered Guard ago and aren't
// due within ReplyWait, then the source stops, then the destination
// starts. A stop unseen in StopBound resumes the source; a start
// unconfirmed in HandoffBound, or refused, rolls back.
func nextHandoff(at HandoffAt, f Facts, now time.Time, r Rules) Action {
	h := f.Handoff
	switch {
	case h.Removed:
		return Rollback{Code: SourceGone}
	case h.Gone:
		return Rollback{Code: DestinationGone}
	}
	switch at.Step {
	case HandoffGuard:
		return nextGuard(h, f, now, r)
	case HandoffStopping:
		bound := at.At.Add(r.StopBound)
		switch {
		case h.PausedAt != nil:
			return AskStart{}
		case !now.Before(bound):
			return Resume{}
		}
		return Hold{Wait: WaitGuard, Until: bound}
	case HandoffStarting:
	}
	bound := at.At.Add(r.HandoffBound)
	switch {
	case h.Arrival != nil && *h.Arrival == downloadclients.ArrivalStarted:
		return SwitchTorrent{}
	case h.Arrival != nil && *h.Arrival == downloadclients.ArrivalRefused:
		return Rollback{Code: SwitchRefused, Detail: h.ArrivalWhy}
	case !now.Before(bound):
		return Rollback{Code: DestinationUnreachable}
	}
	return Hold{Wait: WaitGuard, Until: bound}
}

func nextGuard(h downloadclients.Handoff, f Facts, now time.Time, r Rules) Action {
	switch {
	case f.Importing:
		return Hold{Wait: WaitImporting}
	case h.Wanted == downloadclients.Paused:
		return SwitchTorrent{}
	case !f.NodesUp:
		return Hold{Wait: WaitNodeOffline}
	case h.AnnouncedAt == nil:
		return Hold{Wait: WaitGuard, Until: now.Add(r.Guard)}
	}
	if ends := h.AnnouncedAt.Add(r.Guard); now.Before(ends) {
		return Hold{Wait: WaitGuard, Until: ends}
	}
	if h.AnnounceDueAt != nil && h.AnnounceDueAt.Sub(now) <= r.ReplyWait {
		return Hold{Wait: WaitGuard, Until: h.AnnounceDueAt.Add(r.Guard)}
	}
	return Stop{}
}
