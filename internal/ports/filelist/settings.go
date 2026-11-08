// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/FileList/FileListSettings.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package filelist

import "github.com/jatassi/slipstream/internal/ports/port"

// SlipStream: Prowlarr's validator is the required flag on the
// credentials, and its default query limit is Entry.QueryLimit. The
// username is tagged key because FileList checks the pair as one Basic
// credential, so the stop names both.
type fileListSettings struct {
	Username          string `port:"username,key" label:"Username" help:"Your FileList username."`
	port.PasskeyField `help:"Your passkey is the string of letters and digits in the tracker URL your download client shows for a FileList torrent."`
	FreeleechOnly     bool `port:"freeleechOnly" label:"Freeleech only" help:"Search only freeleech Releases."`
}
