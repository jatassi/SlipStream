package filesystem

import "errors"

// Errors returned by the filesystem service
var (
	ErrPathNotFound  = errors.New("path does not exist")
	ErrNotDirectory  = errors.New("path is not a directory")
	ErrAccessDenied  = errors.New("access denied")
	ErrInvalidPath   = errors.New("invalid path")
)

// DirectoryEntry represents a single directory in a browse result
type DirectoryEntry struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	IsDir bool   `json:"isDir"`
}

// DriveInfo represents a drive on Windows systems
type DriveInfo struct {
	Letter    string `json:"letter"`
	Label     string `json:"label,omitempty"`
	Type      string `json:"type,omitempty"` // "fixed", "removable", "network", "cdrom"
	FreeSpace int64  `json:"freeSpace,omitempty"`
}

// BrowseResult contains the result of browsing a directory
type BrowseResult struct {
	Path    string           `json:"path"`
	Parent  string           `json:"parent,omitempty"`
	Entries []DirectoryEntry `json:"entries"`
	Drives  []DriveInfo      `json:"drives,omitempty"` // Only populated on Windows when browsing root
}
