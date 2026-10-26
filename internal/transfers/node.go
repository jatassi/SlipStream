package transfers

import (
	"context"
	"log/slog"
	"net"

	"github.com/jatassi/slipstream/internal/link"
	transferv1 "github.com/jatassi/slipstream/internal/pb/slipstream/transfer/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/transfers/internal/engine"
	"github.com/jatassi/slipstream/internal/transfers/source"
)

// Opener resolves a source on this Node: a path, or *source.Absent.
// downloading.Client is the Torrent one.
type Opener interface {
	Open(ref source.Ref) (string, error)
}

// Stager is a root destination's gate on this Node: the folder a Transfer
// stages in at place, or *storing.NotReady while the root takes no bytes.
// storing.Store is one.
type Stager interface {
	Staging(ctx context.Context, root storage.UUID, place int64, transfer storage.UUID) (string, error)
}

// NodeConfig builds a Node's half of Transfers. Every Node has one.
type NodeConfig struct {
	Local   *storage.DB[file]
	Disk    storing.Disk
	Link    *link.Link
	Sources Opener
	Roots   Stager
	Timings Timings
	Hooks   Hooks
	Log     *slog.Logger
}

// Node sends and receives the Transfers its transfers intent makes
// active. It is the link's LaneHandler, and serves route 3's kind-2
// streams with ServeStream.
type Node struct{ e *engine.Engine }

func NewNode(cfg NodeConfig) *Node {
	t := cfg.Timings
	return &Node{e: engine.New(engine.Config{
		Local:   cfg.Local,
		Disk:    cfg.Disk,
		Link:    cfg.Link,
		Sources: cfg.Sources,
		Roots:   cfg.Roots,
		Timings: engine.Timings{
			Chunk: t.Chunk, Report: t.Report, Dial: t.Dial, Estimate: t.Estimate, Ahead: t.Ahead,
			SyncBytes: t.SyncBytes, SyncEvery: t.SyncEvery, RetryMin: t.RetryMin, RetryMax: t.RetryMax,
			Damaged: t.Damaged, Piece: t.Piece,
		},
		Hooks: engine.Hooks{Read: cfg.Hooks.Read, Sent: cfg.Hooks.Sent},
		Log:   cfg.Log,
	})}
}

// Run follows the transfers intent until ctx ends: it runs a sender or a
// receiver for each active Transfer while this Node is in step, stops
// them and keeps what they verified while it isn't, and deletes the
// staging of each Transfer the intent no longer names.
func (n *Node) Run(ctx context.Context, h role.Holder[*transferv1.Intent, *transferv1.Facts, *transferv1.Result]) error {
	return n.e.Run(ctx, h)
}

// Release deletes every staging folder and resume record: the Node was
// removed.
func (n *Node) Release(ctx context.Context) error { return n.e.Release(ctx) }

// Admit admits a lane from key while this Node is in step and runs a
// Transfer with key as its peer.
func (n *Node) Admit(key link.Key) bool { return n.e.Admit(key) }

// Serve takes a lane key opened, until the Transfer it names lets it go.
func (n *Node) Serve(ctx context.Context, key link.Key, c net.Conn) { n.e.Serve(ctx, key, c) }

// ServeStream takes a route 3 stream as Serve takes a lane.
func (n *Node) ServeStream(key link.Key, s link.Stream) { n.e.ServeStream(key, s) }
