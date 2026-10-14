package e2e

import (
	"os"
	"syscall"
	"testing"
)

func linkCount(t *testing.T, path string) uint64 {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	var info syscall.ByHandleFileInformation
	if err := syscall.GetFileInformationByHandle(syscall.Handle(f.Fd()), &info); err != nil {
		t.Fatal(err)
	}
	return uint64(info.NumberOfLinks)
}
