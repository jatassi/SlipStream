// Package lifecycle runs one Node of a world, from each spawn to each
// exit, as a small state machine: an action moves a Node between
// running, stopped, killed and exited, and a process that exits while no
// action is under way leaves it exited. Each process writes node.spawned,
// node.ready, node.error and node.exited records, and each stop, kill and
// wipe a node.stopped, node.killed or node.wiped record once done. A Node
// that restarts prints its readiness record again, which writes
// node.restarted and node.ready.
package lifecycle

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/world/internal/proc"
	"github.com/jatassi/slipstream/internal/node"
)

type State string

const (
	Running State = "running"
	// Stopped is a Node a stop action ended, and a Node not yet started.
	Stopped State = "stopped"
	Killed  State = "killed"
	// Exited is a Node whose process ended with no action asking it to.
	Exited State = "exited"
)

type Action string

const (
	Start   Action = "start"
	Stop    Action = "stop"
	Kill    Action = "kill"
	Restart Action = "restart"
	Wipe    Action = "wipe"
)

// Actions is every action, in the order fakeworld lists them.
var Actions = []Action{Stop, Kill, Start, Restart, Wipe}

// from is the states each action may start from.
var from = map[Action][]State{
	Start:   {Stopped, Killed, Exited},
	Stop:    {Running},
	Kill:    {Running},
	Restart: {Running},
	Wipe:    {Stopped},
}

// Config is what a Node runs: the binary's run on its data dir, with its
// stderr appended to Log.
type Config struct {
	Name    string
	Binary  string
	DataDir node.DataDir
	Env     []string
	Log     string
	// Record appends a record an action writes, ctx carrying its step.
	Record func(ctx context.Context, rec events.Record)
	// Append appends a record no action asked for: node.error,
	// node.exited, and a restart's node.restarted and node.ready.
	Append func(rec events.Record)
	// Changed hears a Node that left running with no action asking it to,
	// and a running Node that restarted.
	Changed func()
}

// Spawned is node.spawned's data.
type Spawned struct {
	PID     int    `json:"pid"`
	DataDir string `json:"dataDir"`
}

type pid struct {
	PID int `json:"pid"`
}

type exited struct {
	PID  int `json:"pid"`
	Code int `json:"code"`
}

// restarted is node.restarted's data. A restart in place keeps the PID,
// and a successor has its own.
type restarted struct {
	OldPID int `json:"oldPid"`
	PID    int `json:"pid"`
}

type Node struct {
	cfg Config

	// act serializes actions, each of which runs to its end under it.
	act sync.Mutex
	// ended is a Node its world stopped, which takes no more actions.
	ended bool

	mu    sync.Mutex
	state State
	ready node.Ready
	// proc is the process Start spawned. Its stderr outlives it while a
	// successor it started for a restart holds the same pipe.
	proc *proc.Process
	// pid is the latest process's: proc's, or that of the successor whose
	// readiness record came last.
	pid int
	// successor is the latest process when a restart started it, watched
	// from its readiness record while pid still names it.
	successor *proc.Watched
	// announced is closed by the next readiness record.
	announced chan struct{}
	// leaving is the state an action ending the process moves to on its
	// exit.
	leaving State
	// gone is closed once the process and its successor have exited, its
	// node.exited record is written and its state moved.
	gone chan struct{}
}

func New(cfg Config) *Node {
	return &Node{cfg: cfg, state: Stopped}
}

func (n *Node) State() State {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.state
}

// Ready is the readiness record of the Node's latest process.
func (n *Node) Ready() node.Ready {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.ready
}

// PID is the running Node's process ID, or 0 while no process runs.
func (n *Node) PID() int {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.state != Running {
		return 0
	}
	return n.pid
}

// Live is the readiness record of the running Node.
func (n *Node) Live() (node.Ready, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.state != Running {
		return node.Ready{}, fmt.Errorf("node %s is %s", n.cfg.Name, n.state)
	}
	return n.ready, nil
}

func (n *Node) Do(ctx context.Context, a Action) error {
	switch a {
	case Start:
		return n.Start(ctx)
	case Stop:
		return n.Stop(ctx)
	case Kill:
		return n.Kill(ctx)
	case Restart:
		return n.Restart(ctx)
	case Wipe:
		return n.Wipe(ctx)
	default:
		return fmt.Errorf("%q is not a Node action", a)
	}
}

func (n *Node) may(a Action) error {
	if n.ended {
		return fmt.Errorf("node %s's world has stopped", n.cfg.Name)
	}
	if st := n.State(); !slices.Contains(from[a], st) {
		names := make([]string, len(from[a]))
		for i, s := range from[a] {
			names[i] = string(s)
		}
		return fmt.Errorf("node %s is %s; %s needs it %s", n.cfg.Name, st, a, strings.Join(names, " or "))
	}
	return nil
}

