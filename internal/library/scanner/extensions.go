package scanner

import (
	"github.com/slipstream/slipstream/internal/module/parseutil"
)

// VideoExtensions contains supported video file extensions.
// Kept for backwards compatibility with code that accesses it directly.
var VideoExtensions = map[string]bool{
	".mkv":  true,
	".mp4":  true,
	".avi":  true,
	".m4v":  true,
	".ts":   true,
	".wmv":  true,
	".mov":  true,
	".webm": true,
	".flv":  true,
	".mpg":  true,
	".mpeg": true,
	".m2ts": true,
	".vob":  true,
	".iso":  true,
}

// SampleFileIndicators are strings that indicate a file is a sample.
// Kept for backwards compatibility with code that accesses it directly.
var SampleFileIndicators = []string{
	"sample",
	"trailer",
	"proof",
}

// IsVideoFile checks if a filename has a video extension.
func IsVideoFile(filename string) bool {
	return parseutil.IsVideoFile(filename)
}

// IsSampleFile checks if a filename indicates it's a sample file.
func IsSampleFile(filename string) bool {
	return parseutil.IsSampleFile(filename)
}
