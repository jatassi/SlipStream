package indexers

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/indexers/internal/db"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/storage"
)

// Use is what a query row is for, and which limit it counts against.
type Use string

const (
	// UseSearch and UseRecent count against Searches.
	UseSearch Use = "search"
	UseRecent Use = "recent"
	// UseFetch counts against Grabs.
	UseFetch Use = "fetch"
	// UseTest counts against nothing.
	UseTest Use = "test"
)

// counted is the uses that count against the limit u counts against, nil
// for a Test.
func counted(u Use) []Use {
	switch u {
	case UseSearch, UseRecent:
		return []Use{UseSearch, UseRecent}
	case UseFetch:
		return []Use{UseFetch}
	case UseTest:
	}
	return nil
}

func (idx Indexer) limit(u Use) Limit {
	switch u {
	case UseSearch, UseRecent:
		return idx.Searches
	case UseFetch:
		return idx.Grabs
	case UseTest:
	}
	return Limit{}
}

// LimitError is a call refused because its limit is used up in its
// window.
type LimitError struct {
	// Until is when the oldest counted call leaves the window.
	Until time.Time
}

func (e *LimitError) Error() string {
	return "the Indexer reached its limit until " + e.Until.UTC().Format(time.RFC3339)
}

// free is when one more call fits under l, given the reservation times
// counted against it in any order: the zero time when one fits now. Pure.
func free(l Limit, reserved []time.Time, now time.Time) time.Time {
	if l.Count <= 0 {
		return time.Time{}
	}
	window := l.Per.Duration()
	var in []time.Time
	for _, at := range reserved {
		if now.Sub(at) < window {
			in = append(in, at)
		}
	}
	if len(in) < l.Count {
		return time.Time{}
	}
	slices.SortFunc(in, time.Time.Compare)
	return in[len(in)-l.Count].Add(window)
}

// reserve inserts a query row for u in tx, refusing with *LimitError while
// idx's limit for u is used up. The row's id is the job's id.
func reserve(ctx context.Context, tx *storage.Tx[file], idx Indexer, u Use, now storage.Time) (QueryID, error) {
	if l := idx.limit(u); l.Count > 0 {
		reserved, err := reservedFor(ctx, tx, idx.ID, u, now.Time().Add(-l.Per.Duration()))
		if err != nil {
			return QueryID{}, err
		}
		if until := free(l, reserved, now.Time()); !until.IsZero() {
			return QueryID{}, &LimitError{Until: until}
		}
	}
	q := QueryID{UUID: storage.NewUUID()}
	return q, insertQuery(ctx, tx, q, idx, u, now)
}

func insertQuery(ctx context.Context, tx *storage.Tx[file], q QueryID, idx Indexer, u Use, now storage.Time) error {
	return db.New(tx).InsertQuery(ctx, db.InsertQueryParams{
		ID: q, IndexerID: idx.ID, Use: string(u), ReservedAt: now, Revision: idx.Revision, Sha: orNil(pinned(idx)),
	})
}

func reservedFor(ctx context.Context, r storage.Reader[file], id ID, u Use, since time.Time) ([]time.Time, error) {
	uses := counted(u)
	names := make([]string, len(uses))
	for i, c := range uses {
		names[i] = string(c)
	}
	rows, err := db.New(r).ReservedOfSince(ctx, db.ReservedOfSinceParams{IndexerID: id, Uses: names, Since: storage.FromTime(since)})
	if err != nil {
		return nil, err
	}
	out := make([]time.Time, len(rows))
	for i, at := range rows {
		out[i] = at.Time()
	}
	return out, nil
}

// complete writes a's outcome on query row q once: a repeated answer
// changes nothing.
func complete(ctx context.Context, tx *storage.Tx[file], q QueryID, a answer, now storage.Time) error {
	p := db.CompleteQueryParams{
		ID: q, StartedAt: storage.FromTime(a.started), CompletedAt: now, NodeID: a.node, Outcome: ptr(outcomeWord(a.outcome)),
	}
	if a.outcome == nil {
		p.Releases = ptr(int64(a.releases))
	}
	_, err := db.New(tx).CompleteQuery(ctx, p)
	return err
}

// completeLost completes q as lost: its job never answered.
func completeLost(ctx context.Context, tx *storage.Tx[file], q QueryID, now storage.Time) error {
	_, err := db.New(tx).CompleteQuery(ctx, db.CompleteQueryParams{ID: q, CompletedAt: now, Outcome: ptr("lost")})
	return err
}

func outcomeWord(err error) string {
	var limited *searching.Limited
	var stopped *searching.Stopped
	switch {
	case err == nil:
		return "ok"
	case errors.As(err, &limited):
		return "limited"
	case errors.As(err, &stopped):
		return "stopped"
	}
	return "failing"
}

func ptr[T any](v T) *T { return &v }

// Usage is how much of each limit is used now.
type Usage struct {
	Searches, Grabs int
	// SearchesFree and GrabsFree are zero while under the limit.
	SearchesFree, GrabsFree time.Time
}

// usage is idx's Usage at now, from every counted reservation of the last
// day in r.
func usage(idx Indexer, reserved []db.ReservedSinceRow, now time.Time) Usage {
	var searches, grabs []time.Time
	for _, row := range reserved {
		if row.IndexerID != idx.ID {
			continue
		}
		switch Use(row.Use) {
		case UseSearch, UseRecent:
			searches = append(searches, row.ReservedAt.Time())
		case UseFetch:
			grabs = append(grabs, row.ReservedAt.Time())
		case UseTest:
		}
	}
	return Usage{
		Searches: within(idx.Searches, searches, now), Grabs: within(idx.Grabs, grabs, now),
		SearchesFree: free(idx.Searches, searches, now), GrabsFree: free(idx.Grabs, grabs, now),
	}
}

func within(l Limit, reserved []time.Time, now time.Time) int {
	n := 0
	for _, at := range reserved {
		if now.Sub(at) < l.Per.Duration() {
			n++
		}
	}
	return n
}

// QueryRow is one job's query row, for the Indexer page.
type QueryRow struct {
	ID        QueryID
	Use       Use
	Reserved  time.Time
	Started   time.Time
	Completed time.Time
	// Outcome is "" while the job runs: ok, limited, failing, stopped or
	// lost.
	Outcome  string
	Releases int
}

func queryRowOf(row db.IndexersQuery) QueryRow {
	q := QueryRow{
		ID: row.ID, Use: Use(row.Use), Reserved: row.ReservedAt.Time(), Started: row.StartedAt.Time(),
		Completed: row.CompletedAt.Time(), Outcome: deref(row.Outcome),
	}
	if row.Releases != nil {
		q.Releases = int(*row.Releases)
	}
	return q
}

// Page is one page of a list, newest first.
type Page struct {
	Offset, Limit int
}

func readQueries(ctx context.Context, r storage.Reader[file], id ID, p Page) ([]QueryRow, error) {
	rows, err := db.New(r).ListQueries(ctx, db.ListQueriesParams{IndexerID: id, Lim: int64(p.Limit), Off: int64(p.Offset)})
	if err != nil {
		return nil, fmt.Errorf("list the queries of Indexer %s: %w", id, err)
	}
	out := make([]QueryRow, len(rows))
	for i, row := range rows {
		out[i] = queryRowOf(row)
	}
	return out, nil
}
