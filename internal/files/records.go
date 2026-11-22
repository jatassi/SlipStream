package files

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/files/internal/db"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// ImportRecord is one Import's landing of a file, as its files_imported row
// keeps it.
type ImportRecord struct {
	Import storage.UUID
	Work   works.WorkID
	Items  []works.ItemID
	// Quality is the placed file's, nil on a row written before files
	// 00009 whose file was already gone.
	Quality  *release.Quality
	Replaced []Replaced
	At       time.Time
	key      RecordKey
}

// RecordKey is where a newest-first walk of landings stopped. The zero
// key starts at the newest.
type RecordKey struct {
	at storage.Time
	id storage.UUID
}

// Key is the key a walk that read l goes on from.
func (l ImportRecord) Key() RecordKey { return l.key }

// RecordTally is one Import that landed a file since a time: Upgraded when every
// one of its landings replaced a file.
type RecordTally struct {
	Import   storage.UUID
	Upgraded bool
}

// RecordsBefore is the n newest Import landings older than k, newest
// first, on the files_imported_at index.
func RecordsBefore(ctx context.Context, r storage.Reader[file], k RecordKey, n int) ([]ImportRecord, error) {
	q := db.New(r)
	var rows []db.ListLandingsOfRow
	if k.at.IsZero() {
		got, err := q.ListNewestLandings(ctx, int64(n))
		if err != nil {
			return nil, err
		}
		for _, row := range got {
			rows = append(rows, db.ListLandingsOfRow(row))
		}
	} else {
		got, err := q.ListLandingsBefore(ctx, db.ListLandingsBeforeParams{At: k.at, ID: k.id, Lim: int64(n)})
		if err != nil {
			return nil, err
		}
		for _, row := range got {
			rows = append(rows, db.ListLandingsOfRow(row))
		}
	}
	return recordsOf(rows)
}

// RecordsOf is every landing of imports, each Import's oldest first.
func RecordsOf(ctx context.Context, r storage.Reader[file], imports []storage.UUID) ([]ImportRecord, error) {
	q := db.New(r)
	var out []ImportRecord
	for chunk := range slices.Chunk(imports, works.Batch) {
		rows, err := q.ListLandingsOf(ctx, chunk)
		if err != nil {
			return nil, err
		}
		ls, err := recordsOf(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, ls...)
	}
	return out, nil
}

// TalliesSince is each Import that landed a file at or after since,
// judged on every one of its landings, since or not.
func TalliesSince(ctx context.Context, r storage.Reader[file], since time.Time) ([]RecordTally, error) {
	rows, err := db.New(r).ListTalliesSince(ctx, storage.FromTime(since))
	if err != nil {
		return nil, err
	}
	out := make([]RecordTally, len(rows))
	for i, row := range rows {
		out[i] = RecordTally{Import: row.ImportID, Upgraded: row.Upgraded == 1}
	}
	return out, nil
}

func recordsOf(rows []db.ListLandingsOfRow) ([]ImportRecord, error) {
	out := make([]ImportRecord, len(rows))
	for i, row := range rows {
		l := ImportRecord{Import: row.ImportID, Work: row.WorkID, Quality: row.Quality, At: row.At.Time(), key: RecordKey{at: row.At, id: row.ID}}
		if err := json.Unmarshal([]byte(row.Items), &l.Items); err != nil {
			return nil, fmt.Errorf("import record %s: decode its items: %w", row.ID, err)
		}
		if err := json.Unmarshal([]byte(row.Replaced), &l.Replaced); err != nil {
			return nil, fmt.Errorf("import record %s: decode what it replaced: %w", row.ID, err)
		}
		out[i] = l
	}
	return out, nil
}
