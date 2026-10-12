package loopback

import "testing"

func TestLoopbackPlist(t *testing.T) {
	want := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>org.slipstream.fakeworld.loopback</string>
	<key>ProgramArguments</key>
	<array>
		<string>/bin/sh</string>
		<string>-c</string>
		<string>n=2; while [ $n -le 32 ]; do /sbin/ifconfig lo0 alias 127.0.0.$n up; n=$((n+1)); done</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
</dict>
</plist>
`
	if got := string(Plist()); got != want {
		t.Fatalf("Plist() =\n%s\nwant\n%s", got, want)
	}
}
