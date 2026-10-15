package desktop

import (
	"bytes"
	"encoding/xml"

	"github.com/jatassi/slipstream/internal/install"
)

// launchAgent is the macOS LaunchAgent that starts program at login. It has
// RunAtLoad and no KeepAlive: launchd cannot skip a restart on exit codes
// 3 to 6, so a restart rule would loop on a refused database.
func launchAgent(program string) []byte {
	var path bytes.Buffer
	_ = xml.EscapeText(&path, []byte(program))
	return []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>` + install.BundleID + `</string>
	<key>ProgramArguments</key>
	<array>
		<string>` + path.String() + `</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
</dict>
</plist>
`)
}
