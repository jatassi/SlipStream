// Ported from Prowlarr's src/NzbDrone.Common/Http/CookieUtil.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package parseutil

import (
	"regexp"
	"slices"
	"strings"
)

// The classes spell out .NET's \s, which also takes \v, U+0085 and every Unicode space.
var cookiePair = regexp.MustCompile(`([^\(\)<>@,;:\\"/\[\]\?=\{\}\s\v\x{85}\p{Z}]+)=([^,;\\"\s\v\x{85}\p{Z}]+)`)

var cookieAttributes = []string{"COMMENT", "COMMENTURL", "DISCORD", "DOMAIN", "EXPIRES", "MAX-AGE", "PATH", "PORT", "SECURE", "VERSION", "HTTPONLY", "SAMESITE"}

// CookieHeaderToDictionary is CookieUtil.CookieHeaderToDictionary: every
// name=value pair in a Cookie or Set-Cookie header but the cookie
// attributes. A later pair of the same name wins.
func CookieHeaderToDictionary(header string) map[string]string {
	out := map[string]string{}
	for _, m := range cookiePair.FindAllStringSubmatch(header, -1) {
		if !slices.Contains(cookieAttributes, strings.ToUpper(m[1])) {
			out[m[1]] = m[2]
		}
	}
	return out
}
