package downloadclients

import (
	"context"
	"errors"
	"time"

	"github.com/jatassi/slipstream/internal/downloadclients/internal/db"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// Raw is an external client's raw counters for one torrent at one report.
type Raw struct {
	Uploaded, Downloaded int64
	Active, Seeding      time.Duration
}

// counted is what now adds to a Torrent's totals since last, and the next
// baseline. With no last it adds nothing and now is the baseline, so only
// what the torrent did while tracked counts. A counter that dropped, as
// when the torrent is added again in the client, adds nothing and counts
// on from now. A replay of the same counters adds zero.
func counted(last *Raw, now Raw) (add, next Raw) {
	if last == nil {
		return Raw{}, now
	}
	return Raw{
		Uploaded: rise(last.Uploaded, now.Uploaded), Downloaded: rise(last.Downloaded, now.Downloaded),
		Active: rise(last.Active, now.Active), Seeding: rise(last.Seeding, now.Seeding),
	}, now
}

func rise[N int64 | time.Duration](last, now N) N { return max(now-last, 0) }

// applyClientTotals adds counted to each row's Torrent of node's in an
// external client, and stores its raw counters as the next baseline.
func applyClientTotals(ctx context.Context, q *db.Queries, node role.NodeID, rows []*dlv1.ClientCounters) ([]ID, error) {
	now := storage.Now()
	ids := make([]ID, 0, len(rows))
	for _, row := range rows {
		id, err := idOf(row.GetId())
		if err != nil {
			return nil, err
		}
		stored, err := q.GetRaw(ctx, db.GetRawParams{ID: id, NodeID: node})
		if errors.Is(err, storage.ErrNoRows) {
			continue
		}
		if err != nil {
			return nil, err
		}
		var last *Raw
		if stored.RawUploaded != nil {
			last = &Raw{
				Uploaded: *stored.RawUploaded, Downloaded: *stored.RawDownloaded,
				Active: seconds(*stored.RawActiveSeconds), Seeding: seconds(*stored.RawSeedingSeconds),
			}
		}
		add, next := counted(last, Raw{
			Uploaded: row.GetUploaded(), Downloaded: row.GetDownloaded(),
			Active: seconds(row.GetActiveSeconds()), Seeding: seconds(row.GetSeedingSeconds()),
		})
		err = q.AddCounted(ctx, db.AddCountedParams{
			Uploaded: add.Uploaded, Downloaded: add.Downloaded,
			ActiveSeconds: int64(add.Active / time.Second), SeedingSeconds: int64(add.Seeding / time.Second),
			RawUploaded: &next.Uploaded, RawDownloaded: &next.Downloaded,
			RawActiveSeconds: new(int64(next.Active / time.Second)), RawSeedingSeconds: new(int64(next.Seeding / time.Second)),
			TotalsAt: now, ID: id, NodeID: node,
		})
		if err != nil {
			return nil, err
		}
		if row.GetDone() > 0 {
			if err := q.SetDone(ctx, db.SetDoneParams{Done: row.GetDone(), ID: id, NodeID: node}); err != nil {
				return nil, err
			}
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func seconds(s int64) time.Duration { return time.Duration(s) * time.Second }
