package grabs

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
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
func (gr *Grabs) Blocklist(ctx context.Context, _ identity.Admin, q BlocklistQuery) (BlocklistPage, error) {
	var page BlocklistPage
	err := gr.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		qs := db.New(r)
		total, err := qs.CountBlocked(ctx, q.Title)
		if err != nil {
			return err
		}
		rows, err := qs.ListBlocked(ctx, db.ListBlockedParams{Title: q.Title, Skip: int64(q.Offset), PageSize: BlocklistPageSize})
		if err != nil {
			return err
		}
		ids := make([]works.WorkID, len(rows))
		for i, row := range rows {
			ids[i] = row.WorkID
		}
		summaries, err := works.SummariesOf(ctx, r, ids)
		if err != nil {
			return err
		}
		known, err := gr.cfg.Targets.CutsIn(ctx, r)
		if err != nil {
			return err
		}
		page.Total = int(total)
		for _, row := range rows {
			e := entryOf(row)
			if w, ok := summaries[e.Work]; ok {
				e.WorkTitle = w.Label
				e.Words = release.Describe(release.Title(e.Title, gr.cfg.Names[w.ContentType], known.For(w.ContentType)))
			}
			page.Entries = append(page.Entries, e)
		}
		return nil
	})
	return page, err
}

// Unblock removes the entries of hashes. Removing one twice changes
// nothing.
func (gr *Grabs) Unblock(ctx context.Context, _ identity.Admin, hashes []bencode.InfoHash) error {
	keys := make([][]byte, len(hashes))
	for i := range hashes {
		keys[i] = hashes[i][:]
	}
	err := gr.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		return db.New(tx).DeleteBlocked(ctx, keys)
	})
	if err != nil {
		return err
	}
	gr.cfg.BlocklistChanged.Publish(ctx, BlocklistChanged{})
	return nil
}

func entryOf(row db.GrabsBlocklist) BlockEntry {
	e := BlockEntry{
		InfoHash: bencode.InfoHash(row.InfoHash), Title: row.ReleaseTitle, Size: row.ReleaseSize,
		Indexer: row.IndexerID, IndexerName: deref(row.IndexerName), Tracker: row.TrackerID, Work: row.WorkID, Grab: row.GrabID,
		Cause: row.Cause, Client: deref(row.ClientName), Added: row.AddedAt.Time(),
	}
	if e.Cause == CauseMarkedFailed || e.Cause == CauseByHand {
		e.By = &Actor{Name: deref(row.AddedByName)}
		if !row.AddedBy.IsZero() {
			e.By.Account = &row.AddedBy
		}
	}
	e.Because = e.because()
	return e
}

func (e BlockEntry) because() string {
	switch e.Cause {
	case CauseUnregistered:
		if e.IndexerName == "" {
			return "Its tracker reported it unregistered."
		}
		return e.IndexerName + " reported it unregistered."
	case CauseNoPayload:
		return "It never downloaded any data."
	case CauseEngineError:
		return e.Client + " couldn't read it."
	case CauseMarkedFailed:
		if e.By.Name == "" {
			return "Marked failed through the API key."
		}
		return e.By.Name + " marked its download failed."
	case CauseByHand:
		if e.By.Name == "" {
			return "Blocklisted through the API key when its Grab was canceled."
		}
		return e.By.Name + " blocklisted it when canceling its Grab."
	}
	panic("grabs: no words for the Blocklist cause " + string(e.Cause))
}

// blocks is Blocklist entries by info hash.
type blocks map[bencode.InfoHash]BlockEntry

// blocksOf is the entries, read in r, of every info hash as were listed
// with, their copies' included. A search reads only these, in the Write
// that decides, so an entry added while its Indexers answered still
// counts.
func blocksOf(ctx context.Context, r storage.Reader[file], as []assessed) (blocks, error) {
	var keys [][]byte
	for _, a := range as {
		for _, h := range a.hashes() {
			keys = append(keys, h[:])
		}
	}
	rows, err := db.New(r).ListBlocksOf(ctx, keys)
	if err != nil {
		return nil, err
	}
	out := blocks{}
	for _, row := range rows {
		e := entryOf(row)
		out[e.InfoHash] = e
	}
	return out, nil
}

// of is the entry b holds for a's Release, by its info hash or a copy's.
func (b blocks) of(a assessed) (BlockEntry, bool) {
	for _, h := range a.hashes() {
		if e, ok := b[h]; ok {
			return e, true
		}
	}
	return BlockEntry{}, false
}

// gate is as with the own refusal blocklisted on each Release b holds. It
// isn't hard, so the Release keeps every verdict, and picks skip it.
func (b blocks) gate(as []assessed) []assessed {
	out := slices.Clone(as)
	for i, a := range out {
		if _, ok := b.of(a); ok {
			out[i].refusals = append(slices.Clip(a.refusals), own(targets.Blocklisted, ""))
		}
	}
	return out
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

// Block adds e in tx. It is the Blocklist's one writer, so it holds the
// fence: while a restored Controller is Reconciling it refuses with
// ErrReconciling. A hash already listed keeps its first entry.
func (b BlocklistOn) Block(ctx context.Context, tx *storage.Tx[file], e BlockEntry) error {
	reconciling, err := b.Reconciling(ctx, tx)
	if err != nil {
		return err
	}
	if reconciling {
		return ErrReconciling
	}
	p := db.InsertBlockedParams{
		InfoHash: e.InfoHash[:], ReleaseTitle: e.Title, ReleaseSize: e.Size, IndexerID: e.Indexer, IndexerName: orNil(e.IndexerName),
		TrackerID: e.Tracker, WorkID: e.Work, GrabID: e.Grab, Cause: e.Cause, ClientName: orNil(e.Client), AddedAt: storage.Now(),
	}
	if e.By != nil {
		p.AddedByName = orNil(e.By.Name)
		if e.By.Account != nil {
			p.AddedBy = *e.By.Account
		}
	}
	return db.New(tx).InsertBlocked(ctx, p)
}

// MarkedFailed adds an entry for each of ids whose Torrent a Grab at
// downloading holds, by by, in the caller's Write.
func (b BlocklistOn) MarkedFailed(ctx context.Context, tx *storage.Tx[file], ids []downloadclients.ID, by identity.Admin) error {
	rows, err := db.New(tx).ListDownloadingGrabsOfTorrents(ctx, ids)
	if err != nil {
		return err
	}
	torrents, err := downloadclients.Outcomes(ctx, tx, ids)
	if err != nil {
		return err
	}
	actor := actorOf(by)
	for _, row := range rows {
		e := BlockEntry{
			InfoHash: torrents[row.TorrentID].InfoHash, Title: row.ReleaseTitle, Size: row.ReleaseSize, Indexer: row.IndexerID,
			Tracker: row.TrackerID, Work: row.WorkID, Grab: row.ID, Cause: CauseMarkedFailed, By: &actor,
		}
		if !row.IndexerID.IsZero() {
			idx, err := indexers.Get(ctx, tx, row.IndexerID)
			if err != nil {
				return err
			}
			e.IndexerName = idx.Name
		}
		if err := b.Block(ctx, tx, e); err != nil {
			return err
		}
	}
	return nil
}

// actorOf is by as an Actor: its account and name, or the API key's
// empty Actor.
func actorOf(by identity.Admin) Actor {
	a, ok := by.Account()
	if !ok {
		return Actor{}
	}
	p := a.Profile()
	return Actor{Account: &p.ID, Name: p.DisplayName}
}
