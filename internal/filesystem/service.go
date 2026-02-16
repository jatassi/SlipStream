package filesystem

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/filesystem/mock"
)

const (
	osWindows = "windows"
)

// Service provides filesystem browsing capabilities
type Service struct {
	logger *zerolog.Logger
}

// NewService creates a new filesystem service
func NewService(logger *zerolog.Logger) *Service {
	subLogger := logger.With().Str("component", "filesystem").Logger()
	return &Service{
		logger: &subLogger,
	}
}

// BrowseDirectory lists directories at the given path
// If path is empty, returns root directories (drives on Windows, / on Unix)
func (s *Service) BrowseDirectory(path string) (*BrowseResult, error) {
	if path == "" {
		return s.browseRoot()
	}

	if mock.IsMockPath(path) {
		return s.browseMockDirectory(path)
	}

	cleanPath, err := s.validatePath(path)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(cleanPath)
	if err != nil {
		if os.IsPermission(err) {
			return nil, ErrAccessDenied
		}
		return nil, ErrPathNotFound
	}

	dirEntries := s.filterDirectories(entries, cleanPath)

	sort.Slice(dirEntries, func(i, j int) bool {
		return strings.ToLower(dirEntries[i].Name) < strings.ToLower(dirEntries[j].Name)
	})

	parent := filepath.Dir(cleanPath)
	if parent == cleanPath {
		parent = ""
	}

	return &BrowseResult{
		Path:    cleanPath,
		Parent:  parent,
		Entries: dirEntries,
	}, nil
}

func (s *Service) filterDirectories(entries []os.DirEntry, cleanPath string) []DirectoryEntry {
	var dirEntries []DirectoryEntry
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		if s.shouldSkipDirectory(entry.Name()) {
			continue
		}

		dirEntries = append(dirEntries, DirectoryEntry{
			Name:  entry.Name(),
			Path:  filepath.Join(cleanPath, entry.Name()),
			IsDir: true,
		})
	}
	return dirEntries
}

func (s *Service) shouldSkipDirectory(name string) bool {
	if runtime.GOOS != osWindows && strings.HasPrefix(name, ".") {
		return true
	}
	if runtime.GOOS == osWindows && isWindowsSystemDir(name) {
		return true
	}
	return false
}

func (s *Service) browseMockDirectory(path string) (*BrowseResult, error) {
	vfs := mock.GetInstance()

	if !vfs.IsDirectory(path) {
		return nil, ErrPathNotFound
	}

	files, err := vfs.ListDirectory(path)
	if err != nil {
		return nil, err
	}

	var dirEntries []DirectoryEntry
	for _, f := range files {
		dirEntries = append(dirEntries, DirectoryEntry{
			Name:  f.Name,
			Path:  f.Path,
			IsDir: f.Type == mock.FileTypeDirectory,
		})
	}

	parent := filepath.Dir(path)
	if parent == path || parent == "/" {
		parent = ""
	}

	return &BrowseResult{
		Path:    path,
		Parent:  parent,
		Entries: dirEntries,
	}, nil
}

// browseRoot returns the root browsing result (drives on Windows, / on Unix)
func (s *Service) browseRoot() (*BrowseResult, error) {
	if runtime.GOOS == osWindows {
		drives := s.listDrives()
		return &BrowseResult{
			Path:   "",
			Drives: drives,
		}, nil
	}

	// Unix - browse /
	return s.BrowseDirectory("/")
}

// validatePath validates and cleans a filesystem path
func (s *Service) validatePath(path string) (string, error) {
	// Clean the path to resolve . and ..
	cleaned := filepath.Clean(path)

	// Convert to absolute path
	absPath, err := filepath.Abs(cleaned)
	if err != nil {
		return "", ErrInvalidPath
	}

	// Validate path format based on OS
	if runtime.GOOS == osWindows {
		if !isValidWindowsPath(absPath) {
			return "", ErrInvalidPath
		}
	} else {
		if !strings.HasPrefix(absPath, "/") {
			return "", ErrInvalidPath
		}
	}

	// Verify path exists and is a directory
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrPathNotFound
		}
		if os.IsPermission(err) {
			return "", ErrAccessDenied
		}
		return "", ErrInvalidPath
	}

	if !info.IsDir() {
		return "", ErrNotDirectory
	}

	return absPath, nil
}

// isValidWindowsPath checks if a path is a valid Windows path
func isValidWindowsPath(path string) bool {
	if len(path) < 3 {
		return false
	}

	if isValidWindowsDrivePath(path) {
		return true
	}

	if isValidWindowsUNCPath(path) {
		return true
	}

	return false
}

