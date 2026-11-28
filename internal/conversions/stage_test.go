package conversions

import (
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/conversions/internal/plan"
	"github.com/jatassi/slipstream/internal/encode"
	"github.com/jatassi/slipstream/internal/history/entry/entrytest"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/role"
)

// TestTheStageTable moves a Conversion through each row of the design's
// stage table, through advance on a real library, and reads it back.
func TestTheStageTable(t *testing.T) {
	l := newLife(t)
	attic, den := l.attic, l.den
	proof := &plan.Proof{Pipeline: encode.Pipeline{Path: encode.Hardware, API: encode.QSV}, Speed: 4}
	probe := &media.Record{Version: media.Version, Container: "matroska,webm", Duration: 2 * time.Hour, Video: media.Video{Codec: "h264", Width: 1280, Height: 720}}
	on := func(s Stage) seed { return seed{Stage: s, Node: &attic, Attempt: 1} }
	onDen := func(s Stage) seed { return seed{Stage: s, Node: &den, Attempt: 1} }
	made, _ := l.file(t)
	l.key = l.prove(t, attic)
	l.prove(t, den)
	cases := []struct {
		name    string
		seed    seed
		ev      func(Row) Event
		placing bool
		want    Stage
		check   check
	}{
		{name: "placed with its source on the Node", seed: seed{Stage: StageQueued}, ev: placed(attic, proof, false), want: StageReady,
			check: expect("placed at attempt 1 on Attic in its root, with one open attempt on the proof's pipeline", func(t *testing.T, _, a Row) bool {
				as := l.attempts(t, a.ID)
				return a.Attempt == 1 && *a.Node == attic && *a.Root == l.root && len(as) == 1 && as[0].Pipeline == proof.Pipeline && as[0].Ended == nil
			})},
		{name: "placed with its source elsewhere", seed: seed{Stage: StageQueued}, ev: placed(den, proof, true), want: StageCopying,
			check: expect("a source copy to Den", func(_ *testing.T, _, a Row) bool { return l.transfers.holding(sourceJob(den, *a.SourceFile)) })},
		{name: "placed before its download wait ends", seed: seed{Stage: StageQueued, NotBefore: l.now.Add(time.Hour)}, ev: placed(attic, proof, false), want: StageQueued},
		{name: "a source fault at placement", seed: seed{Stage: StageQueued},
			ev:   func(Row) Event { return Event{On: Fault, Code: CodeContainerAudio, Basis: Basis{Stage: StageQueued}} },
			want: StageFailed, check: l.failedWith(CodeContainerAudio, SourceClass)},
		{name: "a pinned retry unpinned", seed: seed{Stage: StageQueued, Pin: &attic, Failed: []role.NodeID{attic}},
			ev: func(Row) Event { return Event{On: Unpin, Node: attic} }, want: StageQueued,
			check: expect("no pin and Attic excluded", func(_ *testing.T, _, a Row) bool {
				return a.Pin == nil && slices.Equal(a.Exclude, []role.NodeID{attic})
			})},
		{name: "copied", seed: onDen(StageCopying), ev: func(Row) Event { return Event{On: Copied, Node: den} }, want: StageReady},
		{name: "the copy failed", seed: onDen(StageCopying), ev: func(Row) Event { return Event{On: CopyFailed, Node: den} },
			want: StageQueued, check: l.pinnedTo(den, CodeCopyFailed)},
		{name: "started", seed: on(StageReady), ev: report(Started, attic), want: StageRunning,
			check: expect("started now with the Target's version", func(_ *testing.T, _, a Row) bool {
				return a.StartedAt != nil && a.StartedAt.Equal(l.now) && a.Version == l.version
			})},
		{name: "paused", seed: on(StageRunning), ev: report(Paused, attic), want: StagePaused},
		{name: "resumed", seed: on(StagePaused), ev: report(Resumed, attic), want: StageRunning},
		{name: "preempted", seed: on(StageRunning), ev: func(Row) Event { return Event{On: Preempted, Node: attic} },
			want: StageQueued, check: l.putBackWith(AttemptPutBack)},
		{name: "exited in its root's Node", seed: on(StageRunning), ev: report(Exited, attic), want: StageChecking},
		{name: "exited on another Node", seed: onDen(StageRunning), ev: report(Exited, den), want: StageReturning,
			check: expect("an output Transfer to the root, and the source copy ended", func(_ *testing.T, _, a Row) bool {
				job := sourceJob(den, *a.SourceFile)
				return l.transfers.holding(outputJob(a.ID, 1)) && l.transfers.asked(job) && !l.transfers.holding(job)
			})},
		{name: "returned", seed: onDen(StageReturning), ev: report(Returned, den), want: StageChecking},
		{name: "the return failed", seed: onDen(StageReturning), ev: report(ReturnFailed, den), want: StageQueued,
			check: both(l.pinnedTo(den, CodeReturnFailed), expect("the output Transfer ended", func(_ *testing.T, _, a Row) bool { return !l.transfers.holding(outputJob(a.ID, 1)) }))},
		{name: "checked and passed", seed: on(StageChecking), ev: checked(probe, ""), want: StageLanding,
			check: expect("handed off in the root's holding folder, named after its source and Target", func(_ *testing.T, _, a Row) bool {
				h := l.importer.started[len(l.importer.started)-1]
				return h.Conversion == a.ID && h.Holding != nil && h.Sent == nil && h.SourceName == "Heat.1995.2160p.UHD.BluRay - 720p mobile.mkv"
			})},
		{name: "checked short", seed: on(StageChecking), ev: checked(probe, CodeShortOutput), want: StageQueued, check: l.pinnedTo(attic, CodeShortOutput)},
		{name: "checked and refused", seed: on(StageChecking), ev: checked(probe, CodeRefused), want: StageFailed, check: l.failedWith(CodeRefused, SourceClass)},
		{name: "a run fault", seed: on(StageRunning), ev: fault(CodeEncoder, attic), want: StageQueued, check: l.pinnedTo(attic, CodeEncoder)},
		{name: "a source fault", seed: on(StageRunning), ev: fault(CodeUnreadable, attic), want: StageFailed, check: l.failedWith(CodeUnreadable, SourceClass)},
		{name: "a run fault on a second Node", seed: seed{Stage: StageRunning, Node: &den, Attempt: 2, Failed: []role.NodeID{attic}},
			ev: fault(CodeStalled, den), want: StageFailed, check: l.failedWith(CodeStalled, RunClass)},
		{name: "a run fault after a Retry", seed: seed{Stage: StageRunning, Node: &den, Attempt: 3, Failed: []role.NodeID{attic, den}, BudgetFrom: 3},
			ev: fault(CodeEncoder, den), want: StageQueued, check: l.pinnedTo(den, CodeEncoder)},
		{name: "the disk filled", seed: on(StageRunning), ev: report(DiskFull, attic), want: StageQueued, check: l.putBackWith(AttemptDiskFull)},
		{name: "lost in a restart", seed: on(StagePaused), ev: report(Lost, attic), want: StageReady,
			check: expect("attempt 2 open on Attic after 1 put back", func(t *testing.T, _, a Row) bool {
				as := l.attempts(t, a.ID)
				return a.Attempt == 2 && *a.Node == attic && len(as) == 2 && as[0].Outcome == AttemptPutBack && as[1].Node == attic && as[1].Ended == nil
			})},
		{name: "put back", seed: onDen(StageCopying), ev: func(Row) Event { return Event{On: PutBack} }, want: StageQueued,
			check: both(l.putBackWith(AttemptPutBack), expect("the source copy to Den ended", func(_ *testing.T, _, a Row) bool {
				job := sourceJob(den, *a.SourceFile)
				return l.transfers.asked(job) && !l.transfers.holding(job)
			}))},
		{name: "landed", seed: on(StageLanding), ev: func(Row) Event { return Event{On: Landed, File: made} }, want: StageDone,
			check: expect("finished with the made file", func(_ *testing.T, _, a Row) bool { return a.Output != nil && *a.Output == made && a.FinishedAt != nil })},
		{name: "its Import declined", seed: on(StageLanding), ev: func(Row) Event { return Event{On: Ended, Why: ImportDeclined} },
			want: StageCanceled, check: l.canceledWith(ImportDeclined, true)},
		{name: "its Import ended", seed: on(StageLanding), ev: func(Row) Event { return Event{On: Ended, Why: ImportEnded} },
			want: StageCanceled, check: l.canceledWith(ImportEnded, false)},
		{name: "no longer needed", seed: on(StageRunning), ev: func(Row) Event { return Event{On: Unneeded, Why: NotWanted} },
			want: StageCanceled, check: l.canceledWith(NotWanted, false)},
		{name: "no longer needed while its file places", seed: on(StageLanding), placing: true,
			ev: func(Row) Event { return Event{On: Unneeded, Why: SourceGone} }, want: StageLanding},
		{name: "canceled while landing", seed: on(StageLanding), ev: func(Row) Event { return Event{On: Canceled} }, want: StageCanceled,
			check: both(l.canceledWith(CanceledByAdmin, true), expect("its Import ended", func(_ *testing.T, _, a Row) bool { return slices.Contains(l.importer.ended, a.ID) }))},
		{name: "a failed one canceled", seed: seed{Stage: StageFailed, Attempt: 1}, ev: func(Row) Event { return Event{On: Canceled} },
			want: StageCanceled, check: l.canceledWith(CanceledByAdmin, true)},
		{name: "retried", seed: seed{Stage: StageFailed, Attempt: 2, Failed: []role.NodeID{attic}}, ev: func(Row) Event { return Event{On: Retry} },
			want: StageQueued, check: expect("budget from 3, no failure, at the head of its lane", func(_ *testing.T, b, a Row) bool {
				return a.BudgetFrom == 3 && a.Failure == nil && a.Position < b.Position
			})},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			l.importer.placing = tc.placing
			before := l.seed(t, tc.seed)
			if err := l.advance(t, before.ID, tc.ev(before)); err != nil {
				t.Fatal(err)
			}
			after := l.read(t, before.ID)
			if after.Stage != tc.want {
				t.Fatalf("%s, want %s", after.Stage, tc.want)
			}
			if tc.check != nil {
				tc.check(t, before, after)
			}
		})
	}
}

