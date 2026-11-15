package grabs

import (
	"context"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// triedRelease is a Release a failed Grab of the Work tried, which its
// failure rows refuse: known by its info hash, else by its Indexer and
// GUID.
type triedRelease struct {
	hash    *bencode.InfoHash
	indexer indexers.ID
	guid    string
}

// is reports whether r is t's Release.
func (t triedRelease) is(r indexers.Release) bool {
	if t.hash != nil && r.InfoHash != nil {
		return *t.hash == *r.InfoHash
	}
	return t.guid != "" && t.indexer == r.Indexer && t.guid == r.GUID
}

// triedOf is the Releases the failure rows of work refuse, which outlive
// the Grabs that wrote them.
func triedOf(ctx context.Context, r storage.Reader[file], work works.WorkID) ([]triedRelease, error) {
	rows, err := db.New(r).ListTried(ctx, work)
	if err != nil {
		return nil, err
	}
	out := make([]triedRelease, len(rows))
	for i, row := range rows {
		out[i] = triedRelease{indexer: row.IndexerID, guid: deref(row.Guid)}
		if row.InfoHash != nil {
			h := bencode.InfoHash(row.InfoHash)
			out[i].hash = &h
		}
	}
	return out, nil
}
