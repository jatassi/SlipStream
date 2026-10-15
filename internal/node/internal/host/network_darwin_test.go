package host

import "testing"

func TestNetworkFSName(t *testing.T) {
	for name, want := range map[string]bool{"nfs": true, "smbfs": true, "apfs": false, "hfs": false} {
		if got := networkFSName(name); got != want {
			t.Errorf("networkFSName(%q) = %v, want %v", name, got, want)
		}
	}
}
