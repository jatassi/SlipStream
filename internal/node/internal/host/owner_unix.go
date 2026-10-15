//go:build unix

package host

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"strconv"
	"syscall"
)

// ForeignOwner is dir's owner when dir exists and another user owns it,
// else nil.
func ForeignOwner(dir string) (*Owner, error) {
	info, err := os.Stat(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("data dir %s: %w", dir, err)
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return nil, fmt.Errorf("data dir %s: stat returned %T", dir, info.Sys())
	}
	uid := int(st.Uid)
	if uid == os.Getuid() {
		return nil, nil
	}
	name := strconv.Itoa(uid)
	if u, err := user.LookupId(name); err == nil {
		name = u.Username
	}
	return &Owner{UID: uid, Name: name}, nil
}
