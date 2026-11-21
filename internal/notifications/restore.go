package notifications

import (
	"context"
	"time"

	"github.com/jatassi/slipstream/internal/notifications/internal/db"
	"github.com/jatassi/slipstream/internal/notifications/providers"
	"github.com/jatassi/slipstream/internal/storage"
)

// restored is a message given up because a restored Backup holds it as
// due: the old Controller most likely sent it seconds after the Backup.
const restored providers.ReasonCode = "restored"

// GiveUpBefore gives up in tx every message due since before at, the time
// the restored Backup was taken, and drops every entry from before it not
// yet released, so nothing sent before the loss sends twice.
func GiveUpBefore(ctx context.Context, tx *storage.Tx[file], at time.Time) error {
	q := db.New(tx)
	_, err := q.GiveUpDueBefore(ctx, db.GiveUpDueBeforeParams{
		EndedAt: storage.Now(), Reason: jsonText(providers.Reason{Code: restored}), Before: storage.FromTime(at),
	})
	if err != nil {
		return err
	}
	return q.DeletePendingBefore(ctx, storage.FromTime(at))
}
