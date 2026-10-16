package identity

import (
	"context"
	"errors"

	"github.com/jatassi/slipstream/internal/identity/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
)

var (
	ErrLastAdmin = errors.New("identity: no Admin who is turned on would remain")
	ErrNoAccount = errors.New("identity: no Account has that id")
)

func getAccount(ctx context.Context, q *db.Queries, id AccountID) (db.IdentityAccount, error) {
	row, err := q.GetAccount(ctx, id)
	if errors.Is(err, storage.ErrNoRows) {
		return db.IdentityAccount{}, ErrNoAccount
	}
	return row, err
}

// emailFree refuses ErrEmailTaken when an Account other than self has
// email.
func emailFree(ctx context.Context, q *db.Queries, email string, self AccountID) error {
	row, err := q.GetAccountByEmail(ctx, email)
	switch {
	case errors.Is(err, storage.ErrNoRows):
		return nil
	case err != nil:
		return err
	case row.ID != self:
		return ErrEmailTaken
	}
	return nil
}
