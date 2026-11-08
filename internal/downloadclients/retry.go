package downloadclients

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/downloadclients/internal/db"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// Retried is an Admin's Retry of a failed Torrent that had completed, and
// what its check found.
type Retried struct {
	At time.Time `json:"at"`
	// Wanted is the wanted state the Torrent had, which it gets back once
	// its check finds all of its data.
	Wanted Wanted `json:"wanted"`
	// Found is the bytes its check found, nil while its Node has yet to
	// check.
	Found *int64 `json:"found,omitempty"`
}

// retryable is whether a Retry may check a Torrent: one its built-in
// client failed after it completed. A failed download has no complete
// data to check, so it isn't.
func retryable(state State, client *ClientID, completed bool) bool {
	return state == StateFailed && client == nil && completed
}

// settle is the step a report takes r through: once the report names
// r's check, r records what it found, and the answer is the wanted state
// to write back when all of size was there, nil when the Torrent stays
// paused. It is false while the report names another check, or r already
// settled.
func (r Retried) settle(checked, found, size int64) (Retried, *Wanted, bool) {
	if r.Found != nil || checked != r.At.UnixMilli() {
		return r, nil, false
	}
	r.Found = &found
	if found < size {
		return r, nil, true
	}
	return r, &r.Wanted, true
}

func retryOf(stored *string) (*Retried, error) {
	if stored == nil {
		return nil, nil
	}
	var r Retried
	if err := json.Unmarshal([]byte(*stored), &r); err != nil {
		return nil, fmt.Errorf("retry: %w", err)
	}
	return &r, nil
}

func setRetry(ctx context.Context, q *db.Queries, id ID, r Retried) error {
	stored, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("encode the retry: %w", err)
	}
	return q.SetRetry(ctx, db.SetRetryParams{Retry: new(string(stored)), ID: id})
}

// retryAtOnWire is the Retry an intent carries for a Torrent, 0 unless
// one waits for its Node's check.
func retryAtOnWire(stored *string) (int64, error) {
	r, err := retryOf(stored)
	if err != nil || r == nil || r.Found != nil {
		return 0, err
	}
	return r.At.UnixMilli(), nil
}

// retry has each failed Torrent that had completed checked again, in one
// Write: it records the Retry and pauses the Torrent, so the check
// starts no download, and its Node's intent carries the Retry until a
// report names its check. A Retry while one waits keeps the wanted state
// the first one recorded.
func (c *Clients) retry(ctx context.Context, a Action) ([]Result, *Preview, error) {
	var results []Result
	var nodes []role.NodeID
	var changed []ID
	err := c.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		results, nodes, changed = nil, nil, nil
		rows, err := readActed(ctx, tx, a.IDs)
		if err != nil {
			return err
		}
		q, at := db.New(tx), storage.Now().Time()
		for _, id := range a.IDs {
			r := Result{ID: id, Refusal: rows.refusal(id)}
			if r.Refusal == nil {
				if r.Refusal, err = rows.retryRefusal(id); err != nil {
					return err
				}
			}
			results = append(results, r)
			if r.Refusal != nil {
				continue
			}
			if err := startRetry(ctx, q, rows[id], at); err != nil {
				return err
			}
			nodes, changed = append(nodes, rows[id].NodeID), append(changed, id)
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	c.committed(nodes, changed)
	return results, nil, nil
}

// startRetry records row's Retry at and pauses its Torrent.
func startRetry(ctx context.Context, q *db.Queries, row db.ListActedRow, at time.Time) error {
	next, err := nextRetry(ctx, q, row, at)
	if err != nil {
		return err
	}
	if err := setRetry(ctx, q, row.ID, next); err != nil {
		return err
	}
	return q.SetWanted(ctx, db.SetWantedParams{Wanted: string(Paused), ID: row.ID})
}

// nextRetry is row's Retry at: it keeps the wanted state of a Retry still
// waiting, which already paused the Torrent.
func nextRetry(ctx context.Context, q *db.Queries, row db.ListActedRow, at time.Time) (Retried, error) {
	got, err := q.GetRetry(ctx, db.GetRetryParams{ID: row.ID, NodeID: row.NodeID})
	if err != nil {
		return Retried{}, err
	}
	prev, err := retryOf(got.Retry)
	if err != nil {
		return Retried{}, fmt.Errorf("torrent %s: %w", row.ID, err)
	}
	next := Retried{At: at, Wanted: Wanted(row.Wanted)}
	if prev != nil && prev.Found == nil {
		next.Wanted = prev.Wanted
	}
	return next, nil
}

// retryRefusal is why id can't be retried: it isn't a failed Torrent of
// a built-in client that had completed.
func (a acted) retryRefusal(id ID) (*Refusal, error) {
	row := a[id]
	rep, err := reportOf(row.Report)
	if err != nil {
		return nil, fmt.Errorf("torrent %s: %w", id, err)
	}
	if !retryable(StateOf(Facts{Wanted: Wanted(row.Wanted), Report: rep}), row.ClientID, row.Completion != nil) {
		return &Refusal{Code: RefusedNotRetryable}, nil
	}
	return nil, nil
}

// settleRetry records what the check a report names found, for a Retry
// of id waiting for it, and writes back its wanted state when all of
// its data was there. It answers whether it changed id's intent.
func settleRetry(ctx context.Context, q *db.Queries, node role.NodeID, id ID, r *dlv1.TorrentReport) (bool, error) {
	checked := r.GetRetryCheckedUnixMs()
	if checked == 0 {
		return false, nil
	}
	got, err := q.GetRetry(ctx, db.GetRetryParams{ID: id, NodeID: node})
	if err != nil {
		return false, err
	}
	prev, err := retryOf(got.Retry)
	if err != nil || prev == nil {
		return false, err
	}
	rep := reportFrom(r)
	next, wanted, ok := prev.settle(checked, rep.done(got.Size), got.Size)
	if !ok {
		return false, nil
	}
	if err := setRetry(ctx, q, id, next); err != nil {
		return false, err
	}
	if wanted != nil {
		if err := q.SetWanted(ctx, db.SetWantedParams{Wanted: string(*wanted), ID: id}); err != nil {
			return false, err
		}
	}
	return true, nil
}
