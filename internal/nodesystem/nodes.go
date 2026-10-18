package nodesystem

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"time"

	"github.com/jatassi/slipstream/internal/backoff"
	systemv1 "github.com/jatassi/slipstream/internal/pb/slipstream/system/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/system/v1/systemv1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/system"
)

// FollowTimings bound the wait between reads of a log whose Node is
// offline.
type FollowTimings struct {
	RetryMin, RetryMax time.Duration
}

// Nodes is every Node's system as the Controller reaches it, its own
// included. Each method's error is ErrNotFound, ErrOffline, ErrMismatched,
// ErrAlreadyRestarting or a *Refusal, or else a failure.
type Nodes struct {
	on      func(role.NodeID) systemv1connect.NodeSystemServiceHandler
	name    func(context.Context, role.NodeID) (string, error)
	timings FollowTimings
	rand    *rand.Rand
}

// NewNodes reaches each Node through on, and names it by name, which
// answers ErrNotFound for a Node the roster lacks. r draws the retry
// delays.
func NewNodes(on func(role.NodeID) systemv1connect.NodeSystemServiceHandler, name func(context.Context, role.NodeID) (string, error), t FollowTimings, r *rand.Rand) *Nodes {
	return &Nodes{on: on, name: name, timings: t, rand: r}
}

// fail is a call's error as a domain error. A call the Node never
// answered, its link gone before or during it, finds the Node offline.
func fail(err error) error {
	if why, ok := rpc.NotSent(err); ok {
		switch why {
		case rpc.Mismatched:
			return ErrMismatched
		case rpc.NotHolder:
			return ErrNotFound
		case rpc.Disconnected:
		}
		return fmt.Errorf("%w: %w", ErrOffline, err)
	}
	code := rpc.CodeOf(err)
	if code == rpc.CodeAborted {
		return ErrAlreadyRestarting
	}
	if !rpc.Answered(err) || code == rpc.CodeFailedPrecondition || code == rpc.CodeUnavailable || code == rpc.CodeCanceled || code == rpc.CodeDeadlineExceeded {
		return fmt.Errorf("%w: %w", ErrOffline, err)
	}
	return err
}

func (n *Nodes) Facts(ctx context.Context, id role.NodeID) (system.Facts, error) {
	res, err := n.on(id).GetFacts(ctx, &systemv1.GetFactsRequest{})
	if err != nil {
		return system.Facts{}, fail(err)
	}
	return factsFromWire(res)
}

func (n *Nodes) Config(ctx context.Context, id role.NodeID) (Config, error) {
	res, err := n.on(id).GetConfig(ctx, &systemv1.GetConfigRequest{})
	if err != nil {
		return Config{}, fail(err)
	}
	return configFromWire(res.GetConfig()), nil
}

// SetConfig changes every key or none. A link lost mid-call answers
// ErrOffline though the change may have been saved; the Node's next
// settings revision tells.
func (n *Nodes) SetConfig(ctx context.Context, id role.NodeID, changes []Change) (Config, error) {
	req := &systemv1.SetConfigRequest{Changes: make([]*systemv1.Change, len(changes))}
	for i, c := range changes {
		req.Changes[i] = changeToWire(c)
	}
	res, err := n.on(id).SetConfig(ctx, req)
	if err != nil {
		return Config{}, fail(err)
	}
	if r := res.GetRefused(); r != nil {
		refused, err := refusalFromWire(r)
		if err != nil {
			return Config{}, err
		}
		return Config{}, refused
	}
	return configFromWire(res.GetConfig()), nil
}

// Restart answers once the Node has accepted the restart. A Node's stop
// can drop the link before its answer crosses it, so a sent call the Node
// never answered counts as accepted, reported from its facts before the
// call, with no NextPort.
func (n *Nodes) Restart(ctx context.Context, id role.NodeID) (Restarting, error) {
	f, err := n.Facts(ctx, id)
	if err != nil {
		return Restarting{}, err
	}
	res, err := n.on(id).Restart(ctx, &systemv1.RestartRequest{})
	if _, notSent := rpc.NotSent(err); err != nil && !notSent && !rpc.Answered(err) && ctx.Err() == nil {
		return Restarting{StartedAt: f.StartedAt, BoundPort: f.Listeners.HTTP.Port()}, nil
	}
	if err != nil {
		return Restarting{}, fail(err)
	}
	return Restarting{
		StartedAt: time.Unix(0, res.GetStartedAtUnixNano()),
		BoundPort: uint16(res.GetBoundPort()), //nolint:gosec // a port the Node bound
		NextPort:  uint16(res.GetNextPort()),  //nolint:gosec // a port the Node saved
	}, nil
}

