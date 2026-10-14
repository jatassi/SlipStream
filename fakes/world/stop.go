package world

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/world/internal/channel"
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
	// ExitedUnasked is a Node that exited before the run was stopping.
	ExitedUnasked bool `json:"exitedUnasked"`
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
		case n.ExitCode != 0:
			problems = append(problems, fmt.Sprintf("node %s exited with code %d", n.Name, n.ExitCode))
		}
	}
	problems = append(problems, r.Problems...)
	if len(problems) == 0 {
		return nil
	}
	return errors.New(strings.Join(problems, "\n"))
}

// Stop stops every Node in reverse Scenario order, then the fakes, renames
// the manifest to manifest.stopped.json, deletes the scratch dir, writes
// stopped.json and closes the stop channel. The first caller does the
// work; every caller gets the same report.
func (r *Run) Stop() (StopReport, error) {
	r.stopOnce.Do(r.stop)
	return r.report, r.report.Err()
}

// Done is closed once the run has stopped and the stop channel has answered
// every request it took.
func (r *Run) Done() <-chan struct{} { return r.channel.Done() }

func (r *Run) stop() {
	r.record(events.RunStopping, "", struct{}{})
	unasked := make([]bool, len(r.nodes))
	for i, n := range r.nodes {
		select {
		case <-n.proc.Exited():
			unasked[i] = true
		default:
		}
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
	problem(os.Rename(filepath.Join(r.runDir, manifestFile), filepath.Join(r.runDir, stoppedManifestFile)))
	problem(os.RemoveAll(r.scratchDir))
	problem(writeAtomically(filepath.Join(r.runDir, stopReportFile), marshal(r.report)))
	r.record(events.RunStopped, "", stopped{Clean: r.report.Err() == nil})
	r.channel.Close()
	_ = r.events.Close()
	_ = r.logFile.Close()
}

func (r *Run) stopNode(n *nodeRun, unasked bool) NodeStop {
	ns := NodeStop{Name: n.spec.Name, ExitedUnasked: unasked}
	select {
	case <-n.proc.Exited():
	default:
		res, err := proc.Run(r.life, r.binary.Path, []string{"stop", "--data-dir", n.dataDir.Path()}, n.env)
		switch {
		case err != nil:
			ns.StopFailed = err.Error()
		case res.Code != 0:
			ns.StopFailed = fmt.Sprintf("slipstream stop exited with code %d: %s", res.Code, strings.TrimSpace(string(res.Stderr)))
		}
		if ns.StopFailed != "" {
			if err := n.proc.Kill(); err != nil {
				ns.StopFailed += "; " + err.Error()
			}
		}
	}
	<-n.recorded
	ns.ExitCode = n.proc.ExitCode()
	_ = n.logFile.Close()
	r.log.Info("node stopped", "node", ns.Name, "code", ns.ExitCode, "unasked", ns.ExitedUnasked, "stop_failed", ns.StopFailed)
	return ns
}

func (r *Run) stopAnswer() []byte {
	report, _ := r.Stop()
	return marshal(report)
}

// SupervisorGoneError is a run whose stop channel does not answer. It
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

// StopRun stops the run in runDir through its supervisor's stop channel
// and returns the report once the work is done and a supervisor in another
// process has exited. A run that already stopped returns its stopped.json.
// It never stops a Node itself: when the supervisor is gone it returns a
// *SupervisorGoneError.
func StopRun(ctx context.Context, runDir string) (StopReport, error) {
	supervisor, spawns, err := readLaunch(runDir)
	if err != nil {
		return StopReport{}, err
	}
	data, err := os.ReadFile(filepath.Join(runDir, stopReportFile))
	if err != nil {
		data, err = channel.Stop(ctx, supervisor.Control)
	}
	if errors.Is(err, channel.ErrNoAnswer) {
		return StopReport{}, &SupervisorGoneError{PID: supervisor.PID, Spawns: spawns}
	}
	if err != nil {
		return StopReport{}, err
	}
	report, err := decodeReport(data)
	if supervisor.PID != os.Getpid() {
		if werr := proc.WaitExit(ctx, supervisor.PID); werr != nil {
			return report, errors.Join(err, fmt.Errorf("the supervisor: %w", werr))
		}
	}
	return report, err
}

func decodeReport(data []byte) (StopReport, error) {
	var report StopReport
	if err := json.Unmarshal(data, &report); err != nil {
		return StopReport{}, fmt.Errorf("stop report: %w", err)
	}
	return report, report.Err()
}
