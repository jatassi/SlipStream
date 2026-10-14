package world

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/world/internal/control"
	"github.com/jatassi/slipstream/fakes/world/internal/lifecycle"
	"github.com/jatassi/slipstream/fakes/world/internal/proc"
)

const stopReportFile = "stopped.json"

// StopReport is how each Node of a run ended, in Scenario order.
type StopReport struct {
	Nodes []NodeStop `json:"nodes"`
	// Problems are stop steps other than a Node's that failed.
	Problems []string `json:"problems,omitempty"`
}

type NodeStop struct {
	Name     string `json:"name"`
	ExitCode int    `json:"exitCode"`
	// ExitedUnasked is a Node that exited before the run was stopping
	// with no Node action asking it to.
	ExitedUnasked bool `json:"exitedUnasked"`
	// Killed is a Node a kill action left down, whose exit code says
	// nothing.
	Killed bool `json:"killed,omitempty"`
	// StopFailed is slipstream stop's failure when it did not see the Node
	// release its data dir; the world then killed the Node.
	StopFailed string `json:"stopFailed,omitempty"`
}

// Err is every way the run did not end cleanly: a Node that exited unasked,
// exited non-zero or missed its deadline, and any other failed step.
func (r StopReport) Err() error {
	var problems []string
	for _, n := range r.Nodes {
		switch {
		case n.ExitedUnasked:
			problems = append(problems, fmt.Sprintf("node %s exited with code %d before it was asked to stop", n.Name, n.ExitCode))
		case n.StopFailed != "":
			problems = append(problems, fmt.Sprintf("node %s did not stop: %s", n.Name, n.StopFailed))
		case n.ExitCode != 0 && !n.Killed:
			problems = append(problems, fmt.Sprintf("node %s exited with code %d", n.Name, n.ExitCode))
		}
	}
	problems = append(problems, r.Problems...)
	if len(problems) == 0 {
		return nil
	}
	return errors.New(strings.Join(problems, "\n"))
}

// Stop stops every Node in reverse Scenario order, then the fakes and their
// fronts, renames
// the manifest to manifest.stopped.json, deletes the scratch dir, writes
// stopped.json and closes the control socket. The first caller does the
// work, which always runs to its end; every caller gets the same report.
func (r *Run) Stop(context.Context) (StopReport, error) {
	r.stopOnce.Do(r.stop)
	return r.report, r.report.Err()
}

// Done is closed once the run has stopped and the control socket has
// answered every request it took.
func (r *Run) Done() <-chan struct{} { return r.control.Done() }

func (r *Run) stop() {
	r.record(events.RunStopping, struct{}{})
	r.manifestMu.Lock()
	r.published = false
	r.manifestMu.Unlock()
	unasked := make([]bool, len(r.nodes))
	for i, n := range r.nodes {
		unasked[i] = n.life.State() == lifecycle.Exited
	}
	r.report.Nodes = make([]NodeStop, len(r.nodes))
	for i := len(r.nodes) - 1; i >= 0; i-- {
		r.report.Nodes[i] = r.stopNode(r.nodes[i], unasked[i])
	}
	problem := func(err error) {
		if err != nil {
			r.report.Problems = append(r.report.Problems, err.Error())
		}
	}
	problem(r.seeders.Close())
	for _, f := range r.fakes {
		problem(f.stop())
	}
	r.fronts.Close()
	problem(os.Rename(filepath.Join(r.runDir, manifestFile), filepath.Join(r.runDir, stoppedManifestFile)))
	problem(os.RemoveAll(r.scratchDir))
	problem(writeAtomically(filepath.Join(r.runDir, stopReportFile), marshal(r.report)))
	r.record(events.RunStopped, stopped{Clean: r.report.Err() == nil})
	r.control.Close()
	_ = r.events.Close()
	_ = r.logFile.Close()
}

func (r *Run) stopNode(n *nodeRun, unasked bool) NodeStop {
	e := n.life.End(r.life)
	ns := NodeStop{Name: n.spec.Name, ExitCode: e.ExitCode, ExitedUnasked: unasked, Killed: e.Killed, StopFailed: e.StopFailed}
	r.log.Info("node stopped", "node", ns.Name, "code", ns.ExitCode, "unasked", ns.ExitedUnasked, "killed", ns.Killed, "stop_failed", ns.StopFailed)
	return ns
}

// SupervisorGoneError is a run whose control socket does not answer. It
// names the command that stops each Node the event log says was spawned.
type SupervisorGoneError struct {
	PID    int
	Spawns []Spawn
}

func (e *SupervisorGoneError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "the supervisor (PID %d) is not running; stop each Node with:", e.PID)
	for _, s := range e.Spawns {
		fmt.Fprintf(&b, "\n  slipstream stop --data-dir %s", s.DataDir)
	}
	return b.String()
}

// StopRun stops the run in runDir through its supervisor's control
// socket and returns the report once the work is done and a supervisor in
// another process has exited. A run that already stopped returns its
// stopped.json. It never stops a Node itself: when the supervisor is gone
// it returns a *SupervisorGoneError.
func StopRun(ctx context.Context, runDir string) (StopReport, error) {
	supervisor, spawns, err := readLaunch(runDir)
	if err != nil {
		return StopReport{}, err
	}
	var report StopReport
	if data, rerr := os.ReadFile(filepath.Join(runDir, stopReportFile)); rerr == nil {
		err = json.Unmarshal(data, &report)
	} else {
		err = (&Remote{socket: supervisor.Control}).call(ctx, http.MethodPost, "/v1/stop", struct{}{}, &report)
	}
	if errors.Is(err, control.ErrNoAnswer) {
		return StopReport{}, &SupervisorGoneError{PID: supervisor.PID, Spawns: spawns}
	}
	if err != nil {
		return StopReport{}, fmt.Errorf("stop report: %w", err)
	}
	err = report.Err()
	if supervisor.PID != os.Getpid() {
		if werr := proc.WaitExit(ctx, supervisor.PID); werr != nil {
			return report, errors.Join(err, fmt.Errorf("the supervisor: %w", werr))
		}
	}
	return report, err
}
