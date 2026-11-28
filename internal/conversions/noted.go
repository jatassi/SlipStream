package conversions

import "github.com/jatassi/slipstream/internal/role"

// Noted is told each notable facts change: a new signature replaces node's
// proofs, and a media-server-link change Touches every Converting holder.
func (c *Conversions) Noted(role.NodeID, string) {}

// TouchHolders Touches every Converting holder's intent, for
// mediaservers.Config.Touched.
func (c *Conversions) TouchHolders() {}
