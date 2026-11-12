package imports

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/jatassi/slipstream/internal/imports/internal/db"
)

// moves is every change of outcome setFile allows. A file may also be
// written again in the outcome it has. Final outcomes move nowhere, except
// a failed file the Admin retries.
var moves = map[Outcome][]Outcome{
	Pending:   {LeftOut, NotBetter, Waiting, Sending, Placing, FailedOut, Declined},
	Waiting:   {Pending, Sending, Placing, Placed, Declined},
	Sending:   {Placing, FailedOut, Declined},
	Placing:   {Placed, Waiting, FailedOut, Declined},
	FailedOut: {Pending, Declined},
}

func allowed(from, to Outcome) bool { return from == to || slices.Contains(moves[from], to) }

// errMoved is a file whose outcome changed underneath the Write that read
// it.
var errMoved = errors.New("the file's outcome changed while it was being written")

// setFile writes f over its row, which must still read was. A change of
// outcome moves allows nothing is refused.
func setFile(ctx context.Context, q *db.Queries, id ImportID, was Outcome, f File) error {
	if !allowed(was, f.Outcome) {
		return fmt.Errorf("import %s file %d: a %s file can't become %s", id, f.Position, was, f.Outcome)
	}
	row, err := fileRow(id, f)
	if err != nil {
		return err
	}
	n, err := q.SetFile(ctx, db.SetFileParams{
		Outcome: row.Outcome, Reason: row.Reason, Hold: row.Hold, Confirmed: row.Confirmed, Probe: row.Probe, Quality: row.Quality,
		Cut: row.Cut, ReleaseName: row.ReleaseName, RootID: row.RootID, Dest: row.Dest, SentTransfer: row.SentTransfer,
		SentOrder: row.SentOrder, PlacedBy: row.PlacedBy, FileID: row.FileID, SourceRemovedAt: row.SourceRemovedAt,
		ImportID: id, Position: row.Position, Was: string(was),
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("import %s file %d: %w", id, f.Position, errMoved)
	}
	return nil
}

func jsonOf[T any](v *T) (*string, error) {
	if v == nil {
		return nil, nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("encode it: %w", err)
	}
	return new(string(b)), nil
}

func fromJSON[T any](s *string) (*T, error) {
	if s == nil {
		return nil, nil
	}
	v := new(T)
	if err := json.Unmarshal([]byte(*s), v); err != nil {
		return nil, fmt.Errorf("decode it: %w", err)
	}
	return v, nil
}
