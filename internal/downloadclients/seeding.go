package downloadclients

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloadclients/internal/db"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/trackers/seeding"
)

// Seeder is the seeding side of the Trackers, as trackers answers it.
// node/halves.go binds it late, since trackers reads Torrents through
// Clients.
type Seeder interface {
	// Judge is each row's Tracker and its info hash's seeding, in r's
	// snapshot.
	Judge(ctx context.Context, r storage.Reader[file], rows []Row) (map[ID]Judged, error)
	// EndlessUnits counts node's live Torrents with no seeding end: their
	// goal never acts, so they stay until they are removed, moved or
	// written off.
	EndlessUnits(ctx context.Context, r storage.Reader[file], node role.NodeID) (int64, error)
}

type Judged struct {
	Tracker *TrackerRef
	View    seeding.View
	// Plan is what the row's own goal does now.
	Plan seeding.Plan
}

// HashFact is one Torrent of an info hash as its seeding is judged: its
// row before judging, with its record's Tracker, wanted state and
// removal.
type HashFact struct {
	Row
	TrackerID *indexers.TrackerID
	Wanted    Wanted
	Complete  bool
	// Removal is nil while it is live. A Removing Torrent's has no Parted.
	Removal *Removal
}

// HashFacts is every built-in Torrent of hashes that its Node holds,
// Removing ones too, and each hash's latest built-in Torrent its Node
// parted from, with their Grabs, in added order by hash. An external
// client's Torrents are no part of a hash's seeding.
func (c *Clients) HashFacts(ctx context.Context, r storage.Reader[file], hashes []bencode.InfoHash) (map[bencode.InfoHash][]HashFact, error) {
	raw := make([][]byte, len(hashes))
	for i, h := range hashes {
		raw[i] = h[:]
	}
	rows, err := db.New(r).ListHashFacts(ctx, raw)
	if err != nil {
		return nil, err
	}
	ids := make([]ID, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
	}
	uses, err := c.cfg.Uses.Of(ctx, r, ids)
	if err != nil {
		return nil, err
	}
	out := map[bencode.InfoHash][]HashFact{}
	for _, h := range rows {
		row, err := c.rowOf(db.ListHeldRow(h), uses[h.ID])
		if err != nil {
			return nil, fmt.Errorf("torrent %s: %w", h.ID, err)
		}
		f := HashFact{Row: row, TrackerID: h.TrackerID, Wanted: Wanted(h.Wanted), Complete: row.report != nil && row.report.Complete}
		if !h.RemovedAt.IsZero() {
			f.Removal = &Removal{At: h.RemovedAt.Time(), Reason: RemoveReason(*h.RemovedReason)}
			if !h.PartedAt.IsZero() {
				f.Removal.Parted = &Parted{At: h.PartedAt.Time()}
			}
		}
		out[row.InfoHash] = append(out[row.InfoHash], f)
	}
	return out, nil
}

// TrackerHashes is every info hash with a built-in Torrent naming
// tracker, live or removed.
func TrackerHashes(ctx context.Context, r storage.Reader[file], tracker indexers.TrackerID) ([]bencode.InfoHash, error) {
	rows, err := db.New(r).ListTrackerHashes(ctx, &tracker)
	if err != nil {
		return nil, err
	}
	out := make([]bencode.InfoHash, len(rows))
	for i, h := range rows {
		out[i] = bencode.InfoHash(h)
	}
	return out, nil
}

// Trackerless is a Torrent no Tracker is known for, with its announce
// hosts.
type Trackerless struct {
	ID       ID
	InfoHash bencode.InfoHash
	Hosts    []string
}

// TrackerlessIn is every Torrent with no Tracker, live or removed, in
// added order. A Torrent added before its trackers were stored reads its
// hosts from its metainfo.
func TrackerlessIn(ctx context.Context, r storage.Reader[file]) ([]Trackerless, error) {
	rows, err := db.New(r).ListTrackerless(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Trackerless, len(rows))
	for i, row := range rows {
		hosts, err := hostsOf(row)
		if err != nil {
			return nil, fmt.Errorf("torrent %s: %w", row.ID, err)
		}
		out[i] = Trackerless{ID: row.ID, InfoHash: bencode.InfoHash(row.InfoHash), Hosts: hosts}
	}
	return out, nil
}

func hostsOf(row db.ListTrackerlessRow) ([]string, error) {
	if row.Metainfo == nil {
		var hosts []string
		if err := json.Unmarshal([]byte(row.Trackers), &hosts); err != nil {
			return nil, fmt.Errorf("decode the trackers: %w", err)
		}
		return hosts, nil
	}
	m, err := bencode.DecodeMetainfo(row.Metainfo)
	if err != nil {
		return nil, fmt.Errorf("decode the metainfo: %w", err)
	}
	return HostsOf(m.Trackers()), nil
}

// AssignTracker names tracker as Torrent id's Tracker while it names none,
// inside tx.
func AssignTracker(ctx context.Context, tx *storage.Tx[file], id ID, tracker indexers.TrackerID) error {
	return db.New(tx).SetTrackerIfNone(ctx, db.SetTrackerIfNoneParams{TrackerID: &tracker, ID: id})
}
