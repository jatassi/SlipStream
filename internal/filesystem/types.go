package filesystem

import "errors"

// Errors returned by the filesystem service
var (
	ErrPathNotFound = errors.New("path does not exist")
	ErrNotDirectory = errors.New("path is not a directory")
	ErrAccessDenied = errors.New("access denied")
	ErrInvalidPath  = errors.New("invalid path")
)

// DirectoryEntry represents a single directory in a browse result
type DirectoryEntry struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	IsDir bool   `json:"isDir"`
}

// DriveInfo represents a drive on Windows systems
type DriveInfo struct {
	Letter     string `json:"letter"`
	Label      string `json:"label,omitempty"`
	Type       string `json:"type,omitempty"` // "fixed", "removable", "network", "cdrom"
	FreeSpace  int64  `json:"freeSpace,omitempty"`
	TotalSpace int64  `json:"totalSpace,omitempty"`
}

// VolumeInfo represents a volume/mount point on Unix systems
type VolumeInfo struct {
	Path       string `json:"path"`
	MountPoint string `json:"mountPoint"`
	Filesystem string `json:"filesystem"`
	Type       string `json:"type"` // "local", "network", "removable"
	FreeSpace  int64  `json:"freeSpace,omitempty"`
	TotalSpace int64  `json:"totalSpace,omitempty"`
}

// StorageInfo represents aggregated storage information for a volume/drive
type StorageInfo struct {
	// Common fields
	Label       string  `json:"label"`
	Path        string  `json:"path"` // Drive letter on Windows, mount point on Unix
	FreeSpace   int64   `json:"freeSpace"`
	TotalSpace  int64   `json:"totalSpace"`
	UsedSpace   int64   `json:"usedSpace"`
	UsedPercent float64 `json:"usedPercent"`
	Type        string  `json:"type"` // "fixed", "removable", "network"

	// Associated root folders
	RootFolders []RootFolderRef `json:"rootFolders"`
}

// RootFolderRef represents a reference to a root folder
type RootFolderRef struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Path      string `json:"path"` // Full path for proper volume/drive matching
	MediaType string `json:"mediaType"`
}

// BrowseResult contains the result of browsing a directory
type BrowseResult struct {
	Path    string           `json:"path"`
	Parent  string           `json:"parent,omitempty"`
	Entries []DirectoryEntry `json:"entries"`
	Drives  []DriveInfo      `json:"drives,omitempty"` // Only populated on Windows when browsing root
}
