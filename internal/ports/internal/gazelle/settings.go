// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/Gazelle/GazelleSettings.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package gazelle

import "github.com/jatassi/slipstream/internal/ports/port"

// Settings is GazelleSettings. A site's settings embed it, or
// UserPassOrCookieSettings.
// SlipStream: no AuthKey or PassKey, which no Gazelle site reads.
type Settings struct {
	port.UserPass
	UseFreeleechToken GazelleFreeleechTokenAction `port:"useFreeleechToken" label:"Use freeleech tokens" default:"0" help:"With Always, SlipStream lists only Releases that are freeleech or can take a token."`
}

type GazelleFreeleechTokenAction int

const (
	Never GazelleFreeleechTokenAction = iota
	Preferred
	Required
)

// Options are the enum's FieldOption labels.
// SlipStream: an option has no hint, so each label says what the C#
// hint says.
func (GazelleFreeleechTokenAction) Options() []port.Option {
	return []port.Option{
		{Value: "0", Label: "Never"},
		{Value: "1", Label: "When possible"},
		{Value: "2", Label: "Always"},
	}
}

// settings is what the family reads of a site's settings, which an
// embedded Settings or UserPassOrCookieSettings provides.
type settings interface {
	gazelle() Settings
	cookie() string
}

func (s Settings) gazelle() Settings { return s }

func (Settings) cookie() string { return "" }
