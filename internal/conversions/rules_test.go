package conversions

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/history/entry/entrytest"
	cv1 "github.com/jatassi/slipstream/internal/pb/slipstream/converting/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

var allStages = []Stage{StageQueued, StageCopying, StageReady, StageRunning, StagePaused, StageChecking, StageReturning, StageLanding, StageDone, StageFailed, StageCanceled}

var allOns = []On{Placed, Unpin, Copied, CopyFailed, Started, Paused, Resumed, Preempted, Exited, Returned, ReturnFailed, Checked,
	Fault, DiskFull, Lost, PutBack, Landed, Ended, Unneeded, Canceled, Retry}

// TestEveryStageAndEventHasARule: every event moves some stage, every
// stage but done and canceled has a way out, no (stage, event) pair has
// two rules, and done and canceled have none.
func TestEveryStageAndEventHasARule(t *testing.T) {
	for _, on := range allOns {
		if !slices.ContainsFunc(transitions, func(tr transition) bool { return tr.On == on }) {
			t.Errorf("no rule takes %s", on)
		}
	}
	for _, s := range allStages {
		outs := rulesFrom(t, s)
		if terminal := s == StageDone || s == StageCanceled; terminal != (len(outs) == 0) {
			t.Errorf("%s leaves by %v", s, outs)
		}
	}
}

// rulesFrom is the events with a rule from s, failing t for a pair with
// two.
func rulesFrom(t *testing.T, s Stage) []On {
	t.Helper()
	var outs []On
	for _, on := range allOns {
		n := 0
		for _, tr := range transitions {
			if tr.On == on && slices.Contains(tr.From, s) {
				n++
			}
		}
		if n > 1 {
			t.Errorf("%s has %d rules for %s", s, n, on)
		}
		if n == 1 {
			outs = append(outs, on)
		}
	}
	return outs
}

// TestEveryCodeHasAClass: every fault code has a class, and every run and
// source code has the words its failure records.
func TestEveryCodeHasAClass(t *testing.T) {
	codes := []Code{CodeEncoder, CodeFilter, CodeDevice, CodeCrash, CodeStalled, CodeShortOutput, CodeStreams, CodeCopyFailed,
		CodeReturnFailed, CodeUnreadable, CodeContainerAudio, CodeRefused, CodeDiskFull}
	if len(faults) != len(codes) {
		t.Errorf("faults has %d codes, want %d", len(faults), len(codes))
	}
	s := Snapshot{Work: "Heat", Target: "720p mobile"}
	for _, c := range codes {
		rule, ok := faults[c]
		if !ok || !slices.Contains([]Class{RunClass, SourceClass, PutBackClass}, rule.Class) {
			t.Errorf("%s has no class", c)
			continue
		}
		if rule.Class == PutBackClass {
			continue
		}
		if words := failedWords(s, c, rule.Class, []string{"Attic"}, ""); !strings.Contains(words, "Heat") || strings.Contains(words, "%!") || !strings.HasSuffix(words, ".") {
			t.Errorf("%s reads %q", c, words)
		}
	}
}

// TestRetryOf is decision 8 over the attempts a fault counts: once more
// on the same Node, then once on another, and fail after two Nodes, or
// after two on one when no other can run it. A Retry's budget counts
// from its next attempt.
func TestRetryOf(t *testing.T) {
	a, b := role.NodeID{UUID: storage.NewUUID()}, role.NodeID{UUID: storage.NewUUID()}
	run := func(n int, on role.NodeID) Attempt { return Attempt{N: n, Node: on, Class: RunClass} }
	putBack := Attempt{N: 2, Node: b, Outcome: AttemptPutBack}
	cases := []struct {
		name       string
		attempts   []Attempt
		budgetFrom int
		others     bool
		want       RetryPlan
	}{
		{"the first fault", []Attempt{run(1, a)}, 1, true, RetryPlan{Pin: &a}},
		{"twice on one Node with another", []Attempt{run(1, a), run(2, a)}, 1, true, RetryPlan{Exclude: []role.NodeID{a}}},
		{"twice on the only Node", []Attempt{run(1, a), run(2, a)}, 1, false, RetryPlan{Fail: true, Why: NoOtherNode}},
		{"on two Nodes", []Attempt{run(1, a), run(2, b)}, 1, true, RetryPlan{Fail: true, Why: FailedOnTwo}},
		{"a put back doesn't count", []Attempt{run(1, a), putBack, run(3, a)}, 1, true, RetryPlan{Exclude: []role.NodeID{a}}},
		{"after a Retry", []Attempt{run(1, a), run(2, b), run(3, b)}, 3, true, RetryPlan{Pin: &b}},
		{"twice after a Retry", []Attempt{run(1, a), run(2, b), run(3, b), run(4, a)}, 3, true, RetryPlan{Fail: true, Why: FailedOnTwo}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := retryOf(counted(tc.attempts, tc.budgetFrom), tc.others)
			if got.Fail != tc.want.Fail || got.Why != tc.want.Why || !slices.Equal(got.Exclude, tc.want.Exclude) ||
				(got.Pin == nil) != (tc.want.Pin == nil) || (got.Pin != nil && *got.Pin != *tc.want.Pin) {
				t.Errorf("%+v, want %+v", got, tc.want)
			}
		})
	}
}

