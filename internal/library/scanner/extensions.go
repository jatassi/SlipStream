package scanner

import (
	"path/filepath"
	"strings"
)

// VideoExtensions contains supported video file extensions.
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
var SampleFileIndicators = []string{
	"sample",
	"trailer",
	"proof",
}

// IsVideoFile checks if a filename has a video extension.
func IsVideoFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return VideoExtensions[ext]
}

// IsSampleFile checks if a filename indicates it's a sample file.
func IsSampleFile(filename string) bool {
	lower := strings.ToLower(filename)
	for _, indicator := range SampleFileIndicators {
		if strings.Contains(lower, indicator) {
			return true
		}
	}
	return false
}
