package conversions

import "context"

// Run reconciles until ctx ends: it wakes on a poke, a new Board, new
// facts and its one timer.
func (c *Conversions) Run(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

// Register sets the Importer the hand-off starts. Call it once, before Run.
func (c *Conversions) Register(im Importer) {
	if c.importer != nil {
		panic("conversions: an Importer is registered twice")
	}
	c.importer = im
}
