package identity

import (
	"context"
	"errors"

	"github.com/jatassi/slipstream/internal/identity/internal/db"
)

// ErrCurrentPINMissing is a Change PIN without the current PIN, which only
// a session that must set a new PIN may leave out.
var ErrCurrentPINMissing = errors.New("identity: the current PIN is missing")

// Reconfirm checks a's PIN and stamps its session confirmed. A wrong PIN
// counts toward the Account's limits as at sign-in.
func (s *Service) Reconfirm(ctx context.Context, a Account, pin PIN) error {
	pass, err := s.checkOwnPIN(ctx, a.client, a.s.profile.ID, pin)
	if err != nil {
		return err
	}
	return s.afterPIN(ctx, pass, func(ctx context.Context, q *db.Queries, fx *effects) error {
		return q.ConfirmSession(ctx, db.ConfirmSessionParams{ConfirmedAt: fx.now, ID: a.s.session})
	})
}

// ChangePIN sets b's Account's PIN to next. current must be right, unless
// b is a session that must set a new PIN, which needs none.
func (s *Service) ChangePIN(ctx context.Context, b Bearer, current *PIN, next PIN) error {
	if !b.s.pinRequired && current == nil {
		return ErrCurrentPINMissing
	}
	hash, err := s.pins.hash(ctx, next)
	if err != nil {
		return err
	}
	set := func(ctx context.Context, q *db.Queries, _ *effects) error {
		if err := q.SetPIN(ctx, db.SetPINParams{PinHash: hash, ID: b.s.profile.ID}); err != nil {
			return err
		}
		return q.ClearPINRequired(ctx, b.s.profile.ID)
	}
	if b.s.pinRequired {
		return s.write(ctx, set)
	}
	pass, err := s.checkOwnPIN(ctx, b.client, b.s.profile.ID, *current)
	if err != nil {
		return err
	}
	return s.afterPIN(ctx, pass, set)
}

func (s *Service) checkOwnPIN(ctx context.Context, c client, account AccountID, pin PIN) (pinPass, error) {
	q := db.New(s.lib)
	set, err := readSettings(ctx, q)
	if err != nil {
		return pinPass{}, err
	}
	row, err := q.GetAccount(ctx, account)
	if err != nil {
		return pinPass{}, err
	}
	return s.checkPIN(ctx, c, subjectOf(row), pin, set)
}
