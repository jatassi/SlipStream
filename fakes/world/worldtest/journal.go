package worldtest

import (
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"

	"github.com/jatassi/slipstream/fakes/world"
)

// journals are the runs each test launched, into which its API calls and
// its Until waits journal, and how many Until waits it is inside, whose
// polling journals only as the wait's note.
var journals = struct {
	mu      sync.Mutex
	runs    map[*testing.T][]*world.Run
	waiting map[*testing.T]int
}{runs: map[*testing.T][]*world.Run{}, waiting: map[*testing.T]int{}}

// journalInto has t's API calls and waits journal into run until t ends.
func journalInto(t *testing.T, run *world.Run) {
	journals.mu.Lock()
	defer journals.mu.Unlock()
	journals.runs[t] = append(journals.runs[t], run)
	t.Cleanup(func() {
		journals.mu.Lock()
		defer journals.mu.Unlock()
		if journals.runs[t] = slices.DeleteFunc(journals.runs[t], func(r *world.Run) bool { return r == run }); len(journals.runs[t]) == 0 {
			delete(journals.runs, t)
		}
	})
}

func launchedBy(t *testing.T) []*world.Run {
	journals.mu.Lock()
	defer journals.mu.Unlock()
	return slices.Clone(journals.runs[t])
}

func waiting(t *testing.T) bool {
	journals.mu.Lock()
	defer journals.mu.Unlock()
	return journals.waiting[t] > 0
}

func wait(t *testing.T, d int) {
	journals.mu.Lock()
	defer journals.mu.Unlock()
	if journals.waiting[t] += d; journals.waiting[t] == 0 {
		delete(journals.waiting, t)
	}
}

// startStep journals s on run and returns what ends it: its exit, and
// the stdout it keeps, if any.
func startStep(t *testing.T, run *world.Run, s world.StepStart) func(exit int, stdout []byte) {
	t.Helper()
	ctx := Context(t)
	step, err := run.StartStep(ctx, s)
	if err != nil {
		t.Fatal(err)
	}
	return func(exit int, stdout []byte) {
		t.Helper()
		var files map[string]string
		if stdout != nil {
			out := world.StepOutputs(step.N, "")["stdout"]
			files = map[string]string{"stdout": out}
			path := filepath.Join(run.Manifest().RunDir, filepath.FromSlash(out))
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, stdout, 0o644); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := run.EndStep(ctx, step.N, world.StepEnd{Exit: exit, Files: files}); err != nil {
			t.Fatal(err)
		}
	}
}
