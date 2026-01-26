package config

// Build-time values injected via ldflags.
// These serve as defaults and can be overridden by environment
// variables or config file.
//
// Build with:
//
//	go build -ldflags "-X 'github.com/slipstream/slipstream/internal/config.Version=1.2.3' \
//	                   -X 'github.com/slipstream/slipstream/internal/config.EmbeddedTMDBKey=xxx' \
//	                   -X 'github.com/slipstream/slipstream/internal/config.EmbeddedTVDBKey=yyy' \
//	                   -X 'github.com/slipstream/slipstream/internal/config.EmbeddedOMDBKey=zzz'"
var (
	// Version is the application version, injected at build time.
	// Defaults to "dev" if not set.
	Version = "dev"

	EmbeddedTMDBKey string
	EmbeddedTVDBKey string
	EmbeddedOMDBKey string
)
