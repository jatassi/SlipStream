package downloadclients

import "slices"

// State is what the Admin sees, in order of precedence.
type State string

const (
	StateRemoving    State = "removing"
	StateFailed      State = "failed"
	StateTrackerErr  State = "tracker-error"
	StatePaused      State = "paused"
	StateChecking    State = "checking"
	StateQueued      State = "queued"
	StateStalled     State = "stalled"
	StateDownloading State = "downloading"
	StateSeeding     State = "seeding"
)

// States is every State in order of precedence.
var States = []State{StateRemoving, StateFailed, StateTrackerErr, StatePaused, StateChecking, StateQueued, StateStalled, StateDownloading, StateSeeding}

// Facts is everything StateOf reads.
type Facts struct {
	// Removing is a Removal not yet Parted.
	Removing bool
	Wanted   Wanted
	// Report is nil before the Node's first.
	Report *Report
}

// StateOf is the one place a Torrent's state word is decided. An offline
// Node doesn't change it: the word stands on the last report.
func StateOf(f Facts) State {
	r := f.Report
	switch {
	case f.Removing:
		return StateRemoving
	case r != nil && r.Engine == EngineFailed:
		return StateFailed
	case r != nil && trackerErr(r.Trackers):
		return StateTrackerErr
	case f.Wanted == Paused || r != nil && slices.Contains(pausedHolds, r.Hold):
		return StatePaused
	case r == nil:
		return StateQueued
	case r.Engine == EngineChecking:
		return StateChecking
	case slices.Contains(queuedHolds, r.Hold):
		return StateQueued
	case r.Stalled:
		return StateStalled
	case !r.Complete:
		return StateDownloading
	}
	return StateSeeding
}

var (
	pausedHolds = []Hold{HoldWanted, HoldWindow, HoldDiskFull}
	queuedHolds = []Hold{HoldSlot, HoldCheckSlot, HoldDiskGuard, HoldDraining, HoldController}
)

// trackerErr is some line failing while none works: per BEP 12 a working
// tier serves the torrent.
func trackerErr(lines []TrackerLine) bool {
	failing := false
	for _, l := range lines {
		switch l.Status {
		case TrackerWorking:
			return false
		case TrackerError, TrackerUnregistered:
			failing = true
		case TrackerUpdating, TrackerNotContacted:
		}
	}
	return failing
}

// Tab is a set of the Torrents view's tabs.
type Tab uint8

const (
	TabDownloading Tab = 1 << iota
	TabSeeding
	TabProblems
	TabAll
)

var (
	downloadingTab = []State{StateDownloading, StateStalled, StateQueued, StateChecking, StatePaused}
	seedingTab     = []State{StateSeeding, StateChecking, StatePaused}
)

func TabsOf(f Facts, s State) Tab {
	t := TabAll
	complete := f.Report != nil && f.Report.Complete
	if !complete && slices.Contains(downloadingTab, s) {
		t |= TabDownloading
	}
	if complete && slices.Contains(seedingTab, s) {
		t |= TabSeeding
	}
	if SeverityOf(f, s) != SeverityNone {
		t |= TabProblems
	}
	return t
}

type Severity uint8

const (
	SeverityNone Severity = iota
	SeverityWarning
	SeverityFailure
)

func SeverityOf(f Facts, s State) Severity {
	switch s {
	case StateFailed:
		return SeverityFailure
	case StateTrackerErr:
		if slices.ContainsFunc(f.Report.Trackers, func(l TrackerLine) bool { return l.Status == TrackerUnregistered }) {
			return SeverityFailure
		}
		return SeverityWarning
	case StateStalled:
		return SeverityWarning
	case StatePaused, StateQueued:
		if f.Report != nil && (f.Report.Hold == HoldDiskFull || f.Report.Hold == HoldDiskGuard) {
			return SeverityWarning
		}
	case StateRemoving, StateChecking, StateDownloading, StateSeeding:
	}
	return SeverityNone
}
