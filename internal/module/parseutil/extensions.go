package parseutil

import (
	"path/filepath"
	"strings"
)

var videoExtensions = map[string]bool{
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

var sampleFileIndicators = []string{
	"sample",
	"trailer",
	"proof",
}

// IsVideoFile checks whether a filename has a recognized video extension.
func IsVideoFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return videoExtensions[ext]
}

// IsSampleFile checks whether a filename appears to be a sample, trailer, or proof file.
func IsSampleFile(filename string) bool {
	lower := strings.ToLower(filename)
	for _, indicator := range sampleFileIndicators {
		if strings.Contains(lower, indicator) {
			return true
		}
	}
	return false
}

// VideoExtensions returns the list of recognized video file extensions (including the leading dot).
func VideoExtensions() []string {
	exts := make([]string, 0, len(videoExtensions))
	for ext := range videoExtensions {
		exts = append(exts, ext)
	}
	return exts
}
