package storing

import "testing"

func TestNetworkFSName(t *testing.T) {
	for name, want := range map[string]bool{"nfs": true, "smbfs": true, "afpfs": true, "webdav": true, "apfs": false, "hfs": false} {
		if got := networkFSName(name); got != want {
			t.Errorf("networkFSName(%q) = %v, want %v", name, got, want)
		}
	}
}
