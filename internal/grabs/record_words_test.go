package grabs

import "testing"

func TestACutIsNamedAnEditionOnce(t *testing.T) {
	for cut, want := range map[string]string{
		"Extended":        "Extended Edition",
		"Director's Cut":  "Director's Cut Edition",
		"Special Edition": "Special Edition",
	} {
		if got := editionWords(cut); got != want {
			t.Errorf("editionWords(%q) = %q, want %q", cut, got, want)
		}
	}
}
