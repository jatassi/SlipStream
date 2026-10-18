package storing

import "strings"

// Mount is one line of a Unix mount table.
type Mount struct {
	// Device is the filesystem's device, "8:1" in /proc/self/mountinfo.
	Device string
	Point  string
	Type   string
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

// realMounts drops pseudo filesystems by type, and every mount of a device
// but its shortest mount point, so bind mounts list once.
func realMounts(ms []Mount) []Mount {
	best := map[string]int{}
	var kept []Mount
	for _, m := range ms {
		if pseudo[strings.ToLower(m.Type)] {
			continue
		}
		i, seen := best[m.Device]
		switch {
		case !seen:
			best[m.Device] = len(kept)
			kept = append(kept, m)
		case len(m.Point) < len(kept[i].Point):
			kept[i] = m
		}
	}
	return kept
}
