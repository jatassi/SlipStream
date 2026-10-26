package downloadclients

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloadclients/internal/db"
	"github.com/jatassi/slipstream/internal/magnet"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
)

// Resolution is how a Downloading Node's search for a magnet link's
// metadata stands.
type Resolution interface{ resolution() }

type (
	// Resolved is the info dictionary a peer sent.
	Resolved struct{ Info []byte }
	// Searching is a search still running on the Node.
	Searching struct {
		Since time.Time
		// Peers is the peers connected now.
		Peers int
	}
	// Unresolved is a search that ended without the metadata.
	Unresolved struct {
		Reason UnresolvedReason
		// MostPeers is the most peers connected at once.
		MostPeers int
		// TrackerMessage is the last failure reason a tracker responded
		// with, verbatim.
		TrackerMessage string
		// Error is the engine's error text for EngineError.
		Error string
	}
)

func (Resolved) resolution()   {}
func (Searching) resolution()  {}
func (Unresolved) resolution() {}

type UnresolvedReason uint8

const (
	// TimedOut: no peer sent the metadata by the search's deadline.
	TimedOut UnresolvedReason = iota + 1
	// NoPeerSource: the link names no tracker and the Node's DHT is off.
	NoPeerSource
	// EngineError: the Node's torrent engine failed the search.
	EngineError
)

var reasonOnWire = map[dlv1.UnresolvedReason]UnresolvedReason{
	dlv1.UnresolvedReason_UNRESOLVED_REASON_TIMED_OUT:      TimedOut,
	dlv1.UnresolvedReason_UNRESOLVED_REASON_NO_PEER_SOURCE: NoPeerSource,
	dlv1.UnresolvedReason_UNRESOLVED_REASON_ENGINE_ERROR:   EngineError,
}

var (
	// ErrHeld is a Node that already holds a torrent of the link's info
	// hash, whose .torrent KnownInfo reads.
	ErrHeld = errors.New("the Node already holds a torrent of this info hash")
	// ErrNotTaking is a Node whose torrent client takes no new torrent now.
	ErrNotTaking = errors.New("the Node's torrent client takes no new torrent now")
)

// Resolve is one long poll of node's search for link's metadata, which
// ends at endsAt unless a peer sent it. It fails with ErrHeld,
// ErrNotTaking or the call's error.
func (c *Clients) Resolve(ctx context.Context, node role.NodeID, link magnet.Link, endsAt time.Time) (Resolution, error) {
	res, err := c.cfg.Remote(node).Resolve(ctx, &dlv1.ResolveRequest{
		InfoHash: link.InfoHash[:], Trackers: link.Trackers, Name: link.Name, EndsAtUnixMs: endsAt.UnixMilli(),
	})
	if rpc.Answered(err) && rpc.CodeOf(err) == rpc.CodeUnavailable {
		return nil, ErrNotTaking
	}
	if err != nil {
		return nil, err
	}
	switch a := res.GetAnswer().(type) {
	case *dlv1.ResolveResponse_Info:
		return Resolved{Info: a.Info}, nil
	case *dlv1.ResolveResponse_Searching:
		return Searching{Since: time.UnixMilli(a.Searching.GetSinceUnixMs()).UTC(), Peers: int(a.Searching.GetPeers())}, nil
	case *dlv1.ResolveResponse_Held:
		return nil, ErrHeld
	case *dlv1.ResolveResponse_Unresolved:
		u := a.Unresolved
		reason, ok := reasonOnWire[u.GetReason()]
		if !ok {
			return nil, fmt.Errorf("the Node's metadata search ended for an unknown reason %v", u.GetReason())
		}
		return Unresolved{Reason: reason, MostPeers: int(u.GetMostPeers()), TrackerMessage: u.GetTrackerMessage(), Error: u.GetError()}, nil
	}
	return nil, errors.New("the Node's response to Resolve has no answer")
}

// Words is the failure message of a Grab whose magnet link's search ended
// as u after running wait.
func (u Unresolved) Words(wait time.Duration) string {
	var msg string
	switch u.Reason {
	case NoPeerSource:
		return "The magnet link names no tracker, and DHT is off on the Node that downloads it"
	case EngineError:
		return "The metadata search failed with a torrent engine error: " + u.Error
	case TimedOut:
		msg = "No peer sent the metadata in " + setting.Words(wait)
	}
	switch {
	case u.MostPeers == 1:
		msg += ", and 1 peer connected"
	case u.MostPeers > 1:
		msg += fmt.Sprintf(", and at most %d peers connected at once", u.MostPeers)
	}
	if u.TrackerMessage != "" {
		msg += ". The tracker responded: " + u.TrackerMessage
	}
	return msg
}

// KnownInfo is the info dictionary of the newest Torrent of h on any
// Node, live or removed, false when the client never recorded one.
func KnownInfo(ctx context.Context, r storage.Reader[file], h bencode.InfoHash) (info []byte, known bool, err error) {
	raw, err := db.New(r).GetMetainfoOfHash(ctx, h[:])
	if errors.Is(err, storage.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	m, err := bencode.DecodeMetainfo(raw)
	if err != nil {
		return nil, false, fmt.Errorf("the metainfo of torrent %s: %w", h, err)
	}
	return m.RawInfo, true, nil
}