func placed(node role.NodeID, p *plan.Proof, copied bool) func(Row) Event {
	return func(Row) Event {
		return Event{On: Placed, Node: node, Proof: p, Copy: copied, Basis: Basis{Stage: StageQueued}}
	}
}

func checked(probe *media.Record, code Code) func(Row) Event {
	return func(r Row) Event {
		return Event{On: Checked, Probe: probe, Verdict: Verdict{Code: code}, Basis: Basis{r.Stage, r.Attempt}}
	}
}

// expect checks what ok says of the row, before and after.
func expect(what string, ok func(t *testing.T, before, after Row) bool) check {
	return func(t *testing.T, before, after Row) {
		t.Helper()
		if !ok(t, before, after) {
			t.Errorf("%+v, want %s", after, what)
		}
	}
}

func both(a, b check) check {
	return func(t *testing.T, before, after Row) {
		t.Helper()
		a(t, before, after)
		b(t, before, after)
	}
}

func report(on On, node role.NodeID) func(Row) Event {
	return func(r Row) Event { return Event{On: on, Node: node, Attempt: r.Attempt, Fenced: true} }
}

func fault(code Code, node role.NodeID) func(Row) Event {
	return func(r Row) Event { return Event{On: Fault, Code: code, Node: node, Attempt: r.Attempt, Fenced: true} }
}