// Start spawns the Node and returns once it is ready. A Node that exits
// first fails Start with the tail of its stderr.
func (n *Node) Start(ctx context.Context) error {
	n.act.Lock()
	defer n.act.Unlock()
	if err := n.may(Start); err != nil {
		return err
	}
	name := n.cfg.Name
	logFile, err := os.OpenFile(n.cfg.Log, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o644) //nolint:gosec // a Node's log is evidence anyone on the machine may read
	if err != nil {
		return fmt.Errorf("node %s: %w", name, err)
	}
	ready := n.expectReady()
	found := false
	watch := func(line []byte) {
		if errorLine(line) {
			n.cfg.Append(events.Record{Kind: events.NodeError, Node: name, Data: line})
		}
		if rec, err := node.ParseReady(line); err == nil {
			n.adopt(rec, found)
			found = true
		}
	}
	p, err := proc.Start(ctx, n.cfg.Binary, []string{"run", "--data-dir", n.cfg.DataDir.Path()}, n.cfg.Env, logFile, watch)
	if err != nil {
		_ = logFile.Close()
		return fmt.Errorf("node %s: %w", name, err)
	}
	gone := make(chan struct{})
	n.mu.Lock()
	n.state, n.proc, n.pid, n.gone, n.leaving = Running, p, p.PID, gone, ""
	n.mu.Unlock()
	n.cfg.Record(ctx, events.Record{Kind: events.NodeSpawned, Node: name, Data: data(Spawned{PID: p.PID, DataDir: n.cfg.DataDir.Path()})})
	go n.await(p, logFile, gone)
	select {
	case <-ready:
		n.cfg.Record(ctx, events.Record{Kind: events.NodeReady, Node: name, Data: data(pid{n.Ready().PID})})
		return nil
	case <-gone:
		return fmt.Errorf("node %s exited with code %d before it was ready\n%s", name, p.ExitCode(), strings.Join(p.Tail(), "\n"))
	case <-ctx.Done():
		return fmt.Errorf("node %s was not ready: %w", name, ctx.Err())
	}
}

func (n *Node) expectReady() <-chan struct{} {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.announced = make(chan struct{})
	return n.announced
}

// adopt takes a readiness record as the Node's latest. One past the
// process's first is a restart, which the Node announces itself.
func (n *Node) adopt(ready node.Ready, again bool) {
	n.mu.Lock()
	old := n.pid
	n.ready, n.pid = ready, ready.PID
	announced := n.announced
	n.announced = make(chan struct{})
	var watchErr error
	if again && ready.PID != old {
		if n.successor != nil {
			n.successor.Close()
		}
		n.successor, watchErr = proc.Watch(ready.PID)
	}
	n.mu.Unlock()
	if watchErr != nil {
		n.fault(watchErr)
	}
	if again {
		n.cfg.Changed()
		n.cfg.Append(events.Record{Kind: events.NodeRestarted, Node: n.cfg.Name, Data: data(restarted{OldPID: old, PID: ready.PID})})
		n.cfg.Append(events.Record{Kind: events.NodeReady, Node: n.cfg.Name, Data: data(pid{ready.PID})})
	}
	close(announced)
}

// await records p's exit and moves the Node to the state the action that
// ended it named, or to exited. A successor holding p's stderr closes it
// before it counts as exited, so await waits for the successor too.
func (n *Node) await(p *proc.Process, logFile *os.File, gone chan struct{}) {
	<-p.Exited()
	n.mu.Lock()
	successor := n.successor
	n.successor = nil
	n.mu.Unlock()
	if successor != nil {
		if err := successor.Wait(); err != nil {
			n.fault(err)
		}
		successor.Close()
	}
	_ = logFile.Close()
	n.cfg.Append(events.Record{Kind: events.NodeExited, Node: n.cfg.Name, Data: data(exited{PID: p.PID, Code: p.ExitCode()})})
	n.mu.Lock()
	to := n.leaving
	if to == "" {
		to = Exited
	}
	n.state, n.leaving = to, ""
	n.mu.Unlock()
	close(gone)
	if to == Exited {
		n.cfg.Changed()
	}
}

// fault records the harness failing to follow the Node's processes.
func (n *Node) fault(err error) {
	n.cfg.Append(events.Record{Kind: events.NodeError, Node: n.cfg.Name, Data: data(err.Error())})
}

// leave marks the running Node as leaving for to and returns its process
// and the channel closed once it is gone.
func (n *Node) leave(to State) (p *proc.Process, gone <-chan struct{}) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.leaving = to
	return n.proc, n.gone
}

// stay undoes leave for a Node whose stop failed and still runs.
func (n *Node) stay() {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.state == Running {
		n.leaving = ""
	}
}

// Stop runs slipstream stop on the Node's data dir and returns once its
// process has exited.
func (n *Node) Stop(ctx context.Context) error {
	n.act.Lock()
	defer n.act.Unlock()
	if err := n.may(Stop); err != nil {
		return err
	}
	p, gone := n.leave(Stopped)
	if err := n.command(ctx, "stop"); err != nil {
		n.stay()
		return err
	}
	<-gone
	n.cfg.Record(ctx, events.Record{Kind: events.NodeStopped, Node: n.cfg.Name, Data: data(pid{p.PID})})
	return nil
}

