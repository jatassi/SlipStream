package transfers

import (
	"context"
	"slices"

	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/transfers/internal/ledger"
	"github.com/jatassi/slipstream/internal/transfers/source"
)

// Sending is which of refs a live record still sends, read in rd: a
// file neither placed nor dropped.
func Sending(ctx context.Context, rd storage.Reader[storage.Library], refs []source.Library) (map[source.Library]bool, error) {
	paths := make([]*string, 0, len(refs))
	for _, r := range refs {
		if !slices.ContainsFunc(paths, func(p *string) bool { return *p == r.Rel }) {
			paths = append(paths, &r.Rel)
		}
	}
	out := map[source.Library]bool{}
	if len(paths) == 0 {
		return out, nil
	}
	rows, err := ledger.New(rd).LibrarySending(ctx, paths)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		ref := source.Library{Root: [16]byte(row.SourceRoot), Rel: *row.Path}
		if slices.Contains(refs, ref) {
			out[ref] = true
		}
	}
	return out, nil
}

// UsingRoot counts the live records whose destination is root or that
// still send a library file from it, read in rd: the re-home guard.
func UsingRoot(ctx context.Context, rd storage.Reader[storage.Library], root roots.ID) (int, error) {
	n, err := ledger.New(rd).CountUsingRoot(ctx, &root)
	return int(n), err
}
