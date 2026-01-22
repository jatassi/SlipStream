package config

// Embedded API keys injected at build time via ldflags.
// These serve as defaults and can be overridden by environment
// variables or config file.
//
// Build with:
//   go build -ldflags "-X 'github.com/slipstream/slipstream/internal/config.EmbeddedTMDBKey=xxx' \
//                      -X 'github.com/slipstream/slipstream/internal/config.EmbeddedTVDBKey=yyy'"
var (
	EmbeddedTMDBKey string
	EmbeddedTVDBKey string
)
