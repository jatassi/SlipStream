package host

import "testing"

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
