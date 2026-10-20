package downloading

import (
	"fmt"
	"os"
	"syscall"
)

// links is how many names the file at path has.
func links(path string) (uint64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err //nolint:wrapcheck // the path is in the error
	}
	defer func() { _ = f.Close() }()
	var info syscall.ByHandleFileInformation
	if err := syscall.GetFileInformationByHandle(syscall.Handle(f.Fd()), &info); err != nil {
		return 0, fmt.Errorf("file information of %s: %w", path, err)
	}
	return uint64(info.NumberOfLinks), nil
}
