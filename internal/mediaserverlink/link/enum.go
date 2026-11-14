package link

import (
	"fmt"
	"slices"
)

// ChangeKind is the one list of Media Server change kinds. The queue, the
// schema's CHECK, the proto and the API all use it.
type ChangeKind string

const (
	ChangeAdded           ChangeKind = "added"
	ChangeReplaced        ChangeKind = "replaced"
	ChangeRemoved         ChangeKind = "removed"
	ChangeMoved           ChangeKind = "moved"
	ChangeSidecarAdded    ChangeKind = "sidecar_added"
	ChangeSidecarReplaced ChangeKind = "sidecar_replaced"
	ChangeSidecarRemoved  ChangeKind = "sidecar_removed"
	// ChangeLibrary is a whole library refresh: the Refresh button, and a
	// mapping confirmed or switched to shown.
	ChangeLibrary ChangeKind = "library"
)

func ChangeKinds() []ChangeKind {
	return []ChangeKind{ChangeAdded, ChangeReplaced, ChangeRemoved, ChangeMoved,
		ChangeSidecarAdded, ChangeSidecarReplaced, ChangeSidecarRemoved, ChangeLibrary}
}

func ParseChangeKind(s string) (ChangeKind, error) { return parse(s, ChangeKinds(), "change kind") }

// Brings is whether a change puts a file at its path: every kind but the
// removals and a whole library refresh.
func (k ChangeKind) Brings() bool {
	return k != ChangeRemoved && k != ChangeSidecarRemoved && k != ChangeLibrary
}

// Step is where a change's Job stopped. Steps only move forward.
type Step string

const (
	StepNone          Step = "none"
	StepSnapshotted   Step = "snapshotted"
	StepScanned       Step = "scanned"
	StepSourceScanned Step = "source_scanned"
	StepConfirmed     Step = "confirmed"
	StepIdle          Step = "idle"
	StepReported      Step = "reported"
	StepSettled       Step = "settled"
	StepOldRemoved    Step = "old_removed"
	StepNewAdded      Step = "new_added"
	StepNewFound      Step = "new_found"
	StepRestored      Step = "restored"
)

func Steps() []Step {
	return []Step{StepNone, StepSnapshotted, StepScanned, StepSourceScanned, StepConfirmed,
		StepIdle, StepReported, StepSettled, StepOldRemoved, StepNewAdded, StepNewFound, StepRestored}
}

func ParseStep(s string) (Step, error) { return parse(s, Steps(), "step") }

// Started is whether a step changed something on the server. A row at a
// started step no longer folds: a new change waits behind it.
func (s Step) Started() bool { return s != StepNone && s != StepSnapshotted }

// Result is how a sent change ended.
type Result string

const (
	ResultAccepted Result = "accepted"
	// ResultCovered is a change a whole library refresh sent.
	ResultCovered Result = "covered"
	// ResultConverted is a carry whose old path held no item, sent as
	// added.
	ResultConverted Result = "converted"
)

func Results() []Result { return []Result{ResultAccepted, ResultCovered, ResultConverted} }

func ParseResult(s string) (Result, error) { return parse(s, Results(), "result") }

// Loss is what a carry couldn't keep.
type Loss string

const (
	LostWatchState  Loss = "watch_state"
	LostPlayCount   Loss = "play_count"
	LostLastPlayed  Loss = "last_played"
	LostRating      Loss = "rating"
	LostCollections Loss = "collections"
)

func Losses() []Loss {
	return []Loss{LostWatchState, LostPlayCount, LostLastPlayed, LostRating, LostCollections}
}

func ParseLoss(s string) (Loss, error) { return parse(s, Losses(), "loss") }

// Decision is how a server plays a session.
type Decision string

const (
	DecisionDirectPlay Decision = "direct_play"
	DecisionCopy       Decision = "copy"
	DecisionTranscode  Decision = "transcode"
)

func Decisions() []Decision { return []Decision{DecisionDirectPlay, DecisionCopy, DecisionTranscode} }

func ParseDecision(s string) (Decision, error) { return parse(s, Decisions(), "decision") }

// Contact is a server as its reaching Node last reported it. Unknown is
// stored until the first report and never reported.
type Contact string

const (
	ContactUnknown       Contact = "unknown"
	ContactResponding    Contact = "responding"
	ContactNotResponding Contact = "not_responding"
	ContactSignInRefused Contact = "sign_in_refused"
	ContactOtherServer   Contact = "other_server"
)

func Contacts() []Contact {
	return []Contact{ContactUnknown, ContactResponding, ContactNotResponding, ContactSignInRefused, ContactOtherServer}
}

func ParseContact(s string) (Contact, error) { return parse(s, Contacts(), "contact") }

func parse[T ~string](s string, all []T, what string) (T, error) {
	if i := slices.Index(all, T(s)); i >= 0 {
		return all[i], nil
	}
	return "", fmt.Errorf("%q isn't a Media Server %s", s, what)
}
