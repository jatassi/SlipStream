package desktop

import (
	"slices"
	"testing"
)

func TestOpener(t *testing.T) {
	const url = "http://localhost:8484/setup#code=ABCD-EFGH-JKMN"
	for _, c := range []struct {
		goos, browser string
		want          []string
	}{
		{"windows", "", []string{"rundll32", "url.dll,FileProtocolHandler", url}},
		{"windows", "firefox", []string{"rundll32", "url.dll,FileProtocolHandler", url}},
		{"darwin", "", []string{"/usr/bin/open", url}},
		{"linux", "", []string{"xdg-open", url}},
		{"linux", "/tmp/browser-stub", []string{"/tmp/browser-stub", url}},
	} {
		if got := opener(c.goos, c.browser, url); !slices.Equal(got, c.want) {
			t.Errorf("opener(%s, %q) = %q, want %q", c.goos, c.browser, got, c.want)
		}
	}
}

func TestLaunchAgent(t *testing.T) {
	got := string(launchAgent("/Applications/R&D SlipStream.app/Contents/MacOS/slipstream-app"))
	want := `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>io.github.jatassi.slipstream</string>
	<key>ProgramArguments</key>
	<array>
		<string>/Applications/R&amp;D SlipStream.app/Contents/MacOS/slipstream-app</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
</dict>
</plist>
`
	if got != want {
		t.Errorf("launchAgent =\n%s\nwant\n%s", got, want)
	}
}
