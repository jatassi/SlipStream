// Package loopback aliases the Node addresses 127.0.0.2 to 127.0.0.32 on
// lo0 on macOS, where only 127.0.0.1 answers until an alias exists.
package loopback

import "fmt"

const (
	Label     = "org.slipstream.fakeworld.loopback"
	PlistPath = "/Library/LaunchDaemons/" + Label + ".plist"

	first = 2
	last  = 32
)

// program adds every alias. ifconfig is idempotent for an alias that
// already exists, so it runs at every boot and on every setup.
var program = fmt.Sprintf("n=%d; while [ $n -le %d ]; do /sbin/ifconfig lo0 alias 127.0.0.$n up; n=$((n+1)); done", first, last)

// Plist is the LaunchDaemon that runs program once at every boot.
func Plist() []byte {
	return fmt.Appendf(nil, `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>%s</string>
	<key>ProgramArguments</key>
	<array>
		<string>/bin/sh</string>
		<string>-c</string>
		<string>%s</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
</dict>
</plist>
`, Label, program)
}
