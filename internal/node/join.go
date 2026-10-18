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
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/joincode"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/node/internal/control"
	"github.com/jatassi/slipstream/internal/node/internal/lock"
	"github.com/jatassi/slipstream/internal/node/surface"
	"github.com/jatassi/slipstream/internal/pairing"
	linkv1 "github.com/jatassi/slipstream/internal/pb/slipstream/link/v1"
	"github.com/jatassi/slipstream/internal/system"
)

// awaitingEvery is how often a Join awaiting its Controller's dial says
// so again: to the log under join.code, and to slipstream join, whose
// hang-up a failed send finds.
const awaitingEvery = time.Minute

// join runs pairing.Claim under the claim lock. A running Worker half
// stays bound to its Controller, so a code that would re-pair it is
// refused. A Claim awaiting its Controller's dial calls awaiting and
// releases the lock for the wait, so a Controller claim meanwhile answers
// join_pending. Once Claim ends, a Node holding a record and no half
// starts the Worker half on it, whether the Join paired it or a re-pair at
// start was refused. A Join that never awaited closes the Setup gate; one
// that did leaves it to its next Setup operation, which getWorkerClaim
// reads the outcome through.
func (n *Node) join(ctx context.Context, code joincode.Code, awaiting func(pairing.Pending)) (pairing.Joined, error) {
	n.claimMu.Lock()
	pended, waited := false, false
	defer func() {
		if !pended {
			n.claimMu.Unlock()
		}
	}()
	if n.half.Load() != nil && n.pairing.Repairs(code) {
		return pairing.Joined{}, pairing.ErrRestartRequired
	}
	wasWorker := n.pairing.Paired()
	joined, err := n.pairing.Claim(ctx, code, func(p pairing.Pending) {
		pended, waited = true, true
		n.claimMu.Unlock()
		awaiting(p)
	})
	if pended {
		n.claimMu.Lock()
		pended = false
	}
	if n.half.Load() == nil && n.pairing.Paired() {
		if err := n.startWorker(context.WithoutCancel(ctx)); err != nil {
			n.stack.fail(fmt.Errorf("start the Worker half: %w", err))
			return pairing.Joined{}, err
		}
		if !wasWorker {
			n.log.Info("claimed", "claim", "worker")
		}
	}
	if err != nil {
		return pairing.Joined{}, err
	}
	if !waited {
		n.gate.Close()
	}
	return joined, nil
}

// repeat calls say at once and every awaitingEvery until the returned
// stop, which waits for it.
func repeat(say func() error) (stop func()) {
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Go(func() {
		tick := time.NewTicker(awaitingEvery)
		defer tick.Stop()
		for {
			if say() != nil {
				return
			}
			select {
			case <-tick.C:
			case <-done:
				return
			}
		}
	})
	return func() {
		close(done)
		wg.Wait()
	}
}

// joinAtStart runs join.code's Join. A refusal leaves the Node as it was,
// logged once, so a container never restart-loops on it. A Join awaiting
// its Controller's dial logs its lines at Warn every minute until it ends.
func (n *Node) joinAtStart(ctx context.Context, code joincode.Code) {
	stop := func() {}
	_, err := n.join(ctx, code, func(p pairing.Pending) {
		stop = repeat(func() error {
			for _, line := range p.Lines() {
				n.log.Warn(line)
			}
			return nil
		})
	})
	stop()
	refused, ok := errors.AsType[*pairing.Error](err)
	switch {
	case err == nil, ctx.Err() != nil:
	case ok && refused.Kind.Ignored():
		n.log.Info("join code ignored: " + refused.Reason)
	default:
		n.log.Warn("join refused", "reason", err.Error())
	}
}

