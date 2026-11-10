package grabs

import (
	"context"
	"errors"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// BlockCauses is every BlockCause, the list the schema's CHECK holds.
var BlockCauses = []BlockCause{CauseUnregistered, CauseNoPayload, CauseEngineError, CauseMarkedFailed, CauseByHand}

// BlockEntry is one Release on the Blocklist.
type BlockEntry struct {
	InfoHash bencode.InfoHash
	Title    string
	Size     int64
	// Words is the Release in words, as release.Describe writes it.
	Words string
	// Indexer is zero once the Indexer is deleted. IndexerName is its name
	// then, "" when no Indexer is known.
	Indexer     indexers.ID
	IndexerName string
	Tracker     indexers.TrackerID
	// Work is zero once the Work is deleted.
	Work      works.WorkID
	WorkTitle string
	Grab      GrabID
	Cause     BlockCause
	// Because is the cause in a sentence: "TorrentLeech reported it
	// unregistered.", "Jackson marked its download failed."
	Because string
	Client  string
	// By is the Admin who added it, nil for a release fault.
	By    *Actor
	Added time.Time
}

// BlocklistQuery is a page of the Blocklist: entries whose title holds
// Title, from Offset.
type BlocklistQuery struct {
	Title  string
	Offset int
}

// BlocklistPage is up to BlocklistPageSize entries, newest first, and how
// many match in all.
type BlocklistPage struct {
	Entries []BlockEntry
	Total   int
}

const BlocklistPageSize = 50

// Blocklist is a page of entries, newest first.
func (gr *Grabs) Blocklist(context.Context, identity.Admin, BlocklistQuery) (BlocklistPage, error) {
	return BlocklistPage{}, ErrNotBuilt
}

// Unblock removes the entries of hashes. Removing one twice changes
// nothing.
func (gr *Grabs) Unblock(context.Context, identity.Admin, []bencode.InfoHash) error {
	return ErrNotBuilt
}

// errBlocked rolls back a fetch's commit whose .torrent the Blocklist
// holds. commit commits it again as a drop with Blocklisted.
var errBlocked = errors.New("the Blocklist holds the fetched torrent")

const blockedWords = "The Release's torrent is on the Blocklist."

// blockedHash reports whether h is on the Blocklist, read in r: the check
// after a fetch, which a Release listed without an info hash can't pass
// unseen.
func blockedHash(ctx context.Context, r storage.Reader[file], h bencode.InfoHash) (bool, error) {
	return db.New(r).IsBlocked(ctx, h[:])
}

// BlocklistOn is downloadclients.Blocklist over grabs' tables. It is
// stateless, as TorrentUses is.
type BlocklistOn struct {
	// Reconciling is whether a restored Controller is Reconciling, read in
	// the caller's snapshot. Nothing is blocklisted while it is.
	Reconciling func(ctx context.Context, r storage.Reader[file]) (bool, error)
}

// MarkedFailed adds an entry for each of ids whose Torrent a Grab at
// downloading holds, by by, in the caller's Write.
func (BlocklistOn) MarkedFailed(context.Context, *storage.Tx[file], []downloadclients.ID, identity.Admin) error {
	return nil
}
