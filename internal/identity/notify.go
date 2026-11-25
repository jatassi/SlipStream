package identity

import (
	"context"

	"github.com/jatassi/slipstream/internal/identity/internal/db"
	"github.com/jatassi/slipstream/internal/mail"
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

// Contacted is an Account as another subsystem names and mails it.
type Contacted struct {
	Name     string
	Email    string
	Admin    bool
	Disabled bool
}

// Contact is account's name, email, role and whether it is disabled, read
// through r.
func Contact(ctx context.Context, r storage.Reader[file], account AccountID) (Contacted, error) {
	a, err := db.New(r).GetAccount(ctx, account)
	if err != nil {
		return Contacted{}, err
	}
	return Contacted{Name: a.DisplayName, Email: a.Email, Admin: a.Admin, Disabled: !a.DisabledAt.IsZero()}, nil
}

// Contacts is Contact of each of ids.
func Contacts(ctx context.Context, r storage.Reader[file], ids []AccountID) (map[AccountID]Contacted, error) {
	out := make(map[AccountID]Contacted, len(ids))
	for _, id := range ids {
		c, err := Contact(ctx, r, id)
		if err != nil {
			return nil, err
		}
		out[id] = c
	}
	return out, nil
}

// EnabledAdminAddresses is every enabled Admin's email, for a message to the
// Admins.
func EnabledAdminAddresses(ctx context.Context, r storage.Reader[file]) ([]mail.Address, error) {
	rows, err := db.New(r).ListAccounts(ctx)
	if err != nil {
		return nil, err
	}
	var out []mail.Address
	for _, a := range rows {
		if a.Admin && a.DisabledAt.IsZero() {
			out = append(out, mail.Address{Name: a.DisplayName, Email: a.Email})
		}
	}
	return out, nil
}
