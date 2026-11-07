// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/PassThePopcorn/PassThePopcornSettings.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package passthepopcorn

import "github.com/jatassi/slipstream/internal/ports/port"

// SlipStream: Prowlarr's validator is gone, because the field model has
// no required flag. An empty API user or API key reaches the site.
type passThePopcornSettings struct {
	port.APIUserFields
	FreeleechOnly     bool `port:"freeleechOnly" label:"Freeleech only" help:"Search only freeleech Releases."`
	GoldenPopcornOnly bool `port:"goldenPopcornOnly" label:"Golden Popcorn only" help:"Search only Golden Popcorn Releases."`
}
