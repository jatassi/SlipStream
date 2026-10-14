//go:build unix

package fsinfo

import (
	"fmt"
	"os"
	"syscall"
)

func links(_ *os.File, fi os.FileInfo) (uint64, error) {
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, fmt.Errorf("%T is not a Stat_t", fi.Sys())
	}
	return widen(st.Nlink), nil
}

// widen takes Nlink's type, which is uint64 on linux-amd64 and narrower
// elsewhere, so the conversion is never redundant on one target only.
func widen[T uint16 | uint32 | uint64](n T) uint64 { return uint64(n) }
