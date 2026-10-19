package storing

import "testing"

func TestIsUNC(t *testing.T) {
	for p, want := range map[string]bool{
		`\\server\share\slipstream`:              true,
		`//server/share/slipstream`:              true,
		`\\?\UNC\server\share\slipstream`:        true,
		`\\?\unc\server\share`:                   true,
		`C:\Users\ana\AppData\Local\SlipStream2`: false,
		`\\?\C:\Users\ana`:                       false,
		`\\.\C:\Users\ana`:                       false,
	} {
		if got := isUNC(p); got != want {
			t.Errorf("isUNC(%q) = %v, want %v", p, got, want)
		}
	}
}

func TestWindowsVolumeName(t *testing.T) {
	for _, c := range []struct{ root, label, want string }{
		{`D:\`, "Media", "D: (Media)"},
		{`C:\`, "", "C:"},
		{`\\?\D:\`, "Media", "D: (Media)"},
		{`\\nas\media\`, "Volume_1", `\\nas\media`},
		{`\\?\UNC\nas\media\`, "", `\\nas\media`},
		{`C:\mnt\disk\`, "Archive", `C:\mnt\disk (Archive)`},
	} {
		if got := volumeName(c.root, c.label); got != c.want {
			t.Errorf("volumeName(%q, %q) = %q, want %q", c.root, c.label, got, c.want)
		}
	}
}
