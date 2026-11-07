// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/Avistaz/AvistazSettings.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package avistaz

import "github.com/jatassi/slipstream/internal/ports/port"

// Settings is AvistazSettings. The token it holds in C# is the session
// the framework keeps, so it isn't a setting.
type Settings struct {
	port.UserPass `help.username:"Only accounts of Member rank or higher can use the site's API."`
	port.PIDField `help:"Find your PID on your My Account or My Profile page."`
	FreeleechOnly bool `port:"freeleechOnly" label:"Freeleech only" help:"Search only freeleech Releases."`
}
