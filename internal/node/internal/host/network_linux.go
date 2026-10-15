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
	return networkMagic(st.Type), nil
}

// networkMagic is NFS, SMB, CIFS or SMB2 (which SMB3 shares) by its
// statfs f_type.
func networkMagic(t int64) bool {
	switch t {
	case unix.NFS_SUPER_MAGIC, unix.SMB_SUPER_MAGIC, unix.CIFS_SUPER_MAGIC, unix.SMB2_SUPER_MAGIC:
		return true
	}
	return false
}
