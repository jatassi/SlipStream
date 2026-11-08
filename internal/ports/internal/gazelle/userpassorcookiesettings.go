// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/Gazelle/GazelleUserPassOrCookieSettings.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package gazelle

import "github.com/jatassi/slipstream/internal/ports/port"

// UserPassOrCookieSettings is GazelleUserPassOrCookieSettings: a cookie
// that, when set, is the session in place of the sign-in form.
// SlipStream: the username and password stay required, since a setting
// can't be required only when another is empty.
type UserPassOrCookieSettings struct {
	Settings
	port.CookieField `help:"Use the cookie only if your account has two-factor sign-in, and leave it empty otherwise."`
}

func (s UserPassOrCookieSettings) cookie() string { return s.Cookie }
