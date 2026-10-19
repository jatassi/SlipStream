package storing

import (
	"slices"
	"strings"
	"testing"
)

func TestParseMountinfo(t *testing.T) {
	got, err := parseMountinfo(strings.NewReader(strings.Join([]string{
		`28 1 254:0 / / rw,relatime - ext4 /dev/vda rw`,
		`36 28 98:0 /mnt1 /mnt2 rw,noatime master:1 - ext3 /dev/root rw,errors=continue`,
		`40 28 0:50 / /mnt/My\040Media rw shared:7 master:2 - cifs //nas/My\040Media rw`,
		`41 28 8:17 /back\134slash /mnt/tab\011and\012line rw - xfs /dev/sdb1 rw`,
		`42 28 8:18 / /mnt/not\08octal rw - xfs /dev/sdb2 rw`,
	}, "\n")))
	if err != nil {
		t.Fatal(err)
	}
	want := []Mount{
		{"254:0", "/", "/", "ext4"},
		{"98:0", "/mnt1", "/mnt2", "ext3"},
		{"0:50", "/", "/mnt/My Media", "cifs"},
		{"8:17", `/back\slash`, "/mnt/tab\tand\nline", "xfs"},
		{"8:18", "/", `/mnt/not\08octal`, "xfs"},
	}
	if !slices.Equal(got, want) {
		t.Errorf("parseMountinfo =\n%q\nwant\n%q", got, want)
	}
	if _, err := parseMountinfo(strings.NewReader("36 35 98:0 /mnt1 /mnt2 rw")); err == nil {
		t.Error("a line with no separator parsed")
	}
}

func TestMountOfIsTheLongestMountAbove(t *testing.T) {
	ms := []Mount{
		{"8:1", "/", "/", "ext4"},
		{"8:17", "/", "/mnt/media", "xfs"},
		{"8:18", "/", "/mnt/media-2", "xfs"},
		{"0:50", "/", "/mnt/media/nas", "cifs"},
		{"0:51", "/", "/mnt/media/nas", "nfs"},
		{"0:40", "/", "/data", "btrfs"},
	}
	for _, c := range []struct{ device, path, want string }{
		{"8:1", "/home/ana/Movies", "/"},
		{"8:17", "/mnt/media", "/mnt/media"},
		{"8:17", "/mnt/media/Movies", "/mnt/media"},
		{"8:18", "/mnt/media-2/TV", "/mnt/media-2"},
		{"0:51", "/mnt/media/nas/TV", "/mnt/media/nas"},
		{"0:44", "/data/subvolume/Movies", "/data"},
		{"8:17", "/mnt/media/nas/TV", "/mnt/media"},
	} {
		m, ok := mountOf(ms, c.device, c.path)
		if !ok || m.Point != c.want {
			t.Errorf("mountOf(%s, %s) = %v, %v; want %s", c.device, c.path, m, ok, c.want)
		}
	}
	if m, _ := mountOf(ms, "0:51", "/mnt/media/nas"); m.Type != "nfs" {
		t.Errorf("over-mounted point: %v, want the later nfs line", m)
	}
}