// command runs slipstream cmd on the Node's data dir.
func (n *Node) command(ctx context.Context, cmd string) error {
	res, err := proc.Run(ctx, n.cfg.Binary, []string{cmd, "--data-dir", n.cfg.DataDir.Path()}, n.cfg.Env)
	if err != nil {
		return fmt.Errorf("node %s: %w", n.cfg.Name, err)
	}
	if res.Code != 0 {
		return fmt.Errorf("node %s: slipstream %s exited with code %d: %s", n.cfg.Name, cmd, res.Code, strings.TrimSpace(string(res.Stderr)))
	}
	return nil
}

// Restart runs slipstream restart on the Node's data dir and returns once
// the Node's next readiness record is in.
func (n *Node) Restart(ctx context.Context) error {
	n.act.Lock()
	defer n.act.Unlock()
	if err := n.may(Restart); err != nil {
		return err
	}
	n.mu.Lock()
	p, ready, gone := n.proc, n.announced, n.gone
	n.mu.Unlock()
	if err := n.command(ctx, "restart"); err != nil {
		return err
	}
	select {
	case <-ready:
		return nil
	case <-gone:
		return fmt.Errorf("node %s exited with code %d before it was ready again\n%s", n.cfg.Name, p.ExitCode(), strings.Join(p.Tail(), "\n"))
	case <-ctx.Done():
		return fmt.Errorf("node %s was not ready again: %w", n.cfg.Name, ctx.Err())
	}
}

// Kill kills the Node's process and returns once it has exited.
func (n *Node) Kill(ctx context.Context) error {
	n.act.Lock()
	defer n.act.Unlock()
	if err := n.may(Kill); err != nil {
		return err
	}
	p, gone := n.leave(Killed)
	if err := n.kill(p); err != nil {
		n.stay()
		return fmt.Errorf("node %s: %w", n.cfg.Name, err)
	}
	<-gone
	n.cfg.Record(ctx, events.Record{Kind: events.NodeKilled, Node: n.cfg.Name, Data: data(pid{p.PID})})
	return nil
}

// kill kills p and the successor a restart left holding its stderr.
func (n *Node) kill(p *proc.Process) error {
	n.mu.Lock()
	latest := n.pid
	n.mu.Unlock()
	if latest != p.PID {
		if err := proc.Kill(latest); err != nil {
			return err
		}
	}
	return p.Kill()
}

// Wipe empties the stopped Node's data dir but its config.yaml, so it
// starts again Unclaimed on the same addresses.
func (n *Node) Wipe(ctx context.Context) error {
	n.act.Lock()
	defer n.act.Unlock()
	if err := n.may(Wipe); err != nil {
		return err
	}
	dir := n.cfg.DataDir.Path()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("node %s: %w", n.cfg.Name, err)
	}
	for _, e := range entries {
		if e.Name() == "config.yaml" {
			continue
		}
		if err := os.RemoveAll(filepath.Join(dir, e.Name())); err != nil {
			return fmt.Errorf("node %s: %w", n.cfg.Name, err)
		}
	}
	n.cfg.Record(ctx, events.Record{Kind: events.NodeWiped, Node: n.cfg.Name, Data: data(struct {
		DataDir string `json:"dataDir"`
	}{dir})})
	return nil
}

// Ending is how a Node ended when its world stopped.
type Ending struct {
	ExitCode int
	// Killed is a Node a kill action left down.
	Killed bool
	// StopFailed is slipstream stop's failure; the world then killed the
	// Node.
	StopFailed string
}

// End stops a running Node, killing it when slipstream stop fails, and
// reports how its last process ended. A Node no process ran reports
// exit code 0.
func (n *Node) End(ctx context.Context) Ending {
	n.act.Lock()
	defer n.act.Unlock()
	n.ended = true
	var e Ending
	if n.State() == Running {
		p, gone := n.leave(Stopped)
		if err := n.command(ctx, "stop"); err != nil {
			e.StopFailed = strings.TrimPrefix(err.Error(), "node "+n.cfg.Name+": ")
			if kerr := n.kill(p); kerr != nil {
				e.StopFailed += "; " + kerr.Error()
			}
		}
		<-gone
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.proc != nil {
		e.ExitCode = n.proc.ExitCode()
	}
	e.Killed = n.state == Killed
	return e
}

// errorLine is a Node log line at level ERROR.
func errorLine(line []byte) bool {
	var rec struct {
		Level string `json:"level"`
	}
	return json.Unmarshal(line, &rec) == nil && rec.Level == slog.LevelError.String()
}

func ParseAction(s string) (Action, error) {
	if a := Action(s); slices.Contains(Actions, a) {
		return a, nil
	}
	return "", fmt.Errorf("%q is not a Node action: stop, kill, start, restart or wipe", s)
}

func data(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic("lifecycle: " + err.Error())
	}
	return b
}