// Reach dials t from id.
func (n *Nodes) Reach(ctx context.Context, id role.NodeID, t Target) (Reached, error) {
	res, err := n.on(id).Reach(ctx, &systemv1.ReachRequest{Host: t.Host, Port: uint32(t.Port), Tls: t.TLS})
	if err != nil {
		return Reached{}, fail(err)
	}
	return Reached{OK: res.GetOk(), Took: time.Duration(res.GetMillis()) * time.Millisecond, Err: res.GetError()}, nil
}

// Follow sends id's log after after until ctx ends: the ring first, then
// each record as it comes, a reset when after was lapped or came from an
// earlier process, and an empty batch whenever the Node's poll passes
// with none. It rides out the Node going offline, resuming from the last
// record sent, and returns ErrNotFound or ErrMismatched.
func (n *Nodes) Follow(ctx context.Context, id role.NodeID, after uint64, send func(system.LogBatch) error) error {
	call := n.on(id)
	for retry := 0; ctx.Err() == nil; {
		res, err := call.ReadLogs(ctx, &systemv1.ReadLogsRequest{After: after})
		if err != nil {
			if err := fail(err); ctx.Err() == nil && !errors.Is(err, ErrOffline) {
				return err
			}
			select {
			case <-ctx.Done():
			case <-time.After(backoff.Delay(n.timings.RetryMin, n.timings.RetryMax, retry, n.rand)):
			}
			retry++
			continue
		}
		retry = 0
		b, err := batchFromWire(res)
		if err != nil {
			return err
		}
		if b.Reset != nil {
			after = b.Reset.After
		}
		if len(b.Records) > 0 {
			after = b.Records[len(b.Records)-1].Seq
		}
		if err := send(b); err != nil {
			return err
		}
	}
	return nil
}

// NamedArchive is a Node's log zip as a download: its file name and a
// body that reads it from the Node chunk by chunk.
type NamedArchive struct {
	Name string
	Size int64
	Body io.ReadCloser
}

// Archive zips id's log folder. The name is the roster's name for id and
// the stamp of the Node that zipped it.
func (n *Nodes) Archive(ctx context.Context, id role.NodeID) (NamedArchive, error) {
	name, err := n.name(ctx, id)
	if err != nil {
		return NamedArchive{}, err
	}
	call := n.on(id)
	res, err := call.OpenLogArchive(ctx, &systemv1.OpenLogArchiveRequest{})
	if err != nil {
		return NamedArchive{}, fail(err)
	}
	return NamedArchive{
		Name: "slipstream-logs-" + name + "-" + res.GetStamp() + ".zip",
		Size: res.GetSize(),
		Body: &archiveReader{ctx: ctx, call: call, id: res.GetId(), size: res.GetSize()},
	}, nil
}

// archiveReader reads an open archive from its Node one chunk at a time.
// Closing it early leaves the rest to the Node's TTL.
type archiveReader struct {
	ctx    context.Context //nolint:containedctx // the download's request, which each chunk's call serves
	call   systemv1connect.NodeSystemServiceHandler
	id     string
	size   int64
	offset int64
	chunk  []byte
}

func (a *archiveReader) Read(p []byte) (int, error) {
	if len(a.chunk) == 0 {
		if a.offset >= a.size {
			return 0, io.EOF
		}
		res, err := a.call.ReadLogArchive(a.ctx, &systemv1.ReadLogArchiveRequest{Id: a.id, Offset: a.offset})
		if err != nil {
			return 0, fail(err)
		}
		if len(res.GetData()) == 0 {
			return 0, io.ErrUnexpectedEOF
		}
		a.chunk = res.GetData()
		a.offset += int64(len(a.chunk))
	}
	n := copy(p, a.chunk)
	a.chunk = a.chunk[n:]
	return n, nil
}

func (a *archiveReader) Close() error { return nil }
