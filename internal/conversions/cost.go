package conversions

import (
	"context"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/storage"
)

func (Seam) Cost(context.Context, *storage.Tx[storage.Library], []files.MadePair) (files.MadeCost, error) {
	return files.MadeCost{}, errNotBuilt
}
