package host

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"golang.org/x/sys/unix"

	"github.com/jatassi/slipstream/internal/install"
)

var (
	glibcFloor  = mustVersion(install.GlibcFloor)
	kernelFloor = mustVersion(install.KernelFloor)
)

func mustVersion(s string) version {
	v, err := parseVersion(s)
	if err != nil {
		panic("host: " + err.Error())
	}
	return v
}

// BelowFloor names each part of the host older than the floor, or that it
// cannot read. The binary is static, so glibc's version comes from the
// host's getconf.
func BelowFloor(ctx context.Context) []string {
	var below []string
	var uts unix.Utsname
	if err := unix.Uname(&uts); err != nil {
		below = append(below, fmt.Sprintf("cannot read the kernel version: %v", err))
	} else if msg := belowFloor("kernel", unix.ByteSliceToString(uts.Release[:]), kernelFloor); msg != "" {
		below = append(below, msg)
	}
	out, err := exec.CommandContext(ctx, "getconf", "GNU_LIBC_VERSION").Output()
	if err != nil {
		return append(below, fmt.Sprintf("cannot read the glibc version with getconf GNU_LIBC_VERSION: %v", err))
	}
	if msg := belowFloor("glibc", strings.TrimPrefix(strings.TrimSpace(string(out)), "glibc "), glibcFloor); msg != "" {
		below = append(below, msg)
	}
	return below
}

func belowFloor(what, release string, floor version) string {
	v, err := parseVersion(release)
	switch {
	case err != nil:
		return fmt.Sprintf("cannot read the %s version: %v", what, err)
	case v.less(floor):
		return fmt.Sprintf("%s %s is older than %s, the oldest SlipStream supports", what, release, floor)
	}
	return ""
}
