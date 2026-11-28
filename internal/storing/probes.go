package storing

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jatassi/slipstream/internal/media"
	mediav1 "github.com/jatassi/slipstream/internal/pb/slipstream/media/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/media/v1/mediav1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/storage"
)

// Probes is MediaService as the Controller reaches it on any Node, in
// process for its own files. It lives here, beside Remote, so a probe
// names its file by the Source a place does.
type Probes func(role.NodeID) mediav1connect.MediaServiceHandler

// Probe reads the media record of the file from on node. A file ffprobe
// can't read is *media.UnreadableError, a probe that ran out of time
// wraps media.ErrProbeTimeout, and a NodePath outside node's library roots,
// download folders and scratch folder is *OutsideError. Any other error is
// the call's, for rpc.NotSent and rpc.Lost to read.
func (p Probes) Probe(ctx context.Context, node role.NodeID, from Source) (media.Record, error) {
	req := &mediav1.ProbeRequest{}
	switch from := from.(type) {
	case Payload:
		req.Source = &mediav1.ProbeRequest_Payload{Payload: &mediav1.Payload{InfoHash: from.InfoHash[:], FileIndex: int64(from.FileIndex)}}
	case NodePath:
		req.Source = &mediav1.ProbeRequest_Path{Path: from.Abs}
	}
	return p.probe(ctx, node, req)
}

// ProbeIn reads the media record of rel in root, which node resolves under
// its own folders: Probe's errors, and *NotReady while node doesn't hold
// root ready.
func (p Probes) ProbeIn(ctx context.Context, node role.NodeID, root storage.UUID, rel Rel) (media.Record, error) {
	return p.probe(ctx, node, &mediav1.ProbeRequest{Source: &mediav1.ProbeRequest_Library{Library: &mediav1.Library{RootId: root.Bytes(), Rel: rel.String()}}})
}

func (p Probes) probe(ctx context.Context, node role.NodeID, req *mediav1.ProbeRequest) (media.Record, error) {
	res, err := p(node).Probe(ctx, req)
	if d, ok := rpc.DetailOf[*mediav1.Unprobed](err); ok {
		return media.Record{}, &media.UnreadableError{Line: d.GetLine()}
	}
	if _, ok := rpc.DetailOf[*mediav1.TimedOut](err); ok {
		return media.Record{}, fmt.Errorf("%w: %w", media.ErrProbeTimeout, err)
	}
	if d, ok := rpc.DetailOf[*mediav1.Outside](err); ok {
		return media.Record{}, &OutsideError{Path: d.GetPath()}
	}
	if err != nil {
		return media.Record{}, answer(err)
	}
	var rec media.Record
	if err := json.Unmarshal(res.GetRecord(), &rec); err != nil {
		return media.Record{}, fmt.Errorf("read the media record: %w", err)
	}
	return rec, nil
}
