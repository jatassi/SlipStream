package roots

import (
	"context"

	"github.com/jatassi/slipstream/internal/roots/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// unreported is the path limit of a Node that hasn't reported one: Windows'
// without long paths, the strictest, so nothing is planned that a Windows
// Node can't open.
const unreported = 259

// BudgetsIn is each root's path budget, the bytes a root-relative path in
// it may take: its Node's last reported path limit, 259 until one reports,
// less the root folder's bytes and a separator.
func BudgetsIn(ctx context.Context, r storage.Reader[file]) (map[ID]int, error) {
	rows, err := db.New(r).ListRootLimits(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[ID]int, len(rows))
	for _, row := range rows {
		out[row.ID] = budgetOf(row)
	}
	return out, nil
}

// SmallestBudget is the least path budget among t's roots, for the save
// check of t's naming templates. A type with no root yet measures against
// a Node that hasn't reported, under a root of no bytes.
func SmallestBudget(ctx context.Context, r storage.Reader[file], t works.TypeID) (int, error) {
	rows, err := db.New(r).ListRootLimits(ctx)
	if err != nil {
		return 0, err
	}
	smallest, found := unreported-1, false
	for _, row := range rows {
		if b := budgetOf(row); row.ContentType == t && (!found || b < smallest) {
			smallest, found = b, true
		}
	}
	return smallest, nil
}

func budgetOf(row db.ListRootLimitsRow) int {
	limit := int64(unreported)
	if row.PathLimit.Valid {
		limit = row.PathLimit.Int64
	}
	return int(limit) - len(row.Path) - 1
}
