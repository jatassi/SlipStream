// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/Avistaz/AvistazSettings.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package avistaz

import "github.com/jatassi/slipstream/internal/ports/port"

// Settings is AvistazSettings. The token it holds in C# is the session
// the framework keeps, so it isn't a setting.
type Settings struct {
	port.UserPass
	port.PIDField
	FreeleechOnly bool `port:"freeleechOnly" label:"Freeleech only" help:"Search only freeleech Releases."`
}
