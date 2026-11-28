package conversions

import (
	"context"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/storage"
)

// Report is imports' word on a conversion Import: placed as File, or ended,
// Declined when the Admin declined it.
type Report struct {
	Placed   bool
	File     files.ID
	Declined bool
}

// Followed is what Follow changed, published after its Write commits.
type Followed struct {
	Conversion ID
}

// Follow takes a landing Conversion to done or canceled in imports' Write.
func (c *Conversions) Follow(context.Context, *storage.Tx[storage.Library], ID, Report) (Followed, error) {
	return Followed{}, errNotBuilt
}

// Followed publishes what Follow changed, after its Write commits.
func (c *Conversions) Followed(context.Context, ...Followed) {}