// TestAFencedReportFromAnotherNodeOrAttemptChangesNothing applies a
// holder's reports to a Conversion running on Attic at attempt 2.
func TestAFencedReportFromAnotherNodeOrAttemptChangesNothing(t *testing.T) {
	l := newLife(t)
	l.key = l.prove(t, l.attic)
	cases := []struct {
		name    string
		from    role.NodeID
		attempt uint32
		moves   bool
	}{
		{"from another Node", l.den, 2, false},
		{"for an older attempt", l.attic, 1, false},
		{"for a later attempt", l.attic, 3, false},
		{"from its Node at its attempt", l.attic, 2, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := l.seed(t, seed{Stage: StageRunning, Node: &l.attic, Attempt: 2, Failed: []role.NodeID{l.den}})
			res := &cv1.Result{ConversionId: r.ID.Bytes(), Attempt: tc.attempt, Result: &cv1.Result_Fault{Fault: &cv1.Fault{Code: "encoder"}}}
			if err := l.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
				return l.c.apply(ctx, tx, tc.from, res)
			}); err != nil {
				t.Fatal(err)
			}
			after := l.read(t, r.ID)
			if moved := after.Stage != StageRunning; moved != tc.moves {
				t.Fatalf("stage %s, want moved %v", after.Stage, tc.moves)
			}
			if lost := l.proofLost(t, l.attic, l.key); lost != tc.moves {
				t.Errorf("proof lost %v, want %v", lost, tc.moves)
			}
			l.prove(t, l.attic)
		})
	}
}

// TestConversionFailedIsRecordedOnceForADoubledReport: a fault that
// fails a Conversion, reported twice, records one entry.
func TestConversionFailedIsRecordedOnceForADoubledReport(t *testing.T) {
	l := newLife(t)
	l.prove(t, l.den)
	for _, tc := range []struct {
		name    string
		code    string
		attempt uint32
		seed    seed
	}{
		{"a source fault", "unreadable", 1, seed{Stage: StageRunning, Node: &l.attic, Attempt: 1}},
		{"a run fault on a second Node", "encoder", 2, seed{Stage: StageRunning, Node: &l.den, Attempt: 2, Failed: []role.NodeID{l.attic}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := l.seed(t, tc.seed)
			res := &cv1.Result{ConversionId: r.ID.Bytes(), Attempt: tc.attempt, Result: &cv1.Result_Fault{Fault: &cv1.Fault{Code: tc.code}}}
			for range 2 {
				if err := l.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
					return l.c.apply(ctx, tx, *r.Node, res)
				}); err != nil {
					t.Fatal(err)
				}
			}
			n := 0
			for _, f := range entrytest.Of(l.history, ConversionFailed) {
				if string(f.Code) == tc.code {
					n++
				}
			}
			if n != 1 {
				t.Errorf("recorded %d times, want once", n)
			}
		})
	}
}

// TestReportsBeyondTheirCaps: an unknown code reads as crash, disk_full
// as a put back, an output past 2^50 bytes and a report with no result
// are refused, and a tail keeps its last 2 KiB.
func TestReportsBeyondTheirCaps(t *testing.T) {
	id := storage.NewUUID().Bytes()
	long := strings.Repeat("é", 2000) + "end"
	cases := []struct {
		name    string
		res     *cv1.Result
		on      On
		code    Code
		refused bool
	}{
		{"an unknown code", &cv1.Result{ConversionId: id, Result: &cv1.Result_Fault{Fault: &cv1.Fault{Code: "gremlins"}}}, Fault, CodeCrash, false},
		{"disk full as a fault", &cv1.Result{ConversionId: id, Result: &cv1.Result_Fault{Fault: &cv1.Fault{Code: "disk_full"}}}, DiskFull, "", false},
		{"a long tail", &cv1.Result{ConversionId: id, Result: &cv1.Result_Fault{Fault: &cv1.Fault{Code: "encoder", Tail: long}}}, Fault, CodeEncoder, false},
		{"an output too large", &cv1.Result{ConversionId: id, Result: &cv1.Result_Exited{Exited: &cv1.Exited{Bytes: 1<<50 + 1}}}, "", "", true},
		{"a negative output", &cv1.Result{ConversionId: id, Result: &cv1.Result_Exited{Exited: &cv1.Exited{Bytes: -1}}}, "", "", true},
		{"no result", &cv1.Result{ConversionId: id}, "", "", true},
		{"no Conversion", &cv1.Result{ConversionId: []byte{1}, Result: &cv1.Result_Started{Started: &cv1.Started{}}}, "", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ev, err := eventOf(tc.res)
			if (err != nil) != tc.refused {
				t.Fatalf("refused %v, want %v", err, tc.refused)
			}
			if ev.On != tc.on || ev.Code != tc.code {
				t.Errorf("%s %s, want %s %s", ev.On, ev.Code, tc.on, tc.code)
			}
			if len(ev.Tail) > 2<<10 || !strings.HasSuffix(ev.Tail, tc.res.GetFault().GetTail()[max(0, len(tc.res.GetFault().GetTail())-3):]) {
				t.Errorf("a tail of %d bytes, want its end within 2 KiB", len(ev.Tail))
			}
		})
	}
}
