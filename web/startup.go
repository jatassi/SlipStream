package web

import _ "embed"

//go:embed harness/startup-played.js
var startupPlayed []byte

// StartupPlayed returns harness/startup-played.js, the init script that
// opens a verify tab past the startup sequence.
func StartupPlayed() []byte {
	return startupPlayed
}
