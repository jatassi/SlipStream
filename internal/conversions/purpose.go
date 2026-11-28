package conversions

import "github.com/jatassi/slipstream/internal/transfers"

// Purpose is the conversion Transfers' purpose: source copies to scratch
// and outputs returned to their root.
func (c *Conversions) Purpose() transfers.Purpose { return transfers.Purpose{} }
