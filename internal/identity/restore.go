package identity

import (
	"context"

	"github.com/jatassi/slipstream/internal/identity/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
)

// Carried is what a restore in place keeps from the library it replaces:
// Accounts with their PINs, passkeys and the API key, matched by id, so a
// credential revoked after the Backup never returns and no Grab loses its
// Account. Accounts come first, since a passkey names its Account.
var Carried = []storage.Carry{
	{Table: "identity_account", Key: "id"},
	{Table: "identity_passkey", Key: "id"},
	{Table: "identity_api_key", Key: "id"},
}

// ReplaceAPIKey replaces the API key in tx with one nobody sees: a manual
// rollback must not bring back a key revoked since its Backup. The Admin
// regenerates it to read one.
func ReplaceAPIKey(ctx context.Context, tx *storage.Tx[file]) error {
	return db.New(tx).ReplaceAPIKey(ctx, db.ReplaceAPIKeyParams{TokenHash: digest(newAPIKey().token), CreatedAt: storage.Now()})
}

// DeleteSessions signs out every session in tx.
func DeleteSessions(ctx context.Context, tx *storage.Tx[file]) error {
	return db.New(tx).DeleteAllSessions(ctx)
}
