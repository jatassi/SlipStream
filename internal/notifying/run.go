package notifying

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/proctree"
)

// Script is one run of an Admin's program for Custom Script.
type Script struct {
	Path string
	Args []string
	// Event is NOTIFICATION_EVENT, the kind.
	Event string
	// EventFile is written to a file NOTIFICATION_EVENT_FILE names.
	EventFile []byte
	// Stop is how long the program and everything it started may run.
	Stop time.Duration
}

// Ran is how a Script's run ended. Exit is -1 when it never exited on its
// own, and Err is why it didn't start for any reason other than Missing.
type Ran struct {
	Exit     int
	Tail     []byte
	TimedOut bool
	Missing  bool
	Err      string
}

// MaxTail is how much of the end of a script's output comes back: enough
// for its last error, little enough to show in a reason.
const MaxTail = 1 << 10

// Runner runs a Script and stops its whole process tree after Stop.
type Runner interface {
	Run(ctx context.Context, s Script) Ran
}

// ProcessRunner runs a Script as a process with a clean environment:
// PATH and the home and temp variables from Environ, SystemRoot on
// Windows, and the two NOTIFICATION_ variables. Nothing else from the
// Node, its SLIPSTREAM_ variables included, reaches the script.
type ProcessRunner struct {
	Environ []string
	// TempDir is where the event file is written.
	TempDir string
}

// stopGrace is how long a killed script's output may take to close, which
// a grandchild outside the tree could hold open.
const stopGrace = 2 * time.Second

func (p ProcessRunner) Run(ctx context.Context, s Script) Ran {
	if _, err := os.Stat(s.Path); errors.Is(err, fs.ErrNotExist) {
		return Ran{Exit: -1, Missing: true}
	}
	event, err := os.CreateTemp(p.TempDir, "notification-*.json")
	if err != nil {
		return Ran{Exit: -1, Err: err.Error()}
	}
	defer func() { _ = os.Remove(event.Name()) }()
	_, werr := event.Write(s.EventFile)
	if err := errors.Join(werr, event.Close()); err != nil {
		return Ran{Exit: -1, Err: err.Error()}
	}

	ctx, cancel := context.WithTimeout(ctx, s.Stop)
	defer cancel()
	cmd := exec.CommandContext(ctx, s.Path, s.Args...) //nolint:gosec // the program an Admin named for this Custom Script
	cmd.Env = append(p.passed(), "NOTIFICATION_EVENT="+s.Event, "NOTIFICATION_EVENT_FILE="+event.Name())
	out := &tail{}
	cmd.Stdout, cmd.Stderr = out, out
	cmd.WaitDelay = stopGrace
	end, err := proctree.Start(cmd)
	if err != nil {
		return Ran{Exit: -1, Err: err.Error()}
	}
	_ = cmd.Wait()
	end()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return Ran{Exit: -1, Tail: out.b, TimedOut: true}
	}
	return Ran{Exit: cmd.ProcessState.ExitCode(), Tail: out.b}
}

// passed is the Node's variables a script gets.
func (p ProcessRunner) passed() []string {
	var env []string
	for _, kv := range p.Environ {
		name, _, _ := strings.Cut(kv, "=")
		if slices.ContainsFunc(passedVars, func(v string) bool { return sameVar(v, name) }) {
			env = append(env, kv)
		}
	}
	return env
}

// tail keeps the last MaxTail bytes written to it.
type tail struct{ b []byte }

func (t *tail) Write(p []byte) (int, error) {
	t.b = append(t.b, p...)
	if over := len(t.b) - MaxTail; over > 0 {
		t.b = slices.Clone(t.b[over:])
	}
	return len(p), nil
}
