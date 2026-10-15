package host

import "testing"

func TestIsUNC(t *testing.T) {
	for p, want := range map[string]bool{
		`\\server\share\slipstream`:              true,
		`\\?\UNC\server\share\slipstream`:        true,
		`C:\Users\ana\AppData\Local\SlipStream2`: false,
		`\\?\C:\Users\ana`:                       false,
	} {
		if got := isUNC(p); got != want {
			t.Errorf("isUNC(%q) = %v, want %v", p, got, want)
		}
	}
}
