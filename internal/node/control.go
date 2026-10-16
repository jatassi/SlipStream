package node

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/node/internal/control"
	"github.com/jatassi/slipstream/internal/node/op"
	"github.com/jatassi/slipstream/internal/pairing"
)

type StopRequest struct{}

// StopReply comes before the stop runs. Deadline is the Node's
// timings.node.stop_deadline.
type StopReply struct {
	PID      int           `json:"pid"`
	Deadline time.Duration `json:"deadline"`
}

// StopOp stops the Node. The Node replies, stops, and then sends one more
// reply once it has released its data dir.
var StopOp = op.Op[StopRequest, StopReply]{Name: "stop"}

type released struct {
	Released bool `json:"released"`
}

type ControlError = control.Error

// stopGrace is how long past the Node's own deadline the stop client waits
// for the Node to exit by itself.
const stopGrace = 5 * time.Second

// Call sends o to the Node on dir through network, the one the Node's Env
// serves its control socket on.
func Call[Req, Resp any](ctx context.Context, network link.Network, dir DataDir, o op.Op[Req, Resp], req Req) (Resp, error) {
	var resp Resp
	c, err := dial(ctx, network, dir)
	if err != nil {
		return resp, err
	}
	defer func() { _ = c.Close() }()
	if err := c.Send(o.Name, req); err != nil {
		return resp, err
	}
	if err := c.Recv(&resp); err != nil {
		return resp, fmt.Errorf("%s: %w", o.Name, err)
	}
	return resp, nil
}

func dial(ctx context.Context, network link.Network, dir DataDir) (*control.Conn, error) {
	c, err := control.Dial(ctx, network.DialContext, dir.ControlSocket())
	if err != nil {
		return nil, fmt.Errorf("no Node answered on %s: %w", dir, err)
	}
	return c, nil
}

// StopNode stops the Node on dir and returns once it has released the data
// dir, or an error when no Node answered or its stop deadline passed.
func StopNode(ctx context.Context, network link.Network, dir DataDir) error {
	dialCtx, cancel := context.WithTimeout(ctx, stopGrace)
	defer cancel()
	c, err := dial(dialCtx, network, dir)
	if err != nil {
		return err
	}
	defer func() { _ = c.Close() }()
	if err := c.Send(StopOp.Name, StopRequest{}); err != nil {
		return err
	}
	var reply StopReply
	if err := c.Recv(&reply); err != nil {
		return fmt.Errorf("stop: %w", err)
	}
	if err := c.SetDeadline(time.Now().Add(reply.Deadline + stopGrace)); err != nil {
		return err
	}
	var done released
	err = c.Recv(&done)
	if errors.Is(err, io.EOF) {
		return fmt.Errorf("node PID %d exited without releasing %s", reply.PID, dir)
	}
	if err != nil {
		return fmt.Errorf("stop node PID %d: %w", reply.PID, err)
	}
	return nil
}

// dispatch serves stop at any time and every other op once the Node is ready.
func (n *Node) dispatch(name string) (control.Handler, error) {
	if name == StopOp.Name {
		return n.serveStop, nil
	}
	if !n.isReady.Load() {
		return nil, &control.Error{Code: control.CodeNotReady, Message: fmt.Sprintf("node is starting; it serves only %s until ready", StopOp.Name)}
	}
	switch name {
	case pairing.JoinOp.Name:
		return n.serveJoin, nil
	case StatusOp.Name:
		return n.serveStatus, nil
	case ConfigOp.Name:
		return n.serveConfig, nil
	case RestartOp.Name:
		return n.serveRestart, nil
	}
	return nil, &control.Error{Code: control.CodeUnknownOp, Message: fmt.Sprintf("unknown op %q; node version %s", name, n.env.Version)}
}

func (n *Node) serveStop(_ context.Context, body json.RawMessage, send func(any) error) error {
	var req StopRequest
	if err := control.Decode(body, &req); err != nil {
		return err
	}
	n.Stop()
	if err := send(StopReply{PID: os.Getpid(), Deadline: n.deadline}); err != nil {
		return err
	}
	<-n.drained
	select {
	case <-n.released:
		return send(released{Released: true})
	default:
		return &control.Error{Code: control.CodeFailed, Message: "the stop deadline passed before the data dir was released"}
	}
}
