package grabs

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/magnet"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/storage"
)

const (
	MagnetInvalid Code = "magnet_invalid"
	NoMetadata    Code = "no_metadata"
	NoPeerSource  Code = "no_peer_source"
)

var unresolvedCode = map[downloadclients.UnresolvedReason]Code{
	downloadclients.TimedOut:     NoMetadata,
	downloadclients.NoPeerSource: NoPeerSource,
	downloadclients.EngineError:  DownloadFailed,
}

// again runs the same step at once, with no backoff: a long poll paced
// it.
type again struct{}

func (again) outcome() {}

// resolve gets the .torrent of link: from a Torrent of its info hash the
// client recorded, else from peers through the Downloading Node to,
// until the deadline the metadata wait sets from the Grab's first ask.
// Each call is one long poll, and the step runs again at once while the
// Node is still looking.
func (gr *Grabs) resolve(ctx context.Context, g Grab, link magnet.Link, to role.NodeID) outcome {
	if h := g.Release.InfoHash; h != nil && *h != link.InfoHash {
		return fail{InfohashMismatch, fmt.Sprintf("The magnet link's info hash is %s and the Indexer's is %s", link.InfoHash, h)}
	}
	var (
		info  []byte
		known bool
		wait  time.Duration
	)
	err := gr.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		if info, known, err = downloadclients.KnownInfo(ctx, r, link.InfoHash); err != nil || known {
			return err
		}
		th, err := downloadclients.ThresholdsIn(ctx, r)
		wait = th.MetadataWait
		return err
	})
	if err != nil {
		return local(ctx, err)
	}
	if known {
		return composed(g, link, info, to)
	}
	started := g.MetadataSince
	if started.IsZero() {
		var at storage.Time
		err := gr.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
			var err error
			at, err = db.New(tx).StartMetadataSearch(ctx, db.StartMetadataSearchParams{At: storage.Now(), ID: g.ID})
			return err
		})
		if errors.Is(err, storage.ErrNoRows) {
			return stopped{}
		}
		if err != nil {
			return local(ctx, err)
		}
		started = at.Time()
	}
	endsAt := started.Add(wait)
	timedOut := fail{NoMetadata, downloadclients.Unresolved{Reason: downloadclients.TimedOut}.Words(wait)}
	if !time.Now().Before(endsAt) {
		return timedOut
	}
	res, err := gr.cfg.Downloads.Resolve(ctx, to, link, endsAt)
	if err != nil {
		return unresolvable(ctx, err)
	}
	switch res := res.(type) {
	case downloadclients.Resolved:
		return composed(g, link, res.Info, to)
	case downloadclients.Searching:
		if !time.Now().Before(endsAt) {
			return timedOut
		}
		if gr.searching.set(g.ID, res.Peers) {
			gr.cfg.Changed.Publish(ctx, Changed{GrabID: g.ID.String(), WorkID: g.WorkID.String()})
		}
		return again{}
	case downloadclients.Unresolved:
		return fail{unresolvedCode[res.Reason], res.Words(wait)}
	}
	panic("grabs: unknown resolution")
}

// composed is the advance to Downloading with the .torrent of link and
// info, which the Downloading Node to fetched from peers.
func composed(g Grab, link magnet.Link, info []byte, to role.NodeID) outcome {
	raw, err := link.Torrent(info)
	switch {
	case errors.Is(err, magnet.ErrWrongInfo):
		return fail{InfohashMismatch, err.Error()}
	case errors.Is(err, bencode.ErrV2Only):
		return fail{TorrentV2Only, err.Error()}
	case err != nil:
		return fail{TorrentInvalid, err.Error()}
	}
	return decoded(g, raw, fetched{by: to, to: to})
}

// unresolvable is the outcome of a Resolve call that failed. A Node that
// holds the hash waits for KnownInfo to read it, and a Node that takes no
// new torrent, or that the call never reached, waits for the roster. A
// call lost or cut short asks again, since Resolve joins the search.
func unresolvable(ctx context.Context, err error) outcome {
	_, notSent := rpc.NotSent(err)
	switch {
	case ctx.Err() != nil:
		return stopped{}
	case errors.Is(err, downloadclients.ErrNotTaking), notSent:
		return wait{cause: err, roster: true}
	case errors.Is(err, downloadclients.ErrHeld), rpc.Lost(err), rpc.CutShort(err):
		return wait{cause: err}
	}
	return fail{DownloadFailed, err.Error()}
}

// searching is the peers each fetching Grab's metadata search had at its
// Downloading Node's last response, for the Grab's row.
type searching struct {
	mu    sync.Mutex
	peers map[GrabID]int
}

// set records peers for id and reports whether they moved.
func (s *searching) set(id GrabID, peers int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	was, ok := s.peers[id]
	s.peers[id] = peers
	return !ok || was != peers
}

func (s *searching) drop(id GrabID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.peers, id)
}

func (s *searching) of(id GrabID) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.peers[id]
}

// MetadataSearch is a fetching Grab's search for its magnet link's
// metadata.
type MetadataSearch struct {
	// Peers is the peers connected at the Downloading Node's last
	// response.
	Peers int
	// EndsAt is when the Grab fails unless a peer has sent the metadata.
	EndsAt time.Time
}

// metadataSearch is g's search while it fetches from a magnet link and
// has asked for its metadata, nil otherwise.
func (gr *Grabs) metadataSearch(g Grab, wait time.Duration) *MetadataSearch {
	if g.Stage != Fetching || g.MetadataSince.IsZero() {
		return nil
	}
	return &MetadataSearch{Peers: gr.searching.of(g.ID), EndsAt: g.MetadataSince.Add(wait)}
}
