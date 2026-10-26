package downloadclients

import (
	"context"
	"fmt"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloadclients/internal/db"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/transfers/source"
)

// ReadinessIn is each of refs' readiness on node, read in rd's snapshot
// from node's latest Torrent of the ref's hash: WrittenOff from its
// removal, then FailedSource with the report's error, Checking,
// Incomplete and Ready from its latest report. A ref with no Torrent or
// no report, or one removed other than by a write-off, is Unknown. Moved
// closes when a report or removal changes what this reads.
func (c *Clients) ReadinessIn(ctx context.Context, rd storage.Reader[file], node role.NodeID, refs []source.Ref) (map[source.Ref]source.Readiness, error) {
	torrents := make([]source.Torrent, len(refs))
	hashes := make([][]byte, len(refs))
	for i, ref := range refs {
		t, ok := ref.(source.Torrent)
		if !ok {
			return nil, fmt.Errorf("a %s source isn't a Torrent", ref.Kind())
		}
		torrents[i], hashes[i] = t, t.Hash[:]
	}
	rows, err := db.New(rd).ListSources(ctx, db.ListSourcesParams{NodeID: node, Hashes: hashes})
	if err != nil {
		return nil, err
	}
	latest := make(map[bencode.InfoHash]source.Readiness, len(rows))
	for _, row := range rows {
		r, err := readinessOf(row)
		if err != nil {
			return nil, fmt.Errorf("torrent %x: %w", row.InfoHash, err)
		}
		latest[bencode.InfoHash(row.InfoHash)] = r
	}
	out := make(map[source.Ref]source.Readiness, len(refs))
	for _, t := range torrents {
		out[t] = latest[t.Hash]
	}
	return out, nil
}

func readinessOf(row db.ListSourcesRow) (source.Readiness, error) {
	r := source.Readiness{Name: row.Name}
	if row.RemovedReason != nil {
		if RemoveReason(*row.RemovedReason) == NodeWrittenOff {
			r.State = source.WrittenOff
		}
		return r, nil
	}
	rep, err := reportOf(row.Report)
	if err != nil || rep == nil {
		return r, err
	}
	switch {
	case rep.Engine == EngineFailed:
		r.State, r.Detail = source.FailedSource, rep.Error
	case rep.Engine == EngineChecking:
		r.State = source.Checking
	case !rep.Complete:
		r.State = source.Incomplete
	default:
		r.State = source.Ready
	}
	return r, nil
}