func isValidWindowsDrivePath(path string) bool {
	if len(path) < 3 || path[1] != ':' {
		return false
	}
	if path[2] != '\\' && path[2] != '/' {
		return false
	}
	letter := path[0]
	return (letter >= 'A' && letter <= 'Z') || (letter >= 'a' && letter <= 'z')
}

func isValidWindowsUNCPath(path string) bool {
	if !strings.HasPrefix(path, "\\\\") {
		return false
	}
	parts := strings.Split(path[2:], "\\")
	return len(parts) >= 2 && parts[0] != "" && parts[1] != ""
}

// GetStorageInfo returns storage information for all drives/volumes
func (s *Service) GetStorageInfo(rootFolders []RootFolderRef) ([]StorageInfo, error) {
	var storage []StorageInfo

	if runtime.GOOS == osWindows {
		storage = s.getWindowsStorageInfo(rootFolders)
	} else {
		storage = s.getUnixStorageInfo(rootFolders)
	}

	return storage, nil
}

// getWindowsStorageInfo aggregates storage info for Windows drives
func (s *Service) getWindowsStorageInfo(rootFolders []RootFolderRef) []StorageInfo {
	var storage []StorageInfo
	drives := s.listDrives()

	for _, drive := range drives {
		// Calculate used space and percentage
		var usedSpace int64
		var usedPercent float64
		if drive.TotalSpace > 0 {
			usedSpace = drive.TotalSpace - drive.FreeSpace
			usedPercent = float64(usedSpace) / float64(drive.TotalSpace) * 100
		}

		// Find root folders on this drive
		var driveRootFolders []RootFolderRef
		for _, rf := range rootFolders {
			if strings.HasPrefix(strings.ToUpper(rf.Name), drive.Letter) {
				// This is a simplified check - in reality we'd need the full path
				driveRootFolders = append(driveRootFolders, rf)
			}
		}

		// Determine label (use drive letter if no label)
		label := drive.Letter
		if drive.Label != "" {
			label = drive.Label
		}

		storage = append(storage, StorageInfo{
			Label:       label,
			Path:        drive.Letter,
			FreeSpace:   drive.FreeSpace,
			TotalSpace:  drive.TotalSpace,
			UsedSpace:   usedSpace,
			UsedPercent: usedPercent,
			Type:        drive.Type,
			RootFolders: driveRootFolders,
		})
	}

	return storage
}

// getUnixStorageInfo aggregates storage info for Unix volumes
func (s *Service) getUnixStorageInfo(rootFolders []RootFolderRef) []StorageInfo {
	var storage []StorageInfo
	volumes := s.ListVolumes()

	for _, volume := range volumes {
		// Calculate used space and percentage
		var usedSpace int64
		var usedPercent float64
		if volume.TotalSpace > 0 {
			usedSpace = volume.TotalSpace - volume.FreeSpace
			usedPercent = float64(usedSpace) / float64(volume.TotalSpace) * 100
		}

		// Find root folders on this volume
		var volumeRootFolders []RootFolderRef
		for _, rf := range rootFolders {
			// This is a simplified check - in reality we'd need the full path
			if strings.HasPrefix(rf.Name, volume.MountPoint) {
				volumeRootFolders = append(volumeRootFolders, rf)
			}
		}

		// Use mount point as label if no better label available
		label := volume.MountPoint
		if label == "/" {
			label = "Root"
		} else {
			// Extract just the last component for cleaner display
			label = filepath.Base(volume.MountPoint)
		}

		storage = append(storage, StorageInfo{
			Label:       label,
			Path:        volume.MountPoint,
			FreeSpace:   volume.FreeSpace,
			TotalSpace:  volume.TotalSpace,
			UsedSpace:   usedSpace,
			UsedPercent: usedPercent,
			Type:        volume.Type,
			RootFolders: volumeRootFolders,
		})
	}

	return storage
}

// isWindowsSystemDir checks if a directory name is a Windows system directory to hide
func isWindowsSystemDir(name string) bool {
	systemDirs := map[string]bool{
		"$Recycle.Bin":              true,
		"System Volume Information": true,
		"$WinREAgent":               true,
		"Config.Msi":                true,
	}
	return systemDirs[name]
}