type check = func(t *testing.T, before, after Row)

// failedWith: the Conversion failed with code, its attempt closed, and
// Conversion failed recorded for it.
func (l *life) failedWith(code Code, class Class) check {
	return func(t *testing.T, _, a Row) {
		t.Helper()
		if a.Failure == nil || a.Failure.Code != code || a.Failure.Class != class || a.Node != nil {
			t.Errorf("failure %+v on %v, want %s %s off its Node", a.Failure, a.Node, code, class)
		}
		for _, at := range l.attempts(t, a.ID) {
			if at.Ended == nil {
				t.Errorf("attempt %d is still open", at.N)
			}
		}
		recorded := 0
		for _, f := range entrytest.Of(l.history, ConversionFailed) {
			if f.Work.ID == a.Work && f.Code == code {
				recorded++
			}
		}
		if recorded == 0 {
			t.Error("no Conversion failed recorded")
		}
	}
}

// pinnedTo: the fault closed the attempt with code and queued the retry
// on node, which lost its proof when the code costs one.
func (l *life) pinnedTo(node role.NodeID, code Code) check {
	return func(t *testing.T, _, a Row) {
		t.Helper()
		if a.Pin == nil || *a.Pin != node || a.Node != nil {
			t.Errorf("pinned to %v on %v, want pinned to %v and off its Node", a.Pin, a.Node, node)
		}
		as := l.attempts(t, a.ID)
		if last := as[len(as)-1]; last.Code != code || last.Class != RunClass || last.Ended == nil {
			t.Errorf("last attempt %+v, want it closed with %s", last, code)
		}
		if lost := l.proofLost(t, node, l.key); lost != faults[code].Costs {
			t.Errorf("proof lost %v, want %v", lost, faults[code].Costs)
		}
		l.prove(t, node)
	}
}

// putBackWith: queued off its Node, its attempt closed with o.
func (l *life) putBackWith(o Outcome) check {
	return func(t *testing.T, _, a Row) {
		t.Helper()
		as := l.attempts(t, a.ID)
		if a.Node != nil || a.Pin != nil || as[len(as)-1].Outcome != o {
			t.Errorf("on %v pinned to %v with attempts %+v, want queued off its Node with %s", a.Node, a.Pin, as, o)
		}
	}
}

// canceledWith: canceled for why, and its pair declined when declined.
func (l *life) canceledWith(why CancelWhy, declined bool) check {
	return func(t *testing.T, _, a Row) {
		t.Helper()
		if a.CancelWhy != why || a.FinishedAt == nil {
			t.Errorf("canceled for %q finished %v, want %q, finished", a.CancelWhy, a.FinishedAt, why)
		}
		if l.declined(t, a) != declined {
			t.Errorf("declined %v, want %v", !declined, declined)
		}
	}
}
