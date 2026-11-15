package grabs

import (
	"context"
	"slices"

	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// HasFailureRule reports whether the failure table has a row for c.
func HasFailureRule(c Code) bool {
	_, ok := afterFailure[c]
	return ok
}

// FailWith fails Grab id with code, as a step whose outcome fails it does.
func (gr *Grabs) FailWith(ctx context.Context, id GrabID, code Code) error {
	var g Grab
	err := gr.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		g, err = grabIn(ctx, r, id)
		return err
	})
	if err != nil {
		return err
	}
	return gr.commit(ctx, g, fail{code, "It failed in a test."})
}

// RecordFailure records Grab id's failure with code again, in its own
// Write, as a second report of one failure would.
func (gr *Grabs) RecordFailure(ctx context.Context, id GrabID, code Code) error {
	return gr.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		g, err := grabIn(ctx, tx, id)
		if err != nil {
			return err
		}
		return gr.failed(ctx, tx, g, code)
	})
}

// Tried reports whether the failure rows of work refuse r.
func (gr *Grabs) Tried(ctx context.Context, work works.WorkID, r indexers.Release) (bool, error) {
	var tried []triedRelease
	err := gr.cfg.Library.Read(ctx, func(ctx context.Context, rd storage.Reader[file]) error {
		var err error
		tried, err = triedOf(ctx, rd, work)
		return err
	})
	return slices.ContainsFunc(tried, func(t triedRelease) bool { return t.is(r) }), err
}
