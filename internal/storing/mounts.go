package storing

import (
	"bufio"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
)

// Mount is one line of a Unix mount table.
type Mount struct {
	// Device is the filesystem's device, "8:1" in /proc/self/mountinfo.
	Device string
	// Root is the folder of the filesystem mounted at Point: "/", or the
	// folder a bind mount shows.
	Root  string
	Point string
	Type  string
}

// pseudo are filesystem types that hold no media: kernel views, memory,
// containers' layers, and desktop helpers.
var pseudo = map[string]bool{
	"autofs": true, "binfmt_misc": true, "bpf": true, "cgroup": true, "cgroup2": true, "configfs": true,
	"debugfs": true, "devfs": true, "devpts": true, "devtmpfs": true, "efivarfs": true, "fuse.gvfsd-fuse": true,
	"fuse.lxcfs": true, "fuse.portal": true, "fusectl": true, "hugetlbfs": true, "mqueue": true, "nsfs": true,
	"overlay": true, "proc": true, "pstore": true, "ramfs": true, "rootfs": true, "rpc_pipefs": true,
	"securityfs": true, "selinuxfs": true, "squashfs": true, "sysfs": true, "tmpfs": true, "tracefs": true,
}

// realMounts drops pseudo filesystems by type, and every mount whose
// folders another mount of its device already shows, keeping the shortest
// mount point, so a bind mount of a folder already listed lists once while
// a container's bind mounts of different host folders each list.
func realMounts(ms []Mount) []Mount {
	var fss []Mount
	for _, m := range ms {
		if !pseudo[strings.ToLower(m.Type)] {
			fss = append(fss, m)
		}
	}
	shown := func(i int) bool {
		m := fss[i]
		for j, n := range fss {
			if j == i || n.Device != m.Device || !underUnix(m.Root, n.Root) {
				continue
			}
			if n.Root != m.Root || len(n.Point) < len(m.Point) || len(n.Point) == len(m.Point) && j < i {
				return true
			}
		}
		return false
	}
	var kept []Mount
	for i, m := range fss {
		if !shown(i) {
			kept = append(kept, m)
		}
	}
	return kept
}

// underUnix is a Unix path equal to or under dir.
func underUnix(path, dir string) bool {
	return dir == "/" || path == dir || strings.HasPrefix(path, dir+"/")
}

// mountOf is the mount holding path, a real path on device: the longest
// mount point above it, preferring one of its device since btrfs
// subvolumes stat as devices of their own. A later line over-mounts an
// earlier one at the same point.
func mountOf(ms []Mount, device, path string) (Mount, bool) {
	best := -1
	better := func(i int) bool {
		if best < 0 {
			return true
		}
		mine, theirs := ms[i].Device == device, ms[best].Device == device
		if mine != theirs {
			return mine
		}
		return len(ms[i].Point) >= len(ms[best].Point)
	}
	for i, m := range ms {
		if underUnix(path, m.Point) && better(i) {
			best = i
		}
	}
	if best < 0 {
		return Mount{}, false
	}
	return ms[best], true
}

// parseMountinfo reads /proc/self/mountinfo: "36 35 98:0 /mnt1 /mnt2 rw
// master:1 - ext3 /dev/root rw", the optional fields ending at "-", with
// spaces and other bytes in paths written as octal escapes such as \040.
func parseMountinfo(r io.Reader) ([]Mount, error) {
	var ms []Mount
	lines := bufio.NewScanner(r)
	for lines.Scan() {
		f := strings.Fields(lines.Text())
		sep := slices.Index(f, "-")
		if sep < 6 || sep+1 >= len(f) {
			return nil, fmt.Errorf("mountinfo line %q: no filesystem type", lines.Text())
		}
		ms = append(ms, Mount{Device: f[2], Root: unescapeOctal(f[3]), Point: unescapeOctal(f[4]), Type: f[sep+1]})
	}
	if err := lines.Err(); err != nil {
		return nil, fmt.Errorf("read mountinfo: %w", err)
	}
	return ms, nil
}

func unescapeOctal(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+4 <= len(s) {
			if n, err := strconv.ParseUint(s[i+1:i+4], 8, 8); err == nil {
				b.WriteByte(byte(n))
				i += 3
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
