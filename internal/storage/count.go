package storage

import (
	"context"
	"sync/atomic"
)

type countKey struct{}

// Counted is ctx with a count of the queries run under it, on either file,
// and the count so far, for a test that holds a call to a query budget.
func Counted(ctx context.Context) (context.Context, func() int64) {
	n := new(atomic.Int64)
	return context.WithValue(ctx, countKey{}, n), n.Load
}

func count(ctx context.Context) {
	if n, ok := ctx.Value(countKey{}).(*atomic.Int64); ok {
		n.Add(1)
	}
}
