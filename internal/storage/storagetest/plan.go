package storagetest

import (
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/storage"
)

// QueryPlan is one query a read ran and its EXPLAIN QUERY PLAN lines.
type QueryPlan struct {
	Query string
	Lines []string
}

// Uses is whether p reads through the index named, as a search or a
// covering read.
func (p QueryPlan) Uses(index string) bool {
	return slices.ContainsFunc(p.Lines, func(l string) bool {
		return strings.HasSuffix(l, "INDEX "+index) || strings.Contains(l, "INDEX "+index+" ")
	})
}

// Plan runs read against r, explaining each query before it runs it, and
// answers the plans in the order read ran them.
func Plan[F storage.File](t testing.TB, r storage.Reader[F], read func(storage.Reader[F]) error) []QueryPlan {
	t.Helper()
	var plans []QueryPlan
	if err := read(storage.Explaining(r, func(query string, lines []string) {
		plans = append(plans, QueryPlan{Query: query, Lines: lines})
	})); err != nil {
		t.Fatal(err)
	}
	return plans
}
