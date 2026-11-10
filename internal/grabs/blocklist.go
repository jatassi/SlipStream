package grabs

import (
	"context"
	"errors"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/storage"
)

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
