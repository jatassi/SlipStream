package health

import (
	"encoding/base64"
	"fmt"
	"strings"
)

// KeyOf joins parts with ".", so a key travels as one URL path segment
// unescaped. Each part is non-empty and only letters, digits, "-", "_" and
// "~"; pass anything else through Opaque. It panics otherwise: a bad part
// is a build mistake.
func KeyOf(parts ...string) Key {
	for _, p := range parts {
		if p == "" || strings.IndexFunc(p, reserved) >= 0 {
			panic(fmt.Sprintf("health: key part %q is not unreserved; pass it through Opaque", p))
		}
	}
	return Key(strings.Join(parts, "."))
}

func reserved(r rune) bool {
	return (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '-' && r != '_' && r != '~'
}

// Opaque is s as a key part: unpadded base64url, for a mount such as
// "D:\" or "/mnt/media".
func Opaque(s string) string { return base64.RawURLEncoding.EncodeToString([]byte(s)) }
