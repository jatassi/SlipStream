package node

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/node/internal/control"
	"github.com/jatassi/slipstream/internal/node/internal/lock"
	"github.com/jatassi/slipstream/internal/node/internal/logs"
	"github.com/jatassi/slipstream/internal/node/op"
)

// restartOfVar carries the PID a restarted process follows. ProcessEnv
// removes it before the config loader sees the environment.
const restartOfVar = "SLIPSTREAM_RESTART_OF"

var (
	ErrAlreadyRestarting = errors.New("the Node is already restarting")
	ErrStopping          = errors.New("the Node is stopping")
	// ErrCannotRestart is an Env with no Restart: a Node a test started
	// without one.
	ErrCannotRestart = errors.New("this Node cannot start its own successor")
)

// Restart stops the Node to start it again. Once the stop has unwound
// cleanly and released the data dir, the Node calls Env.Restart, which
// starts the successor. Only the call that stops the Node restarts it.
func (n *Node) Restart() error {
	if n.env.Restart == nil {
		return ErrCannotRestart
	}
	if n.stop(true) {
		return nil
	}
	if n.restart {
		return ErrAlreadyRestarting
	}
	return ErrStopping
}

// stop closes stopping once, recording whether a restart asked for it, and
// reports whether this call did.
func (n *Node) stop(restart bool) bool {
	stopped := false
	n.stopOnce.Do(func() {
		n.restart = restart
		close(n.stopping)
		stopped = true
	})
	return stopped
}

// handoff is a stopped Node's last step: the successor a restart asked
// for, when the unwind was clean and released the data dir.
func (n *Node) handoff(err error) error {
	if !n.restart || err != nil {
		return err
	}
	select {
	case <-n.released:
		return n.env.Restart()
	default:
		return errors.New("the data dir was not released, so no successor starts")
	}
}

type RestartRequest struct{}

// RestartReply comes before the stop runs. Within is how long the Node may
// take to stop and its successor to take the data dir.
type RestartReply struct {
	PID       int           `json:"pid"`
	StartedAt time.Time     `json:"started_at"`
	Within    time.Duration `json:"within"`
}

// RestartOp restarts the Node: it stops, then the same command starts
// again on the same data dir.
var RestartOp = op.Op[RestartRequest, RestartReply]{Name: "restart"}

func (n *Node) serveRestart(_ context.Context, body json.RawMessage, send func(any) error) error {
	var req RestartRequest
	if err := control.Decode(body, &req); err != nil {
		return err
	}
	if err := n.Restart(); err != nil {
		return &control.Error{Code: control.CodeFailed, Message: err.Error()}
	}
	return send(RestartReply{PID: os.Getpid(), StartedAt: n.started, Within: n.deadline + n.cfg.Timings.Node.RestartLockWait})
}

// RestartNode restarts the Node on env's data dir and returns its
// successor's status once it answers.
func RestartNode(ctx context.Context, env Env) (StatusReply, error) {
	reply, err := Call(ctx, env.Net, env.DataDir, RestartOp, RestartRequest{})
	if err != nil {
		return StatusReply{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, reply.Within+serviceStartWait)
	defer cancel()
	tick := time.NewTicker(500 * time.Millisecond)
	defer tick.Stop()
	for {
		status, err := Call(ctx, env.Net, env.DataDir, StatusOp, StatusRequest{})
		if err == nil && status.StartedAt.After(reply.StartedAt) {
			return status, nil
		}
		select {
		case <-ctx.Done():
			return StatusReply{}, fmt.Errorf("node PID %d stopped to restart and no successor answered within %s", reply.PID, reply.Within+serviceStartWait)
		case <-tick.C:
		}
	}
}

// acquireLock takes the data dir lock. A restarted process first waits up
// to timings.node.restart_lock_wait for the process it follows to release
// it, reading that from a load that takes no lock.
func (n *Node) acquireLock() (*lock.Lock, error) {
	lk, err := lock.Acquire(n.env.DataDir.RunDir())
	if n.env.RestartOf == 0 {
		return lk, err
	}
	wait := Defaults(n.env.Types, n.env.Install).Timings.Node.RestartLockWait
	if cfg, _, cfgErr := LoadConfig(n.env.DataDir, n.env.Environ, n.env.Types, n.env.Install); cfgErr == nil {
		wait = cfg.Timings.Node.RestartLockWait
	}
	deadline := time.Now().Add(wait)
	for {
		if _, held := errors.AsType[*lock.HeldError](err); !held || time.Now().After(deadline) {
			return lk, err
		}
		time.Sleep(100 * time.Millisecond)
		lk, err = lock.Acquire(n.env.DataDir.RunDir())
	}
}

// restartOf reads and removes restartOfVar from environ.
func restartOf(environ []string) (rest []string, pid int, err error) {
	i := slices.IndexFunc(environ, func(kv string) bool { return strings.HasPrefix(kv, restartOfVar+"=") })
	if i < 0 {
		return environ, 0, nil
	}
	pid, err = strconv.Atoi(strings.TrimPrefix(environ[i], restartOfVar+"="))
	if err != nil || pid <= 0 {
		return nil, 0, fmt.Errorf("%s is not a PID", environ[i])
	}
	return slices.Delete(slices.Clone(environ), i, i+1), pid, nil
}

// successorEnviron is environ for the process that follows this one.
func successorEnviron(environ []string) []string {
	return append(slices.Clone(environ), restartOfVar+"="+strconv.Itoa(os.Getpid()))
}

// restartFailed reports a successor that did not start. The Node's log
// has closed, so the record goes straight to the end of its file.
func restartFailed(logsDir string, stderr io.Writer, err error) error {
	err = fmt.Errorf("restart: %w", err)
	_, _ = fmt.Fprintln(stderr, err)
	f, openErr := os.OpenFile(filepath.Join(logsDir, logs.File), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600) //nolint:gosec // the Node's own logs dir
	if openErr != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	slog.New(slog.NewJSONHandler(f, nil)).Error("restart failed", "subsystem", "node", "err", err.Error())
	return err
}
