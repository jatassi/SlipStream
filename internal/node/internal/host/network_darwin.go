package host

import (
	"fmt"

	"golang.org/x/sys/unix"
)

func networkFS(dir string) (bool, error) {
	var st unix.Statfs_t
	if err := unix.Statfs(dir, &st); err != nil {
		return false, fmt.Errorf("statfs %s: %w", dir, err)
	}
	return networkFSName(unix.ByteSliceToString(st.Fstypename[:])), nil
}

func networkFSName(name string) bool { return name == "nfs" || name == "smbfs" }
