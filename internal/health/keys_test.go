package health_test

import (
	"encoding/base64"
	"net/url"
	"testing"

	"github.com/jatassi/slipstream/internal/health"
)

func TestKeysTravelAsOnePathSegment(t *testing.T) {
	for _, mount := range []string{`D:\`, "/mnt/media", `\\nas\media`, "/Volumes/Archive 2", "/mnt/ünïcode"} {
		key := health.KeyOf("v", "0199f0a4-5c3b-7a51-9d1e-2b3c4d5e6f70", health.Opaque(mount))
		if url.PathEscape(string(key)) != string(key) {
			t.Errorf("key %q for %q needs escaping in a path", key, mount)
		}
		back, err := base64.RawURLEncoding.DecodeString(health.Opaque(mount))
		if err != nil || string(back) != mount {
			t.Errorf("Opaque(%q) decodes to %q, %v", mount, back, err)
		}
	}
	if got := health.KeyOf("r", "abc"); got != "r.abc" {
		t.Errorf(`KeyOf("r", "abc") = %q, want "r.abc"`, got)
	}
}

func TestKeyOfRefusesAReservedPart(t *testing.T) {
	for _, part := range []string{"", "a.b", "a/b", `C:\`, "a b"} {
		t.Run(part, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Errorf("KeyOf(%q) did not panic", part)
				}
			}()
			health.KeyOf("v", part)
		})
	}
}
