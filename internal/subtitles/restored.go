package subtitles

import (
	"context"

	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/subtitles/internal/db"
)

// Restored marks every file for a survey, in S49's (Backups and restore)
// restore Write. A companion the Storage Node's ledger knows comes back as
// SlipStream's own, timed again; every other one reads found.
func (s *Subtitles) Restored(ctx context.Context, tx *storage.Tx[storage.Library]) error {
	if err := db.New(tx).MarkAllSurveyDue(ctx); err != nil {
		return err
	}
	return ClearLeases(ctx, tx)
}
