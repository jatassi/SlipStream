package providers

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestCutNeverSplitsARuneAndEndsAtItsLimit(t *testing.T) {
	for _, c := range []struct {
		in   string
		n    int
		want string
	}{
		{"Heat (1995)", 11, "Heat (1995)"},
		{"Heat (1995)", 10, "Heat (199…"},
		{"Amélie", 4, "Amé…"},
		{"🎬🎬🎬", 5, "🎬🎬…"},
		{"🎬🎬🎬", 4, "🎬…"},
		{"a b c", 3, "a…"},
	} {
		if got := cut(c.in, c.n); got != c.want {
			t.Errorf("cut(%q, %d) = %q, want %q", c.in, c.n, got, c.want)
		}
	}
	long := strings.Repeat("é", 5000)
	for _, n := range []int{discordTitle, discordDescription, telegramCaption, telegramText, slackSection, pushoverMessage, pushoverTitle} {
		if got := cut(long, n); units(got) != n || !utf8.ValidString(got) {
			t.Errorf("cut to %d has %d characters", n, units(got))
		}
	}
	if got := cutBytes(strings.Repeat("é", 3000), ntfyMessage); len(got) > ntfyMessage || !utf8.ValidString(got) {
		t.Errorf("cutBytes to %d has %d bytes", ntfyMessage, len(got))
	}
}

func TestFitEscapedKeepsEscapesWhole(t *testing.T) {
	got := fitEscaped(strings.Repeat("*", 300), discordTitle, escapeMarkdown)
	if units(got) > discordTitle || strings.HasSuffix(strings.TrimSuffix(got, ellipsis), `\`) {
		t.Errorf("fitEscaped = %q (%d characters)", got, units(got))
	}
}
