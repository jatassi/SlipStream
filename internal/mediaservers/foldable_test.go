package mediaservers_test

import (
	"fmt"
	"testing"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers/queue"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
)

// TestFoldablePredicate pins the partial unique indexes on
// mediaservers_change to queue.Change.Foldable: a second open row at a path,
// or a second open carry of a file, is refused exactly when the first row is
// foldable.
func TestFoldablePredicate(t *testing.T) {
	f := seed(t)
	n := 0
	for _, state := range []queue.State{queue.Prepared, queue.Due, queue.Sending, queue.Sent} {
		for _, step := range link.Steps() {
			for _, carry := range []bool{false, true} {
				if state == queue.Prepared && !carry {
					continue
				}
				n++
				first := queue.Change{State: state, Step: step}
				if carry {
					first.From = mustRel(t, fmt.Sprintf("Old/%d.mkv", n))
				}
				if refused := f.refusesASecondOpenRow(t, first, n); refused != first.Foldable() {
					t.Errorf("beside a %s row at %s (carry %v), a second open row refused %v, want %v", state, step, carry, refused, first.Foldable())
				}
			}
		}
	}
}

type changeRow struct {
	kind, path string
	from       any
	file       storage.UUID
}

// refusesASecondOpenRow stores c, then reports whether the schema refuses a
// due row at its path, or a due carry of its file when c is a carry.
func (f fixture) refusesASecondOpenRow(t *testing.T, c queue.Change, n int) bool {
	t.Helper()
	path := fmt.Sprintf("Path/%d.mkv", n)
	first, second := changeRow{"added", path, nil, storage.NewUUID()}, changeRow{"removed", path, nil, storage.NewUUID()}
	if c.Carry() {
		first = changeRow{"moved", fmt.Sprintf("New/%d.mkv", n), c.From.String(), storage.NewUUID()}
		second = changeRow{"moved", fmt.Sprintf("Other/%d.mkv", n), fmt.Sprintf("Another/%d.mkv", n), first.file}
	}
	if err := f.insertRow(t, first, c.State, c.Step); err != nil {
		t.Fatalf("a %s row at %s wasn't stored: %v", c.State, c.Step, err)
	}
	return f.insertRow(t, second, queue.Due, link.StepNone) != nil
}

func (f fixture) insertRow(t *testing.T, r changeRow, state queue.State, step link.Step) error {
	t.Helper()
	var stepAt, result, sentAt any
	if step != link.StepNone {
		stepAt = 1
	}
	if state == queue.Sent {
		result, sentAt = link.ResultAccepted, 1
	}
	return exec(t, f.lib, `INSERT INTO mediaservers_change (id, server_id, kind, root_id, path, from_path, file_id, state, step,
		step_at, result, sent_at, next_attempt_at, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, 1)`,
		storage.NewUUID(), f.server, r.kind, f.root, r.path, r.from, r.file, state, step, stepAt, result, sentAt)
}

func mustRel(t *testing.T, s string) storing.Rel {
	t.Helper()
	r, err := storing.ParseRel(s)
	if err != nil {
		t.Fatal(err)
	}
	return r
}
