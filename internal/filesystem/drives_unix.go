//go:build !windows

package filesystem

// listDrives returns nil on Unix systems (no drive concept)
func (s *Service) listDrives() []DriveInfo {
	return nil
}
