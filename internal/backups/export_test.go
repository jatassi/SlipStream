package backups

import (
	"context"

	"github.com/jatassi/slipstream/internal/tasks"
)

func (b *Backups) Settle(ctx context.Context) error { return b.settle(ctx) }

func DestinationUnit(d DestinationID) tasks.UnitKey { return destinationUnit(d) }
