// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/PassThePopcorn/PassThePopcornSettings.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package passthepopcorn

import "github.com/jatassi/slipstream/internal/ports/port"

// SlipStream: Prowlarr's validator is the required flag on the credentials.
type passThePopcornSettings struct {
	port.APIUserFields `help.apiUser:"Find your API user and API key in your PassThePopcorn security settings, under Edit Profile > Security."`
	FreeleechOnly      bool `port:"freeleechOnly" label:"Freeleech only" help:"Search only freeleech Releases."`
	GoldenPopcornOnly  bool `port:"goldenPopcornOnly" label:"Golden Popcorn only" help:"Search only Golden Popcorn Releases."`
}
