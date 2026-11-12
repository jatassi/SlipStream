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
// Incomplete and Ready from its latest report. A Torrent ref with no
// Torrent or no report, or one removed other than by a write-off, is
// Unknown, and so is every Path ref. Moved closes when a report or
// removal changes what this reads.
func (c *Clients) ReadinessIn(ctx context.Context, rd storage.Reader[file], node role.NodeID, refs []source.Ref) (map[source.Ref]source.Readiness, error) {
	out := make(map[source.Ref]source.Readiness, len(refs))
	var torrents []source.Torrent
	var hashes [][]byte
	for _, ref := range refs {
		switch r := ref.(type) {
		case source.Torrent:
			torrents, hashes = append(torrents, r), append(hashes, r.Hash[:])
		case source.Path:
			out[r] = source.Readiness{}
		}
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
