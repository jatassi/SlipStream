package host

import (
	"fmt"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
)

func networkFS(dir string) (bool, error) {
	if isUNC(dir) {
		return true, nil
	}
	root, err := windows.UTF16PtrFromString(filepath.VolumeName(dir) + `\`)
	if err != nil {
		return false, fmt.Errorf("data dir %s: %w", dir, err)
	}
	return windows.GetDriveType(root) == windows.DRIVE_REMOTE, nil
}

// isUNC is a \\server\share path, including the \\?\UNC\ long form.
func isUNC(p string) bool {
	vol := filepath.VolumeName(p)
	return strings.HasPrefix(vol, `\\`) && !strings.HasPrefix(vol, `\\?\`) || strings.HasPrefix(strings.ToUpper(p), `\\?\UNC\`)
}