// serveJoin is JoinOp. A Join awaiting its Controller's dial sends its
// lines every minute, and stops when a send fails, as slipstream join's
// Ctrl-C makes it. A --node-port this Node doesn't listen on is written to
// config.yaml and refused until a restart binds it.
func (n *Node) serveJoin(ctx context.Context, body json.RawMessage, send func(any) error) error {
	var req pairing.JoinRequest
	if err := control.Decode(body, &req); err != nil {
		return err
	}
	code, err := joincode.Parse(req.Code)
	if err != nil {
		return &control.Error{Code: control.CodeBadRequest, Message: err.Error()}
	}
	if req.NodePort != 0 && req.NodePort != n.linkBound.Port() {
		n.cfgMu.Lock()
		listen, err := n.env.configFile().setNodePort(req.NodePort)
		n.cfgMu.Unlock()
		if err != nil {
			return err
		}
		return send(pairing.JoinReply{Refused: &pairing.Error{
			Kind:   pairing.RestartRequired,
			Reason: fmt.Sprintf("the link listens on port %d until a restart binds link.listen %s: restart the Node, then run slipstream join again", n.linkBound.Port(), listen),
		}})
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	stop := func() {}
	joined, err := n.join(ctx, code, func(p pairing.Pending) {
		stop = repeat(func() error {
			err := send(pairing.JoinReply{Awaiting: p.Lines()})
			if err != nil {
				cancel()
			}
			return err
		})
	})
	stop()
	if refused, ok := errors.AsType[*pairing.Error](err); ok {
		return send(pairing.JoinReply{Refused: refused})
	}
	if err != nil {
		return err
	}
	return send(pairing.JoinReply{Joined: joined.String()})
}

// Join joins the Node on env's data dir while no Node holds it: it takes
// the data dir lock, writes nodePort to link.listen unless it is 0, loads
// config and node.key, builds a link on env.Net that listens on
// link.listen, runs pairing.Claim and releases the lock. A held data dir
// is an *InUseError.
func Join(ctx context.Context, env Env, code joincode.Code, nodePort uint16, awaiting func(pairing.Pending)) (pairing.Joined, error) {
	lk, err := lock.Acquire(env.DataDir.RunDir())
	if held, ok := errors.AsType[*lock.HeldError](err); ok {
		err = &InUseError{Dir: env.DataDir, PID: held.PID}
	}
	if err != nil {
		return pairing.Joined{}, err
	}
	joined, err := joinOffline(ctx, env, code, nodePort, awaiting)
	return joined, errors.Join(err, lk.Release())
}

func joinOffline(ctx context.Context, env Env, code joincode.Code, nodePort uint16, awaiting func(pairing.Pending)) (pairing.Joined, error) {
	if nodePort != 0 {
		if _, err := env.configFile().setNodePort(nodePort); err != nil {
			return pairing.Joined{}, err
		}
	}
	cfg, _, err := LoadConfig(env.DataDir, env.Environ, env.Types, env.Install)
	if err != nil {
		return pairing.Joined{}, err
	}
	key, err := openKey(env.DataDir)
	if err != nil {
		return pairing.Joined{}, err
	}
	refuse := offline{env}
	l, err := link.New(link.Config{
		Key:      key,
		Node:     nodeInfo(env.Version, env.Install),
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
	ln, err := env.Net.Listen("tcp", cfg.Link.Listen.String())
	if err != nil {
		return pairing.Joined{}, errors.Join(fmt.Errorf("listen for the link on %s: %w", cfg.Link.Listen, err), l.Close())
	}
	serve := l.Serve(ln)
	serving := make(chan struct{})
	go func() {
		defer close(serving)
		_ = serve(ctx)
	}()
	defer func() { <-serving }()
	defer func() { _ = l.Close() }()
	p, err := pairing.Open(pairing.Config{
		DataDir:   env.DataDir.Path(),
		Key:       key.Public(),
		Link:      l,
		Name:      cfg.Name,
		Claims:    refuse,
		AwaitDial: cfg.Timings.Link.AwaitDial,
		Log:       slog.New(slog.DiscardHandler),
	})
	if _, malformed := errors.AsType[*pairing.RecordError](err); malformed {
		err = &RecordError{Err: err}
	}
	if err != nil {
		return pairing.Joined{}, err
	}
	return p.Claim(ctx, code, awaiting)
}

// NodeAddress is the Node address of the Node on env's data dir: its key
// and the addresses its link advertises on link.listen, as a pending
// claim prints it.
func NodeAddress(env Env) (joincode.NodeAddress, error) {
	cfg, _, err := LoadConfig(env.DataDir, env.Environ, env.Types, env.Install)
	if err != nil {
		return joincode.NodeAddress{}, err
	}
	key, err := openKey(env.DataDir)
	if err != nil {
		return joincode.NodeAddress{}, err
	}
	addrs, ok := link.Advertising(cfg.Link)
	if !ok {
		return joincode.NodeAddress{}, fmt.Errorf("link.listen %s covers no address another machine can dial: set link.advertise", cfg.Link.Listen)
	}
	pub := key.Public()
	return joincode.NewNodeAddress(pub[:], addrs.All())
}

// offline is a stopped Node's claim, read from its files as a running
// Node reads its own: a Controller holds an Admin in library.db. Its link
// Trust admits no peer. Only a pending claim's await admits its
// Controller, ahead of Trust.
type offline struct{ env Env }

func (o offline) Claim(ctx context.Context) (system.Claim, error) {
	dir := o.env.DataDir
	worker, err := pairing.IsWorker(dir.Path())
	switch {
	case err != nil:
		return 0, err
	case worker:
		return system.Worker, nil
	case !exists(dir.LibraryDB()):
		return system.Unclaimed, nil
	}
	cfg, _, err := LoadConfig(dir, o.env.Environ, o.env.Types, o.env.Install)
	if err != nil {
		return 0, err
	}
	lib, err := openLibraryFiles(ctx, libraryFiles{
		dir:     dir,
		schemas: append(LibrarySchemas(o.env.Types), o.env.Test.Library...),
		timings: cfg.Timings,
		hub:     live.NewHub(surface.Events),
		log:     slog.New(slog.DiscardHandler),
	})
	if err != nil {
		return 0, err
	}
	admin, err := lib.ids.HasAdmin(ctx)
	if err := errors.Join(err, lib.close(ctx)); err != nil {
		return 0, err
	}
	if admin {
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

// openKey is link.OpenKey with a node.key the Node cannot read or use as a
// *RecordError.
func openKey(dir DataDir) (link.PrivateKey, error) {
	key, err := link.OpenKey(dir.Path())
	_, malformed := errors.AsType[*link.MalformedKeyError](err)
	_, unreadable := errors.AsType[*link.UnreadableKeyError](err)
	if malformed || unreadable {
		return key, &RecordError{Err: err}
	}
	return key, err
}