// BrowseForImport lists directories and video files at the given path.
// This is used for manual import functionality.
func (s *Service) BrowseForImport(path string) (*ImportBrowseResult, error) {
	if path == "" {
		return s.browseRootForImport()
	}

	if mock.IsMockPath(path) {
		return s.browseMockForImport(path)
	}

	cleanPath, err := s.validatePath(path)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(cleanPath)
	if err != nil {
		if os.IsPermission(err) {
			return nil, ErrAccessDenied
		}
		return nil, ErrPathNotFound
	}

	dirEntries, fileEntries := s.processImportEntries(entries, cleanPath)

	parent := filepath.Dir(cleanPath)
	if parent == cleanPath {
		parent = ""
	}

	return &ImportBrowseResult{
		Path:        cleanPath,
		Parent:      parent,
		Directories: dirEntries,
		Files:       fileEntries,
	}, nil
}

func (s *Service) processImportEntries(entries []os.DirEntry, cleanPath string) ([]DirectoryEntry, []FileEntry) {
	var dirEntries []DirectoryEntry
	var fileEntries []FileEntry

	for _, entry := range entries {
		name := entry.Name()

		if s.shouldSkipImportEntry(name) {
			continue
		}

		if entry.IsDir() {
			if !isSampleDirectory(name) {
				dirEntries = append(dirEntries, DirectoryEntry{
					Name:  name,
					Path:  filepath.Join(cleanPath, name),
					IsDir: true,
				})
			}
		} else {
			if fileEntry, ok := s.buildFileEntry(entry, name, cleanPath); ok {
				fileEntries = append(fileEntries, fileEntry)
			}
		}
	}

	sort.Slice(dirEntries, func(i, j int) bool {
		return strings.ToLower(dirEntries[i].Name) < strings.ToLower(dirEntries[j].Name)
	})
	sort.Slice(fileEntries, func(i, j int) bool {
		return strings.ToLower(fileEntries[i].Name) < strings.ToLower(fileEntries[j].Name)
	})

	return dirEntries, fileEntries
}

func (s *Service) shouldSkipImportEntry(name string) bool {
	if runtime.GOOS != osWindows && strings.HasPrefix(name, ".") {
		return true
	}
	if runtime.GOOS == osWindows && isWindowsSystemDir(name) {
		return true
	}
	return false
}

func isSampleDirectory(name string) bool {
	return strings.EqualFold(name, "sample") || strings.EqualFold(name, "samples")
}

func (s *Service) buildFileEntry(entry os.DirEntry, name, cleanPath string) (FileEntry, bool) {
	ext := strings.ToLower(filepath.Ext(name))
	if !isVideoExtension(ext) {
		return FileEntry{}, false
	}

	info, err := entry.Info()
	if err != nil {
		return FileEntry{}, false
	}

	return FileEntry{
		Name:    name,
		Path:    filepath.Join(cleanPath, name),
		Size:    info.Size(),
		ModTime: info.ModTime().Unix(),
	}, true
}

func (s *Service) browseMockForImport(path string) (*ImportBrowseResult, error) {
	vfs := mock.GetInstance()

	if !vfs.IsDirectory(path) {
		return nil, ErrPathNotFound
	}

	files, err := vfs.ListDirectory(path)
	if err != nil {
		return nil, err
	}

	var dirEntries []DirectoryEntry
	var fileEntries []FileEntry

	for _, f := range files {
		name := f.Name

		// Skip sample directories
		if f.Type == mock.FileTypeDirectory {
			if strings.EqualFold(name, "sample") || strings.EqualFold(name, "samples") {
				continue
			}
			dirEntries = append(dirEntries, DirectoryEntry{
				Name:  name,
				Path:  f.Path,
				IsDir: true,
			})
		} else {
			ext := strings.ToLower(filepath.Ext(name))
			if isVideoExtension(ext) {
				fileEntries = append(fileEntries, FileEntry{
					Name:    name,
					Path:    f.Path,
					Size:    f.Size,
					ModTime: f.ModTime.Unix(),
				})
			}
		}
	}

	parent := filepath.Dir(path)
	if parent == path || parent == "/" {
		parent = ""
	}

	return &ImportBrowseResult{
		Path:        path,
		Parent:      parent,
		Directories: dirEntries,
		Files:       fileEntries,
	}, nil
}

// browseRootForImport returns root for import browsing
func (s *Service) browseRootForImport() (*ImportBrowseResult, error) {
	if runtime.GOOS == osWindows {
		drives := s.listDrives()
		return &ImportBrowseResult{
			Path:   "",
			Drives: drives,
		}, nil
	}

	result, err := s.BrowseForImport("/")
	if err != nil {
		return nil, err
	}
	return result, nil
}

// isVideoExtension checks if the extension is a supported video format
func isVideoExtension(ext string) bool {
	videoExts := map[string]bool{
		".mkv":  true,
		".mp4":  true,
		".avi":  true,
		".m4v":  true,
		".mov":  true,
		".wmv":  true,
		".ts":   true,
		".m2ts": true,
		".webm": true,
		".flv":  true,
		".ogm":  true,
		".divx": true,
		".vob":  true,
	}
	return videoExts[ext]
}

