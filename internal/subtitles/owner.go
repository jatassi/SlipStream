package subtitles

import (
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
)

// Owner is the Subtitles Role's owner. A holder keeps no work between
// calls, so taking the Role off a Node moves nothing.
var Owner = nodes.Owner{Role: role.Subtitles, Held: nodes.HoldsNothing, Unassigned: nodes.MovesNothing}
