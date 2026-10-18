package storing

import (
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/storage"
)

func TestRealMountsDropPseudoFilesystems(t *testing.T) {
	got := realMounts([]Mount{
		{"0:22", "/proc", "proc"},
		{"0:23", "/sys", "sysfs"},
		{"8:1", "/", "ext4"},
		{"0:5", "/dev", "devtmpfs"},
		{"0:30", "/run", "tmpfs"},
		{"0:31", "/sys/fs/cgroup", "cgroup2"},
		{"0:40", "/var/lib/docker/overlay2/x/merged", "overlay"},
		{"7:0", "/snap/core/1", "squashfs"},
		{"8:1", "/etc/hosts", "ext4"},
		{"8:17", "/mnt/media/a/bind", "xfs"},
		{"8:17", "/mnt/media", "xfs"},
		{"0:50", "/mnt/nas", "cifs"},
		{"0:51", "/mnt/pool", "fuse.mergerfs"},
		{"0:52", "/run/user/1000/doc", "fuse.portal"},
	})
	want := []Mount{
		{"8:1", "/", "ext4"},
		{"8:17", "/mnt/media", "xfs"},
		{"0:50", "/mnt/nas", "cifs"},
		{"0:51", "/mnt/pool", "fuse.mergerfs"},
	}
	if !slices.Equal(got, want) {
		t.Errorf("realMounts = %v, want %v", got, want)
	}
}

func TestHiddenRule(t *testing.T) {
	for _, c := range []struct {
		style PathStyle
		name  string
		attrs uint32
		want  bool
	}{
		{Unix, ".config", 0, true},
		{Unix, "Movies", 0, false},
		{Unix, "@eaDir", 0, true},
		{Unix, "#recycle", 0, true},
		{Unix, "Movies", attrHidden, false},
		{Windows, "$RECYCLE.BIN", 0, true},
		{Windows, "$Recycle.Bin", 0, true},
		{Windows, "System Volume Information", 0, true},
		{Windows, ".git", 0, false},
		{Windows, "ProgramData", attrHidden, true},
		{Windows, "pagefile", attrSystem, true},
		{Windows, "Movies", 0x10, false},
	} {
		if got := hidden(c.style, c.name, c.attrs); got != c.want {
			t.Errorf("style %d hidden(%q, %#x) = %v, want %v", c.style, c.name, c.attrs, got, c.want)
		}
	}
}

func TestNaturalOrder(t *testing.T) {
	want := []string{
		"a", "B", "Season 1", "season 2", "Season 02x", "Season 10", "Season 100",
		"Show", "show 1", "Show 9", "Show 10", "x99999999999999999999", "x100000000000000000000", "Émile",
	}
	got := slices.Clone(want)
	slices.Reverse(got)
	slices.SortFunc(got, func(a, b string) int {
		switch {
		case naturalLess(a, b):
			return -1
		case naturalLess(b, a):
			return 1
		}
		return 0
	})
	if !slices.Equal(got, want) {
		t.Errorf("natural order = %q, want %q", got, want)
	}
	for _, p := range [][2]string{{"Movies", "movies"}, {"01", "1"}} {
		if naturalLess(p[0], p[1]) == naturalLess(p[1], p[0]) {
			t.Errorf("naturalLess(%q, %q) doesn't order them", p[0], p[1])
		}
	}
}

func TestVolumesGroupRootsOnOneFilesystem(t *testing.T) {
	a, b, c := storage.NewUUID(), storage.NewUUID(), storage.NewUUID()
	early, late := time.UnixMilli(1000), time.UnixMilli(2000)
	vols, of := volumesOf([]rootSample{
		{a, Volume{ID: "fs-2", Mount: "/mnt/media", Free: 10, SampledAt: early}},
		{c, Volume{ID: "fs-1", Mount: "/", Free: 5, SampledAt: early}},
		{b, Volume{ID: "fs-2", Mount: "/mnt/media", Free: 8, SampledAt: late}},
	})
	want := []Volume{
		{ID: "fs-1", Mount: "/", Free: 5, SampledAt: early},
		{ID: "fs-2", Mount: "/mnt/media", Free: 8, SampledAt: late},
	}
	if !slices.Equal(vols, want) {
		t.Errorf("volumes = %v, want %v", vols, want)
	}
	if of[a] != "fs-2" || of[b] != "fs-2" || of[c] != "fs-1" || of[storage.NewUUID()] != "" {
		t.Errorf("root volumes = %v", of)
	}
}
