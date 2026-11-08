package trackers

import (
	"context"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/trackers/seeding"
)

// Planned is what the next Seeding goals run would do to one live
// Torrent: an Act, a Keep with its Downgrade, or a Wait with its Why.
type Planned struct {
	ID   downloadclients.ID
	Name string
	// Tracker is nil for a torrent with no known Tracker.
	Tracker *downloadclients.TrackerRef
	Plan    seeding.Plan
}

// Preview is what the next Seeding goals run would do, in added order:
// every Torrent its goal acts on or keeps, and every Torrent whose goal
// is met but waits.
func (t *Trackers) Preview(ctx context.Context, _ identity.Admin) ([]Planned, error) {
	var out []Planned
	err := t.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		rows, err := t.cfg.Clients.Rows(ctx, r, downloadclients.Filter{})
		if err != nil {
			return err
		}
		hashes := make([]bencode.InfoHash, len(rows))
		for i, row := range rows {
			hashes[i] = row.InfoHash
		}
		j, err := t.judge(ctx, r, hashes, t.cfg.Roster)
		if err != nil {
			return err
		}
		plans := map[downloadclients.ID]seeding.Plan{}
		for _, h := range j.hashes {
			for i, c := range h.copies {
				plans[c.ID] = h.verdict.Plans[i]
			}
		}
		for _, row := range rows {
			p := plans[row.ID]
			if _, waits := p.(seeding.Wait); waits && !row.Seeding.GoalMet {
				continue
			}
			out = append(out, Planned{ID: row.ID, Name: row.Name, Tracker: row.Tracker, Plan: p})
		}
		return nil
	})
	return out, err
}
