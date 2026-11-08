// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/BroadcastheNet/BroadcastheNetSettings.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package broadcasthenet

import "github.com/jatassi/slipstream/internal/ports/port"

// SlipStream: Prowlarr's validator is the required flag on the API key,
// and its default query limit is Entry.QueryLimit.
type broadcastheNetSettings struct {
	port.APIKeyField
}
