package fsinfo

import (
	"fmt"
	"os"
	"syscall"
)

func links(f *os.File, _ os.FileInfo) (uint64, error) {
	var info syscall.ByHandleFileInformation
	if err := syscall.GetFileInformationByHandle(syscall.Handle(f.Fd()), &info); err != nil {
		return 0, fmt.Errorf("file information: %w", err)
	}
	return uint64(info.NumberOfLinks), nil
}
