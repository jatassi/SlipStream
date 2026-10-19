package storing

import (
	"context"
	"errors"
	"time"
)

// errUnresponsive is a Disk call that missed its deadline. The call goes
// on in its goroutine until the kernel answers.
var errUnresponsive = errors.New("the disk did not answer in time")

// bounded runs call under limit and answers at once when limit passes,
// with errUnresponsive, or with ctx's error when ctx ends first. A filesystem
// call can't be interrupted, so a miss leaves it running; callers that
// must not start a second call on a stuck path track that themselves.
func bounded[T any](ctx context.Context, limit time.Duration, call func(context.Context) (T, error)) (T, error) {
	callCtx, cancel := context.WithTimeout(ctx, limit)
	defer cancel()
	type answer struct {
		v   T
		err error
	}
	done := make(chan answer, 1)
	go func() {
		v, err := call(callCtx)
		done <- answer{v, err}
	}()
	select {
	case a := <-done:
		return a.v, a.err
	case <-callCtx.Done():
		var zero T
		if err := ctx.Err(); err != nil {
			return zero, err //nolint:wrapcheck // the caller's own context
		}
		return zero, errUnresponsive
	}
}
