package conversions

import "github.com/jatassi/slipstream/internal/housekeeping"

// Retention prunes done and canceled Conversions, and backfills with none
// left, past Housekeeping's Keep history for.
func (c *Conversions) Retention() housekeeping.Step { return housekeeping.Step{} }
