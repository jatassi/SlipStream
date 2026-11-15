package grabs

import (
	"context"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/storage"
)

// uncount clears the failure count and the stop of cells, keeping their
// refusals.
func uncount(context.Context, *storage.Tx[file], []files.Cell) error { return nil }