// samplePatterns are used to detect sample files
var sampleDirNames = map[string]bool{
	"sample":  true,
	"samples": true,
}

// isSamplePath checks if a path is a sample file based on folder or filename
func isSamplePath(path string) bool {
	lower := strings.ToLower(path)

	// Check for sample in directory path
	parts := strings.Split(lower, string(filepath.Separator))
	for _, part := range parts {
		if sampleDirNames[part] {
			return true
		}
	}

	// Check for sample in filename
	base := strings.ToLower(filepath.Base(path))
	return strings.Contains(base, "sample")
}

// ScanForMedia recursively scans a directory for video files and parses their metadata.
// This is used for manual import functionality.
func (s *Service) ScanForMedia(path string, parser func(string) *ParsedInfo) (*MediaScanResult, error) {
	if mock.IsMockPath(path) {
		return s.scanMockForMedia(path, parser)
	}

	cleanPath, err := s.validatePath(path)
	if err != nil {
		return nil, err
	}

	result := &MediaScanResult{
		Path:  cleanPath,
		Files: make([]ScannedMediaFile, 0),
	}

	err = filepath.Walk(cleanPath, func(filePath string, info os.FileInfo, walkErr error) error {
		return s.processScanEntry(filePath, info, walkErr, parser, result)
	})

	if err != nil {
		return nil, err
	}

	result.TotalFiles = len(result.Files)

	sort.Slice(result.Files, func(i, j int) bool {
		return strings.ToLower(result.Files[i].Path) < strings.ToLower(result.Files[j].Path)
	})

	return result, nil
}

func (s *Service) processScanEntry(filePath string, info os.FileInfo, walkErr error, parser func(string) *ParsedInfo, result *MediaScanResult) error {
	if walkErr != nil {
		return nil //nolint:nilerr // Skip filesystem errors during walk
	}

	if info.IsDir() {
		return s.handleScanDirectory(info)
	}

	if scanned, ok := s.buildScannedFile(filePath, info, parser); ok {
		result.Files = append(result.Files, scanned)
	}

	return nil
}

func (s *Service) handleScanDirectory(info os.FileInfo) error {
	baseName := strings.ToLower(info.Name())
	if sampleDirNames[baseName] {
		return filepath.SkipDir
	}
	if runtime.GOOS != osWindows && strings.HasPrefix(info.Name(), ".") {
		return filepath.SkipDir
	}
	return nil
}

func (s *Service) buildScannedFile(filePath string, info os.FileInfo, parser func(string) *ParsedInfo) (ScannedMediaFile, bool) {
	ext := strings.ToLower(filepath.Ext(filePath))
	if !isVideoExtension(ext) {
		return ScannedMediaFile{}, false
	}

	scanned := ScannedMediaFile{
		Path:      filePath,
		Name:      info.Name(),
		Size:      info.Size(),
		ModTime:   info.ModTime().Unix(),
		Extension: ext,
		IsSample:  isSamplePath(filePath),
	}

	if parser != nil {
		scanned.Parsed = parser(info.Name())
	}

	return scanned, true
}

func (s *Service) scanMockForMedia(path string, parser func(string) *ParsedInfo) (*MediaScanResult, error) {
	vfs := mock.GetInstance()

	if !vfs.IsDirectory(path) {
		return nil, ErrPathNotFound
	}

	result := &MediaScanResult{
		Path:  path,
		Files: make([]ScannedMediaFile, 0),
	}

	err := vfs.WalkDir(path, func(filePath string, file *mock.VirtualFile) error {
		if file.Type == mock.FileTypeDirectory {
			baseName := strings.ToLower(file.Name)
			if sampleDirNames[baseName] {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(filePath))
		if !isVideoExtension(ext) {
			return nil
		}

		scanned := ScannedMediaFile{
			Path:      filePath,
			Name:      file.Name,
			Size:      file.Size,
			ModTime:   file.ModTime.Unix(),
			Extension: ext,
			IsSample:  isSamplePath(filePath),
		}

		if parser != nil {
			scanned.Parsed = parser(file.Name)
		}

		result.Files = append(result.Files, scanned)
		return nil
	})

	if err != nil {
		return nil, err
	}

	result.TotalFiles = len(result.Files)

	sort.Slice(result.Files, func(i, j int) bool {
		return strings.ToLower(result.Files[i].Path) < strings.ToLower(result.Files[j].Path)
	})

	return result, nil
}
