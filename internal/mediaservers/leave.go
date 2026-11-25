package mediaservers

import (
	"context"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/mediaservers/access"
	"github.com/jatassi/slipstream/internal/mediaservers/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
)

// Leaving removes account's access before identity deletes it, and holds
// the Account's turn until done, so no operation gives access between the
// removals and the delete. A row whose removal fails outlives the Account
// as removing, or making for the sweep.
func (g granter) Leaving(ctx context.Context, account identity.AccountID) (func(), error) {
	release, err := g.m.turns.take(ctx, accountTurn(account))
	if err != nil {
		return nil, err
	}
	if err := g.m.leaveAll(ctx, account); err != nil {
		release()
		return nil, err
	}
	return release, nil
}

func (m *MediaServers) leaveAll(ctx context.Context, account identity.AccountID) error {
	rows, err := m.rowsOf(ctx, account)
	if err != nil {
		return err
	}
	for _, r := range rows {
		if _, err := m.actLocked(ctx, r, access.Leave, m.takeAway); err != nil {
			return err
		}
	}
	return nil
}

// takeAway is the remote part of Remove, Try again and Leave.
func (m *MediaServers) takeAway(ctx context.Context, r accessRow) (step, error) {
	if r.State == access.Offered || r.State == access.Removed {
		return step{}, nil
	}
	if r.State == access.Making {
		return m.unmake(ctx, r)
	}
	return m.wayOf(r.server.Kind).remove(ctx, r)
}

func (m *MediaServers) rowsOf(ctx context.Context, account identity.AccountID) ([]accessRow, error) {
	var out []accessRow
	err := m.cfg.Library.Read(ctx, func(ctx context.Context, rd storage.Reader[storage.Library]) error {
		q := db.New(rd)
		as, err := q.ListAccessOf(ctx, account)
		if err != nil {
			return err
		}
		for _, a := range as {
			r, err := rowOn(ctx, q, a)
			if err != nil {
				return err
			}
			out = append(out, r)
		}
		return nil
	})
	return out, err
}
