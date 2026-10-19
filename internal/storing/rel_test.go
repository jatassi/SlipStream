package storing

import (
	"errors"
	"testing"
)

func TestParseRelRefusesEscapes(t *testing.T) {
	for _, s := range []string{
		"", "/abs", "a//b", "a/", "./a", "a/./b", "..", "a/../../b", `a\b`, `..\x`, "C:/x", "c:x",
	} {
		if r, err := ParseRel(s); !errors.Is(err, ErrBadRel) {
			t.Errorf("ParseRel(%q) = %q, %v; want ErrBadRel", s, r, err)
		}
	}
	for _, s := range []string{"a", "Heat (1995)/Heat (1995).mkv", ".slipstream-incoming/x.mkv", "Star Trek: TNG/s01", "a..b/...c"} {
		if r, err := ParseRel(s); err != nil || r.String() != s {
			t.Errorf("ParseRel(%q) = %q, %v", s, r, err)
		}
	}
}
