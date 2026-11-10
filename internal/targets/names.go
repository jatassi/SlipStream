package targets

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"golang.org/x/text/cases"

	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets/internal/db"
)

// TargetNameTakenError is Name folding to Other, another Target's name.
// {Target} prints the name in file names, and NTFS and APFS compare those
// without case.
type TargetNameTakenError struct{ Name, Other string }

func (e *TargetNameTakenError) Error() string {
	return fmt.Sprintf("another Target is named %q", e.Other)
}

func foldName(name string) string { return cases.Fold().String(name) }

// claimName answers name's fold, or *TargetNameTakenError when a Target
// other than id has a name with that fold.
func claimName(ctx context.Context, q *db.Queries, id TargetID, name string) (string, error) {
	folded := foldName(name)
	switch other, err := q.OtherTargetFoldedAs(ctx, db.OtherTargetFoldedAsParams{NameFolded: &folded, ID: id}); {
	case errors.Is(err, storage.ErrNoRows):
		return folded, nil
	case err != nil:
		return "", err
	default:
		return "", &TargetNameTakenError{Name: name, Other: other}
	}
}

// foldNames gives each Target without a folded name its name's fold, in id
// order, and logs each rename. A Target whose fold an earlier one has takes
// the first of "<name> 2", "<name> 3" and on that folds like no other
// Target's name. It answers whether it renamed any.
func foldNames(ctx context.Context, q *db.Queries, log *slog.Logger) (bool, error) {
	rows, err := q.ListTargetNames(ctx)
	if err != nil {
		return false, err
	}
	named := make(map[string]int, len(rows))
	claimed := map[string]bool{}
	for _, r := range rows {
		named[foldName(r.Name)]++
		if r.NameFolded != nil {
			claimed[*r.NameFolded] = true
		}
	}
	renamed := false
	for _, r := range rows {
		if r.NameFolded != nil {
			continue
		}
		name, folded := r.Name, foldName(r.Name)
		if claimed[folded] {
			for n := 2; named[folded] > 0; n++ {
				name = fmt.Sprintf("%s %d", r.Name, n)
				folded = foldName(name)
			}
			named[folded]++
			log.Warn("renamed a Target whose name differs from another Target's only by case", "from", r.Name, "to", name)
			renamed = true
		}
		claimed[folded] = true
		if err := q.FoldTargetName(ctx, db.FoldTargetNameParams{ID: r.ID, Name: name, NameFolded: &folded}); err != nil {
			return false, err
		}
	}
	return renamed, nil
}
