package storing

import "strings"

// Windows file attributes the hidden rule reads.
const (
	attrHidden = 0x2
	attrSystem = 0x4
)

// clutter are folders NAS boxes and Windows leave everywhere, hidden on
// every OS.
var clutter = map[string]bool{
	"$recycle.bin":              true,
	"system volume information": true,
	"@eadir":                    true,
	"#recycle":                  true,
}

// hidden is the hidden rule for one entry: a dot-name on Unix, the hidden
// or system attribute on Windows, and the clutter table on both.
func hidden(style PathStyle, name string, attrs uint32) bool {
	if clutter[strings.ToLower(name)] {
		return true
	}
	if style == Windows {
		return attrs&(attrHidden|attrSystem) != 0
	}
	return strings.HasPrefix(name, ".")
}
