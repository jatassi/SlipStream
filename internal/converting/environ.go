package converting

import (
	"slices"
	"strings"
)

// environ is the Node's environment less every SLIPSTREAM_ variable, in
// any case, since Windows reads names without it. ffmpeg, ffprobe and
// dovi_tool run in it.
func environ(env []string) []string {
	return slices.DeleteFunc(slices.Clone(env), func(kv string) bool {
		return strings.HasPrefix(strings.ToUpper(kv), "SLIPSTREAM_")
	})
}
