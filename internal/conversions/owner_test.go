package conversions

import (
	"context"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/history"
	"github.com/jatassi/slipstream/internal/housekeeping"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks"
)

// TestHeldCountsEveryPlacedConversionAndUnassignedPutsThemBack: Attic
// holds its copying, ready, running, paused and returning Conversions, of
// which the running and paused ones have started. Taking the Role queues
// them all again, and leaves the checking and landing ones, whose bytes
// are in the root.
func TestHeldCountsEveryPlacedConversionAndUnassignedPutsThemBack(t *testing.T) {
	l := newLife(t)
	stages := []Stage{StageCopying, StageReady, StageRunning, StagePaused, StageReturning, StageChecking, StageLanding, StageQueued}
	rows := make(map[Stage]Row, len(stages))
	for _, s := range stages {
		sd := seed{Stage: s, Node: &l.attic, Attempt: 1}
		if s == StageQueued {
			sd = seed{Stage: s}
		}
		rows[s] = l.seed(t, sd)
	}
	l.seed(t, seed{Stage: StageRunning, Node: &l.den, Attempt: 1})
	heldOn := func() nodes.Held {
		var h nodes.Held
		err := l.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
			var err error
			h, err = Owner.Held(ctx, r, l.attic)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		return h
	}
	if h := heldOn(); h.Units != 5 || h.Started != 2 {
		t.Fatalf("holds %d with %d started, want 5 with 2 started", h.Units, h.Started)
	}
	l.write(t, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return Owner.Unassigned(ctx, tx, l.attic, nodes.ID{})
	})
	if h := heldOn(); h.Units != 0 {
		t.Errorf("holds %d after the Role went, want none", h.Units)
	}
	for s, r := range rows {
		want := StageQueued
		if s == StageChecking || s == StageLanding {
			want = s
		}
		if got := l.read(t, r.ID).Stage; got != want {
			t.Errorf("a %s Conversion is %s, want %s", s, got, want)
		}
	}
}

type keepDefault struct{}

func (keepDefault) Fields(context.Context) ([]setting.Field, error) { return nil, nil }
func (keepDefault) Apply(context.Context, setting.Patch) error      { return nil }
func (keepDefault) Keep(context.Context) (time.Duration, error)     { return history.KeepFor.Default, nil }

// TestRetentionPrunesFinishedConversionsPastKeepHistoryFor: done and
// canceled Conversions older than Keep history for go with their
// attempts, and backfills older than it with none left. Failed and
// unfinished ones stay, however old.
func TestRetentionPrunesFinishedConversionsPastKeepHistoryFor(t *testing.T) {
	l := newLife(t)
	old := time.Now().Add(-history.KeepFor.Default - 24*time.Hour)
	young := time.Now().Add(-time.Hour)
	cases := []struct {
		name string
		seed seed
		goes bool
	}{
		{"done long ago", seed{Stage: StageDone, Node: &l.attic, Attempt: 1, Finished: old}, true},
		{"canceled long ago", seed{Stage: StageCanceled, Finished: old}, true},
		{"done an hour ago", seed{Stage: StageDone, Node: &l.attic, Attempt: 1, Finished: young}, false},
		{"failed", seed{Stage: StageFailed, Attempt: 1}, false},
		{"queued", seed{Stage: StageQueued}, false},
	}
	ids := make([]ID, len(cases))
	for i, tc := range cases {
		ids[i] = l.seed(t, tc.seed).ID
	}
	empty, kept := BackfillID{UUID: storage.NewUUID()}, BackfillID{UUID: storage.NewUUID()}
	l.write(t, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		for _, b := range []BackfillID{empty, kept} {
			if _, err := tx.ExecContext(ctx, `INSERT INTO conversions_backfill (id, kind, created_at, count) VALUES (?, 'backfill', ?, 1)`, b, storage.FromTime(old)); err != nil {
				return err
			}
		}
		_, err := tx.ExecContext(ctx, `UPDATE conversions_conversion SET backfill_id = ? WHERE id = ?`, kept, ids[4])
		return err
	})
	task := housekeeping.Task(housekeeping.Config{Reconciling: func(context.Context) (bool, error) { return false, nil }, Keep: keepDefault{}, Batch: 1, Steps: []housekeeping.Step{l.c.Retention()}})
	if err := task.Run(t.Context(), tasks.Unit{Key: "finished-conversions"}, &tasks.Progress{}); err != nil {
		t.Fatal(err)
	}
	count := func(q string, args ...any) int {
		var n int
		if err := l.lib.QueryRowContext(t.Context(), q, args...).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	for i, tc := range cases {
		if left := count(`SELECT count(*) FROM conversions_conversion WHERE id = ?`, ids[i]) == 1; left == tc.goes {
			t.Errorf("a Conversion %s stays %v, want %v", tc.name, left, !tc.goes)
		}
		if tc.goes && count(`SELECT count(*) FROM conversions_attempt WHERE conversion_id = ?`, ids[i]) != 0 {
			t.Errorf("a Conversion %s left its attempts", tc.name)
		}
	}
	if count(`SELECT count(*) FROM conversions_backfill WHERE id = ?`, empty) != 0 || count(`SELECT count(*) FROM conversions_backfill WHERE id = ?`, kept) != 1 {
		t.Error("want the empty backfill gone and the one with a Conversion kept")
	}
}
