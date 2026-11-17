package identity

import (
	"context"

	"github.com/jatassi/slipstream/internal/identity/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
)

// ReadAddress is the Address setting through r, zero when unset. Unlike
// Address, it reads through any Reader, so a sender outside a request can
// build links.
func ReadAddress(ctx context.Context, r storage.Reader[file]) (Address, error) {
	set, err := readSettings(ctx, db.New(r))
	return set.Address, err
}

// CreatedAt is when account was created.
func CreatedAt(ctx context.Context, r storage.Reader[file], account AccountID) (storage.Time, error) {
	row, err := db.New(r).GetAccount(ctx, account)
	if err != nil {
		return storage.Time{}, err
	}
	return row.CreatedAt, nil
}
