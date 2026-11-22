package reconciling

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/reconciling/internal/db"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// Holdings stores one page of node's Holdings, as downloadclients checked
// and clamped it, as node's own facts in tx, the Write that acks it, and
// decides nothing. A page of another round than the one asked for is
// dropped, and a repeated page counts once. A refused page, or one that
// would take the round past Max, stores nothing and refuses the round:
// the Node keeps every torrent.
func (s *Service) Holdings(ctx context.Context, tx *storage.Tx[file], node role.NodeID, p downloadclients.HoldingsPage) error {
	q := db.New(tx)
	row, err := q.GetNode(ctx, node)
	switch {
	case errors.Is(err, storage.ErrNoRows):
		return nil
	case err != nil:
		return fmt.Errorf("read the Node's Holdings round: %w", err)
	}
	if !row.AdoptedAt.IsZero() || !row.RefusedAt.IsZero() || uint64(row.Round) != p.Round { //nolint:gosec // round is at least 1
		return nil
	}
	if p.Refused == "" && row.Pages != nil && *row.Pages != int64(p.Pages) {
		p.Refused = fmt.Sprintf("One page of the Node's list gives %d pages in all, and a later one gives %d.", *row.Pages, p.Pages)
	}
	entries := once(p.Entries)
	if p.Refused == "" {
		if row.Held, p.Refused, err = s.grown(ctx, q, node, row.Held, entries); err != nil {
			return err
		}
	}
	if p.Refused != "" {
		s.cfg.Log.Warn("SlipStream refused a Node's list of torrents, so the Node keeps all of them", "node", node, "reason", p.Refused)
		return q.Refuse(ctx, db.RefuseParams{At: storage.Now(), Refused: &p.Refused, NodeID: node})
	}
	for _, h := range entries {
		if err := putHeld(ctx, q, node, h); err != nil {
			return err
		}
	}
	seen, err := q.SeePage(ctx, db.SeePageParams{NodeID: node, Round: row.Round, Page: int64(p.Page)})
	if err != nil {
		return err
	}
	pages := int64(p.Pages)
	params := db.SetRoundParams{Pages: &pages, PagesSeen: row.PagesSeen + seen, Held: row.Held, NodeID: node, ReportedAt: row.ReportedAt}
	if params.ReportedAt.IsZero() && params.PagesSeen == pages {
		params.ReportedAt = storage.Now()
		s.reported.Store(true)
	}
	return q.SetRound(ctx, params)
}

// grown is how many torrents node's round lists once entries are in, and
// why the round is refused when that passes Max.
func (s *Service) grown(ctx context.Context, q *db.Queries, node role.NodeID, held int64, entries []downloadclients.Holding) (grown int64, refused string, err error) {
	if len(entries) == 0 {
		return held, "", nil
	}
	hashes := make([][]byte, len(entries))
	for i := range entries {
		hashes[i] = entries[i].InfoHash[:]
	}
	known, err := q.CountHeldOf(ctx, db.CountHeldOfParams{NodeID: node, Hashes: hashes})
	if err != nil {
		return held, "", err
	}
	held += int64(len(entries)) - known
	if held > int64(s.cfg.Max) {
		return held, fmt.Sprintf("The Node's list is over the limit of %d torrents.", s.cfg.Max), nil
	}
	return held, "", nil
}

// once is entries with each info hash once, the last kept.
func once(entries []downloadclients.Holding) []downloadclients.Holding {
	at := make(map[bencode.InfoHash]int, len(entries))
	var out []downloadclients.Holding
	for _, e := range entries {
		if i, ok := at[e.InfoHash]; ok {
			out[i] = e
			continue
		}
		at[e.InfoHash] = len(out)
		out = append(out, e)
	}
	return out
}

func putHeld(ctx context.Context, q *db.Queries, node role.NodeID, h downloadclients.Holding) error {
	hosts, err := json.Marshal(append([]string{}, h.Hosts...))
	if err != nil {
		return fmt.Errorf("encode a torrent's hosts: %w", err)
	}
	p := db.PutHeldParams{
		NodeID: node, InfoHash: h.InfoHash[:], ClientID: h.Client, DonePpm: int64(math.Round(h.Done * 1e6)),
		Private: boolInt(h.Private), SeedSeconds: int64(h.Seed.Seconds()), Hosts: string(hosts),
	}
	if h.Content != nil {
		p.Name, p.Size = &h.Content.Name, &h.Content.Size
	}
	if h.Folder != "" {
		p.SaveFolder = &h.Folder
	}
	return q.PutHeld(ctx, p)
}

// holdingOf is a stored held row as downloadclients reads a Holding.
func holdingOf(row db.ReconcilingHeld) (downloadclients.Holding, error) {
	h := downloadclients.Holding{
		InfoHash: bencode.InfoHash(row.InfoHash), Client: row.ClientID, Done: float64(row.DonePpm) / 1e6,
		Private: row.Private == 1, Seed: seconds(row.SeedSeconds),
	}
	if err := json.Unmarshal([]byte(row.Hosts), &h.Hosts); err != nil {
		return downloadclients.Holding{}, fmt.Errorf("decode a torrent's hosts: %w", err)
	}
	if row.Name != nil {
		h.Content = &downloadclients.HoldingContent{Name: *row.Name, Size: *row.Size}
	}
	if row.SaveFolder != nil {
		h.Folder = *row.SaveFolder
	}
	return h, nil
}
