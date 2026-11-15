package grabs

import (
	"context"

	"github.com/jatassi/slipstream/internal/bencode"
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

// triedOf is the Releases work's refusing failure rows name, read in r.
// No failure row is written yet, so none refuses.
func triedOf(context.Context, storage.Reader[file], works.WorkID) ([]triedRelease, error) {
	return nil, nil
}
