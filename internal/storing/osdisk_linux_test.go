package storing

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestNetworkMagic(t *testing.T) {
	for magic, want := range map[int64]bool{
		0x6969:     true,  // nfs
		0x517b:     true,  // smb
		0xff534d42: true,  // cifs
		0xfe534d42: true,  // smb2 and smb3
		0xef53:     false, // ext4
		0x9123683e: false, // btrfs
		0x01021994: false, // tmpfs
	} {
		if got := networkMagic(magic); got != want {
			t.Errorf("networkMagic(%#x) = %v, want %v", magic, got, want)
		}
	}
}

func TestStableMagic(t *testing.T) {
	for magic, want := range map[int64]bool{
		0xef53:     true,  // ext4
		0x9123683e: true,  // btrfs
		0x58465342: true,  // xfs
		0x01021994: true,  // tmpfs
		0x4d44:     false, // vfat
		0x2011bab0: false, // exfat
		0x6969:     false, // nfs
		0xff534d42: false, // cifs
		0xfe534d42: false, // smb2 and smb3
	} {
		if got := stableMagic(magic); got != want {
			t.Errorf("stableMagic(%#x) = %v, want %v", magic, got, want)
		}
	}
}

func TestOSDiskVolumeMountHoldsTheFolder(t *testing.T) {
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	v, err := newOSDisk(t).Volume(t.Context(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if !underUnix(dir, v.Mount) || v.Name != v.Mount || !strings.HasPrefix(v.Mount, "/") {
		t.Errorf("Volume(%s) mount %q name %q", dir, v.Mount, v.Name)
	}
}
