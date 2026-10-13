package node

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/jatassi/slipstream/internal/joincode"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/node/internal/control"
	"github.com/jatassi/slipstream/internal/node/internal/lock"
	"github.com/jatassi/slipstream/internal/pairing"
	linkv1 "github.com/jatassi/slipstream/internal/pb/slipstream/link/v1"
	"github.com/jatassi/slipstream/internal/system"
)

// join runs pairing.Claim under the claim lock. The first Join starts the
// Worker half; every Join closes the Setup gate.
func (n *Node) join(ctx context.Context, code joincode.Code) (pairing.Joined, error) {
	n.claimMu.Lock()
	defer n.claimMu.Unlock()
	joined, err := n.pairing.Claim(ctx, code)
	if err != nil {
		return pairing.Joined{}, err
	}
	if n.half.Load() == nil {
		if err := n.startWorker(context.WithoutCancel(ctx)); err != nil {
			n.stack.fail(fmt.Errorf("start the Worker half: %w", err))
			return pairing.Joined{}, err
		}
		n.log.Info("claimed", "claim", "worker")
	}
	n.gate.Close()
	return joined, nil
}

// joinAtStart runs join.code's Join. A refusal leaves the Node as it was,
// logged once, so a container never restart-loops on it.
func (n *Node) joinAtStart(ctx context.Context, code joincode.Code) {
	_, err := n.join(ctx, code)
	refused, ok := errors.AsType[*pairing.Error](err)
	switch {
	case err == nil, ctx.Err() != nil:
	case ok && refused.Kind.Ignored():
		n.log.Info("join code ignored: " + refused.Reason)
	default:
		n.log.Warn("join refused", "reason", err.Error())
	}
}

func (n *Node) serveJoin(ctx context.Context, body json.RawMessage, send func(any) error) error {
	var req pairing.JoinRequest
	if err := control.Decode(body, &req); err != nil {
		return err
	}
	code, err := joincode.Parse(req.Code)
	if err != nil {
		return &control.Error{Code: control.CodeBadRequest, Message: err.Error()}
	}
	joined, err := n.join(ctx, code)
	if refused, ok := errors.AsType[*pairing.Error](err); ok {
		return send(pairing.JoinReply{Refused: refused})
	}
	if err != nil {
		return err
	}
	return send(pairing.JoinReply{Joined: joined.String()})
}

// Join joins the Node on env's data dir while no Node holds it: it takes
// the data dir lock, loads config and node.key, builds a link on env.Net,
// runs pairing.Claim and releases the lock. A held data dir is an
// *InUseError.
func Join(ctx context.Context, env Env, code joincode.Code) (pairing.Joined, error) {
	lk, err := lock.Acquire(env.DataDir.RunDir())
	if held, ok := errors.AsType[*lock.HeldError](err); ok {
		err = &InUseError{Dir: env.DataDir, PID: held.PID}
	}
	if err != nil {
		return pairing.Joined{}, err
	}
	joined, err := joinOffline(ctx, env, code)
	return joined, errors.Join(err, lk.Release())
}

func joinOffline(ctx context.Context, env Env, code joincode.Code) (pairing.Joined, error) {
	cfg, _, err := LoadConfig(env.DataDir, env.Environ)
	if err != nil {
		return pairing.Joined{}, err
	}
	key, err := link.OpenKey(env.DataDir.Path())
	if _, malformed := errors.AsType[*link.MalformedKeyError](err); malformed {
		err = &RecordError{Err: err}
	}
	if err != nil {
		return pairing.Joined{}, err
	}
	refuse := offline{env.DataDir}
	l, err := link.New(link.Config{
		Key:      key,
		Node:     nodeInfo(env.Version),
		Timings:  cfg.Timings.Link,
		Settings: cfg.Link,
		Net:      env.Net,
		Rand:     NewStreams(cfg.Seed).Stream("link"),
		Trust:    refuse,
		Lockstep: http.NotFoundHandler(),
		Pair:     refuse.Pair,
		Log:      slog.New(slog.DiscardHandler),
	})
	if err != nil {
		return pairing.Joined{}, err
	}
	defer func() { _ = l.Close() }()
	p, err := pairing.Open(pairing.Config{DataDir: env.DataDir.Path(), Link: l, Name: cfg.Name, Claims: refuse, Log: slog.New(slog.DiscardHandler)})
	if _, malformed := errors.AsType[*pairing.RecordError](err); malformed {
		err = &RecordError{Err: err}
	}
	if err != nil {
		return pairing.Joined{}, err
	}
	return p.Claim(ctx, code)
}

// offline is a stopped Node's claim, read from its files, and its link's
// Trust, which admits no peer: only this Node dials.
type offline struct{ dir DataDir }

func (o offline) Claim(context.Context) (system.Claim, error) {
	switch {
	case exists(o.dir.Path(), pairing.RecordFile):
		return system.Worker, nil
	case exists(o.dir.LibraryDB()):
		return system.Controller, nil
	}
	return system.Unclaimed, nil
}

func (offline) Admit(link.Key) link.Admission { return link.Refused{} }

func (offline) Pair(context.Context, link.Key, *linkv1.PairRequest) (*linkv1.PairResponse, error) {
	return notAController(), nil
}

func exists(elem ...string) bool {
	_, err := os.Stat(filepath.Join(elem...))
	return err == nil
}
