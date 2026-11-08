// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/BroadcastheNet/BroadcastheNetTorrents.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package broadcasthenet

import (
	"encoding/json"

	"github.com/jatassi/slipstream/internal/prowlarr/serializer"
)

// SlipStream: Torrents stays raw so the parser reads it in the site's
// order, as C#'s Dictionary keeps it and a Go map doesn't.
type broadcastheNetTorrents struct {
	Torrents json.RawMessage
	Results  serializer.Int
}
