package link_test

import (
	"testing"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
)

func TestParseVersionReadsTheLeadingNumbers(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"1.43.5.11029-1bd7c5a7e", "1.43.5.11029"},
		{"12.2.0", "12.2.0"},
		{"4.10.1.0", "4.10.1.0"},
		{"10.10.7", "10.10.7"},
		{"12.4.0-beta1", "12.4.0"},
	} {
		v, err := link.ParseVersion(c.in)
		if err != nil || v.String() != c.want {
			t.Errorf("ParseVersion(%q) = %q, %v, want %q", c.in, v, err, c.want)
		}
	}
	if _, err := link.ParseVersion("beta"); err == nil {
		t.Error("ParseVersion(beta) parsed")
	}
}

func TestVersionsCompareByNumber(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want int
	}{
		{"10.10.7", "12.2", -1},
		{"12.2.0", "12.2", 0},
		{"1.43.5.11029", "1.43", 1},
		{"4.9.9", "4.10", -1},
	} {
		if got := link.MustVersion(c.a).Compare(link.MustVersion(c.b)); got != c.want {
			t.Errorf("%s compared with %s = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}
