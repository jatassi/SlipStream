// Package notices embeds the third-party notices file shipped with every binary.
package notices

import _ "embed"

// Text is the generated notices file.
//
//go:embed NOTICES.txt
var Text string
