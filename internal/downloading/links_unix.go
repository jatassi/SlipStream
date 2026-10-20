//go:build unix

package downloading

import (
	"fmt"
	"os"
	"syscall"
)

// links is how many names the file at path has.
func links(path string) (uint64, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return 0, err //nolint:wrapcheck // the path is in the error
	}
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, fmt.Errorf("%T is not a Stat_t", fi.Sys())
	}
	return widen(st.Nlink), nil
}

// widen takes Nlink's type, which is uint64 on linux-amd64 and narrower
// elsewhere, so the conversion is never redundant on one target only.
func widen[T uint16 | uint32 | uint64](n T) uint64 { return uint64(n) }
