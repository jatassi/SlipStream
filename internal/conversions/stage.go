package conversions

import "context"

// Fence is what an event must match on the row before its transition runs.
type Fence uint8

const (
	// FenceNone runs on any row in the From stages.
	FenceNone Fence = iota
	// FenceNode: the event's Node is the row's, or its pin's while queued.
	FenceNode
	// FenceAttempt: the event's Node and attempt are the row's.
	FenceAttempt
	// FenceBasis: the stage and attempt the event's snapshot read are the
	// row's.
	FenceBasis
)

// transition is one row of the stage table. To is where the row goes, or
// "" when the event changes nothing. Do runs in order once To is set.
type transition struct {
	From  []Stage
	On    On
	Fence Fence
	To    func(ctx context.Context, m *move) (Stage, error)
	Do    []effect
}

type effect func(ctx context.Context, m *move) error

var (
	placedStages = []Stage{StageCopying, StageReady, StageRunning, StagePaused, StageReturning}
	// faultStages are copying to returning, where a run or source fault
	// closes the attempt.
	faultStages = []Stage{StageCopying, StageReady, StageRunning, StagePaused, StageChecking, StageReturning}
	openStages  = []Stage{StageQueued, StageCopying, StageReady, StageRunning, StagePaused, StageChecking, StageReturning, StageLanding, StageFailed}
)

func to(s Stage) func(context.Context, *move) (Stage, error) {
	return func(context.Context, *move) (Stage, error) { return s, nil }
}

var transitions = []transition{
	{From: []Stage{StageQueued}, On: Placed, Fence: FenceBasis, To: placedTo, Do: []effect{place, wantSource}},
	{From: []Stage{StageQueued}, On: Fault, Fence: FenceBasis, To: unfitTo, Do: []effect{fail}},
	{From: []Stage{StageQueued}, On: Unpin, Fence: FenceNode, To: unpinTo, Do: []effect{unpinned}},
	{From: []Stage{StageCopying}, On: Copied, Fence: FenceNode, To: to(StageReady)},
	{From: []Stage{StageCopying}, On: CopyFailed, Fence: FenceNode, To: faultTo, Do: []effect{faulted}},
	{From: []Stage{StageReady}, On: Started, Fence: FenceAttempt, To: to(StageRunning), Do: []effect{started}},
	{From: []Stage{StageRunning}, On: Paused, Fence: FenceAttempt, To: to(StagePaused), Do: []effect{pausedBy}},
	{From: []Stage{StagePaused}, On: Resumed, Fence: FenceAttempt, To: to(StageRunning), Do: []effect{resumed}},
	{From: []Stage{StageRunning, StagePaused}, On: Preempted, Fence: FenceNode, To: to(StageQueued), Do: []effect{leave(AttemptPutBack)}},
	{From: []Stage{StageRunning}, On: Exited, Fence: FenceAttempt, To: exitedTo, Do: []effect{moveSpeed, returnOutput}},
	{From: []Stage{StageReturning}, On: Returned, Fence: FenceAttempt, To: to(StageChecking)},
	{From: []Stage{StageReturning}, On: ReturnFailed, Fence: FenceAttempt, To: faultTo, Do: []effect{faulted}},
	{From: []Stage{StageChecking}, On: Checked, Fence: FenceBasis, To: checkedTo, Do: []effect{faulted, handoff}},
	{From: faultStages, On: Fault, Fence: FenceAttempt, To: faultTo, Do: []effect{faulted}},
	{From: []Stage{StageRunning}, On: DiskFull, Fence: FenceAttempt, To: to(StageQueued), Do: []effect{leave(AttemptDiskFull)}},
	{From: []Stage{StageRunning, StagePaused}, On: Lost, Fence: FenceAttempt, To: to(StageReady), Do: []effect{relaunch}},
	{From: placedStages, On: PutBack, Fence: FenceNone, To: to(StageQueued), Do: []effect{leave(AttemptPutBack)}},
	{From: []Stage{StageLanding}, On: Landed, Fence: FenceNone, To: to(StageDone), Do: []effect{landed}},
	{From: []Stage{StageLanding}, On: Ended, Fence: FenceNone, To: to(StageCanceled), Do: []effect{leave(AttemptPutBack), canceled, declineIf(ImportDeclined)}},
	{From: openStages, On: Unneeded, Fence: FenceNone, To: unneededTo, Do: []effect{leave(AttemptPutBack), canceled}},
	{From: openStages, On: Canceled, Fence: FenceNone, To: canceledTo, Do: []effect{leave(AttemptPutBack), canceled, declineIf(CanceledByAdmin)}},
	{From: []Stage{StageFailed}, On: Retry, Fence: FenceNone, To: to(StageQueued), Do: []effect{retried}},
}

// rule is the transition ev takes from stage, if any.
func rule(stage Stage, on On) (transition, bool) {
	for _, t := range transitions {
		if t.On == on {
			for _, s := range t.From {
				if s == stage {
					return t, true
				}
			}
		}
	}
	return transition{}, false
}
